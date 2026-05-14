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

package common

import (
	"context"
	"strings"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource/fake"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/provider-sonarqube/apis/v1alpha1"
)

// newSonarTestClient creates a fake Kubernetes client for SonarQube testing.
func newSonarTestClient(objs ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = v1alpha1.SchemeBuilder.AddToScheme(scheme)

	return fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&v1alpha1.ProviderConfig{}, &v1alpha1.ClusterProviderConfig{}).
		Build()
}

// TestNewClient tests creating a new SonarQube client.
func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config    Config
		wantPanic bool
	}{
		"PersonalAccessTokenCreatesClient": {
			config: Config{
				AuthType: PersonalAccessToken,
				Token:    "my-token",
				BaseURL:  "http://localhost:9000",
			},
			wantPanic: false,
		},
		"BasicAuthCreatesClient": {
			config: Config{
				AuthType: BasicAuth,
				BasicAuth: &BasicAuthArgs{
					Username: "admin",
					Password: "admin",
				},
				BaseURL: "http://localhost:9000",
			},
			wantPanic: false,
		},
		"BasicAuthWithInsecureSkipVerify": {
			config: Config{
				AuthType: BasicAuth,
				BasicAuth: &BasicAuthArgs{
					Username: "admin",
					Password: "admin",
				},
				BaseURL:            "https://localhost:9000",
				InsecureSkipVerify: true,
			},
			wantPanic: false,
		},
		"PersonalAccessTokenWithInsecureSkipVerify": {
			config: Config{
				AuthType:           PersonalAccessToken,
				Token:              "my-token",
				BaseURL:            "https://localhost:9000",
				InsecureSkipVerify: true,
			},
			wantPanic: false,
		},
		"BasicAuthNilBasicAuthPanics": {
			config: Config{
				AuthType:  BasicAuth,
				BasicAuth: nil,
				BaseURL:   "http://localhost:9000",
			},
			wantPanic: true,
		},
		"UnsupportedAuthTypePanics": {
			config: Config{
				AuthType: "UnsupportedType",
				BaseURL:  "http://localhost:9000",
			},
			wantPanic: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("NewClient() expected panic, but did not panic")
					}
				}()

				NewClient(tc.config)
			} else {
				got := NewClient(tc.config)
				if got == nil {
					t.Errorf("NewClient() returned nil, want non-nil client")
				}
			}
		})
	}
}

