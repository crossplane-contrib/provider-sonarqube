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

package jiraorganizationbinding

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
	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
	sonarfake "github.com/crossplane/provider-sonarqube/internal/fake"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// testID is the ID of the managed binding.
	testID = "binding-id"
	// testOrg is the organization identifier used by tests.
	testOrg = "00000000-0000-4000-0000-000000000000"
)

// errBoom is the error returned by failing mocks.
var errBoom = errors.New("boom")

// response returns a response with the given status code.
func response(status int) *http.Response {
	return &http.Response{StatusCode: status, Body: http.NoBody}
}

// newBinding returns a JiraOrganizationBinding with the given external name.
func newBinding(externalName string, shared *bool) *v1alpha1.JiraOrganizationBinding {
	binding := &v1alpha1.JiraOrganizationBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "binding", Namespace: "default"},
		Spec: v1alpha1.JiraOrganizationBindingSpec{
			ForProvider: v1alpha1.JiraOrganizationBindingParameters{SonarOrganizationUUID: testOrg, IsTokenShared: shared},
		},
	}
	if externalName != "" {
		meta.SetExternalName(binding, externalName)
	}

	return binding
}

// getting returns a client holding the binding with the given ID.
func getting(id string) *sonarfake.MockJiraClient {
	return &sonarfake.MockJiraClient{
		GetOrganizationBindingFn: func(opt *sonar.JiraOrganizationBindingOptions) (*sonar.JiraOrganizationBinding, *http.Response, error) {
			if opt.SonarOrganizationUuid != testOrg {
				return nil, response(http.StatusBadRequest), errBoom
			}

			return &sonar.JiraOrganizationBinding{"id": id, "isTokenShared": true}, response(http.StatusOK), nil
		},
	}
}

// failingGet returns a client whose retrieval fails with the given status.
func failingGet(status int) *sonarfake.MockJiraClient {
	return &sonarfake.MockJiraClient{
		GetOrganizationBindingFn: func(*sonar.JiraOrganizationBindingOptions) (*sonar.JiraOrganizationBinding, *http.Response, error) {
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

	deleted := newBinding(testID, new(false))
	deleted.SetDeletionTimestamp(&metav1.Time{Time: time.Unix(1, 0)})

	cases := map[string]struct {
		client  *sonarfake.MockJiraClient
		binding *v1alpha1.JiraOrganizationBinding
		want    managed.ExternalObservation
		wantErr string
	}{
		"NoExternalName": {client: &sonarfake.MockJiraClient{}, binding: newBinding("", nil)},
		"NotFound":       {client: failingGet(http.StatusNotFound), binding: newBinding(testID, nil)},
		"EmptyResponse":  {client: getting(""), binding: newBinding(testID, nil)},
		"OtherBinding":   {client: getting("other"), binding: newBinding(testID, nil)},
		"Error":          {client: failingGet(http.StatusForbidden), binding: newBinding(testID, nil), wantErr: errGetBinding},
		"LateInitialized": {
			client:  getting(testID),
			binding: newBinding(testID, nil),
			want:    managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true},
		},
		"OutOfDate": {
			client:  getting(testID),
			binding: newBinding(testID, new(false)),
			want:    managed.ExternalObservation{ResourceExists: true},
		},
		"BeingDeleted": {
			client:  getting(testID),
			binding: deleted,
			want:    managed.ExternalObservation{ResourceExists: true},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}

			got, err := e.Observe(context.Background(), tc.binding)
			checkError(t, tc.wantErr, err)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Observe() mismatch (-want +got):\n%s", diff)
			}

			if tc.want.ResourceExists {
				if tc.binding.Status.AtProvider.ID != testID {
					t.Errorf("observation id = %q, want %q", tc.binding.Status.AtProvider.ID, testID)
				}

				if tc.binding.GetCondition(xpv1.TypeReady).Status != corev1.ConditionTrue {
					t.Error("resource is not Ready")
				}
			}
		})
	}
}

// TestCreate tests Create.
func TestCreate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client  *sonarfake.MockJiraClient
		wantErr string
	}{
		"Exists":  {client: getting(testID), wantErr: "already has a Jira organization binding (id \"binding-id\")"},
		"Missing": {client: failingGet(http.StatusNotFound), wantErr: errCannotCreate},
		"Error":   {client: failingGet(http.StatusInternalServerError), wantErr: errGetBinding},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}
			binding := newBinding("", nil)

			_, err := e.Create(context.Background(), binding)
			checkError(t, tc.wantErr, err)

			if meta.GetExternalName(binding) != "" {
				t.Error("Create() must not set the external name")
			}
		})
	}
}

// TestUpdate tests Update.
func TestUpdate(t *testing.T) {
	t.Parallel()

	updating := func(err error) *sonarfake.MockJiraClient {
		return &sonarfake.MockJiraClient{
			UpdateOrganizationBindingFn: func(body sonar.JiraOrganizationBinding) (*sonar.JiraOrganizationBinding, *http.Response, error) {
				if body["isTokenShared"] != true || body["sonarOrganizationUuid"] != testOrg {
					return nil, response(http.StatusBadRequest), errors.New("unexpected body")
				}

				return nil, response(http.StatusOK), err
			},
		}
	}

	cases := map[string]struct {
		client  *sonarfake.MockJiraClient
		shared  *bool
		wantErr string
	}{
		"NothingToUpdate": {client: &sonarfake.MockJiraClient{}},
		"Updated":         {client: updating(nil), shared: new(true)},
		"Error":           {client: updating(errBoom), shared: new(true), wantErr: errUpdateBinding},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{client: tc.client}

			_, err := e.Update(context.Background(), newBinding(testID, tc.shared))
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
		deleteStatus int
		wantDeleted  bool
		wantErr      string
	}{
		"NoExternalName": {client: &sonarfake.MockJiraClient{}},
		"AlreadyGone":    {externalName: testID, client: failingGet(http.StatusNotFound)},
		"OtherBinding":   {externalName: testID, client: getting("other")},
		"GetError":       {externalName: testID, client: failingGet(http.StatusInternalServerError), wantErr: errGetBinding},
		"Deleted":        {externalName: testID, client: getting(testID), deleteStatus: http.StatusNoContent, wantDeleted: true},
		"DeleteNotFound": {externalName: testID, client: getting(testID), deleteStatus: http.StatusNotFound, wantDeleted: true},
		"DeleteError": {
			externalName: testID, client: getting(testID), deleteStatus: http.StatusInternalServerError,
			wantDeleted: true, wantErr: errDeleteBinding,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			deleted := false
			tc.client.DeleteOrganizationBindingFn = func(*sonar.JiraOrganizationBindingOptions) (*http.Response, error) {
				deleted = true

				var err error
				if tc.deleteStatus >= http.StatusBadRequest {
					err = errBoom
				}

				return response(tc.deleteStatus), err //nolint:nilnil // the endpoint answers with both
			}

			e := &external{client: tc.client}

			_, err := e.Delete(context.Background(), newBinding(tc.externalName, nil))
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
				kube:         kube,
				usage:        resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
				newServiceFn: func(common.Config) integration.JiraClient { return &sonarfake.MockJiraClient{} },
			}

			binding := newBinding(testID, nil)
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

		_, err := c.Connect(context.Background(), newBinding(testID, nil))
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

	if diff := cmp.Diff([]schema.GroupVersionKind{v1alpha1.JiraOrganizationBindingGroupVersionKind}, gate.gvks); diff != "" {
		t.Errorf("SetupGated() GVKs mismatch (-want +got):\n%s", diff)
	}
}
