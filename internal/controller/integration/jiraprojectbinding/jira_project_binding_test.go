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

package jiraprojectbinding

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
	sonarfake "github.com/crossplane/provider-sonarqube/internal/fake"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// testID is the ID of the managed binding.
	testID = "binding-id"
	// testProject is the key of the bound project.
	testProject = "project-key"
	// testUUID is the ID of the bound project.
	testUUID = "project-uuid"
	// testOrg is the ID of the Jira organization binding.
	testOrg = "organization-binding-id"
	// testJiraKey is the key of the bound Jira project.
	testJiraKey = "ABC"
)

// errBoom is the error returned by failing mocks.
var errBoom = errors.New("boom")

// response returns a response with the given status code.
func response(status int) *http.Response {
	return &http.Response{StatusCode: status, Body: http.NoBody}
}

// newBinding returns a JiraProjectBinding with the given external name.
func newBinding(externalName string) *v1alpha1.JiraProjectBinding {
	binding := &v1alpha1.JiraProjectBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "binding", Namespace: "default"},
		Spec: v1alpha1.JiraProjectBindingSpec{
			ForProvider: v1alpha1.JiraProjectBindingParameters{
				ProjectKey:                new(testProject),
				JiraOrganizationBindingID: new(testOrg),
				JiraProjectKey:            testJiraKey,
			},
		},
	}
	if externalName != "" {
		meta.SetExternalName(binding, externalName)
	}

	return binding
}

// searching returns a projects client answering searches with the given
// components.
func searching(components ...sonar.ProjectSearchComponent) *sonarfake.MockProjectsClient {
	return &sonarfake.MockProjectsClient{
		SearchFn: func(*sonar.ProjectsSearchOptions) (*sonar.ProjectsSearch, *http.Response, error) {
			return &sonar.ProjectsSearch{Components: components}, response(http.StatusOK), nil
		},
	}
}

// foundProject returns a projects client holding the test project.
func foundProject() *sonarfake.MockProjectsClient {
	return searching(
		sonar.ProjectSearchComponent{Key: "other", ProjectUuid: "other-uuid"},
		sonar.ProjectSearchComponent{Key: testProject, ProjectUuid: testUUID},
	)
}

// failingSearch returns a projects client whose search fails.
func failingSearch() *sonarfake.MockProjectsClient {
	return &sonarfake.MockProjectsClient{
		SearchFn: func(*sonar.ProjectsSearchOptions) (*sonar.ProjectsSearch, *http.Response, error) {
			return nil, response(http.StatusForbidden), errBoom
		},
	}
}

// emptySearch returns a projects client answering searches with nothing.
func emptySearch() *sonarfake.MockProjectsClient {
	return &sonarfake.MockProjectsClient{
		SearchFn: func(*sonar.ProjectsSearchOptions) (*sonar.ProjectsSearch, *http.Response, error) {
			return nil, response(http.StatusOK), nil
		},
	}
}

// remote returns the binding SonarQube holds, with the given ID.
func remote(id string) sonar.JiraProjectBinding {
	return sonar.JiraProjectBinding{
		"id": id, "sonarProjectId": testUUID, "jiraOrganizationId": testOrg, "jiraProjectKey": testJiraKey,
	}
}

// getting returns a client holding a binding with the given ID.
func getting(id string) *sonarfake.MockJiraClient {
	return &sonarfake.MockJiraClient{
		GetProjectBindingFn: func(opt *sonar.JiraProjectBindingOptions) (*sonar.JiraProjectBinding, *http.Response, error) {
			if opt.SonarProjectId != testUUID {
				return nil, response(http.StatusBadRequest), errBoom
			}

			return new(remote(id)), response(http.StatusOK), nil
		},
	}
}

// failingGet returns a client whose retrieval fails with the given status.
func failingGet(status int) *sonarfake.MockJiraClient {
	return &sonarfake.MockJiraClient{
		GetProjectBindingFn: func(*sonar.JiraProjectBindingOptions) (*sonar.JiraProjectBinding, *http.Response, error) {
			return nil, response(status), errBoom
		},
	}
}