// TestDetermineAuthType tests authentication type determination.
func TestDetermineAuthType(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec    v1alpha1.ProviderConfigSpec
		want    AuthType
		wantErr bool
	}{
		"TokenAuthWithValidSecret": {
			spec: v1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
				Token: &v1alpha1.ProviderCredentials{
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{
								Name:      "my-secret",
								Namespace: "default",
							},
							Key: "token",
						},
					},
					Source: xpv1.CredentialsSourceSecret,
				},
			},
			want:    PersonalAccessToken,
			wantErr: false,
		},
		"TokenAuthWithNonSecretSource": {
			spec: v1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
				Token: &v1alpha1.ProviderCredentials{
					Source: xpv1.CredentialsSourceNone,
				},
			},
			want:    "",
			wantErr: true,
		},
		"TokenAuthWithNilSecretRef": {
			spec: v1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
				Token: &v1alpha1.ProviderCredentials{
					Source: xpv1.CredentialsSourceSecret,
				},
			},
			want:    "",
			wantErr: true,
		},
		"BasicAuthWithValidSecrets": {
			spec: v1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
				Username: &v1alpha1.ProviderCredentials{
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{
								Name:      "my-secret",
								Namespace: "default",
							},
							Key: "username",
						},
					},
					Source: xpv1.CredentialsSourceSecret,
				},
				Password: &v1alpha1.ProviderCredentials{
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{
								Name:      "my-secret",
								Namespace: "default",
							},
							Key: "password",
						},
					},
					Source: xpv1.CredentialsSourceSecret,
				},
			},
			want:    BasicAuth,
			wantErr: false,
		},
		"BasicAuthUsernameNonSecretSource": {
			spec: v1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
				Username: &v1alpha1.ProviderCredentials{
					Source: xpv1.CredentialsSourceNone,
				},
				Password: &v1alpha1.ProviderCredentials{
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{
								Name:      "my-secret",
								Namespace: "default",
							},
							Key: "password",
						},
					},
					Source: xpv1.CredentialsSourceSecret,
				},
			},
			want:    "",
			wantErr: true,
		},
		"BasicAuthUsernameNilSecretRef": {
			spec: v1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
				Username: &v1alpha1.ProviderCredentials{
					Source: xpv1.CredentialsSourceSecret,
				},
				Password: &v1alpha1.ProviderCredentials{
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{
								Name:      "my-secret",
								Namespace: "default",
							},
							Key: "password",
						},
					},
					Source: xpv1.CredentialsSourceSecret,
				},
			},
			want:    "",
			wantErr: true,
		},
		"BasicAuthPasswordNonSecretSource": {
			spec: v1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
				Username: &v1alpha1.ProviderCredentials{
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{
								Name:      "my-secret",
								Namespace: "default",
							},
							Key: "username",
						},
					},
					Source: xpv1.CredentialsSourceSecret,
				},
				Password: &v1alpha1.ProviderCredentials{
					Source: xpv1.CredentialsSourceNone,
				},
			},
			want:    "",
			wantErr: true,
		},
		"BasicAuthPasswordNilSecretRef": {
			spec: v1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
				Username: &v1alpha1.ProviderCredentials{
					CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
						SecretRef: &xpv1.SecretKeySelector{
							SecretReference: xpv1.SecretReference{
								Name:      "my-secret",
								Namespace: "default",
							},
							Key: "username",
						},
					},
					Source: xpv1.CredentialsSourceSecret,
				},
				Password: &v1alpha1.ProviderCredentials{
					Source: xpv1.CredentialsSourceSecret,
				},
			},
			want:    "",
			wantErr: true,
		},
		"NoAuthMethodReturnsError": {
			spec: v1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
			},
			want:    "",
			wantErr: true,
		},
		"OnlyUsernameNoPasswordReturnsError": {
			spec: v1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
				Username: &v1alpha1.ProviderCredentials{
					Source: xpv1.CredentialsSourceSecret,
				},
			},
			want:    "",
			wantErr: true,
		},
		"OnlyPasswordNoUsernameReturnsError": {
			spec: v1alpha1.ProviderConfigSpec{
				BaseURL: "http://localhost:9000",
				Password: &v1alpha1.ProviderCredentials{
					Source: xpv1.CredentialsSourceSecret,
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := determineAuthType(tc.spec)
			if (err != nil) != tc.wantErr {
				t.Errorf("determineAuthType() error = %v, wantErr %v", err, tc.wantErr)

				return
			}

			if got != tc.want {
				t.Errorf("determineAuthType() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGetConfigNonModernManaged tests getting config from non-modern managed.
func TestGetConfigNonModernManaged(t *testing.T) {
	t.Parallel()

	kubeClient := newSonarTestClient()

	_, err := GetConfig(context.Background(), kubeClient, &fake.Managed{})
	if err == nil {
		t.Errorf("GetConfig() expected error for non-ModernManaged, got nil")
	}

	if !strings.Contains(err.Error(), "unknown managed resource type") {
		t.Errorf("GetConfig() error = %v, want error containing 'unknown managed resource type'", err)
	}
}

// getConfigTestCase holds test case data for config retrieval tests.
type getConfigTestCase struct {
	managed     *fake.ModernManaged
	setupClient func() client.Client
	wantErr     bool
	errMsg      string
	wantConfig  *Config
}

// runGetConfigTests runs a suite of config retrieval tests.
func runGetConfigTests(t *testing.T, tests map[string]getConfigTestCase) {
	t.Helper()

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			kubeClient := tc.setupClient()
			got, err := GetConfig(context.Background(), kubeClient, tc.managed)

			if (err != nil) != tc.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tc.wantErr)

				return
			}

			if tc.wantErr && tc.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("GetConfig() error = %v, want error containing %v", err, tc.errMsg)
				}

				return
			}

			if tc.wantConfig != nil {
				if diff := cmp.Diff(tc.wantConfig, got); diff != "" {
					t.Errorf("GetConfig() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// TestGetConfigModernManagedRefErrors tests config retrieval ref errors.
func TestGetConfigModernManagedRefErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]getConfigTestCase{
		"NoProviderConfigRefReturnsError": {
			managed: &fake.ModernManaged{},
			setupClient: func() client.Client {
				return newSonarTestClient()
			},
			wantErr: true,
			errMsg:  "providerConfigRef is not given",
		},
		"ProviderConfigNotFoundReturnsError": {
			managed: func() *fake.ModernManaged {
				m := &fake.ModernManaged{}
				m.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "nonexistent", Kind: "ProviderConfig"})

				return m
			}(),
			setupClient: func() client.Client {
				return newSonarTestClient()
			},
			wantErr: true,
			errMsg:  "cannot get referenced ProviderConfig",
		},
		"ClusterProviderConfigNotFoundReturnsError": {
			managed: func() *fake.ModernManaged {
				m := &fake.ModernManaged{}
				m.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "nonexistent", Kind: clusterProviderConfigKind})

				return m
			}(),
			setupClient: func() client.Client {
				return newSonarTestClient()
			},
			wantErr: true,
			errMsg:  "cannot get referenced ClusterProviderConfig",
		},
		"TrackingFailsReturnsError": {
			managed: func() *fake.ModernManaged {
				m := &fake.ModernManaged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-resource",
						Namespace: "default",
					},
				}
				m.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "my-pc", Kind: "ProviderConfig"})

				return m
			}(),
			setupClient: func() client.Client {
				return newSonarTestClient(
					&v1alpha1.ProviderConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-pc",
							Namespace: "default",
						},
						Spec: v1alpha1.ProviderConfigSpec{
							BaseURL: "http://localhost:9000",
							Token: &v1alpha1.ProviderCredentials{
								CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
									SecretRef: &xpv1.SecretKeySelector{
										SecretReference: xpv1.SecretReference{
											Name:      "my-secret",
											Namespace: "default",
										},
										Key: "token",
									},
								},
								Source: xpv1.CredentialsSourceSecret,
							},
						},
					},
				)
			},
			wantErr: true,
			errMsg:  "cannot track ProviderConfig usage",
		},
	}

	runGetConfigTests(t, tests)
}

