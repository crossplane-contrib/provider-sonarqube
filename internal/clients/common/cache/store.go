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
	"slices"
	"sync"
	"time"

	k8scache "k8s.io/apimachinery/pkg/util/cache"
)

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

// ttlStore is a Store backed by an LRUExpireCache, which bounds the number
// of entries and expires them after a TTL. ttlStore adds per-bucket
// generations on top of it, so that a value fetched before an Invalidate is
// never stored after it.
type ttlStore struct {
	// mu makes the generation check of Set atomic with Invalidate.
	mu sync.RWMutex
	// ttl is the lifetime of an entry.
	ttl time.Duration
	// entries holds the cached values, keyed by Key.
	entries *k8scache.LRUExpireCache
	// generations records, for every invalidated bucket, the clock value
	// of its latest invalidation. Buckets never invalidated are at 0.
	generations map[bucketKey]uint64
	// clock is incremented by every invalidation.
	clock uint64
}

// newTTLStore returns a ttlStore holding its values in entries.
func newTTLStore(ttl time.Duration, entries *k8scache.LRUExpireCache) *ttlStore {
	return &ttlStore{
		ttl:         ttl,
		entries:     entries,
		generations: make(map[bucketKey]uint64),
	}
}

// Get returns the live value stored under key, if any.
func (s *ttlStore) Get(key Key) (value any, generation uint64, found bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, found = s.entries.Get(key)

	return value, s.generations[bucketKey{scope: key.Scope, namespace: key.Namespace}], found
}

// Set stores value under key unless its bucket was invalidated since
// generation was observed.
func (s *ttlStore) Set(key Key, value any, generation uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.generations[bucketKey{scope: key.Scope, namespace: key.Namespace}] != generation {
		return false
	}

	s.entries.Add(key, value, s.ttl)

	return true
}

// Invalidate drops every entry of scope under the given namespaces and
// bumps their generation, so that in-flight fetches started before the
// invalidation are not stored.
func (s *ttlStore) Invalidate(scope string, namespaces ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, namespace := range namespaces {
		s.clock++
		s.generations[bucketKey{scope: scope, namespace: namespace}] = s.clock

		invalidationsTotal.WithLabelValues(namespace).Inc()
	}

	s.entries.RemoveAll(func(cached any) bool {
		key, _ := cached.(Key)

		return key.Scope == scope && slices.Contains(namespaces, key.Namespace)
	})
}

// Len returns the number of unexpired entries currently held.
func (s *ttlStore) Len() int {
	return len(s.entries.Keys())
}
