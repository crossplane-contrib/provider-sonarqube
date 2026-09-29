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
	"container/list"
	"encoding/json"
	"sync"
	"time"
)

// sizeOverheadFactor converts the JSON-encoded size of a value into an
// estimate of the memory it takes once decoded: Go strings, slices, maps and
// struct padding carry headers the JSON encoding does not.
const sizeOverheadFactor = 2

// Store holds cached values. Implementations are safe for concurrent use.
//
// Consumers normally do not call Get and Set directly: they read through
// Fetch (or FetchWithResponse) and only call Invalidate from their write
// paths.
type Store interface {
	// Get returns the live value stored under key, if any. The returned
	// generation identifies the current state of the key's
	// (Scope, Namespace) pair; it must be passed back to Set so that a
	// value fetched before an Invalidate is never stored after it.
	Get(key Key) (value any, generation uint64, found bool)
	// Set stores value under key, unless the key's (Scope, Namespace) pair
	// was invalidated since generation was returned by Get. It reports
	// whether the value was stored.
	Set(key Key, value any, generation uint64) bool
	// Invalidate drops every entry of the given scope under the given
	// namespaces, whatever their Params.
	Invalidate(scope string, namespaces ...string)
	// Len returns the number of entries currently held.
	Len() int
}

// noopStore is the Store used when the cache is disabled: it always misses
// and never stores.
type noopStore struct{}

// NewNoopStore returns a Store that always misses and never stores.
func NewNoopStore() Store {
	return noopStore{}
}

// Get always misses.
func (noopStore) Get(Key) (value any, generation uint64, found bool) {
	return nil, 0, false
}

// Set never stores.
func (noopStore) Set(Key, any, uint64) bool {
	return false
}

// Invalidate does nothing.
func (noopStore) Invalidate(string, ...string) {}

// Len always returns 0.
func (noopStore) Len() int {
	return 0
}

// bucketKey identifies the set of entries sharing a Scope and a Namespace,
// which is the unit of invalidation.
type bucketKey struct {
	// scope is the Key.Scope of the bucket's entries.
	scope string
	// namespace is the Key.Namespace of the bucket's entries.
	namespace string
}

// entry is a cached value, stored in ttlStore.order.
type entry struct {
	// key is the entry's key, used to remove it from its bucket on eviction.
	key Key
	// value is the cached value.
	value any
	// expiresAt is the time after which the entry must not be served.
	expiresAt time.Time
	// size is the estimated memory footprint of value, in bytes.
	size int64
}

// ttlStore is an in-memory Store with a per-entry time to live, a bound on
// the number of entries and a bound on their estimated total size. Expired
// entries are evicted lazily on read and periodically by Sweep. When a bound
// is reached, the oldest entries are evicted first.
type ttlStore struct {
	// mu guards every field below.
	mu sync.RWMutex
	// ttl is the lifetime of an entry.
	ttl time.Duration
	// maxEntries is the maximum number of entries held.
	maxEntries int
	// maxBytes is the maximum estimated size of the entries held.
	maxBytes int64
	// bytes is the estimated size of the entries held.
	bytes int64
	// sizeOf estimates the memory footprint of a value, or reports that it
	// cannot; injectable for tests.
	sizeOf func(value any) (size int64, ok bool)
	// now returns the current time; injectable for tests.
	now func() time.Time
	// buckets indexes entries by (Scope, Namespace), then by Params.
	buckets map[bucketKey]map[string]*list.Element
	// order lists entries from the oldest to the most recently stored.
	order *list.List
	// generations records, for every invalidated bucket, the clock value
	// of its latest invalidation. Buckets never invalidated are at 0.
	generations map[bucketKey]uint64
	// clock is incremented by every invalidation.
	clock uint64
}

// newTTLStore returns an empty ttlStore.
func newTTLStore(ttl time.Duration, maxEntries int, maxBytes int64, now func() time.Time) *ttlStore {
	return &ttlStore{
		ttl:         ttl,
		maxEntries:  maxEntries,
		maxBytes:    maxBytes,
		sizeOf:      estimateSize,
		now:         now,
		buckets:     make(map[bucketKey]map[string]*list.Element),
		order:       list.New(),
		generations: make(map[bucketKey]uint64),
	}
}