// checkError verifies that err matches the wanted substring.
func checkError(t *testing.T, want string, err error) {
	t.Helper()

	switch {
	case want == "" && err != nil:
		t.Errorf("unexpected error: %v", err)
	case want != "" && (err == nil || !strings.Contains(err.Error(), want)):
		t.Errorf("error = %v, want substring %q", err, want)
	}
}

// TestObserve tests Observe.
func TestObserve(t *testing.T) {
	t.Parallel()

	outdated := newBinding(testID)
	outdated.Spec.ForProvider.JiraProjectKey = "XYZ"

	unresolved := newBinding(testID)
	unresolved.Spec.ForProvider.ProjectKey = nil

	deleted := newBinding(testID)
	deleted.SetDeletionTimestamp(&metav1.Time{Time: time.Unix(1, 0)})

	cases := map[string]struct {
		client   *sonarfake.MockJiraClient
		projects *sonarfake.MockProjectsClient
		binding  *v1alpha1.JiraProjectBinding
		want     managed.ExternalObservation
		wantErr  string
	}{
		"NoExternalName": {binding: newBinding("")},
		"NoProjectKey":   {binding: unresolved},
		"ProjectGone":    {projects: searching(), binding: newBinding(testID)},
		"SearchError":    {projects: failingSearch(), binding: newBinding(testID), wantErr: errSearchProject},
		"EmptySearch":    {projects: emptySearch(), binding: newBinding(testID), wantErr: errEmptySearch},
		"NoProjectID":    {projects: searching(sonar.ProjectSearchComponent{Key: testProject}), binding: newBinding(testID), wantErr: "2025.6"},
		"NotFound":       {client: failingGet(http.StatusNotFound), projects: foundProject(), binding: newBinding(testID)},
		"EmptyResponse":  {client: getting(""), projects: foundProject(), binding: newBinding(testID)},
		"OtherBinding":   {client: getting("other"), projects: foundProject(), binding: newBinding(testID)},
		"GetError":       {client: failingGet(http.StatusForbidden), projects: foundProject(), binding: newBinding(testID), wantErr: errGetBinding},
		"UpToDate": {
			client: getting(testID), projects: foundProject(), binding: newBinding(testID),
			want: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
		},
		"OutOfDate": {
			client: getting(testID), projects: foundProject(), binding: outdated,
			want: managed.ExternalObservation{ResourceExists: true},
		},
		"BeingDeleted": {
			client: getting(testID), projects: foundProject(), binding: deleted,
			want: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client, projects: tc.projects}

			got, err := e.Observe(context.Background(), tc.binding)
			checkError(t, tc.wantErr, err)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Observe() mismatch (-want +got):\n%s", diff)
			}

			if tc.want.ResourceExists {
				if tc.binding.Status.AtProvider.SonarProjectID != testUUID {
					t.Errorf("observed project id = %q, want %q", tc.binding.Status.AtProvider.SonarProjectID, testUUID)
				}

				if tc.binding.GetCondition(xpv1.TypeReady).Status != corev1.ConditionTrue {
					t.Error("resource is not Ready")
				}
			}
		})
	}
}

// wantBody is the body binding the test project.
func wantBody() sonar.JiraProjectBinding {
	return sonar.JiraProjectBinding{"sonarProjectId": testUUID, "jiraOrganizationId": testOrg, "jiraProjectKey": testJiraKey}
}

// withPut makes client answer binding writes with result and err, checking the
// body sent.
func withPut(client *sonarfake.MockJiraClient, result sonar.JiraProjectBinding, err error) *sonarfake.MockJiraClient {
	client.CreateProjectBindingFn = func(body sonar.JiraProjectBinding) (*sonar.JiraProjectBinding, *http.Response, error) {
		if !cmp.Equal(body, wantBody()) {
			return nil, response(http.StatusBadRequest), errors.New("unexpected body")
		}

		return &result, response(http.StatusCreated), err
	}

	return client
}

