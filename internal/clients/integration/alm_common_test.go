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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
)

func TestIsALMUpToDate(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.ALMCommonParameters{URL: "https://gitlab.example.com", Key: "gitlab-main"}
	obs := &v1alpha1.ALMCommonObservation{URL: "https://gitlab.example.com", Key: "gitlab-main"}

	cases := map[string]struct {
		spec          *v1alpha1.ALMCommonParameters
		specAPIToken  string
		observation   *v1alpha1.ALMCommonObservation
		savedAPIToken string
		want          bool
	}{
		"NilSpec": {
			spec:          nil,
			specAPIToken:  "token",
			observation:   obs,
			savedAPIToken: "token",
			want:          true,
		},
		"NilObservation": {
			spec:          spec,
			specAPIToken:  "token",
			observation:   nil,
			savedAPIToken: "token",
			want:          false,
		},
		"UpToDate": {
			spec:          spec,
			specAPIToken:  "token",
			observation:   obs,
			savedAPIToken: "token",
			want:          true,
		},
		"URLChanged": {
			spec:          spec,
			specAPIToken:  "token",
			observation:   &v1alpha1.ALMCommonObservation{URL: "https://other.example.com", Key: "gitlab-main"},
			savedAPIToken: "token",
			want:          false,
		},
		"TokenChanged": {
			spec:          spec,
			specAPIToken:  "token-a",
			observation:   obs,
			savedAPIToken: "token-b",
			want:          false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsALMUpToDate(tc.spec, tc.specAPIToken, tc.observation, tc.savedAPIToken); got != tc.want {
				t.Fatalf("IsALMUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLateInitializeALM(t *testing.T) {
	t.Parallel()

	LateInitializeALM(nil, nil)
	LateInitializeALM(&v1alpha1.ALMCommonParameters{}, nil)
	LateInitializeALM(nil, &v1alpha1.ALMCommonObservation{})

	spec := &v1alpha1.ALMCommonParameters{URL: "https://gitlab.example.com", Key: "gitlab-main"}
	obs := &v1alpha1.ALMCommonObservation{URL: "https://other.example.com", Key: "other"}
	LateInitializeALM(spec, obs)

	if spec.URL != "https://gitlab.example.com" || spec.Key != "gitlab-main" {
		t.Fatalf("LateInitializeALM() mutated spec unexpectedly: %+v", spec)
	}
}

func TestIsALMLateInitialized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		former  *v1alpha1.ALMCommonParameters
		current *v1alpha1.ALMCommonParameters
		want    bool
	}{
		"NilFormer": {
			former:  nil,
			current: &v1alpha1.ALMCommonParameters{},
			want:    true,
		},
		"NilCurrent": {
			former:  &v1alpha1.ALMCommonParameters{},
			current: nil,
			want:    true,
		},
		"NoChanges": {
			former:  &v1alpha1.ALMCommonParameters{URL: "https://gitlab.example.com", Key: "gitlab-main"},
			current: &v1alpha1.ALMCommonParameters{URL: "https://gitlab.example.com", Key: "gitlab-main"},
			want:    false,
		},
		"ChangesOccurred": {
			former:  &v1alpha1.ALMCommonParameters{URL: "https://gitlab.example.com", Key: "gitlab-main"},
			current: &v1alpha1.ALMCommonParameters{URL: "https://other.example.com", Key: "other"},
			want:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsALMLateInitialized(tc.former, tc.current); got != tc.want {
				t.Fatalf("IsALMLateInitialized() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGenerateALMDeleteOptions(t *testing.T) {
	t.Parallel()

	got := GenerateALMDeleteOptions("gitlab-main")
	if got == nil {
		t.Fatal("GenerateALMDeleteOptions() returned nil")
	}

	if got.Key != "gitlab-main" {
		t.Fatalf("GenerateALMDeleteOptions() key = %q, want %q", got.Key, "gitlab-main")
	}
}
