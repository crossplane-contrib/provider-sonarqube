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
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AssertReady fails the test if obj does not currently report Ready=True.
func AssertReady(t *testing.T, obj Conditioned) {
	t.Helper()
	AssertCondition(t, obj, xpv1.TypeReady, corev1.ConditionTrue)
}

// AssertSynced fails the test if obj does not currently report Synced=True.
func AssertSynced(t *testing.T, obj Conditioned) {
	t.Helper()
	AssertCondition(t, obj, xpv1.TypeSynced, corev1.ConditionTrue)
}

// AssertCondition fails the test if the named condition is not at the
// expected status, surfacing the reason and message for fast triage.
func AssertCondition(t *testing.T, obj Conditioned, ct xpv1.ConditionType, want corev1.ConditionStatus) {
	t.Helper()
	got := obj.GetCondition(ct)
	if got.Status != want {
		t.Errorf("%s condition = %s (reason=%s, message=%q); want %s",
			ct, got.Status, got.Reason, got.Message, want)
	}
}

// AssertExternalName fails the test if obj's crossplane.io/external-name
// annotation does not equal want.
func AssertExternalName(t *testing.T, obj client.Object, want string) {
	t.Helper()
	if got := meta.GetExternalName(obj); got != want {
		t.Errorf("external-name = %q, want %q", got, want)
	}
}
