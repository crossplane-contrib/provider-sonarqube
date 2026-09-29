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
	"sync"
	"testing"
	"time"
)

// fakeClock is a manually advanced clock for ttlStore tests.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// newFakeClock returns a fakeClock set to an arbitrary fixed time.
func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)}
}

// Now returns the current fake time.
func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

// Advance moves the fake time forward by d.
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = c.now.Add(d)
}

// set stores value under key at the current generation of key.
func set(t *testing.T, s Store, key Key, value any) {
	t.Helper()

	_, generation, _ := s.Get(key)
	if !s.Set(key, value, generation) {
		t.Fatalf("Set(%v) was rejected", key)
	}
}

// assertValue checks that key holds want, or is missing when want is nil.
func assertValue(t *testing.T, s Store, key Key, want any) {
	t.Helper()

	got, _, found := s.Get(key)

	switch {
	case want == nil && found:
		t.Errorf("Get(%v) = %v, want miss", key, got)
	case want != nil && !found:
		t.Errorf("Get(%v) missed, want %v", key, want)
	case want != nil && got != want:
		t.Errorf("Get(%v) = %v, want %v", key, got, want)
	}
}

// TestTTLStoreHitMiss tests that a ttlStore serves stored values and misses
// on unknown keys.
func TestTTLStoreHitMiss(t *testing.T) {
	t.Parallel()

	s := newTTLStore(time.Minute, 10, newFakeClock().Now)
	key := Key{Scope: "scope", Namespace: "ns", Params: "a=1"}

	assertValue(t, s, key, nil)
	set(t, s, key, "value")
	assertValue(t, s, key, "value")
	assertValue(t, s, Key{Scope: "scope", Namespace: "ns", Params: "a=2"}, nil)
	assertValue(t, s, Key{Scope: "other", Namespace: "ns", Params: "a=1"}, nil)

	set(t, s, key, "replaced")
	assertValue(t, s, key, "replaced")

	if got := s.Len(); got != 1 {
		t.Errorf("Len() = %d, want 1", got)
	}
}

// TestTTLStoreExpiry tests that expired entries are evicted lazily on read.
func TestTTLStoreExpiry(t *testing.T) {
	t.Parallel()

	clock := newFakeClock()
	s := newTTLStore(10*time.Second, 10, clock.Now)
	key := Key{Scope: "scope", Namespace: "ns"}

	set(t, s, key, "value")

	clock.Advance(9 * time.Second)
	assertValue(t, s, key, "value")

	clock.Advance(time.Second)
	assertValue(t, s, key, nil)

	if got := s.Len(); got != 0 {
		t.Errorf("Len() after lazy eviction = %d, want 0", got)
	}
}

// TestTTLStoreSweep tests that Sweep evicts expired entries only.
func TestTTLStoreSweep(t *testing.T) {
	t.Parallel()

	clock := newFakeClock()
	s := newTTLStore(10*time.Second, 10, clock.Now)
	old := Key{Scope: "scope", Namespace: "ns", Params: "old"}
	recent := Key{Scope: "scope", Namespace: "ns", Params: "recent"}

	set(t, s, old, "old")
	clock.Advance(5 * time.Second)
	set(t, s, recent, "recent")
	clock.Advance(5 * time.Second)

	s.Sweep()

	if got := s.Len(); got != 1 {
		t.Fatalf("Len() after Sweep = %d, want 1", got)
	}

	assertValue(t, s, old, nil)
	assertValue(t, s, recent, "recent")
}

// TestTTLStoreRunSweeper tests that the background sweeper evicts expired
// entries and stops.
func TestTTLStoreRunSweeper(t *testing.T) {
	t.Parallel()

	clock := newFakeClock()
	s := newTTLStore(time.Second, 10, clock.Now)
	set(t, s, Key{Scope: "scope", Namespace: "ns"}, "value")
	clock.Advance(time.Second)

	stop := make(chan struct{})
	done := make(chan struct{})

	go func() {
		s.runSweeper(time.Millisecond, stop)
		close(done)
	}()

	deadline := time.Now().Add(5 * time.Second)
	for s.Len() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("sweeper did not evict the expired entry")
		}

		time.Sleep(time.Millisecond)
	}

	close(stop)
	<-done
}

