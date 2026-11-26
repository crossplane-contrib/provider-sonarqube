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

package integration

import (
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// TestNewJiraClient tests NewJiraClient.
func TestNewJiraClient(t *testing.T) {
	t.Parallel()

	if NewJiraClient(common.Config{AuthType: common.PersonalAccessToken, Token: "token", BaseURL: "http://localhost:9000"}) == nil {
		t.Error("NewJiraClient() = nil")
	}
}

// TestParseJiraOrganizationBinding tests ParseJiraOrganizationBinding.
func TestParseJiraOrganizationBinding(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		raw     *sonar.JiraOrganizationBinding
		want    *JiraOrganizationBinding
		wantErr bool
	}{
		"Nil":   {},
		"Empty": {raw: &sonar.JiraOrganizationBinding{}},
		"Binding": {
			raw: &sonar.JiraOrganizationBinding{
				"id": "id", "sonarOrganizationUuid": "org", "jiraInstanceUrl": "https://jira",
				"createdBy": "admin", "createdAt": float64(1000), "isTokenShared": true,
			},
			want: &JiraOrganizationBinding{
				ID: "id", SonarOrganizationUUID: "org", JiraInstanceURL: "https://jira",
				CreatedBy: "admin", CreatedAt: 1000, IsTokenShared: true,
			},
		},
		"WrongType":     {raw: &sonar.JiraOrganizationBinding{"id": 1}, wantErr: true},
		"Unmarshalable": {raw: &sonar.JiraOrganizationBinding{"id": make(chan int)}, wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseJiraOrganizationBinding(tc.raw)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseJiraOrganizationBinding() error = %v, wantErr %t", err, tc.wantErr)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseJiraOrganizationBinding() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestParseJiraProjectBinding tests ParseJiraProjectBinding.
func TestParseJiraProjectBinding(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		raw     *sonar.JiraProjectBinding
		want    *JiraProjectBinding
		wantErr bool
	}{
		"Nil":   {},
		"Empty": {raw: &sonar.JiraProjectBinding{}},
		"Binding": {
			raw:  &sonar.JiraProjectBinding{"id": "id", "sonarProjectId": "project", "jiraOrganizationId": "org", "jiraProjectKey": "ABC"},
			want: &JiraProjectBinding{ID: "id", SonarProjectID: "project", JiraOrganizationID: "org", JiraProjectKey: "ABC"},
		},
		"WrongType": {raw: &sonar.JiraProjectBinding{"id": 1}, wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseJiraProjectBinding(tc.raw)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseJiraProjectBinding() error = %v, wantErr %t", err, tc.wantErr)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseJiraProjectBinding() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestJiraOrganizationBindingHelpers tests JiraOrganizationBindingHelpers.
func TestJiraOrganizationBindingHelpers(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.JiraOrganizationBindingParameters{SonarOrganizationUUID: "org"}

	if got := GenerateJiraOrganizationBindingOptions(spec); got.SonarOrganizationUuid != "org" {
		t.Errorf("GenerateJiraOrganizationBindingOptions() = %+v", got)
	}

	observation := GenerateJiraOrganizationBindingObservation(&JiraOrganizationBinding{ID: "id", CreatedAt: 1000, IsTokenShared: true})
	want := v1alpha1.JiraOrganizationBindingObservation{ID: "id", CreatedAt: new(metav1.NewTime(time.UnixMilli(1000))), IsTokenShared: true}

	if diff := cmp.Diff(want, observation); diff != "" {
		t.Errorf("GenerateJiraOrganizationBindingObservation() mismatch (-want +got):\n%s", diff)
	}

	if got := GenerateJiraOrganizationBindingObservation(&JiraOrganizationBinding{}); got.CreatedAt != nil {
		t.Errorf("CreatedAt = %v, want nil", got.CreatedAt)
	}

	if !IsJiraOrganizationBindingUpToDate(spec, &observation) {
		t.Error("an unset isTokenShared must be up to date")
	}

	if !LateInitializeJiraOrganizationBinding(spec, &observation) || !ptr.Deref(spec.IsTokenShared, false) {
		t.Errorf("isTokenShared not late initialized: %+v", spec)
	}

	spec.IsTokenShared = new(false)

	if LateInitializeJiraOrganizationBinding(spec, &observation) || *spec.IsTokenShared {
		t.Error("a set isTokenShared must not be late initialized")
	}

	if IsJiraOrganizationBindingUpToDate(spec, &observation) {
		t.Error("a differing isTokenShared must be out of date")
	}

	wantBody := sonar.JiraOrganizationBinding{"sonarOrganizationUuid": "org", "isTokenShared": false}
	if diff := cmp.Diff(wantBody, GenerateJiraOrganizationBindingUpdate(spec)); diff != "" {
		t.Errorf("GenerateJiraOrganizationBindingUpdate() mismatch (-want +got):\n%s", diff)
	}
}

// TestJiraProjectBindingHelpers tests JiraProjectBindingHelpers.
func TestJiraProjectBindingHelpers(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.JiraProjectBindingParameters{JiraOrganizationBindingID: new("org"), JiraProjectKey: "ABC"}

	if got := GenerateJiraProjectBindingOptions("project"); got.SonarProjectId != "project" {
		t.Errorf("GenerateJiraProjectBindingOptions() = %+v", got)
	}

	wantBody := sonar.JiraProjectBinding{"sonarProjectId": "project", "jiraOrganizationId": "org", "jiraProjectKey": "ABC"}
	if diff := cmp.Diff(wantBody, GenerateJiraProjectBindingBody("project", spec)); diff != "" {
		t.Errorf("GenerateJiraProjectBindingBody() mismatch (-want +got):\n%s", diff)
	}

	observation := GenerateJiraProjectBindingObservation(&JiraProjectBinding{ID: "id", SonarProjectID: "project", JiraOrganizationID: "org", JiraProjectKey: "ABC"})
	want := v1alpha1.JiraProjectBindingObservation{ID: "id", SonarProjectID: "project", JiraOrganizationBindingID: "org", JiraProjectKey: "ABC"}

	if diff := cmp.Diff(want, observation); diff != "" {
		t.Errorf("GenerateJiraProjectBindingObservation() mismatch (-want +got):\n%s", diff)
	}

	cases := map[string]struct {
		observation v1alpha1.JiraProjectBindingObservation
		want        bool
	}{
		"UpToDate":            {observation: want, want: true},
		"ProjectKeyDiffers":   {observation: v1alpha1.JiraProjectBindingObservation{JiraOrganizationBindingID: "org", JiraProjectKey: "XYZ"}},
		"OrganizationDiffers": {observation: v1alpha1.JiraProjectBindingObservation{JiraOrganizationBindingID: "other", JiraProjectKey: "ABC"}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsJiraProjectBindingUpToDate(spec, &tc.observation); got != tc.want {
				t.Errorf("IsJiraProjectBindingUpToDate() = %t, want %t", got, tc.want)
			}
		})
	}
}