// TestGetConfigModernManagedAuthErrors tests config retrieval auth errors.
func TestGetConfigModernManagedAuthErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]getConfigTestCase{
		"TokenSecretNotFoundReturnsError": {
			managed: func() *fake.ModernManaged {
				m := &fake.ModernManaged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-resource",
						Namespace: "default",
						UID:       "test-uid-4",
					},
				}
				m.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "my-pc-missing-secret", Kind: "ProviderConfig"})

				return m
			}(),
			setupClient: func() client.Client {
				return newSonarTestClient(
					&v1alpha1.ProviderConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-pc-missing-secret",
							Namespace: "default",
						},
						Spec: v1alpha1.ProviderConfigSpec{
							BaseURL: "http://localhost:9000",
							Token: &v1alpha1.ProviderCredentials{
								CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
									SecretRef: &xpv1.SecretKeySelector{
										SecretReference: xpv1.SecretReference{
											Name:      "nonexistent-secret",
											Namespace: "default",
										},
										Key: "token",
									},
								},
								Source: xpv1.CredentialsSourceSecret,
							},
						},
					},
				)
			},
			wantErr: true,
			errMsg:  "cannot get token from secret",
		},
		"BasicAuthUsernameSecretNotFoundReturnsError": {
			managed: func() *fake.ModernManaged {
				m := &fake.ModernManaged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-resource",
						Namespace: "default",
						UID:       "test-uid-5",
					},
				}
				m.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "my-pc-missing-user-secret", Kind: "ProviderConfig"})

				return m
			}(),
			setupClient: func() client.Client {
				return newSonarTestClient(
					&v1alpha1.ProviderConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-pc-missing-user-secret",
							Namespace: "default",
						},
						Spec: v1alpha1.ProviderConfigSpec{
							BaseURL: "http://localhost:9000",
							Username: &v1alpha1.ProviderCredentials{
								CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
									SecretRef: &xpv1.SecretKeySelector{
										SecretReference: xpv1.SecretReference{
											Name:      "nonexistent-secret",
											Namespace: "default",
										},
										Key: "username",
									},
								},
								Source: xpv1.CredentialsSourceSecret,
							},
							Password: &v1alpha1.ProviderCredentials{
								CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
									SecretRef: &xpv1.SecretKeySelector{
										SecretReference: xpv1.SecretReference{
											Name:      "my-secret",
											Namespace: "default",
										},
										Key: "password",
									},
								},
								Source: xpv1.CredentialsSourceSecret,
							},
						},
					},
				)
			},
			wantErr: true,
			errMsg:  "cannot get username from secret",
		},
		"BasicAuthPasswordSecretNotFoundReturnsError": {
			managed: func() *fake.ModernManaged {
				m := &fake.ModernManaged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-resource",
						Namespace: "default",
						UID:       "test-uid-6",
					},
				}
				m.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "my-pc-missing-pwd-secret", Kind: "ProviderConfig"})

				return m
			}(),
			setupClient: func() client.Client {
				return newSonarTestClient(
					&v1alpha1.ProviderConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-pc-missing-pwd-secret",
							Namespace: "default",
						},
						Spec: v1alpha1.ProviderConfigSpec{
							BaseURL: "http://localhost:9000",
							Username: &v1alpha1.ProviderCredentials{
								CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
									SecretRef: &xpv1.SecretKeySelector{
										SecretReference: xpv1.SecretReference{
											Name:      "my-secret",
											Namespace: "default",
										},
										Key: "username",
									},
								},
								Source: xpv1.CredentialsSourceSecret,
							},
							Password: &v1alpha1.ProviderCredentials{
								CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
									SecretRef: &xpv1.SecretKeySelector{
										SecretReference: xpv1.SecretReference{
											Name:      "nonexistent-secret",
											Namespace: "default",
										},
										Key: "password",
									},
								},
								Source: xpv1.CredentialsSourceSecret,
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-secret",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"username": []byte("admin"),
						},
					},
				)
			},
			wantErr: true,
			errMsg:  "cannot get password from secret",
		},
		"InvalidAuthTypeInProviderConfigReturnsError": {
			managed: func() *fake.ModernManaged {
				m := &fake.ModernManaged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-resource",
						Namespace: "default",
						UID:       "test-uid-7",
					},
				}
				m.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "my-pc-no-auth", Kind: "ProviderConfig"})

				return m
			}(),
			setupClient: func() client.Client {
				return newSonarTestClient(
					&v1alpha1.ProviderConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-pc-no-auth",
							Namespace: "default",
						},
						Spec: v1alpha1.ProviderConfigSpec{
							BaseURL: "http://localhost:9000",
						},
					},
				)
			},
			wantErr: true,
			errMsg:  "cannot determine authentication type",
		},
	}

	runGetConfigTests(t, tests)
}

