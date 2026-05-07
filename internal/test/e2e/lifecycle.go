//go:build e2e

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

package e2e

import (
	"context"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DefaultDeleteTimeout bounds how long Cleanup waits for the controller
// to finalize a managed resource before logging the failure and moving on.
const DefaultDeleteTimeout = 2 * time.Minute

// Apply creates obj on the cluster, failing the test on any error.
// Callers that also want automatic cleanup + readiness wait should use
// CreateAndWaitForReady instead.
func (f *Framework) Apply(t *testing.T, obj client.Object) {
	t.Helper()
	if err := f.Kube.Create(context.Background(), obj); err != nil {
		t.Fatalf("creating %s/%s: %v", obj.GetNamespace(), obj.GetName(), err)
	}
}

// Update sends a full update of obj, failing the test on any error.
// The object's resourceVersion must already be populated (call Get or
// re-fetch the object first).
func (f *Framework) Update(t *testing.T, obj client.Object) {
	t.Helper()
	if err := f.Kube.Update(context.Background(), obj); err != nil {
		t.Fatalf("updating %s/%s: %v", obj.GetNamespace(), obj.GetName(), err)
	}
}

// Delete removes obj from the cluster, tolerating NotFound so callers can
// safely invoke this after a previous cleanup.
func (f *Framework) Delete(t *testing.T, obj client.Object) {
	t.Helper()
	if err := f.Kube.Delete(context.Background(), obj); err != nil && !apierrors.IsNotFound(err) {
		t.Fatalf("deleting %s/%s: %v", obj.GetNamespace(), obj.GetName(), err)
	}
}

// CreateAndWaitForReady creates obj, registers a t.Cleanup that deletes it
// and waits for the controller to drop the finalizer, then blocks until
// obj reports Ready+Synced.
//
// On a failed wait the test is marked Fatal but cleanup still runs so the
// next test starts from a clean slate.
func (f *Framework) CreateAndWaitForReady(t *testing.T, obj Conditioned, timeout time.Duration) {
	t.Helper()
	ctx := context.Background()

	if err := f.Kube.Create(ctx, obj); err != nil {
		t.Fatalf("creating %s/%s: %v", obj.GetNamespace(), obj.GetName(), err)
	}
	t.Cleanup(func() { f.cleanup(t, obj) })

	if err := f.WaitForReady(ctx, obj, timeout); err != nil {
		t.Fatalf("waiting for %s/%s to be Ready: %v\n  conditions: %s",
			obj.GetNamespace(), obj.GetName(), err, SummariseConditions(obj))
	}
}

// cleanup deletes obj and waits for the API server to confirm its removal.
// Failures are logged rather than failing the test - by the time cleanup
// runs the assertions of interest have already happened.
func (f *Framework) cleanup(t *testing.T, obj client.Object) {
	ctx := context.Background()
	if err := f.Kube.Delete(ctx, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return
		}
		t.Logf("cleanup: deleting %s/%s: %v", obj.GetNamespace(), obj.GetName(), err)
		return
	}
	if err := f.WaitForDeletion(ctx, obj, DefaultDeleteTimeout); err != nil {
		t.Logf("cleanup: waiting for deletion of %s/%s: %v",
			obj.GetNamespace(), obj.GetName(), err)
	}
}
