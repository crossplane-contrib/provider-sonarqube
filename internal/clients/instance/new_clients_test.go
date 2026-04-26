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

// newTestConfig creates a test configuration with a personal access token.
func newTestConfig() common.Config {
	return common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "test-token",
		BaseURL:  "http://localhost:9000",
	}
}

// newBasicAuthTestConfig creates a test configuration with basic auth.
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

// TestNewProjectsClient tests creating a new projects client.
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

// TestNewProjectBranchesClient tests creating a new project branches client.
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

// TestNewProjectLinksClient tests creating a new project links client.
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

// TestNewProjectTagsClient tests creating a new project tags client.
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

// TestNewQualityGatesClient tests creating a new quality gates client.
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

// TestNewQualityProfilesClient tests creating a new quality profiles client.
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

// TestNewRulesClient tests creating a new rules client.
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

// TestNewSettingsClient tests creating a new settings client.
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

// TestNewNewCodePeriodsClient tests creating a new code periods client.
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