// TestGetConfigModernManagedSuccess tests successful config retrieval.
func TestGetConfigModernManagedSuccess(t *testing.T) {
	t.Parallel()

	tests := map[string]getConfigTestCase{
		"ValidProviderConfigWithTokenAuth": {
			managed: func() *fake.ModernManaged {
				m := &fake.ModernManaged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-resource",
						Namespace: "default",
						UID:       "test-uid-1",
					},
				}
				m.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "my-pc", Kind: "ProviderConfig"})

				return m
			}(),
			setupClient: func() client.Client {
				return newSonarTestClient(
					&v1alpha1.ProviderConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-pc",
							Namespace: "default",
						},
						Spec: v1alpha1.ProviderConfigSpec{
							BaseURL: "http://localhost:9000",
							Token: &v1alpha1.ProviderCredentials{
								CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
									SecretRef: &xpv1.SecretKeySelector{
										SecretReference: xpv1.SecretReference{
											Name:      "my-secret",
											Namespace: "default",
										},
										Key: "token",
									},
								},
								Source: xpv1.CredentialsSourceSecret,
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-secret",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"token": []byte("my-token-value"),
						},
					},
				)
			},
			wantErr: false,
			wantConfig: &Config{
				AuthType:           PersonalAccessToken,
				Token:              "my-token-value",
				BaseURL:            "http://localhost:9000",
				InsecureSkipVerify: false,
			},
		},
		"ValidProviderConfigWithBasicAuth": {
			managed: func() *fake.ModernManaged {
				m := &fake.ModernManaged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-resource",
						Namespace: "default",
						UID:       "test-uid-2",
					},
				}
				m.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "my-pc-basic", Kind: "ProviderConfig"})

				return m
			}(),
			setupClient: func() client.Client {
				return newSonarTestClient(
					&v1alpha1.ProviderConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-pc-basic",
							Namespace: "default",
						},
						Spec: v1alpha1.ProviderConfigSpec{
							BaseURL:            "http://localhost:9000",
							InsecureSkipVerify: new(true),
							Username: &v1alpha1.ProviderCredentials{
								CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
									SecretRef: &xpv1.SecretKeySelector{
										SecretReference: xpv1.SecretReference{
											Name:      "my-secret",
											Namespace: "default",
										},
										Key: "username",
									},
								},
								Source: xpv1.CredentialsSourceSecret,
							},
							Password: &v1alpha1.ProviderCredentials{
								CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
									SecretRef: &xpv1.SecretKeySelector{
										SecretReference: xpv1.SecretReference{
											Name:      "my-secret",
											Namespace: "default",
										},
										Key: "password",
									},
								},
								Source: xpv1.CredentialsSourceSecret,
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-secret",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"username": []byte("admin"),
							"password": []byte("admin-pass"),
						},
					},
				)
			},
			wantErr: false,
			wantConfig: &Config{
				AuthType: BasicAuth,
				BasicAuth: &BasicAuthArgs{
					Username: "admin",
					Password: "admin-pass",
				},
				BaseURL:            "http://localhost:9000",
				InsecureSkipVerify: true,
			},
		},
		"ValidClusterProviderConfigWithTokenAuth": {
			managed: func() *fake.ModernManaged {
				m := &fake.ModernManaged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-resource",
						Namespace: "default",
						UID:       "test-uid-3",
					},
				}
				m.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "my-cpc", Kind: clusterProviderConfigKind})

				return m
			}(),
			setupClient: func() client.Client {
				return newSonarTestClient(
					&v1alpha1.ClusterProviderConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-cpc",
						},
						Spec: v1alpha1.ProviderConfigSpec{
							BaseURL: "http://localhost:9000",
							Token: &v1alpha1.ProviderCredentials{
								CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
									SecretRef: &xpv1.SecretKeySelector{
										SecretReference: xpv1.SecretReference{
											Name:      "my-secret",
											Namespace: "default",
										},
										Key: "token",
									},
								},
								Source: xpv1.CredentialsSourceSecret,
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-secret",
							Namespace: "default",
						},
						Data: map[string][]byte{
							"token": []byte("cluster-token"),
						},
					},
				)
			},
			wantErr: false,
			wantConfig: &Config{
				AuthType:           PersonalAccessToken,
				Token:              "cluster-token",
				BaseURL:            "http://localhost:9000",
				InsecureSkipVerify: false,
			},
		},
	}

	runGetConfigTests(t, tests)
}
