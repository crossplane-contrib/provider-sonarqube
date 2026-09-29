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

package cache

import (
	"strings"
	"testing"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// TestScopeFromConfig tests that every connection identity gets its own
// scope, which never contains a secret.
func TestScopeFromConfig(t *testing.T) {
	t.Parallel()

	const (
		token    = "squ_super-secret-token"
		username = "admin-user"
		password = "super-secret-password"
	)

	base := common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    token,
		BaseURL:  "https://sonar.example.com/api",
	}
	basicAuth := common.Config{
		AuthType:  common.BasicAuth,
		BasicAuth: &common.BasicAuthArgs{Username: username, Password: password},
		BaseURL:   base.BaseURL,
	}

	variants := map[string]common.Config{
		"Base":          base,
		"OtherToken":    {AuthType: base.AuthType, Token: "other", BaseURL: base.BaseURL},
		"OtherURL":      {AuthType: base.AuthType, Token: token, BaseURL: "https://other.example.com/api"},
		"OtherAuthType": {AuthType: common.BasicAuth, Token: token, BaseURL: base.BaseURL},
		"BasicAuth":     basicAuth,
		"OtherUsername": {AuthType: common.BasicAuth, BasicAuth: &common.BasicAuthArgs{Username: "other", Password: password}, BaseURL: base.BaseURL},
		"OtherPassword": {AuthType: common.BasicAuth, BasicAuth: &common.BasicAuthArgs{Username: username, Password: "other"}, BaseURL: base.BaseURL},
		"FieldShift":    {AuthType: common.BasicAuth, BasicAuth: &common.BasicAuthArgs{Username: username + password}, BaseURL: base.BaseURL},
	}

	seen := make(map[string]string, len(variants))

	for name, config := range variants {
		scope := ScopeFromConfig(config)

		if other, ok := seen[scope]; ok {
			t.Errorf("configs %q and %q share scope %q", name, other, scope)
		}

		seen[scope] = name

		for _, secret := range []string{token, username, password} {
			if strings.Contains(scope, secret) {
				t.Errorf("scope of %q contains secret %q", name, secret)
			}
		}

		if again := ScopeFromConfig(config); again != scope {
			t.Errorf("ScopeFromConfig(%q) is not deterministic: %q != %q", name, again, scope)
		}
	}

	// InsecureSkipVerify does not change the identity of the data.
	insecure := base
	insecure.InsecureSkipVerify = true

	if ScopeFromConfig(insecure) != ScopeFromConfig(base) {
		t.Error("InsecureSkipVerify changed the scope")
	}
}

// TestEncodeParams tests the canonical encoding of query parameters.
func TestEncodeParams(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params map[string]string
		want   string
	}{
		"Nil":    {params: nil, want: ""},
		"Sorted": {params: map[string]string{"q": "a b", "language": "go", "p": "1"}, want: "language=go&p=1&q=a+b"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for range 10 {
				if got := EncodeParams(tc.params); got != tc.want {
					t.Fatalf("EncodeParams() = %q, want %q", got, tc.want)
				}
			}
		})
	}
}

// TestKeyString tests that distinct keys have distinct string
// representations.
func TestKeyString(t *testing.T) {
	t.Parallel()

	a := Key{Scope: "a", Namespace: "b/c", Params: ""}
	b := Key{Scope: "a", Namespace: "b", Params: "c"}

	if a.String() == b.String() {
		t.Errorf("distinct keys share String() %q", a.String())
	}
}
