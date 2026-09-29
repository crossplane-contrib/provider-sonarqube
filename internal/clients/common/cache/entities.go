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
)

// Entity is a kind of SonarQube object that cached datasets mention.
// Creating, renaming or deleting an object of that kind makes every
// dataset depending on the kind stale, even though the dataset's own
// writes were not called: deleting a project removes its permissions and
// webhooks, renaming a group changes the names permission searches return.
type Entity string

const (
	// EntityProject is a project, identified by its key.
	EntityProject Entity = "project"
	// EntityUser is a user, identified by its login.
	EntityUser Entity = "user"
	// EntityGroup is a group, identified by its name.
	EntityGroup Entity = "group"
	// EntityQualityGate is a Quality Gate, identified by its name.
	EntityQualityGate Entity = "qualitygate"
	// EntityQualityProfile is a Quality Profile, identified by its
	// language and name.
	EntityQualityProfile Entity = "qualityprofile"
)

var (
	// dependentsMu guards dependents.
	dependentsMu sync.RWMutex
	// dependents lists, for every entity kind, the cache namespaces that
	// depend on it.
	dependents = map[Entity][]string{}
)

// DependOn declares that the datasets cached under namespace mention
// objects of the given kinds, so that Scoped.InvalidateEntities drops
// them when such an object is created, renamed or deleted.
//
// Packages defining a cached dataset call it from an init function, next
// to the namespace constant, so that the writers of those objects never
// need to know which datasets exist.
func DependOn(namespace string, entities ...Entity) {
	dependentsMu.Lock()
	defer dependentsMu.Unlock()

	for _, entity := range entities {
		if !slices.Contains(dependents[entity], namespace) {
			dependents[entity] = append(dependents[entity], namespace)
		}
	}
}

// Dependents returns the namespaces declared as depending on any of the
// given entity kinds, see DependOn.
func Dependents(entities ...Entity) []string {
	dependentsMu.RLock()
	defer dependentsMu.RUnlock()

	var namespaces []string

	for _, entity := range entities {
		for _, namespace := range dependents[entity] {
			if !slices.Contains(namespaces, namespace) {
				namespaces = append(namespaces, namespace)
			}
		}
	}

	return namespaces
}

// InvalidateEntities drops every entry of s's scope under the namespaces
// depending on the given entity kinds, see DependOn. Decorators of the
// clients creating, renaming or deleting such objects call it after the
// write, whether it succeeded or not.
func (s Scoped) InvalidateEntities(entities ...Entity) {
	namespaces := Dependents(entities...)
	if len(namespaces) > 0 {
		s.Invalidate(namespaces...)
	}
}
