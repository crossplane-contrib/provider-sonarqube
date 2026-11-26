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

package instance

import (
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// TestNewInstanceConfigurationClients tests that every client of an
// InstanceConfiguration is created.
func TestNewInstanceConfigurationClients(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config common.Config
	}{
		"PersonalAccessToken": {
			config: newTestConfig(),
		},
		"BasicAuth": {
			config: newBasicAuthTestConfig(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			clients := NewInstanceConfigurationClients(tc.config)

			if clients.CleanCodePolicy == nil {
				t.Error("NewInstanceConfigurationClients() CleanCodePolicy is nil")
			}

			if clients.Projects == nil {
				t.Error("NewInstanceConfigurationClients() Projects is nil")
			}

			if clients.Settings == nil {
				t.Error("NewInstanceConfigurationClients() Settings is nil")
			}

			if clients.ComputeEngine == nil {
				t.Error("NewInstanceConfigurationClients() ComputeEngine is nil")
			}

			if clients.SCIM == nil {
				t.Error("NewInstanceConfigurationClients() SCIM is nil")
			}

			if clients.SCA == nil {
				t.Error("NewInstanceConfigurationClients() SCA is nil")
			}
		})
	}
}

// TestGenerateInstanceConfigurationOptions tests the generators of the
// request options of each InstanceConfiguration option.
func TestGenerateInstanceConfigurationOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		got  any
		want any
	}{
		"CleanCodePolicyUpdateMode": {
			got:  GenerateCleanCodePolicyUpdateModeOptions(sonar.InstanceModeStandardExperience),
			want: &sonar.CleanCodePolicyUpdateModeOptions{Mode: sonar.InstanceModeStandardExperience},
		},
		"UpdateDefaultVisibility": {
			got:  GenerateUpdateDefaultVisibilityOptions("private"),
			want: &sonar.ProjectsUpdateDefaultVisibilityOptions{ProjectVisibility: "private"},
		},
		"DefaultProjectVisibilityValues": {
			got:  GenerateDefaultProjectVisibilityValuesOptions(),
			want: &sonar.SettingsValuesOptions{Keys: []string{DefaultProjectVisibilitySettingKey}},
		},
		"SetWorkerCount": {
			got:  GenerateSetWorkerCountOptions(4),
			want: &sonar.CeSetWorkerCountOptions{Count: 4},
		},
		"SetSCAEnablementEnabled": {
			got:  GenerateSetSCAEnablementOptions(true),
			want: &sonar.ScaSetEnablementOptions{Enablement: true},
		},
		"SetSCAEnablementDisabled": {
			got:  GenerateSetSCAEnablementOptions(false),
			want: &sonar.ScaSetEnablementOptions{Enablement: false},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tc.want, tc.got); diff != "" {
				t.Errorf("options mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestExtractDefaultProjectVisibility tests reading the default project
// visibility from a settings values response.
func TestExtractDefaultProjectVisibility(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		values *sonar.SettingsValues
		want   *string
	}{
		"NilResponse": {
			values: nil,
			want:   nil,
		},
		"KeyMissing": {
			values: &sonar.SettingsValues{
				Settings: []sonar.SettingValue{{Key: "sonar.core.serverBaseURL", Value: "http://localhost"}},
			},
			want: nil,
		},
		"KeyPresent": {
			values: &sonar.SettingsValues{
				Settings: []sonar.SettingValue{
					{Key: "sonar.core.serverBaseURL", Value: "http://localhost"},
					{Key: DefaultProjectVisibilitySettingKey, Value: "private"},
				},
			},
			want: new("private"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := ExtractDefaultProjectVisibility(tc.values)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractDefaultProjectVisibility() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestIsInstanceConfigurationOptionUpToDate tests the comparison of a
// desired option with its observed value.
func TestIsInstanceConfigurationOptionUpToDate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		desired  *int64
		observed *int64
		want     bool
	}{
		"NotSet": {
			desired:  nil,
			observed: new(int64(2)),
			want:     true,
		},
		"NotSetNorObserved": {
			desired:  nil,
			observed: nil,
			want:     true,
		},
		"NotObserved": {
			desired:  new(int64(2)),
			observed: nil,
			want:     false,
		},
		"Equal": {
			desired:  new(int64(2)),
			observed: new(int64(2)),
			want:     true,
		},
		"Different": {
			desired:  new(int64(2)),
			observed: new(int64(1)),
			want:     false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsInstanceConfigurationOptionUpToDate(tc.desired, tc.observed)
			if got != tc.want {
				t.Errorf("IsInstanceConfigurationOptionUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}
