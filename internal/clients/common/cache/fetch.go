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
	"strconv"

	"golang.org/x/sync/singleflight"
)

// flights coalesces concurrent misses on the same key and generation.
var flights singleflight.Group

// Fetch returns the value cached under key in store, or calls fetch to
// obtain it and caches the result.
//
// Concurrent misses on the same key share a single call to fetch. Errors are
// never cached, and neither are results obtained while ctx was cancelled.
// With a no-op store, Fetch simply calls fetch.
//
// The returned value is shared with other callers and must be treated as
// read-only, see the package documentation.
func Fetch[T any](ctx context.Context, store Store, key Key, fetch func(context.Context) (T, error)) (T, error) {
	if !IsEnabled(store) {
		return fetch(ctx)
	}

	cached, generation, found := store.Get(key)
	if found {
		if value, ok := cached.(T); ok {
			requestsTotal.WithLabelValues(key.Namespace, resultHit).Inc()

			return value, nil
		}
	}

	// The generation is part of the flight key: a caller arriving after an
	// invalidation must not join a fetch started before it.
	flightKey := key.String() + keySeparator + strconv.FormatUint(generation, 10)
	leader := false

	shared, err, _ := flights.Do(flightKey, func() (any, error) {
		leader = true

		value, fetchErr := fetch(ctx)
		if fetchErr != nil {
			return nil, fetchErr
		}

		if ctx.Err() == nil {
			store.Set(key, value, generation)
		}

		return value, nil
	})

	if !leader && failedOnLeaderContext(ctx, err) {
		requestsTotal.WithLabelValues(key.Namespace, resultMiss).Inc()

		return fetch(ctx)
	}

	result := resultCoalesced
	if leader {
		result = resultMiss
	}

	requestsTotal.WithLabelValues(key.Namespace, result).Inc()

	value, _ := shared.(T)

	return value, err
}

// failedOnLeaderContext reports whether err, returned by a shared fetch that
// ran with the leader's context, comes from that context ending while ctx is
// still live. Such an error is not the caller's to report, so it should fetch
// on its own behalf.
func failedOnLeaderContext(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() != nil {
		return false
	}

	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// FetchWithResponse is Fetch for SDK methods that also return an
// [*http.Response].
//
// The caller that actually performed the request gets its real response
// back. Every other successful caller (cache hit or coalesced request) gets
// HitResponse, so that callers can keep closing and inspecting the response
// as usual. A coalesced caller whose shared request failed gets a nil
// response.
func FetchWithResponse[T any](
	ctx context.Context,
	store Store,
	key Key,
	fetch func(context.Context) (T, *http.Response, error),
) (T, *http.Response, error) {
	// singleflight runs fetch in the goroutine of the caller that performs
	// it, so resp is only ever written by that caller.
	var resp *http.Response

	value, err := Fetch(ctx, store, key, func(ctx context.Context) (T, error) {
		fetched, fetchResp, fetchErr := fetch(ctx) //nolint:bodyclose // returned to the caller, which closes it
		resp = fetchResp

		return fetched, fetchErr
	})

	if resp == nil && err == nil {
		resp = HitResponse()
	}

	return value, resp, err
}

// HitResponse returns the synthetic response handed to callers served from
// the cache: a 200 OK with an empty body.
func HitResponse() *http.Response {
	return &http.Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
		Body:       http.NoBody,
	}
}
