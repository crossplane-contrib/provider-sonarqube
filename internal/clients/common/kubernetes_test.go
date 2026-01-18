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
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newFakeClient(objs ...runtime.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	clientObjs := make([]client.Object, 0, len(objs))
	for _, obj := range objs {
		if co, ok := obj.(client.Object); ok {
			clientObjs = append(clientObjs, co)
		}
	}

	return fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(clientObjs...).
		Build()
}

func TestGetTokenValueFromSecret(t *testing.T) {
	type args struct {
		ctx      context.Context
		client   client.Client
		m        resource.Managed
		selector *xpv1.SecretKeySelector
	}
	tests := map[string]struct {
		args        args
		want        *string
		wantErr     bool
		errContains string
	}{
		"NilSelectorReturnsError": {
			args: args{
				ctx:      context.Background(),
				client:   newFakeClient(),
				m:        &fake.Managed{},
				selector: nil,
			},
			want:        nil,
			wantErr:     true,
			errContains: ErrSecretSelectorNil,
		},
		"SecretNotFoundReturnsError": {
			args: args{
				ctx:    context.Background(),
				client: newFakeClient(),
				m:      &fake.Managed{},
				selector: &xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{
						Name:      "nonexistent",
						Namespace: "default",
					},
					Key: "token",
				},
			},
			want:        nil,
			wantErr:     true,
			errContains: ErrSecretNotFound,
		},
		"KeyNotFoundReturnsError": {
			args: args{
				ctx: context.Background(),
				client: newFakeClient(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"other-key": []byte("value"),
					},
				}),
				m: &fake.Managed{},
				selector: &xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{
						Name:      "my-secret",
						Namespace: "default",
					},
					Key: "token",
				},
			},
			want:        nil,
			wantErr:     true,
			errContains: ErrSecretKeyNotFound,
		},
		"SuccessfulTokenRetrieval": {
			args: args{
				ctx: context.Background(),
				client: newFakeClient(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"token": []byte("my-token-value"),
					},
				}),
				m: &fake.Managed{},
				selector: &xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{
						Name:      "my-secret",
						Namespace: "default",
					},
					Key: "token",
				},
			},
			want:    strPtr("my-token-value"),
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := GetTokenValueFromSecret(tc.args.ctx, tc.args.client, tc.args.m, tc.args.selector)
			if (err != nil) != tc.wantErr {
				t.Errorf("GetTokenValueFromSecret() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr && tc.errContains != "" {
				if err == nil || !containsString(err.Error(), tc.errContains) {
					t.Errorf("GetTokenValueFromSecret() error = %v, should contain %v", err, tc.errContains)
				}
				return
			}
			if tc.want == nil && got != nil {
				t.Errorf("GetTokenValueFromSecret() = %v, want nil", *got)
				return
			}
			if tc.want != nil && got == nil {
				t.Errorf("GetTokenValueFromSecret() = nil, want %v", *tc.want)
				return
			}
			if tc.want != nil && got != nil && *got != *tc.want {
				t.Errorf("GetTokenValueFromSecret() = %v, want %v", *got, *tc.want)
			}
		})
	}
}

func TestGetTokenValueFromLocalSecret(t *testing.T) {
	type args struct {
		ctx    context.Context
		client client.Client
		m      resource.Managed
		l      *xpv1.LocalSecretKeySelector
	}
	tests := map[string]struct {
		args        args
		want        *string
		wantErr     bool
		errContains string
	}{
		"NilSelectorReturnsError": {
			args: args{
				ctx:    context.Background(),
				client: newFakeClient(),
				m:      &fake.Managed{},
				l:      nil,
			},
			want:        nil,
			wantErr:     true,
			errContains: ErrSecretSelectorNil,
		},
		"SuccessfulLocalTokenRetrieval": {
			args: args{
				ctx: context.Background(),
				client: newFakeClient(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "local-secret",
						Namespace: "test-ns",
					},
					Data: map[string][]byte{
						"token": []byte("local-token-value"),
					},
				}),
				m: &fake.Managed{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-ns",
					},
				},
				l: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{
						Name: "local-secret",
					},
					Key: "token",
				},
			},
			want:    strPtr("local-token-value"),
			wantErr: false,
		},
		"SecretNotFoundReturnsError": {
			args: args{
				ctx:    context.Background(),
				client: newFakeClient(),
				m: &fake.Managed{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-ns",
					},
				},
				l: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{
						Name: "nonexistent",
					},
					Key: "token",
				},
			},
			want:        nil,
			wantErr:     true,
			errContains: ErrSecretNotFound,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := GetTokenValueFromLocalSecret(tc.args.ctx, tc.args.client, tc.args.m, tc.args.l)
			if (err != nil) != tc.wantErr {
				t.Errorf("GetTokenValueFromLocalSecret() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr && tc.errContains != "" {
				if err == nil || !containsString(err.Error(), tc.errContains) {
					t.Errorf("GetTokenValueFromLocalSecret() error = %v, should contain %v", err, tc.errContains)
				}
				return
			}
			if tc.want == nil && got != nil {
				t.Errorf("GetTokenValueFromLocalSecret() = %v, want nil", *got)
				return
			}
			if tc.want != nil && got == nil {
				t.Errorf("GetTokenValueFromLocalSecret() = nil, want %v", *tc.want)
				return
			}
			if tc.want != nil && got != nil && *got != *tc.want {
				t.Errorf("GetTokenValueFromLocalSecret() = %v, want %v", *got, *tc.want)
			}
		})
	}
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
