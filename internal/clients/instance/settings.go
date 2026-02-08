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
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// SettingsClient is the interface for interacting with SonarQube Settings API
// It handles all the operations related to Settings in SonarQube, such as creating, updating, deleting, and retrieving Settings.
type SettingsClient interface {
	Set(opt *sonar.SettingsSetOption) (*http.Response, error)
	Values(opt *sonar.SettingsValuesOption) (*sonar.SettingsValues, *http.Response, error)
	Reset(opt *sonar.SettingsResetOption) (*http.Response, error)
}

// NewSettingsClient creates a new SettingsClient with the provided SonarQube client configuration.
func NewSettingsClient(clientConfig common.Config) SettingsClient {
	newClient := common.NewClient(clientConfig)
	return newClient.Settings
}

// GenerateSettingSetOptions generates the options for the Set API call based on the provided settings parameters and component.
func GenerateSettingSetOptions(params v1alpha1.SettingParameters, component *string) *sonar.SettingsSetOption {
	settingsSetOptions := &sonar.SettingsSetOption{
		Key: params.Key,
	}
	helpers.AssignIfNonNil(&settingsSetOptions.Value, params.Value)
	helpers.AssignIfNonNil(&settingsSetOptions.Component, component)

	if params.Values != nil && len(*params.Values) > 0 {
		settingsSetOptions.Values = *params.Values
	}

	if params.FieldValues != nil && len(*params.FieldValues) > 0 {
		settingsSetOptions.FieldValues = sonar.JSONEncodedMap{}
		for k, v := range *params.FieldValues {
			settingsSetOptions.FieldValues[k] = v
		}
	}

	return settingsSetOptions
}

// GenerateSettingsValuesOptions generates the options for the Values API call based on the provided component and keys.
func GenerateSettingsValuesOptions(params *v1alpha1.SettingsParameters) *sonar.SettingsValuesOption {
	keys := make([]string, 0, len(params.Settings))
	for key := range params.Settings {
		keys = append(keys, key)
	}

	settingsValuesOptions := &sonar.SettingsValuesOption{
		Keys: keys,
	}
	helpers.AssignIfNonNil(&settingsValuesOptions.Component, params.Component)

	return settingsValuesOptions
}

// GenerateSettingsResetOptions generates the options for the Reset API call based on the provided settings parameters and component.
func GenerateSettingsResetOptions(params v1alpha1.SettingsParameters) *sonar.SettingsResetOption {
	keys := make([]string, 0, len(params.Settings))
	for key := range params.Settings {
		keys = append(keys, key)
	}
	settingsResetOptions := &sonar.SettingsResetOption{
		Keys: keys,
	}
	helpers.AssignIfNonNil(&settingsResetOptions.Component, params.Component)

	return settingsResetOptions
}

// GenerateSettingsResetOptionsFromList generates the options for the Reset API call based on the provided list of keys and component.
func GenerateSettingsResetOptionsFromList(keys []string, component *string) *sonar.SettingsResetOption {
	settingsResetOptions := &sonar.SettingsResetOption{
		Keys: keys,
	}
	helpers.AssignIfNonNil(&settingsResetOptions.Component, component)

	return settingsResetOptions
}

// GenerateSettingsObservation generates the SettingsObservation based on the observed SettingsValues from SonarQube.
func GenerateSettingsObservation(observed *sonar.SettingsValues) v1alpha1.SettingsObservation {
	settingsObservation := v1alpha1.SettingsObservation{
		Settings: make(map[string]v1alpha1.SettingObservation),
	}

	for _, setting := range observed.Settings {
		settingsObservation.Settings[setting.Key] = GenerateSettingObservation(&setting)
	}

	return settingsObservation
}

// GenerateSettingObservation generates the SettingObservation based on the observed SettingValue from SonarQube.
func GenerateSettingObservation(observed *sonar.SettingValue) v1alpha1.SettingObservation {
	fieldValues := make(map[string]string)
	for _, fieldValue := range observed.FieldValues {
		for k, v := range fieldValue {
			fieldValues[k] = v
		}
	}
	return v1alpha1.SettingObservation{
		Value:       observed.Value,
		Values:      observed.Values,
		FieldValues: fieldValues,
	}
}

// IsSettingUpToDate checks if the observed setting is up to date with the desired setting parameters.
func IsSettingUpToDate(params v1alpha1.SettingParameters, observation v1alpha1.SettingObservation) bool {
	return helpers.IsComparablePtrEqualComparable(params.Value, observation.Value) &&
		helpers.IsComparableSlicePtrEqualComparableSlice(params.Values, observation.Values) &&
		helpers.IsComparableMapPtrEqualComparableMap(params.FieldValues, observation.FieldValues)
}

// AreSettingsUpToDate checks if the observed settings are up to date with the desired settings parameters.
func AreSettingsUpToDate(params v1alpha1.SettingsParameters, observation v1alpha1.SettingsObservation) bool {
	for key, param := range params.Settings {
		observationSetting, exists := observation.Settings[key]
		if !exists || !IsSettingUpToDate(param, observationSetting) {
			return false
		}
	}
	return true
}
