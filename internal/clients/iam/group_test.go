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

package iam

import (
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

// TestLateInitializeGroup tests the LateInitializeGroup function.
func TestLateInitializeGroup(t *testing.T) {
	t.Parallel()

	t.Run("NilSpecNoPanic", func(t *testing.T) {
		t.Parallel()

		obs := &v1alpha1.GroupObservation{Description: "from-api"}
		LateInitializeGroup(nil, obs)
	})

	t.Run("NilObservationNoPanic", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.GroupParameters{Name: "devs"}
		LateInitializeGroup(spec, nil)

		if spec.Description != nil {
			t.Fatalf("LateInitializeGroup() expected nil description, got %q", *spec.Description)
		}
	})

	t.Run("DescriptionInitializedWhenMissing", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.GroupParameters{Name: "devs"}
		obs := &v1alpha1.GroupObservation{Description: "engineering group"}

		LateInitializeGroup(spec, obs)

		if spec.Description == nil || *spec.Description != "engineering group" {
			t.Fatalf("LateInitializeGroup() description = %v, want %q", spec.Description, "engineering group")
		}
	})

	t.Run("DescriptionNotOverwrittenWhenPresent", func(t *testing.T) {
		t.Parallel()

		current := "custom"
		spec := &v1alpha1.GroupParameters{Name: "devs", Description: &current}
		obs := &v1alpha1.GroupObservation{Description: "engineering group"}

		LateInitializeGroup(spec, obs)

		if spec.Description == nil || *spec.Description != "custom" {
			t.Fatalf("LateInitializeGroup() description = %v, want %q", spec.Description, "custom")
		}
	})

	t.Run("EmptyObservationDescriptionIgnored", func(t *testing.T) {
		t.Parallel()

		spec := &v1alpha1.GroupParameters{Name: "devs"}
		obs := &v1alpha1.GroupObservation{Description: ""}

		LateInitializeGroup(spec, obs)

		if spec.Description != nil {
			t.Fatalf("LateInitializeGroup() description = %v, want nil", spec.Description)
		}
	})
}

// TestIsGroupLateInitialized tests the IsGroupLateInitialized function.
func TestIsGroupLateInitialized(t *testing.T) {
	t.Parallel()

	d1 := "d1"
	d2 := "d2"

	cases := map[string]struct {
		former  *v1alpha1.GroupParameters
		current *v1alpha1.GroupParameters
		want    bool
	}{
		"NilFormer": {
			former:  nil,
			current: &v1alpha1.GroupParameters{Name: "devs"},
			want:    true,
		},
		"NilCurrent": {
			former:  &v1alpha1.GroupParameters{Name: "devs"},
			current: nil,
			want:    true,
		},
		"NoChanges": {
			former:  &v1alpha1.GroupParameters{Name: "devs", Description: &d1},
			current: &v1alpha1.GroupParameters{Name: "devs", Description: &d1},
			want:    false,
		},
		"NameChanged": {
			former:  &v1alpha1.GroupParameters{Name: "devs", Description: &d1},
			current: &v1alpha1.GroupParameters{Name: "admins", Description: &d1},
			want:    true,
		},
		"DescriptionChanged": {
			former:  &v1alpha1.GroupParameters{Name: "devs", Description: &d1},
			current: &v1alpha1.GroupParameters{Name: "devs", Description: &d2},
			want:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsGroupLateInitialized(tc.former, tc.current); got != tc.want {
				t.Fatalf("IsGroupLateInitialized() = %t, want %t", got, tc.want)
			}
		})
	}
}

