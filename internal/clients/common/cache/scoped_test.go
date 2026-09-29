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
	"testing"
	"time"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// TestScopedKey tests that Scoped builds keys in its own scope.
func TestScopedKey(t *testing.T) {
	t.Parallel()

	got := NewScoped(NewNoopStore(), "scope").Key("namespace", "params")
	want := Key{Scope: "scope", Namespace: "namespace", Params: "params"}

	if got != want {
		t.Errorf("Key() = %+v, want %+v", got, want)
	}
}

// TestScopedEnabled tests that only a Scoped with an enabled store caches.
func TestScopedEnabled(t *testing.T) {
	t.Parallel()

	if (Scoped{}).Enabled() {
		t.Error("zero Scoped is enabled")
	}

	if NewScoped(NewNoopStore(), "scope").Enabled() {
		t.Error("Scoped with a no-op store is enabled")
	}

	if !NewScoped(newTestStore(time.Minute, 10, newFakeClock()), "scope").Enabled() {
		t.Error("Scoped with a TTL store is disabled")
	}
}

// TestScopedInvalidate tests that Scoped only invalidates its own scope,
// and that invalidating through a disabled Scoped is a no-op.
func TestScopedInvalidate(t *testing.T) {
	t.Parallel()

	store := newTestStore(time.Minute, 10, newFakeClock())
	mine := NewScoped(store, "mine")
	other := NewScoped(store, "other")

	for _, scoped := range []Scoped{mine, other} {
		if !store.Set(scoped.Key("namespace", ""), "value", 0) {
			t.Fatalf("Set(%q) was rejected", scoped.Scope)
		}
	}

	mine.Invalidate("namespace")

	if _, _, found := store.Get(mine.Key("namespace", "")); found {
		t.Error("Invalidate() kept an entry of its own scope")
	}

	if _, _, found := store.Get(other.Key("namespace", "")); !found {
		t.Error("Invalidate() dropped an entry of another scope")
	}

	// Must not panic on a nil store.
	Scoped{}.Invalidate("namespace")
}

// TestForConfig tests that ForConfig binds the default store to the scope
// of the config. It reads the process-wide default store, so it does not
// run in parallel with TestConfigure.
//
//nolint:paralleltest // reads package-level state mutated by TestConfigure
func TestForConfig(t *testing.T) {
	config := common.Config{BaseURL: "http://sonar", AuthType: common.PersonalAccessToken, Token: "token"}

	got := ForConfig(config)

	if got.Scope != ScopeFromConfig(config) {
		t.Errorf("ForConfig().Scope = %q, want %q", got.Scope, ScopeFromConfig(config))
	}

	if got.Store != Default() {
		t.Errorf("ForConfig().Store = %v, want Default()", got.Store)
	}
}
