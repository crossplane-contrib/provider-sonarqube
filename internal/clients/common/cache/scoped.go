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
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// Scoped binds a Store to the scope of one SonarQube connection. Client
// decorators and dataset accessors hold a Scoped rather than a Store and a
// scope, so that every key they build and every invalidation they issue is
// confined to their connection.
//
// The zero value is a disabled Scoped: it never caches.
type Scoped struct {
	// Store holds the cached values.
	Store Store
	// Scope identifies the SonarQube connection, see ScopeFromConfig.
	Scope string
}

// NewScoped returns a Scoped binding store to scope.
func NewScoped(store Store, scope string) Scoped {
	return Scoped{Store: store, Scope: scope}
}

// ForConfig returns a Scoped binding the process-wide Default Store to the
// connection described by config.
func ForConfig(config common.Config) Scoped {
	return NewScoped(Default(), ScopeFromConfig(config))
}

// Enabled reports whether s actually caches values, see IsEnabled.
func (s Scoped) Enabled() bool {
	return IsEnabled(s.Store)
}

// Key returns the key of the given dataset in s's scope. params is the
// canonical encoding of the dataset's query parameters, empty for
// instance-wide lists.
func (s Scoped) Key(namespace, params string) Key {
	return Key{Scope: s.Scope, Namespace: namespace, Params: params}
}

// Invalidate drops every entry of s's scope under the given namespaces. It
// does nothing on a disabled Scoped.
func (s Scoped) Invalidate(namespaces ...string) {
	if s.Enabled() {
		s.Store.Invalidate(s.Scope, namespaces...)
	}
}
