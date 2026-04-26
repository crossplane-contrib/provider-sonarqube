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

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// TestALMClientConstructors tests ALM client constructors.
func TestALMClientConstructors(t *testing.T) {
	t.Parallel()

	config := common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  "http://localhost:9000",
	}

	cases := map[string]func(common.Config) any{
		"ALMIntegrationsAzure":          func(cfg common.Config) any { return NewALMIntegrationsAzureClient(cfg) },
		"ALMSettingsAzure":              func(cfg common.Config) any { return NewALMSettingsAzureClient(cfg) },
		"ALMIntegrationsBitbucket":      func(cfg common.Config) any { return NewALMIntegrationsBitbucketClient(cfg) },
		"ALMSettingsBitbucket":          func(cfg common.Config) any { return NewALMSettingsBitbucketClient(cfg) },
		"ALMIntegrationsBitbucketCloud": func(cfg common.Config) any { return NewALMIntegrationsBitbucketCloudClient(cfg) },
		"ALMSettingsBitbucketCloud":     func(cfg common.Config) any { return NewALMSettingsBitbucketCloudClient(cfg) },
		"ALMIntegrationsGitHub":         func(cfg common.Config) any { return NewALMIntegrationsGitHubClient(cfg) },
		"ALMSettingsGitHub":             func(cfg common.Config) any { return NewALMSettingsGitHubClient(cfg) },
	}

	for name, fn := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := fn(config); got == nil {
				t.Fatalf("%s constructor returned nil", name)
			}
		})
	}
}