// Get returns the live value stored under key, if any, evicting it when it
// has expired.
func (s *ttlStore) Get(key Key) (value any, generation uint64, found bool) {
	bucket := bucketKey{scope: key.Scope, namespace: key.Namespace}

	s.mu.RLock()
	generation = s.generations[bucket]
	element, found := s.buckets[bucket][key.Params]

	if !found {
		s.mu.RUnlock()

		return nil, generation, false
	}

	cached, _ := element.Value.(*entry)
	if s.now().Before(cached.expiresAt) {
		s.mu.RUnlock()

		return cached.value, generation, true
	}

	s.mu.RUnlock()

	// Lazy eviction: re-check under the write lock, the entry may have been
	// replaced in the meantime.
	s.mu.Lock()
	defer s.mu.Unlock()

	if current, ok := s.buckets[bucket][key.Params]; ok && current == element {
		s.removeLocked(element)
	}

	return nil, generation, false
}

// Set stores value under key unless its bucket was invalidated since
// generation was observed, evicting the oldest entries to stay within
// maxEntries and maxBytes. A value whose size cannot be estimated, or that
// is larger than maxBytes on its own, is not stored.
func (s *ttlStore) Set(key Key, value any, generation uint64) bool {
	bucket := bucketKey{scope: key.Scope, namespace: key.Namespace}

	// Estimate outside the lock: it encodes the whole value.
	size, sizeOK := s.sizeOf(value)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.generations[bucket] != generation {
		return false
	}

	// The previous value is older than the one being stored: drop it even
	// when the new one cannot be stored.
	if existing, ok := s.buckets[bucket][key.Params]; ok {
		s.removeLocked(existing)
	}

	if !sizeOK || size > s.maxBytes {
		return false
	}

	for s.order.Len() >= s.maxEntries || s.bytes+size > s.maxBytes {
		oldest := s.order.Front()
		if oldest == nil {
			break
		}

		s.removeLocked(oldest)
	}

	params, ok := s.buckets[bucket]
	if !ok {
		params = make(map[string]*list.Element)
		s.buckets[bucket] = params
	}

	params[key.Params] = s.order.PushBack(&entry{
		key:       key,
		value:     value,
		expiresAt: s.now().Add(s.ttl),
		size:      size,
	})
	s.bytes += size
	s.updateGaugesLocked()

	return true
}

// Invalidate drops every entry of scope under the given namespaces and
// bumps their generation, so that in-flight fetches started before the
// invalidation are not stored.
func (s *ttlStore) Invalidate(scope string, namespaces ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, namespace := range namespaces {
		bucket := bucketKey{scope: scope, namespace: namespace}

		s.clock++
		s.generations[bucket] = s.clock

		for _, element := range s.buckets[bucket] {
			s.removeLocked(element)
		}

		invalidationsTotal.WithLabelValues(namespace).Inc()
	}
}

// Len returns the number of entries currently held, including expired
// entries not evicted yet.
func (s *ttlStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.order.Len()
}

// Bytes returns the estimated size of the entries currently held.
func (s *ttlStore) Bytes() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.bytes
}

// Sweep evicts every expired entry.
func (s *ttlStore) Sweep() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()

	for element := s.order.Front(); element != nil; {
		next := element.Next()

		cached, _ := element.Value.(*entry)
		if !now.Before(cached.expiresAt) {
			s.removeLocked(element)
		}

		element = next
	}
}

// runSweeper calls Sweep every interval until stop is closed.
func (s *ttlStore) runSweeper(interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s.Sweep()
		}
	}
}

// removeLocked removes element from the store. s.mu must be held for
// writing.
func (s *ttlStore) removeLocked(element *list.Element) {
	cached, _ := element.Value.(*entry)
	bucket := bucketKey{scope: cached.key.Scope, namespace: cached.key.Namespace}

	s.order.Remove(element)

	params := s.buckets[bucket]
	delete(params, cached.key.Params)

	if len(params) == 0 {
		delete(s.buckets, bucket)
	}

	s.bytes -= cached.size
	s.updateGaugesLocked()
}

// updateGaugesLocked publishes the size of the store. s.mu must be held.
func (s *ttlStore) updateGaugesLocked() {
	entriesGauge.Set(float64(s.order.Len()))
	bytesGauge.Set(float64(s.bytes))
}

// estimateSize estimates the memory footprint of value from the length of
// its JSON encoding. Values returned by the SonarQube SDK are decoded from
// JSON, so they always encode; ok is false for any value that does not.
func estimateSize(value any) (size int64, ok bool) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return 0, false
	}

	return int64(len(encoded)) * sizeOverheadFactor, true
}