// TestTTLStoreMaxEntries tests that the oldest entries are evicted once
// maxEntries is reached.
func TestTTLStoreMaxEntries(t *testing.T) {
	t.Parallel()

	s := newTTLStore(time.Minute, 2, newFakeClock().Now)
	first := Key{Scope: "scope", Namespace: "ns", Params: "1"}
	second := Key{Scope: "scope", Namespace: "ns", Params: "2"}
	third := Key{Scope: "scope", Namespace: "other", Params: "3"}

	set(t, s, first, "first")
	set(t, s, second, "second")
	// Re-storing first makes second the oldest entry.
	set(t, s, first, "first again")
	set(t, s, third, "third")

	if got := s.Len(); got != 2 {
		t.Fatalf("Len() = %d, want 2", got)
	}

	assertValue(t, s, second, nil)
	assertValue(t, s, first, "first again")
	assertValue(t, s, third, "third")
}

// TestTTLStoreInvalidate tests that Invalidate drops every Params of the
// given namespaces of a single scope.
func TestTTLStoreInvalidate(t *testing.T) {
	t.Parallel()

	s := newTTLStore(time.Minute, 10, newFakeClock().Now)
	keys := map[string]Key{
		"a/ns1/p1": {Scope: "a", Namespace: "ns1", Params: "p1"},
		"a/ns1/p2": {Scope: "a", Namespace: "ns1", Params: "p2"},
		"a/ns2":    {Scope: "a", Namespace: "ns2"},
		"a/ns3":    {Scope: "a", Namespace: "ns3"},
		"b/ns1/p1": {Scope: "b", Namespace: "ns1", Params: "p1"},
	}

	for name, key := range keys {
		set(t, s, key, name)
	}

	s.Invalidate("a", "ns1", "ns2")

	assertValue(t, s, keys["a/ns1/p1"], nil)
	assertValue(t, s, keys["a/ns1/p2"], nil)
	assertValue(t, s, keys["a/ns2"], nil)
	assertValue(t, s, keys["a/ns3"], "a/ns3")
	assertValue(t, s, keys["b/ns1/p1"], "b/ns1/p1")

	if got := s.Len(); got != 2 {
		t.Errorf("Len() = %d, want 2", got)
	}
}

// TestTTLStoreSetAfterInvalidateIsRejected tests that a value fetched before
// an invalidation is not stored.
func TestTTLStoreSetAfterInvalidateIsRejected(t *testing.T) {
	t.Parallel()

	s := newTTLStore(time.Minute, 10, newFakeClock().Now)
	key := Key{Scope: "scope", Namespace: "ns"}

	_, generation, _ := s.Get(key)
	s.Invalidate("scope", "ns")

	if s.Set(key, "stale", generation) {
		t.Error("Set() with a generation older than the last Invalidate was accepted")
	}

	assertValue(t, s, key, nil)

	// Other namespaces of the scope are unaffected.
	other := Key{Scope: "scope", Namespace: "other"}
	_, otherGeneration, _ := s.Get(other)

	if !s.Set(other, "value", otherGeneration) {
		t.Error("Set() on a namespace that was not invalidated was rejected")
	}
}

// TestNoopStore tests that the noop store never stores anything.
func TestNoopStore(t *testing.T) {
	t.Parallel()

	s := NewNoopStore()
	key := Key{Scope: "scope", Namespace: "ns"}

	if s.Set(key, "value", 0) {
		t.Error("noop Set() reported storing a value")
	}

	assertValue(t, s, key, nil)
	s.Invalidate("scope", "ns")

	if got := s.Len(); got != 0 {
		t.Errorf("noop Len() = %d, want 0", got)
	}

	if IsEnabled(s) {
		t.Error("IsEnabled(noop) = true, want false")
	}

	if IsEnabled(nil) {
		t.Error("IsEnabled(nil) = true, want false")
	}

	if !IsEnabled(newTTLStore(time.Minute, 1, time.Now)) {
		t.Error("IsEnabled(ttlStore) = false, want true")
	}
}
