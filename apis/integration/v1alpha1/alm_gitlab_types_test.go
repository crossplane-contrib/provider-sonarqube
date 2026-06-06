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
	"encoding/json"
	"strings"
	"testing"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// TestALMGitLabParametersJSONInline tests inline JSON
// marshaling of ALM GitLab parameters.
func TestALMGitLabParametersJSONInline(t *testing.T) {
	t.Parallel()

	in := ALMGitLabParameters{
		ALMCommonParameters: ALMCommonParameters{
			URL: "https://gitlab.example.com",
			Key: "gitlab-main",
			PersonalAccessTokenSecretRef: &xpv1.LocalSecretKeySelector{
				LocalSecretReference: xpv1.LocalSecretReference{Name: "gitlab-pat"},
				Key:                  "token",
			},
		},
	}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("json.Marshal() unexpected error: %v", err)
	}

	out := string(b)
	if !strings.Contains(out, "\"key\":\"gitlab-main\"") {
		t.Fatalf("json.Marshal() missing inline key field: %s", out)
	}

	if !strings.Contains(out, "\"url\":\"https://gitlab.example.com\"") {
		t.Fatalf("json.Marshal() missing inline url field: %s", out)
	}

	if !strings.Contains(out, "\"personalAccessTokenSecretRef\":") {
		t.Fatalf("json.Marshal() missing inline personalAccessTokenSecretRef field: %s", out)
	}

	if strings.Contains(out, "ALMCommonParameters") || strings.Contains(out, "commonParameters") {
		t.Fatalf("json.Marshal() unexpectedly nested common fields: %s", out)
	}
}

// TestALMGitLabParametersJSONUnmarshalFlatShape tests flat JSON unmarshaling.
func TestALMGitLabParametersJSONUnmarshalFlatShape(t *testing.T) {
	t.Parallel()

	payload := `{"key":"gitlab-main","url":"https://gitlab.example.com","personalAccessTokenSecretRef":{"name":"gitlab-pat","key":"token"}}`

	var got ALMGitLabParameters

	err := json.Unmarshal([]byte(payload), &got)
	if err != nil {
		t.Fatalf("json.Unmarshal() unexpected error: %v", err)
	}

	if got.Key != "gitlab-main" {
		t.Fatalf("json.Unmarshal() key = %q, want %q", got.Key, "gitlab-main")
	}

	if got.URL != "https://gitlab.example.com" {
		t.Fatalf("json.Unmarshal() url = %q, want %q", got.URL, "https://gitlab.example.com")
	}

	if got.PersonalAccessTokenSecretRef == nil || got.PersonalAccessTokenSecretRef.Name != "gitlab-pat" || got.PersonalAccessTokenSecretRef.Key != "token" {
		t.Fatalf("json.Unmarshal() token ref mismatch: %+v", got.PersonalAccessTokenSecretRef)
	}
}
