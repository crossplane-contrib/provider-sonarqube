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

package helpers

import "golang.org/x/sync/errgroup"

// MaxConcurrentRequests bounds how many SonarQube requests a single reconcile
// sends at once when it fans out over a list from the spec, so that the size
// of a tenant-controlled list does not set the number of open connections.
const MaxConcurrentRequests = 8

// NewRequestGroup returns a group that runs at most MaxConcurrentRequests
// goroutines at once. Group.Go blocks while the limit is reached, so a long
// list never spawns more goroutines than the limit.
func NewRequestGroup() *errgroup.Group {
	group := &errgroup.Group{}
	group.SetLimit(MaxConcurrentRequests)

	return group
}
