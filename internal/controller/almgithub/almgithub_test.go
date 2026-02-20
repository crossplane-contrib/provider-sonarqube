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

package almgithub

import (
	"context"
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/fake"
)

type notAlmGithub struct {
	resource.Managed
}

// mockHTTPResponse returns a mock HTTP response for testing.
func mockHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
	}
}

// testSecretRef returns a SecretKeySelector pointing to the test secret.
func testSecretRef(name, namespace, key string) xpv1.SecretKeySelector {
	return xpv1.SecretKeySelector{
		SecretReference: xpv1.SecretReference{
			Name:      name,
			Namespace: namespace,
		},
		Key: key,
	}
}

// newFakeKubeClient creates a fake kube client with the given secrets pre-loaded.
func newFakeKubeClient(secrets ...*corev1.Secret) client.Client {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	objs := make([]client.Object, len(secrets))
	for i, s := range secrets {
		objs[i] = s
	}

	return clientfake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

// testSecret creates a secret for testing.
func testSecret(name, namespace string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

// defaultAlmGithubParams returns AlmGithubParameters with secret refs for testing.
func defaultAlmGithubParams() v1alpha1.AlmGithubParameters {
	return v1alpha1.AlmGithubParameters{
		Key:                    "my-github",
		AppID:                  "12345",
		ClientID:               "client-id",
		ClientSecretSecretRef:  testSecretRef("github-secrets", "default", "client-secret"),
		PrivateKeySecretRef:    testSecretRef("github-secrets", "default", "private-key"),
		URL:                    "https://api.github.com",
		WebhookSecretSecretRef: nil,
	}
}

func TestObserve(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		client *fake.MockAlmClient
		args   args
		want   want
	}{
		"NotAlmGithubError": {
			client: &fake.MockAlmClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notAlmGithub{},
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.New(errNotAlmGithub),
			},
		},
		"EmptyExternalNameReturnsNotExists": {
			client: &fake.MockAlmClient{},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.AlmGithub{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test-alm",
						Annotations: map[string]string{},
					},
				},
			},
			want: want{
				o:   managed.ExternalObservation{ResourceExists: false},
				err: nil,
			},
		},
		"ListDefinitionsFailsReturnsError": {
			client: &fake.MockAlmClient{
				ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
					return nil, nil, errors.New("api error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.AlmGithub {
					ag := &v1alpha1.AlmGithub{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "test-alm",
							Annotations: map[string]string{},
						},
					}
					meta.SetExternalName(ag, "my-github")

					return ag
				}(),
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errors.New("api error"), errObserveAlmGithub),
			},
		},
		"DefinitionNotFoundReturnsNotExists": {
			client: &fake.MockAlmClient{
				ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
					return &sonar.AlmSettingsListDefinitions{
						Github: []sonar.GithubDefinition{},
					}, nil, nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.AlmGithub {
					ag := &v1alpha1.AlmGithub{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "test-alm",
							Annotations: map[string]string{},
						},
					}
					meta.SetExternalName(ag, "my-github")

					return ag
				}(),
			},
			want: want{
				o:   managed.ExternalObservation{ResourceExists: false},
				err: nil,
			},
		},
		"SuccessfulObserveResourceUpToDate": {
			client: &fake.MockAlmClient{
				ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
					return &sonar.AlmSettingsListDefinitions{
						Github: []sonar.GithubDefinition{
							{
								Key:      "my-github",
								AppID:    "12345",
								ClientID: "client-id",
								URL:      "https://api.github.com",
							},
						},
					}, nil, nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.AlmGithub {
					ag := &v1alpha1.AlmGithub{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "test-alm",
							Annotations: map[string]string{},
						},
						Spec: v1alpha1.AlmGithubSpec{
							ForProvider: defaultAlmGithubParams(),
						},
					}
					meta.SetExternalName(ag, "my-github")

					return ag
				}(),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				err: nil,
			},
		},
		"ResourceNotUpToDateWhenURLDiffers": {
			client: &fake.MockAlmClient{
				ListDefinitionsFn: func() (*sonar.AlmSettingsListDefinitions, *http.Response, error) {
					return &sonar.AlmSettingsListDefinitions{
						Github: []sonar.GithubDefinition{
							{
								Key:      "my-github",
								AppID:    "12345",
								ClientID: "client-id",
								URL:      "https://old-api.github.com",
							},
						},
					}, nil, nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.AlmGithub {
					ag := &v1alpha1.AlmGithub{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "test-alm",
							Annotations: map[string]string{},
						},
						Spec: v1alpha1.AlmGithubSpec{
							ForProvider: defaultAlmGithubParams(),
						},
					}
					meta.SetExternalName(ag, "my-github")

					return ag
				}(),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: false,
				},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{almClient: tc.client}
			got, err := e.Observe(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("Observe() error mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Observe() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalCreation
		err error
	}

	kubeClient := newFakeKubeClient(testSecret("github-secrets", "default", map[string][]byte{
		"client-secret": []byte("my-client-secret"),
		"private-key":   []byte("my-private-key"),
	}))

	cases := map[string]struct {
		client *fake.MockAlmClient
		kube   client.Client
		args   args
		want   want
	}{
		"NotAlmGithubError": {
			client: &fake.MockAlmClient{},
			kube:   kubeClient,
			args: args{
				ctx: context.Background(),
				mg:  &notAlmGithub{},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.New(errNotAlmGithub),
			},
		},
		"CreateFails": {
			client: &fake.MockAlmClient{
				CreateGithubFn: func(opt *sonar.AlmSettingsCreateGithubOption) (*http.Response, error) {
					return nil, errors.New("create error")
				},
			},
			kube: kubeClient,
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.AlmGithub{
					ObjectMeta: metav1.ObjectMeta{Name: "test-alm", Namespace: "default"},
					Spec: v1alpha1.AlmGithubSpec{
						ForProvider: defaultAlmGithubParams(),
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errors.New("create error"), errCreateAlmGithub),
			},
		},
		"SuccessfulCreate": {
			client: &fake.MockAlmClient{
				CreateGithubFn: func(opt *sonar.AlmSettingsCreateGithubOption) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
			},
			kube: kubeClient,
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.AlmGithub{
					ObjectMeta: metav1.ObjectMeta{Name: "test-alm", Namespace: "default"},
					Spec: v1alpha1.AlmGithubSpec{
						ForProvider: defaultAlmGithubParams(),
					},
				},
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{almClient: tc.client, kube: tc.kube}
			got, err := e.Create(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("Create() error mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Create() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateSetsExternalNameToKey(t *testing.T) {
	t.Parallel()

	kubeClient := newFakeKubeClient(testSecret("github-secrets", "default", map[string][]byte{
		"client-secret": []byte("my-client-secret"),
		"private-key":   []byte("my-private-key"),
	}))

	almClient := &fake.MockAlmClient{
		CreateGithubFn: func(opt *sonar.AlmSettingsCreateGithubOption) (*http.Response, error) {
			return mockHTTPResponse(), nil
		},
	}

	params := defaultAlmGithubParams()
	params.Key = "my-github-key"

	ag := &v1alpha1.AlmGithub{
		ObjectMeta: metav1.ObjectMeta{Name: "k8s-resource-name", Namespace: "default"},
		Spec: v1alpha1.AlmGithubSpec{
			ForProvider: params,
		},
	}

	e := &external{almClient: almClient, kube: kubeClient}

	_, err := e.Create(context.Background(), ag)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify external name is set to the key
	externalName := meta.GetExternalName(ag)
	if externalName != "my-github-key" {
		t.Errorf("Expected external name 'my-github-key', got '%s'", externalName)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalUpdate
		err error
	}

	kubeClient := newFakeKubeClient(testSecret("github-secrets", "default", map[string][]byte{
		"client-secret": []byte("my-client-secret"),
		"private-key":   []byte("my-private-key"),
	}))

	cases := map[string]struct {
		client *fake.MockAlmClient
		kube   client.Client
		args   args
		want   want
	}{
		"NotAlmGithubError": {
			client: &fake.MockAlmClient{},
			kube:   kubeClient,
			args: args{
				ctx: context.Background(),
				mg:  &notAlmGithub{},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.New(errNotAlmGithub),
			},
		},
		"UpdateFails": {
			client: &fake.MockAlmClient{
				UpdateGithubFn: func(opt *sonar.AlmSettingsUpdateGithubOption) (*http.Response, error) {
					return nil, errors.New("update error")
				},
			},
			kube: kubeClient,
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.AlmGithub{
					ObjectMeta: metav1.ObjectMeta{Name: "test-alm", Namespace: "default"},
					Spec: v1alpha1.AlmGithubSpec{
						ForProvider: defaultAlmGithubParams(),
					},
				},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Wrap(errors.New("update error"), errUpdateAlmGithub),
			},
		},
		"SuccessfulUpdate": {
			client: &fake.MockAlmClient{
				UpdateGithubFn: func(opt *sonar.AlmSettingsUpdateGithubOption) (*http.Response, error) {
					return mockHTTPResponse(), nil
				},
			},
			kube: kubeClient,
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.AlmGithub{
					ObjectMeta: metav1.ObjectMeta{Name: "test-alm", Namespace: "default"},
					Spec: v1alpha1.AlmGithubSpec{
						ForProvider: defaultAlmGithubParams(),
					},
				},
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{almClient: tc.client, kube: tc.kube}
			got, err := e.Update(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("Update() error mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Update() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalDelete
		err error
	}

	cases := map[string]struct {
		client *fake.MockAlmClient
		args   args
		want   want
	}{
		"NotAlmGithubError": {
			client: &fake.MockAlmClient{},
			args: args{
				ctx: context.Background(),
				mg:  &notAlmGithub{},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: errors.New(errNotAlmGithub),
			},
		},
		"EmptyExternalNameDoesNothing": {
			client: &fake.MockAlmClient{},
			args: args{
				ctx: context.Background(),
				mg: &v1alpha1.AlmGithub{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test-alm",
						Annotations: map[string]string{},
					},
				},
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: nil,
			},
		},
		"SuccessfulDelete": {
			client: &fake.MockAlmClient{
				DeleteFn: func(opt *sonar.AlmSettingsDeleteOption) (*http.Response, error) {
					if opt.Key != "my-github" {
						return nil, errors.New("expected key 'my-github' but got: " + opt.Key)
					}

					return mockHTTPResponse(), nil
				},
			},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.AlmGithub {
					ag := &v1alpha1.AlmGithub{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "test-alm",
							Annotations: map[string]string{},
						},
					}
					meta.SetExternalName(ag, "my-github")

					return ag
				}(),
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: nil,
			},
		},
		"DeleteFails": {
			client: &fake.MockAlmClient{
				DeleteFn: func(opt *sonar.AlmSettingsDeleteOption) (*http.Response, error) {
					return nil, errors.New("delete error")
				},
			},
			args: args{
				ctx: context.Background(),
				mg: func() *v1alpha1.AlmGithub {
					ag := &v1alpha1.AlmGithub{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "test-alm",
							Annotations: map[string]string{},
						},
					}
					meta.SetExternalName(ag, "my-github")

					return ag
				}(),
			},
			want: want{
				o:   managed.ExternalDelete{},
				err: errors.Wrap(errors.New("delete error"), errDeleteAlmGithub),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := &external{almClient: tc.client}
			got, err := e.Delete(tc.args.ctx, tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, cmp.Comparer(errComparer)); diff != "" {
				t.Errorf("Delete() error mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Delete() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{almClient: &fake.MockAlmClient{}}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() error = %v, want nil", err)
	}
}

// errComparer compares errors by their message.
func errComparer(a, b error) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return a.Error() == b.Error()
}
