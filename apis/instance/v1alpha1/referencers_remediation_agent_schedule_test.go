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

package v1alpha1

import (
	"context"
	"testing"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestRemediationAgentScheduleResolveReferences verifies that project
// references resolve to project keys.
func TestRemediationAgentScheduleResolveReferences(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	err := SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(): %v", err)
	}

	project := &Project{ObjectMeta: metav1.ObjectMeta{
		Name: "project", Namespace: "default",
		Annotations: map[string]string{"crossplane.io/external-name": "project-key"},
	}}
	kube := fakekube.NewClientBuilder().WithScheme(scheme).WithObjects(project).Build()
	ref := xpv1.Reference{Name: "project"}
	missing := xpv1.Reference{Name: "missing"}
	key := "project-key"

	cases := map[string]struct {
		spec    RemediationAgentScheduleParameters
		want    RemediationAgentScheduleParameters
		wantErr bool
	}{
		"NothingToResolve": {},
		"ProjectKey": {
			spec: RemediationAgentScheduleParameters{ProjectKeyRef: &ref},
			want: RemediationAgentScheduleParameters{ProjectKey: &key, ProjectKeyRef: &ref},
		},
		"SelectedProjects": {
			spec: RemediationAgentScheduleParameters{SelectedProjectKeyRefs: []xpv1.Reference{ref}},
			want: RemediationAgentScheduleParameters{SelectedProjectKeys: []string{key}, SelectedProjectKeyRefs: []xpv1.Reference{ref}},
		},
		"MissingProject":         {spec: RemediationAgentScheduleParameters{ProjectKeyRef: &missing}, wantErr: true},
		"MissingSelectedProject": {spec: RemediationAgentScheduleParameters{SelectedProjectKeyRefs: []xpv1.Reference{missing}}, wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			schedule := &RemediationAgentSchedule{
				ObjectMeta: metav1.ObjectMeta{Name: "schedule", Namespace: "default"},
				Spec:       RemediationAgentScheduleSpec{ForProvider: tc.spec},
			}

			resolveErr := schedule.ResolveReferences(context.Background(), kube)
			if (resolveErr != nil) != tc.wantErr {
				t.Fatalf("ResolveReferences() error = %v, wantErr %v", resolveErr, tc.wantErr)
			}

			if tc.wantErr {
				return
			}

			if diff := cmp.Diff(tc.want, schedule.Spec.ForProvider); diff != "" {
				t.Errorf("-want, +got:\n%s", diff)
			}
		})
	}
}
