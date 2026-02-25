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

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

func newTestConfig() common.Config {
	return common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "test-token",
		BaseURL:  "http://localhost:9000",
	}
}

func newBasicAuthTestConfig() common.Config {
	return common.Config{
		AuthType: common.BasicAuth,
		BasicAuth: &common.BasicAuthArgs{
			Username: "admin",
			Password: "admin",
		},
		BaseURL: "http://localhost:9000",
	}
}

func TestNewProjectsClient(t *testing.T) {
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

			client := NewProjectsClient(tc.config)
			if client == nil {
				t.Error("NewProjectsClient() returned nil")
			}
		})
	}
}

func TestNewProjectBranchesClient(t *testing.T) {
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

			client := NewProjectBranchesClient(tc.config)
			if client == nil {
				t.Error("NewProjectBranchesClient() returned nil")
			}
		})
	}
}

func TestNewProjectLinksClient(t *testing.T) {
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

			client := NewProjectLinksClient(tc.config)
			if client == nil {
				t.Error("NewProjectLinksClient() returned nil")
			}
		})
	}
}

func TestNewProjectTagsClient(t *testing.T) {
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

			client := NewProjectTagsClient(tc.config)
			if client == nil {
				t.Error("NewProjectTagsClient() returned nil")
			}
		})
	}
}

func TestNewQualityGatesClient(t *testing.T) {
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

			client := NewQualityGatesClient(tc.config)
			if client == nil {
				t.Error("NewQualityGatesClient() returned nil")
			}
		})
	}
}

func TestNewQualityProfilesClient(t *testing.T) {
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

			client := NewQualityProfilesClient(tc.config)
			if client == nil {
				t.Error("NewQualityProfilesClient() returned nil")
			}
		})
	}
}

func TestNewRulesClient(t *testing.T) {
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

			client := NewRulesClient(tc.config)
			if client == nil {
				t.Error("NewRulesClient() returned nil")
			}
		})
	}
}

func TestNewSettingsClient(t *testing.T) {
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

			client := NewSettingsClient(tc.config)
			if client == nil {
				t.Error("NewSettingsClient() returned nil")
			}
		})
	}
}

func TestNewNewCodePeriodsClient(t *testing.T) {
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

			client := NewNewCodePeriodsClient(tc.config)
			if client == nil {
				t.Error("NewNewCodePeriodsClient() returned nil")
			}
		})
	}
}
