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
	"time"

	"github.com/pkg/errors"
	k8scache "k8s.io/apimachinery/pkg/util/cache"
)

const (
	// MaxTTL is the exclusive upper bound of Options.TTL: crossplane-runtime's
	// creation grace period. See the package documentation.
	MaxTTL = 30 * time.Second

	// DefaultTTL is the default lifetime of a cached entry.
	DefaultTTL = 20 * time.Second

	// DefaultMaxEntries is the default maximum number of cached entries.
	DefaultMaxEntries = 1000
)

// Options configures the process-wide default Store.
type Options struct {
	// Enabled turns the cache on. When false, Default returns a no-op Store.
	Enabled bool
	// TTL is the lifetime of a cached entry. It must be in (0, MaxTTL).
	TTL time.Duration
	// MaxEntries is the maximum number of cached entries. It must be
	// strictly positive.
	MaxEntries int
}

// Validate checks that the options are usable. Disabled options are always
// valid.
func (o Options) Validate() error {
	if !o.Enabled {
		return nil
	}

	if o.TTL <= 0 || o.TTL >= MaxTTL {
		return errors.Errorf("observe cache TTL must be greater than 0 and lower than %s, got %s", MaxTTL, o.TTL)
	}

	if o.MaxEntries <= 0 {
		return errors.Errorf("observe cache max entries must be greater than 0, got %d", o.MaxEntries)
	}

	return nil
}

var (
	// defaultMu guards defaultStore.
	defaultMu sync.RWMutex
	// defaultStore is the Store returned by Default.
	defaultStore = NewNoopStore()
)

// NewStore returns a Store configured by opts: a no-op Store when the cache
// is disabled, an in-memory TTL Store otherwise.
func NewStore(opts Options) (Store, error) {
	err := opts.Validate()
	if err != nil {
		return nil, err
	}

	if !opts.Enabled {
		return NewNoopStore(), nil
	}

	return newTTLStore(opts.TTL, k8scache.NewLRUExpireCache(opts.MaxEntries)), nil
}

// Configure sets up the process-wide default Store returned by Default. It
// is meant to be called once from main, before the controllers start. With
// disabled options, Default keeps returning a no-op Store.
func Configure(opts Options) error {
	store, err := NewStore(opts)
	if err != nil {
		return err
	}

	if IsEnabled(store) {
		registerMetrics()
	}

	defaultMu.Lock()
	defer defaultMu.Unlock()

	defaultStore = store

	return nil
}

// Default returns the process-wide Store configured by Configure, or a no-op
// Store when Configure was not called or the cache is disabled.
func Default() Store {
	defaultMu.RLock()
	defer defaultMu.RUnlock()

	return defaultStore
}

// IsEnabled reports whether store actually caches values, that is whether it
// is not a no-op Store. Client constructors use it to skip decorating their
// client when the cache is disabled.
func IsEnabled(store Store) bool {
	_, disabled := store.(noopStore)

	return store != nil && !disabled
}