// TestCreate tests Create.
func TestCreate(t *testing.T) {
	t.Parallel()

	missing := func() *sonarfake.MockJiraClient { return failingGet(http.StatusNotFound) }

	cases := map[string]struct {
		client   *sonarfake.MockJiraClient
		projects *sonarfake.MockProjectsClient
		wantName string
		wantErr  string
	}{
		"Created":         {client: withPut(missing(), remote(testID), nil), projects: foundProject(), wantName: testID},
		"SearchError":     {projects: failingSearch(), wantErr: errSearchProject},
		"ProjectNotFound": {projects: searching(), wantErr: "does not exist"},
		"AlreadyBound":    {client: getting("other"), projects: foundProject(), wantErr: "already has a Jira binding (id \"other\")"},
		"CreateError":     {client: withPut(missing(), nil, errBoom), projects: foundProject(), wantErr: errCreateBinding},
		"NoID":            {client: withPut(missing(), sonar.JiraProjectBinding{}, nil), projects: foundProject(), wantErr: errNoBindingID},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client, projects: tc.projects}
			binding := newBinding("")

			_, err := e.Create(context.Background(), binding)
			checkError(t, tc.wantErr, err)

			if got := meta.GetExternalName(binding); got != tc.wantName {
				t.Errorf("external name = %q, want %q", got, tc.wantName)
			}
		})
	}
}

// TestUpdate tests Update.
func TestUpdate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client   *sonarfake.MockJiraClient
		projects *sonarfake.MockProjectsClient
		wantErr  string
	}{
		"Updated":      {client: withPut(getting(testID), remote(testID), nil), projects: foundProject()},
		"SearchError":  {projects: failingSearch(), wantErr: errSearchProject},
		"ProjectGone":  {projects: searching(), wantErr: errBindingChanged},
		"OtherBinding": {client: getting("other"), projects: foundProject(), wantErr: errBindingChanged},
		"UpdateError":  {client: withPut(getting(testID), nil, errBoom), projects: foundProject(), wantErr: errUpdateBinding},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client, projects: tc.projects}

			_, err := e.Update(context.Background(), newBinding(testID))
			checkError(t, tc.wantErr, err)
		})
	}
}

// TestDelete tests Delete.
func TestDelete(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		externalName string
		client       *sonarfake.MockJiraClient
		projects     *sonarfake.MockProjectsClient
		deleteStatus int
		wantDeleted  bool
		wantErr      string
	}{
		"NoExternalName": {client: &sonarfake.MockJiraClient{}},
		"ProjectGone":    {externalName: testID, client: &sonarfake.MockJiraClient{}, projects: searching()},
		"AlreadyGone":    {externalName: testID, client: failingGet(http.StatusNotFound), projects: foundProject()},
		"OtherBinding":   {externalName: testID, client: getting("other"), projects: foundProject()},
		"GetError": {
			externalName: testID, client: failingGet(http.StatusInternalServerError), projects: foundProject(), wantErr: errGetBinding,
		},
		"Deleted": {
			externalName: testID, client: getting(testID), projects: foundProject(),
			deleteStatus: http.StatusNoContent, wantDeleted: true,
		},
		"DeleteNotFound": {
			externalName: testID, client: getting(testID), projects: foundProject(),
			deleteStatus: http.StatusNotFound, wantDeleted: true,
		},
		"DeleteError": {
			externalName: testID, client: getting(testID), projects: foundProject(),
			deleteStatus: http.StatusInternalServerError, wantDeleted: true, wantErr: errDeleteBinding,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			deleted := false
			tc.client.DeleteProjectBindingFn = func(opt *sonar.JiraProjectBindingOptions) (*http.Response, error) {
				deleted = opt.SonarProjectId == testUUID

				var err error
				if tc.deleteStatus >= http.StatusBadRequest {
					err = errBoom
				}

				return response(tc.deleteStatus), err //nolint:nilnil // the endpoint answers with both
			}

			e := &external{client: tc.client, projects: tc.projects}

			_, err := e.Delete(context.Background(), newBinding(tc.externalName))
			checkError(t, tc.wantErr, err)

			if deleted != tc.wantDeleted {
				t.Errorf("deleted = %t, want %t", deleted, tc.wantDeleted)
			}

			err = e.Disconnect(context.Background())
			if err != nil {
				t.Errorf("Disconnect() unexpected error: %v", err)
			}
		})
	}
}

