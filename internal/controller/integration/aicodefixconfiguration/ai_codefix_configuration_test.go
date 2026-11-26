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

package aicodefixconfiguration

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
	"github.com/crossplane/provider-sonarqube/internal/fake"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// mockGate is a mock implementation of the feature gate interface.
type mockGate struct {
	callback func()
	gvks     []schema.GroupVersionKind
}

// Register records the callback and GVKs.
func (m *mockGate) Register(callback func(), gvks ...schema.GroupVersionKind) {
	m.callback = callback
	m.gvks = append(m.gvks, gvks...)
}

// Set implements the gate interface (no-op for tests).
func (m *mockGate) Set(_ schema.GroupVersionKind, _ bool) bool {
	return false
}

// newTestConfiguration returns an AICodeFixConfiguration enabling AI
// CodeFix for the given projects.
func newTestConfiguration(projects ...string) *v1alpha1.AICodeFixConfiguration {
	return &v1alpha1.AICodeFixConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "ai-codefix", Namespace: "default", UID: types.UID("ai-codefix-uid")},
		Spec: v1alpha1.AICodeFixConfigurationSpec{
			ForProvider: v1alpha1.AICodeFixConfigurationParameters{
				Enablement:         v1alpha1.AICodeFixEnabledForSomeProjects,
				EnabledProjectKeys: projects,
			},
		},
	}
}

// enablementClient returns a client reporting the given enablement and
// recording the enablement it is sent.
func enablementClient(enablement string, projects []string, sent **integration.AICodeFixEnablementOptions) *fake.MockFixSuggestionsClient {
	return &fake.MockFixSuggestionsClient{
		GetEnablementFn: func() (*sonar.FixSuggestionsFeatureEnablement, *http.Response, error) {
			return &sonar.FixSuggestionsFeatureEnablement{Enablement: enablement, EnabledProjectKeys: projects}, &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		},
		SetEnablementFn: func(opt *integration.AICodeFixEnablementOptions) (*http.Response, error) {
			if sent != nil {
				*sent = opt
			}

			return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
		},
	}
}

// checkError verifies that an error matches (or does not match) expectations.
func checkError(t *testing.T, method, wantErrSubstr string, gotErr error) {
	t.Helper()

	switch {
	case wantErrSubstr == "" && gotErr != nil:
		t.Errorf("%s() unexpected error: %v", method, gotErr)
	case wantErrSubstr != "" && gotErr == nil:
		t.Errorf("%s() expected error containing %q, got nil", method, wantErrSubstr)
	case wantErrSubstr != "" && !strings.Contains(gotErr.Error(), wantErrSubstr):
		t.Errorf("%s() error = %q, want substring %q", method, gotErr.Error(), wantErrSubstr)
	}
}

// TestObserve tests observing the AI CodeFix enablement.
func TestObserve(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		client       *fake.MockFixSuggestionsClient
		deleting     bool
		wantExists   bool
		wantUpToDate bool
		wantErr      string
	}{
		"GetError": {
			client:  &fake.MockFixSuggestionsClient{},
			wantErr: errGetEnablement,
		},
		"UpToDate": {
			client:       enablementClient(v1alpha1.AICodeFixEnabledForSomeProjects, []string{"b", "a"}, nil),
			wantExists:   true,
			wantUpToDate: true,
		},
		"Drifted": {
			client:     enablementClient(v1alpha1.AICodeFixDisabled, nil, nil),
			wantExists: true,
		},
		"DeletingStillEnabled": {
			client:     enablementClient(v1alpha1.AICodeFixEnabledForAllProjects, nil, nil),
			deleting:   true,
			wantExists: true,
		},
		"DeletingDisabled": {
			client:   enablementClient(v1alpha1.AICodeFixDisabled, nil, nil),
			deleting: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mg := newTestConfiguration("a", "b")
			if tc.deleting {
				mg.SetDeletionTimestamp(new(metav1.Now()))
			}

			got, err := (&external{client: tc.client}).Observe(context.Background(), mg)
			checkError(t, "Observe", tc.wantErr, err)

			if got.ResourceExists != tc.wantExists || got.ResourceUpToDate != tc.wantUpToDate {
				t.Errorf("Observe() = exists %v, upToDate %v, want %v, %v", got.ResourceExists, got.ResourceUpToDate, tc.wantExists, tc.wantUpToDate)
			}
		})
	}
}

