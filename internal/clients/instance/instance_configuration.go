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
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// DefaultProjectVisibilitySettingKey is the key under which SonarQube stores
// the default visibility of new projects. It is written by
// api/projects/update_default_visibility and read through api/settings/values.
const DefaultProjectVisibilitySettingKey = "projects.default.visibility"

// CleanCodePolicyModeClient is the interface for reading and updating the
// Clean Code policy mode (MQR or Standard Experience) of a SonarQube
// instance.
type CleanCodePolicyModeClient interface {
	GetMode(ctx context.Context) (*sonar.CleanCodePolicyMode, *http.Response, error)
	UpdateMode(ctx context.Context, opt *sonar.CleanCodePolicyUpdateModeOptions) (*sonar.CleanCodePolicyMode, *http.Response, error)
}

// DefaultProjectVisibilityClient is the interface for updating the default
// visibility of new SonarQube projects. The current value is read through the
// SettingsClient.
type DefaultProjectVisibilityClient interface {
	UpdateDefaultVisibility(ctx context.Context, opt *sonar.ProjectsUpdateDefaultVisibilityOptions) (*http.Response, error)
}

// ComputeEngineWorkerCountClient is the interface for reading and updating
// the number of Compute Engine workers of a SonarQube instance.
type ComputeEngineWorkerCountClient interface {
	WorkerCount(ctx context.Context) (*sonar.CeWorkerCount, *http.Response, error)
	SetWorkerCount(ctx context.Context, opt *sonar.CeSetWorkerCountOptions) (*http.Response, error)
}

// SCIMManagementClient is the interface for reading and toggling SCIM
// provisioning on a SonarQube instance.
type SCIMManagementClient interface {
	Status(ctx context.Context) (*sonar.ScimManagementStatus, *http.Response, error)
	Enable(ctx context.Context) (*http.Response, error)
	Disable(ctx context.Context) (*http.Response, error)
}

// SCAEnablementClient is the interface for reading and toggling Software
// Composition Analysis on a SonarQube instance.
type SCAEnablementClient interface {
	GetEnablement(ctx context.Context) (*sonar.ScaFeatureEnablement, *http.Response, error)
	SetEnablement(ctx context.Context, opt *sonar.ScaSetEnablementOptions) (*sonar.ScaFeatureEnablement, *http.Response, error)
}

// InstanceConfigurationClients groups the SonarQube clients used to manage an
// InstanceConfiguration, one per option family.
type InstanceConfigurationClients struct {
	// CleanCodePolicy reads and updates the Clean Code policy mode.
	CleanCodePolicy CleanCodePolicyModeClient
	// Projects updates the default project visibility.
	Projects DefaultProjectVisibilityClient
	// Settings reads the default project visibility.
	Settings SettingsClient
	// ComputeEngine reads and updates the Compute Engine worker count.
	ComputeEngine ComputeEngineWorkerCountClient
	// SCIM reads and toggles SCIM provisioning.
	SCIM SCIMManagementClient
	// SCA reads and toggles Software Composition Analysis.
	SCA SCAEnablementClient
}

// NewInstanceConfigurationClients creates the InstanceConfigurationClients
// sharing a single SonarQube client built from the provided configuration.
func NewInstanceConfigurationClients(clientConfig common.Config) InstanceConfigurationClients {
	newClient := common.NewClient(clientConfig)

	return InstanceConfigurationClients{
		CleanCodePolicy: newClient.V2.CleanCodePolicy,
		Projects:        newClient.Projects,
		Settings:        newClient.Settings,
		ComputeEngine:   newClient.Ce,
		SCIM:            newClient.ScimManagement,
		SCA:             newClient.V2.Sca,
	}
}

// GenerateCleanCodePolicyUpdateModeOptions generates the options for
// updating the Clean Code policy mode to the given mode.
func GenerateCleanCodePolicyUpdateModeOptions(mode string) *sonar.CleanCodePolicyUpdateModeOptions {
	return &sonar.CleanCodePolicyUpdateModeOptions{Mode: mode}
}

// GenerateUpdateDefaultVisibilityOptions generates the options for updating
// the default visibility of new projects.
func GenerateUpdateDefaultVisibilityOptions(visibility string) *sonar.ProjectsUpdateDefaultVisibilityOptions {
	return &sonar.ProjectsUpdateDefaultVisibilityOptions{ProjectVisibility: visibility}
}

// GenerateDefaultProjectVisibilityValuesOptions generates the options for
// reading the default visibility of new projects from the instance settings.
func GenerateDefaultProjectVisibilityValuesOptions() *sonar.SettingsValuesOptions {
	return &sonar.SettingsValuesOptions{Keys: []string{DefaultProjectVisibilitySettingKey}}
}

// ExtractDefaultProjectVisibility returns the default visibility of new
// projects from a settings values response, or nil if it is not part of it.
func ExtractDefaultProjectVisibility(values *sonar.SettingsValues) *string {
	if values == nil {
		return nil
	}

	for i := range values.Settings {
		if values.Settings[i].Key == DefaultProjectVisibilitySettingKey {
			return &values.Settings[i].Value
		}
	}

	return nil
}

// GenerateSetWorkerCountOptions generates the options for setting the number
// of Compute Engine workers.
func GenerateSetWorkerCountOptions(count int64) *sonar.CeSetWorkerCountOptions {
	return &sonar.CeSetWorkerCountOptions{Count: count}
}

// GenerateSetSCAEnablementOptions generates the options for enabling or
// disabling Software Composition Analysis.
func GenerateSetSCAEnablementOptions(enabled bool) *sonar.ScaSetEnablementOptions {
	return &sonar.ScaSetEnablementOptions{Enablement: enabled}
}

// IsInstanceConfigurationOptionUpToDate reports whether an observed option
// matches the desired one. An option left out of the spec is always up to
// date, and an option that could not be observed never is.
func IsInstanceConfigurationOptionUpToDate[T comparable](desired, observed *T) bool {
	if desired == nil {
		return true
	}

	return observed != nil && *desired == *observed
}