// TestConnect tests Connect.
func TestConnect(t *testing.T) {
	t.Parallel()

	scheme := runtimeScheme(t)
	providerConfig := &apisv1alpha1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: "default"},
		Spec: apisv1alpha1.ProviderConfigSpec{
			BaseURL: "http://localhost:9000/api",
			Token: &apisv1alpha1.ProviderCredentials{
				Source: xpv1.CredentialsSourceSecret,
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{Name: "token", Namespace: "default"},
						Key:             "token",
					},
				},
			},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "token", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("value")},
	}

	cases := map[string]struct {
		providerConfig string
		wantErr        string
	}{
		"Connected":             {providerConfig: "config"},
		"MissingProviderConfig": {providerConfig: "missing", wantErr: errGetPC},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(providerConfig.DeepCopy(), secret.DeepCopy()).Build()
			c := &connector{
				kube:          kube,
				usage:         resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
				newServiceFn:  func(common.Config) integration.JiraClient { return &sonarfake.MockJiraClient{} },
				newProjectsFn: func(common.Config) instance.ProjectsClient { return &sonarfake.MockProjectsClient{} },
			}

			binding := newBinding(testID)
			binding.SetUID("uid")
			binding.Spec.ProviderConfigReference = &xpv1.ProviderConfigReference{Kind: "ProviderConfig", Name: tc.providerConfig}

			_, err := c.Connect(context.Background(), binding)
			checkError(t, tc.wantErr, err)
		})
	}

	t.Run("UsageError", func(t *testing.T) {
		t.Parallel()

		kube := fake.NewClientBuilder().WithScheme(scheme).Build()
		c := &connector{kube: kube, usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{})}

		_, err := c.Connect(context.Background(), newBinding(testID))
		checkError(t, errTrackPCUsage, err)
	})
}

// runtimeScheme returns a scheme holding the types Connect reads.
func runtimeScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{corev1.AddToScheme, apisv1alpha1.SchemeBuilder.AddToScheme, v1alpha1.SchemeBuilder.AddToScheme} {
		err := add(scheme)
		if err != nil {
			t.Fatalf("AddToScheme() unexpected error: %v", err)
		}
	}

	return scheme
}

// mockGate records the GVKs registered with the feature gate.
type mockGate struct {
	gvks []schema.GroupVersionKind
}

// Register implements the gate interface.
func (m *mockGate) Register(_ func(), gvks ...schema.GroupVersionKind) {
	m.gvks = append(m.gvks, gvks...)
}

// Set implements the gate interface.
func (m *mockGate) Set(_ schema.GroupVersionKind, _ bool) bool {
	return true
}

// TestSetupGated tests SetupGated.
func TestSetupGated(t *testing.T) {
	t.Parallel()

	gate := &mockGate{}
	options := helpers.Options{}
	options.Gate = gate

	err := SetupGated(nil, options)
	if err != nil {
		t.Fatalf("SetupGated() unexpected error: %v", err)
	}

	if diff := cmp.Diff([]schema.GroupVersionKind{v1alpha1.JiraProjectBindingGroupVersionKind}, gate.gvks); diff != "" {
		t.Errorf("SetupGated() GVKs mismatch (-want +got):\n%s", diff)
	}
}
