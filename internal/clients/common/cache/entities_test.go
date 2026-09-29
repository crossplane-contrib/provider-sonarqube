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
	"testing"
	"time"
)

// TestInvalidateEntities tests that invalidating an entity kind drops the
// namespaces declared as depending on it, in the Scoped's scope only.
func TestInvalidateEntities(t *testing.T) {
	t.Parallel()

	// Test-specific kinds and namespaces, so that the process-wide
	// registry does not interfere with other tests.
	const (
		entityA Entity = "test/entities/a"
		entityB Entity = "test/entities/b"
		entityC Entity = "test/entities/c"
	)

	DependOn("test/entities/ab", entityA, entityB)
	DependOn("test/entities/ab", entityA)
	DependOn("test/entities/b", entityB)

	if got := Dependents(entityA, entityB); !slices.Equal(got, []string{"test/entities/ab", "test/entities/b"}) {
		t.Errorf("Dependents(a, b) = %v, want each namespace once", got)
	}

	store := newTestStore(time.Minute, 10, newFakeClock())
	mine := NewScoped(store, "mine")
	other := NewScoped(store, "other")

	for _, key := range []Key{mine.Key("test/entities/ab", ""), mine.Key("test/entities/b", ""), other.Key("test/entities/ab", "")} {
		if !store.Set(key, "value", 0) {
			t.Fatalf("Set(%v) was rejected", key)
		}
	}

	// A kind without dependents invalidates nothing.
	mine.InvalidateEntities(entityC)

	if got := store.Len(); got != 3 {
		t.Fatalf("Len() after invalidating a kind without dependents = %d, want 3", got)
	}

	mine.InvalidateEntities(entityA)

	if _, _, found := store.Get(mine.Key("test/entities/ab", "")); found {
		t.Error("InvalidateEntities(a) kept a dependent namespace")
	}

	if _, _, found := store.Get(mine.Key("test/entities/b", "")); !found {
		t.Error("InvalidateEntities(a) dropped a namespace depending on b only")
	}

	if _, _, found := store.Get(other.Key("test/entities/ab", "")); !found {
		t.Error("InvalidateEntities(a) dropped an entry of another scope")
	}

	// Must not panic on a disabled Scoped.
	Scoped{}.InvalidateEntities(entityA)
}
