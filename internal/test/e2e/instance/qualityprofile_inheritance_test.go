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
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestQualityProfileInheritance creates a profile extending the built-in
// "Sonar way", checks that it inherits its rules, then removes the parent.
func TestQualityProfileInheritance(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const (
		name       = "e2e-qp-go-inheritance"
		language   = "go"
		parentName = "Sonar way"
	)

	qp := &instancev1alpha1.QualityProfile{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: f.Namespace},
		Spec: instancev1alpha1.QualityProfileSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.QualityProfileParameters{
				Name:     name,
				Language: language,
				Parent:   new(parentName),
			},
		},
	}

	f.CreateAndWaitForReady(t, qp, 2*time.Minute)

	// waitForParent waits until SonarQube reports the wanted parent, along
	// with inherited rules if and only if there is a parent.
	waitForParent := func(want string) {
		t.Helper()

		err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
			got, err := f.FindQualityProfile(ctx, name, language)
			if err != nil || got == nil {
				return false, err
			}

			return got.ParentName == want && (got.ActiveRuleCount > 0) == (want != ""), nil
		})
		if err != nil {
			t.Fatalf("waiting for quality profile %q to have parent %q: %v", name, want, err)
		}
	}

	waitForParent(parentName)

	err := f.Kube.Get(context.Background(), client.ObjectKeyFromObject(qp), qp)
	if err != nil {
		t.Fatalf("getting quality profile: %v", err)
	}

	// A merge patch cannot conflict with the controller's concurrent updates.
	base := qp.DeepCopy()
	qp.Spec.ForProvider.Parent = new("")

	err = f.Kube.Patch(context.Background(), qp, client.MergeFrom(base))
	if err != nil {
		t.Fatalf("removing the parent of the quality profile: %v", err)
	}

	waitForParent("")
}
