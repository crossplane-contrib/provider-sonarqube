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

package instance_test

import (
	"context"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestQualityProfileCRUD creates an empty (no custom rules) Go quality
// profile and verifies SonarQube knows about it at the expected name +
// language. Leaving rules out keeps the test hermetic - rule references
// are exercised separately in the rule_test.go suite.
func TestQualityProfileCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName   = "e2e-qp-go-crud"
		qpName   = "e2e-qp-go-crud"
		qpLang   = "go"
		hoursTtl = 2 * time.Minute
	)

	qp := &instancev1alpha1.QualityProfile{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: instancev1alpha1.QualityProfileSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.QualityProfileParameters{
				Name:     qpName,
				Language: qpLang,
				Default:  ptr.To(false),
			},
		},
	}

	f.CreateAndWaitForReady(t, qp, hoursTtl)
	e2e.AssertReady(t, qp)
	e2e.AssertSynced(t, qp)

	got, err := f.FindQualityProfile(context.Background(), qpName, qpLang)
	if err != nil {
		t.Fatalf("searching quality profiles: %v", err)
	}
	if got == nil {
		t.Fatalf("quality profile %q (%s) not found in SonarQube", qpName, qpLang)
	}
	if got.Language != qpLang {
		t.Errorf("quality profile language = %q, want %q", got.Language, qpLang)
	}
	if got.IsBuiltIn {
		t.Errorf("quality profile %q is reported as built-in; want user-created", qpName)
	}
}
