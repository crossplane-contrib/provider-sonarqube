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

//go:build e2e

package instance_test

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// stringPtr returns a pointer to the supplied string — useful for the
// many optional *string fields in the instance API types.
func stringPtr(s string) *string { return &s }

// boolPtr returns a pointer to the supplied bool.
func boolPtr(b bool) *bool { return &b }

// kubeKey is a terse alias for client.ObjectKeyFromObject to keep the
// polling loops in the individual test files readable.
func kubeKey(obj client.Object) client.ObjectKey {
	return client.ObjectKeyFromObject(obj)
}