// TestCreateUpdateDelete tests the enablement sent by each write.
func TestCreateUpdateDelete(t *testing.T) {
	t.Parallel()

	want := &integration.AICodeFixEnablementOptions{Enablement: v1alpha1.AICodeFixEnabledForSomeProjects, EnabledProjectKeys: []string{"a"}}

	var sent *integration.AICodeFixEnablementOptions

	e := &external{client: enablementClient(v1alpha1.AICodeFixDisabled, nil, &sent)}

	_, err := e.Create(context.Background(), newTestConfiguration("a"))
	if err != nil || !reflect.DeepEqual(sent, want) {
		t.Errorf("Create() sent %+v, error %v, want %+v", sent, err, want)
	}

	sent = nil

	_, err = e.Update(context.Background(), newTestConfiguration("a"))
	if err != nil || !reflect.DeepEqual(sent, want) {
		t.Errorf("Update() sent %+v, error %v, want %+v", sent, err, want)
	}

	_, err = e.Delete(context.Background(), newTestConfiguration("a"))
	if err != nil || sent.Enablement != v1alpha1.AICodeFixDisabled || sent.EnabledProjectKeys != nil {
		t.Errorf("Delete() sent %+v, error %v, want DISABLED", sent, err)
	}
}

// TestWriteErrors tests that write failures are wrapped.
func TestWriteErrors(t *testing.T) {
	t.Parallel()

	e := &external{client: &fake.MockFixSuggestionsClient{
		SetEnablementFn: func(*integration.AICodeFixEnablementOptions) (*http.Response, error) {
			return nil, errors.New("boom")
		},
	}}

	_, err := e.Create(context.Background(), newTestConfiguration())
	checkError(t, "Create", errSetEnablement, err)

	_, err = e.Update(context.Background(), newTestConfiguration())
	checkError(t, "Update", errSetEnablement, err)

	_, err = e.Delete(context.Background(), newTestConfiguration())
	checkError(t, "Delete", errDisable, err)

	err = e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() error = %v", err)
	}
}

// newScheme returns a scheme with the provider and integration types.
func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()

	for _, add := range []func(*runtime.Scheme) error{apisv1alpha1.SchemeBuilder.AddToScheme, v1alpha1.SchemeBuilder.AddToScheme, corev1.AddToScheme} {
		err := add(scheme)
		if err != nil {
			t.Fatalf("AddToScheme() error = %v", err)
		}
	}

	return scheme
}

// TestConnect tests connecting to SonarQube.
func TestConnect(t *testing.T) {
	t.Parallel()

	t.Run("TrackUsageError", func(t *testing.T) {
		t.Parallel()

		kube := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
		c := &connector{kube: kube, usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{})}

		_, err := c.Connect(context.Background(), newTestConfiguration())
		checkError(t, "Connect", errTrackPCUsage, err)
	})

	t.Run("GetConfigError", func(t *testing.T) {
		t.Parallel()

		kube := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
		mg := newTestConfiguration()
		mg.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing", Kind: "ProviderConfig"})

		c := &connector{kube: kube, usage: resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{})}

		_, err := c.Connect(context.Background(), mg)
		checkError(t, "Connect", errGetPC, err)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		providerConfig := &apisv1alpha1.ProviderConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "pc", Namespace: "default"},
			Spec: apisv1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
				Token: &apisv1alpha1.ProviderCredentials{
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{Name: "prov-secret", Namespace: "default"},
							Key:             "token",
						},
					},
					Source: xpv1.CredentialsSourceSecret,
				},
			},
		}
		providerSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "prov-secret", Namespace: "default"},
			Data:       map[string][]byte{"token": []byte("t")},
		}

		kube := fakekube.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(providerConfig, providerSecret).Build()
		mg := newTestConfiguration()
		mg.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

		c := &connector{
			kube:         kube,
			usage:        resource.NewProviderConfigUsageTracker(kube, &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: func(common.Config) integration.FixSuggestionsClient { return &fake.MockFixSuggestionsClient{} },
		}

		got, err := c.Connect(context.Background(), mg)
		if err != nil || got == nil {
			t.Fatalf("Connect() = %v, %v", got, err)
		}
	})
}

// TestSetupGated tests that SetupGated registers the AICodeFixConfiguration
// GVK and that the callback panics when setup fails.
func TestSetupGated(t *testing.T) {
	t.Parallel()

	g := &mockGate{}
	o := helpers.Options{Options: controller.DefaultOptions()}
	o.Gate = g

	err := SetupGated(nil, o)
	if err != nil {
		t.Fatalf("SetupGated() error = %v", err)
	}

	if len(g.gvks) != 1 || g.gvks[0] != v1alpha1.AICodeFixConfigurationGroupVersionKind {
		t.Fatalf("SetupGated() GVKs = %v", g.gvks)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected callback to panic when setup fails")
		}
	}()

	g.callback()
}
