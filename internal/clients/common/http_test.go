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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestFetchFromEndpoint tests all branches of FetchFromEndpoint.
func TestFetchFromEndpoint(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		auth       *EndpointAuth
		handler    http.HandlerFunc
		want       string
		wantErrSub string
	}{
		"Unauthenticated": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "" {
					t.Errorf("Authorization header = %q, want empty", got)
				}

				_, _ = w.Write([]byte("  license-key-abc  \n"))
			},
			want: "license-key-abc",
		},
		"BasicAuth": {
			auth: &EndpointAuth{BasicAuthUsername: "alice", BasicAuthPassword: "s3cret"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				if !ok || user != "alice" || pass != "s3cret" {
					t.Errorf("BasicAuth() = (%q, %q, %v), want (\"alice\", \"s3cret\", true)", user, pass, ok)
				}

				_, _ = w.Write([]byte("license-key-basic"))
			},
			want: "license-key-basic",
		},
		"BearerToken": {
			auth: &EndpointAuth{BearerToken: "tok123"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "Bearer tok123" {
					t.Errorf("Authorization header = %q, want %q", got, "Bearer tok123")
				}

				_, _ = w.Write([]byte("license-key-bearer"))
			},
			want: "license-key-bearer",
		},
		"BearerTokenTakesPrecedenceOverBasicAuth": {
			auth: &EndpointAuth{BasicAuthUsername: "alice", BasicAuthPassword: "s3cret", BearerToken: "tok123"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "Bearer tok123" {
					t.Errorf("Authorization header = %q, want %q", got, "Bearer tok123")
				}

				_, _ = w.Write([]byte("license-key-bearer"))
			},
			want: "license-key-bearer",
		},
		"NonSuccessStatusCode": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErrSub: "unexpected status code 401",
		},
		"EmptyResponse": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("   "))
			},
			wantErrSub: "empty response",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(tc.handler)
			defer server.Close()

			got, err := FetchFromEndpoint(context.Background(), server.URL, tc.auth)

			if tc.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Fatalf("FetchFromEndpoint() error = %v, want substring %q", err, tc.wantErrSub)
				}

				return
			}

			if err != nil {
				t.Fatalf("FetchFromEndpoint() unexpected error: %v", err)
			}

			if got != tc.want {
				t.Errorf("FetchFromEndpoint() = %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("InvalidURL", func(t *testing.T) {
		t.Parallel()

		_, err := FetchFromEndpoint(context.Background(), "://bad-url", nil)
		if err == nil {
			t.Fatal("FetchFromEndpoint() expected error for invalid URL, got nil")
		}
	})
}
