/*
Copyright 2026 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cache

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// countingFetch returns a fetch function returning value and err, and a
// counter of its calls.
func countingFetch[T any](value T, err error) (func(context.Context) (T, error), *atomic.Int64) {
	calls := &atomic.Int64{}

	return func(context.Context) (T, error) {
		calls.Add(1)

		return value, err
	}, calls
}

// uniqueKey returns a key unique to the running test, so that tests do not
// share singleflight flights.
func uniqueKey(t *testing.T) Key {
	t.Helper()

	return Key{Scope: t.Name(), Namespace: "ns"}
}

// TestFetchHitMiss tests that Fetch only calls fetch on a miss.
func TestFetchHitMiss(t *testing.T) {
	t.Parallel()

	s := newTestStore(time.Minute, 10, newFakeClock())
	key := uniqueKey(t)
	fetch, calls := countingFetch("value", nil)

	for range 3 {
		got, err := Fetch(context.Background(), s, key, fetch)
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		if got != "value" {
			t.Errorf("Fetch() = %q, want %q", got, "value")
		}
	}

	if got := calls.Load(); got != 1 {
		t.Errorf("fetch called %d times, want 1", got)
	}

	s.Invalidate(key.Scope, key.Namespace)

	_, err := Fetch(context.Background(), s, key, fetch)
	if err != nil {
		t.Fatalf("Fetch() after Invalidate error = %v", err)
	}

	if got := calls.Load(); got != 2 {
		t.Errorf("fetch called %d times after Invalidate, want 2", got)
	}
}

// TestFetchDoesNotCacheErrors tests that failed fetches are not cached.
func TestFetchDoesNotCacheErrors(t *testing.T) {
	t.Parallel()

	s := newTestStore(time.Minute, 10, newFakeClock())
	key := uniqueKey(t)
	wantErr := errors.New("boom")
	fetch, calls := countingFetch("partial", wantErr)

	for range 2 {
		_, err := Fetch(context.Background(), s, key, fetch)
		if !errors.Is(err, wantErr) {
			t.Fatalf("Fetch() error = %v, want %v", err, wantErr)
		}
	}

	if got := calls.Load(); got != 2 {
		t.Errorf("fetch called %d times, want 2", got)
	}

	if got := s.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}
}

// TestFetchDoesNotCacheUnderCancelledContext tests that results fetched
// under a cancelled context are not cached.
func TestFetchDoesNotCacheUnderCancelledContext(t *testing.T) {
	t.Parallel()

	s := newTestStore(time.Minute, 10, newFakeClock())
	key := uniqueKey(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fetch, calls := countingFetch("value", nil)

	_, err := Fetch(ctx, s, key, fetch)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if got := s.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}

	_, err = Fetch(context.Background(), s, key, fetch)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if got := calls.Load(); got != 2 {
		t.Errorf("fetch called %d times, want 2", got)
	}
}

// TestFetchDoesNotStoreAcrossInvalidate tests that a fetch racing with an
// invalidation is not stored.
func TestFetchDoesNotStoreAcrossInvalidate(t *testing.T) {
	t.Parallel()

	s := newTestStore(time.Minute, 10, newFakeClock())
	key := uniqueKey(t)

	// The fetch races with a write: the write invalidates while the fetch
	// is in flight, so its (possibly stale) result must not be stored.
	_, err := Fetch(context.Background(), s, key, func(context.Context) (string, error) {
		s.Invalidate(key.Scope, key.Namespace)

		return "stale", nil
	})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if got := s.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}
}

// TestFetchCoalescesConcurrentMisses tests that concurrent misses on a key
// share a single fetch.
func TestFetchCoalescesConcurrentMisses(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	s := newTestStore(time.Minute, 10, newFakeClock())
	key := uniqueKey(t)
	release := make(chan struct{})
	calls := &atomic.Int64{}

	var started, finished sync.WaitGroup

	started.Add(goroutines)
	finished.Add(goroutines)

	errs := make(chan error, goroutines)

	for range goroutines {
		go func() {
			defer finished.Done()

			started.Done()

			got, err := Fetch(context.Background(), s, key, func(context.Context) (string, error) {
				calls.Add(1)
				<-release

				return "value", nil
			})
			if err == nil && got != "value" {
				err = errors.New("unexpected value " + got)
			}

			errs <- err
		}()
	}

	started.Wait()
	// Give every goroutine the time to reach the in-flight fetch.
	time.Sleep(100 * time.Millisecond)
	close(release)
	finished.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("Fetch() error = %v", err)
		}
	}

	if got := calls.Load(); got != 1 {
		t.Errorf("fetch called %d times, want 1", got)
	}
}

// TestFetchNoopStore tests that Fetch always calls fetch with a disabled
// store.
func TestFetchNoopStore(t *testing.T) {
	t.Parallel()

	fetch, calls := countingFetch("value", nil)

	for _, store := range []Store{NewNoopStore(), nil} {
		for range 2 {
			got, err := Fetch(context.Background(), store, uniqueKey(t), fetch)
			if err != nil || got != "value" {
				t.Fatalf("Fetch() = %q, %v, want %q, nil", got, err, "value")
			}
		}
	}

	if got := calls.Load(); got != 4 {
		t.Errorf("fetch called %d times, want 4", got)
	}
}

// TestFetchWithResponse tests the responses returned on a miss and on a hit.
func TestFetchWithResponse(t *testing.T) {
	t.Parallel()

	s := newTestStore(time.Minute, 10, newFakeClock())
	key := uniqueKey(t)
	realResp := &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
	calls := 0
	fetch := func(context.Context) (string, *http.Response, error) {
		calls++

		return "value", realResp, nil
	}

	got, resp, err := FetchWithResponse(context.Background(), s, key, fetch) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil || got != "value" {
		t.Fatalf("FetchWithResponse() = %q, %v, want %q, nil", got, err, "value")
	}

	if resp != realResp {
		t.Error("FetchWithResponse() on a miss did not return the real response")
	}

	got, resp, err = FetchWithResponse(context.Background(), s, key, fetch) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil || got != "value" {
		t.Fatalf("FetchWithResponse() = %q, %v, want %q, nil", got, err, "value")
	}

	if resp == nil || resp == realResp || resp.StatusCode != http.StatusOK {
		t.Errorf("FetchWithResponse() on a hit returned %v, want a synthetic 200 response", resp)
	}

	if calls != 1 {
		t.Errorf("fetch called %d times, want 1", calls)
	}
}

// TestFetchWithResponseError tests that the real response is returned along
// with an error.
func TestFetchWithResponseError(t *testing.T) {
	t.Parallel()

	s := newTestStore(time.Minute, 10, newFakeClock())
	notFound := &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}
	wantErr := errors.New("not found")

	_, resp, err := FetchWithResponse(context.Background(), s, uniqueKey(t), //nolint:bodyclose // closed via helpers.CloseBody
		func(context.Context) (string, *http.Response, error) {
			return "", notFound, wantErr
		})
	helpers.CloseBody(resp)

	if !errors.Is(err, wantErr) {
		t.Fatalf("FetchWithResponse() error = %v, want %v", err, wantErr)
	}

	if !common.IsResponseNotFound(resp) {
		t.Error("FetchWithResponse() did not return the real error response")
	}
}

// TestHitResponse tests that the synthetic response works with the response
// helpers.
func TestHitResponse(t *testing.T) {
	t.Parallel()

	resp := HitResponse() //nolint:bodyclose // closed via helpers.CloseBody below

	if resp.StatusCode != http.StatusOK {
		t.Errorf("HitResponse().StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if common.IsResponseNotFound(resp) {
		t.Error("IsResponseNotFound(HitResponse()) = true, want false")
	}

	// Closing must not panic and can be repeated, as callers may both defer
	// and explicitly close the response.
	helpers.CloseBody(resp)
	helpers.CloseBody(resp)

	other := HitResponse() //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(other)

	if other == resp {
		t.Error("HitResponse() returned a shared response")
	}
}

// TestFetchCoalescedRetriesOnLeaderCancellation checks that a coalesced
// caller whose own context is live does not inherit the cancellation error
// of the leader's context.
func TestFetchCoalescedRetriesOnLeaderCancellation(t *testing.T) {
	t.Parallel()

	s := newTestStore(time.Minute, 10, newFakeClock())
	key := uniqueKey(t)

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	started := make(chan struct{})

	leaderDone := make(chan error, 1)

	go func() {
		_, err := Fetch(leaderCtx, s, key, func(ctx context.Context) (int, error) {
			close(started)
			<-ctx.Done()

			return 0, ctx.Err()
		})
		leaderDone <- err
	}()

	<-started

	followerDone := make(chan struct{})

	var (
		value int
		err   error
	)

	go func() {
		defer close(followerDone)

		value, err = Fetch(context.Background(), s, key, func(context.Context) (int, error) {
			return 42, nil
		})
	}()

	// Give the follower time to join the in-flight call before cancelling.
	time.Sleep(50 * time.Millisecond)
	cancelLeader()

	leaderErr := <-leaderDone
	if !errors.Is(leaderErr, context.Canceled) {
		t.Fatalf("leader: want context.Canceled, got %v", leaderErr)
	}

	<-followerDone

	if err != nil || value != 42 {
		t.Fatalf("follower: want (42, nil), got (%d, %v)", value, err)
	}
}