// TestIsGroupUpToDate tests the IsGroupUpToDate function.
func TestIsGroupUpToDate(t *testing.T) {
	t.Parallel()

	desc := "engineering"

	cases := map[string]struct {
		spec *v1alpha1.GroupParameters
		obs  *v1alpha1.GroupObservation
		want bool
	}{
		"NilSpec": {
			spec: nil,
			obs:  &v1alpha1.GroupObservation{Name: "devs"},
			want: true,
		},
		"NilObservation": {
			spec: &v1alpha1.GroupParameters{Name: "devs"},
			obs:  nil,
			want: false,
		},
		"UpToDate": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Description: &desc},
			obs:  &v1alpha1.GroupObservation{Name: "devs", Description: "engineering"},
			want: true,
		},
		"NameMismatch": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Description: &desc},
			obs:  &v1alpha1.GroupObservation{Name: "admins", Description: "engineering"},
			want: false,
		},
		"DescriptionMismatch": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Description: &desc},
			obs:  &v1alpha1.GroupObservation{Name: "devs", Description: "different"},
			want: false,
		},
		"NilDescriptionIsComparable": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Description: nil},
			obs:  &v1alpha1.GroupObservation{Name: "devs", Description: "any"},
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsGroupUpToDate(tc.spec, tc.obs); got != tc.want {
				t.Fatalf("IsGroupUpToDate() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestGenerateCreateGroupOptions(t *testing.T) {
	t.Parallel()

	description := "engineering"

	cases := map[string]struct {
		spec *v1alpha1.GroupParameters
		want string
	}{
		"NilSpec": {
			spec: nil,
		},
		"WithoutDescription": {
			spec: &v1alpha1.GroupParameters{Name: "devs"},
			want: "devs",
		},
		"WithDescription": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Description: &description},
			want: "devs",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateCreateGroupOptions(tc.spec)
			if tc.spec == nil {
				if got != nil {
					t.Fatal("GenerateCreateGroupOptions() expected nil, got non-nil")
				}

				return
			}

			if got == nil {
				t.Fatal("GenerateCreateGroupOptions() got nil, want non-nil")
			}

			if got.Name != tc.want {
				t.Fatalf("GenerateCreateGroupOptions() name = %q, want %q", got.Name, tc.want)
			}

			wantDescription := ""
			if tc.spec.Description != nil {
				wantDescription = *tc.spec.Description
			}

			if got.Description != wantDescription {
				t.Fatalf("GenerateCreateGroupOptions() description mismatch: got %q, want %q", got.Description, wantDescription)
			}
		})
	}
}

func TestGenerateUpdateGroupOptions(t *testing.T) {
	t.Parallel()

	description := "engineering"

	cases := map[string]struct {
		spec *v1alpha1.GroupParameters
	}{
		"NilSpec": {
			spec: nil,
		},
		"WithoutDescription": {
			spec: &v1alpha1.GroupParameters{Name: "devs"},
		},
		"WithDescription": {
			spec: &v1alpha1.GroupParameters{Name: "devs", Description: &description},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateUpdateGroupOptions(tc.spec)
			if tc.spec == nil {
				if got != nil {
					t.Fatal("GenerateUpdateGroupOptions() expected nil, got non-nil")
				}

				return
			}

			if got == nil {
				t.Fatal("GenerateUpdateGroupOptions() got nil, want non-nil")
			}

			if got.Name != tc.spec.Name {
				t.Fatalf("GenerateUpdateGroupOptions() name = %q, want %q", got.Name, tc.spec.Name)
			}

			if !cmp.Equal(got.Description, tc.spec.Description) {
				t.Fatalf("GenerateUpdateGroupOptions() description mismatch: got %v, want %v", got.Description, tc.spec.Description)
			}
		})
	}
}

func TestGenerateGroupObservation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		input *sonar.Group
		want  v1alpha1.GroupObservation
	}{
		"NilGroup": {
			input: nil,
			want:  v1alpha1.GroupObservation{},
		},
		"PopulatedGroup": {
			input: &sonar.Group{
				Id:          "group-1",
				Name:        "devs",
				Description: "engineering",
				Managed:     true,
				Default:     false,
			},
			want: v1alpha1.GroupObservation{
				ID:          "group-1",
				Name:        "devs",
				Description: "engineering",
				Managed:     true,
				Default:     false,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GenerateGroupObservation(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("GenerateGroupObservation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewGroupsClient(t *testing.T) {
	t.Parallel()

	client := NewGroupsClient(common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  "http://localhost:9000",
	})

	if client == nil {
		t.Fatal("NewGroupsClient() expected non-nil client")
	}
}
