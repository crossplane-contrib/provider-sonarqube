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

// Package cachetest provides helpers to test cache decorators.
package cachetest

import (
	"testing"
	"time"

	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
)

const (
	// ttl outlives any test, so that entries never expire mid-test.
	ttl = 10 * time.Second
	// maxEntries is large enough for any test.
	maxEntries = 100
)

// NewStore returns an enabled store that is not the process-wide default
// one.
func NewStore(t *testing.T) cache.Store {
	t.Helper()

	store, err := cache.NewStore(cache.Options{Enabled: true, TTL: ttl, MaxEntries: maxEntries})
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	return store
}

// NewScoped returns a Scoped binding a fresh NewStore to a scope named after
// the test.
func NewScoped(t *testing.T) cache.Scoped {
	t.Helper()

	return cache.NewScoped(NewStore(t), t.Name())
}

// Disabled returns a Scoped that never caches.
func Disabled() cache.Scoped {
	return cache.NewScoped(cache.NewNoopStore(), "disabled")
}
