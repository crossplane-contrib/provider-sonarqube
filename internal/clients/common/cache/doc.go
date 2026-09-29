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

// Package cache provides a short-lived, process-wide cache for SonarQube
// list/search responses read during Observe.
//
// Many managed resources of the same kind usually share a single SonarQube
// endpoint that returns the whole dataset (for example the list of installed
// plugins). Without a cache, every reconcile of every resource re-fetches
// that dataset. Client decorators use Fetch (or FetchWithResponse) to share
// one response between reconciles for at most one TTL.
//
// The cache is disabled by default: until Configure is called with
// Options.Enabled set, Default returns a no-op Store that always misses and
// never stores, so call sites never need "if enabled" branches.
//
// The enabled Store is an LRUExpireCache from k8s.io/apimachinery, bounded
// by a number of entries (Options.MaxEntries) and evicting the least
// recently used entries first.
//
// Consumers hold a Scoped, which binds a Store to one connection (see
// ForConfig). Read paths either decorate an SDK client (the decorator
// caches a list method transparently), or, when a whole dataset is better
// fetched once and indexed (for example every group permission of a
// project), expose a dataset accessor taking the Scoped explicitly. In both
// cases, the write paths are decorated to invalidate what they affect.
//
// Entries are addressed by a Key made of:
//   - Scope: the identity of the SonarQube connection (see ScopeFromConfig),
//     so that two credentials never share cached data.
//   - Namespace: the logical dataset, for example "plugins/installed".
//   - Params: a canonical encoding of the query parameters (see
//     EncodeParams), empty for instance-wide lists.
//
// # Contract
//
// Every consumer of this package must respect the following rules:
//
//  1. Cached values are read-only. Consumers must not mutate returned
//     slices, maps or pointed-to structs. Decorators return the shared
//     value, and any caller that needs to mutate it makes a copy first.
//  2. Every write path on a decorated client must invalidate the
//     namespaces it can affect (Store.Invalidate), after the write returns,
//     including when the write returns an error (the write may have
//     partially succeeded).
//  3. Datasets mentioning other objects (projects, users, groups, Quality
//     Gates, Quality Profiles) declare it with DependOn, and every write
//     creating, renaming or deleting such an object invalidates its kind
//     with Scoped.InvalidateEntities: deleting a project must drop its
//     cached permissions, whichever controller deleted it.
//  4. The TTL must stay below crossplane-runtime's creation grace period
//     (30s, see MaxTTL), so that a resource created by this provider is
//     always observed from fresh data once the grace period is over.
package cache
