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

package instanceconfiguration

import (
	"context"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// option reconciles one optional field of an InstanceConfiguration through
// its own SonarQube endpoints. The controller only calls observe, isUpToDate
// and apply for the options whose isSet returns true, so the endpoints of an
// option left out of the spec are never called. Supporting a new option only
// takes a new entry in options.
type option struct {
	// name identifies the option in error messages.
	name string
	// isSet reports whether the option is set in the spec.
	isSet func(params *v1alpha1.InstanceConfigurationParameters) bool
	// observe reads the option from SonarQube into the observation.
	observe func(ctx context.Context, clients *instance.InstanceConfigurationClients, observation *v1alpha1.InstanceConfigurationObservation) error
	// isUpToDate reports whether the observed option matches the spec.
	isUpToDate func(params *v1alpha1.InstanceConfigurationParameters, observation *v1alpha1.InstanceConfigurationObservation) bool
	// apply sets the option in SonarQube to its value in the spec.
	apply func(ctx context.Context, clients *instance.InstanceConfigurationClients, params *v1alpha1.InstanceConfigurationParameters) error
}

// options lists every option an InstanceConfiguration manages, in the order
// they are observed and applied. The Community Edition options come first.
//
//nolint:gochecknoglobals // static table of the managed options
var options = []option{
	cleanCodePolicyModeOption,
	defaultProjectVisibilityOption,
	computeEngineWorkerCountOption,
	scimEnabledOption,
	scaEnabledOption,
}

// cleanCodePolicyModeOption manages the Clean Code policy mode.
//
//nolint:gochecknoglobals,dupl // static option definition, same shape as scaEnabledOption but bound to distinct SonarQube API types
var cleanCodePolicyModeOption = option{
	name: "clean code policy mode",
	isSet: func(params *v1alpha1.InstanceConfigurationParameters) bool {
		return params.CleanCodePolicyMode != nil
	},
	observe: func(ctx context.Context, clients *instance.InstanceConfigurationClients, observation *v1alpha1.InstanceConfigurationObservation) error {
		mode, resp, err := clients.CleanCodePolicy.GetMode(ctx)
		defer helpers.CloseBody(resp)

		if err != nil {
			return err
		}

		if mode != nil {
			observation.CleanCodePolicyMode = &mode.Mode
		}

		return nil
	},
	isUpToDate: func(params *v1alpha1.InstanceConfigurationParameters, observation *v1alpha1.InstanceConfigurationObservation) bool {
		return instance.IsInstanceConfigurationOptionUpToDate(params.CleanCodePolicyMode, observation.CleanCodePolicyMode)
	},
	apply: func(ctx context.Context, clients *instance.InstanceConfigurationClients, params *v1alpha1.InstanceConfigurationParameters) error {
		_, resp, err := clients.CleanCodePolicy.UpdateMode(ctx, instance.GenerateCleanCodePolicyUpdateModeOptions(*params.CleanCodePolicyMode))
		defer helpers.CloseBody(resp)

		return err
	},
}

// defaultProjectVisibilityOption manages the default visibility of new
// projects. SonarQube has no dedicated read endpoint for it, so it is read
// from the instance settings, where update_default_visibility stores it.
//
//nolint:gochecknoglobals // static option definition
var defaultProjectVisibilityOption = option{
	name: "default project visibility",
	isSet: func(params *v1alpha1.InstanceConfigurationParameters) bool {
		return params.DefaultProjectVisibility != nil
	},
	observe: func(ctx context.Context, clients *instance.InstanceConfigurationClients, observation *v1alpha1.InstanceConfigurationObservation) error {
		values, resp, err := clients.Settings.Values(ctx, instance.GenerateDefaultProjectVisibilityValuesOptions())
		defer helpers.CloseBody(resp)

		if err != nil {
			return err
		}

		observation.DefaultProjectVisibility = instance.ExtractDefaultProjectVisibility(values)

		return nil
	},
	isUpToDate: func(params *v1alpha1.InstanceConfigurationParameters, observation *v1alpha1.InstanceConfigurationObservation) bool {
		return instance.IsInstanceConfigurationOptionUpToDate(params.DefaultProjectVisibility, observation.DefaultProjectVisibility)
	},
	apply: func(ctx context.Context, clients *instance.InstanceConfigurationClients, params *v1alpha1.InstanceConfigurationParameters) error {
		resp, err := clients.Projects.UpdateDefaultVisibility(ctx, instance.GenerateUpdateDefaultVisibilityOptions(*params.DefaultProjectVisibility))
		defer helpers.CloseBody(resp)

		return err
	},
}

// computeEngineWorkerCountOption manages the number of Compute Engine
// workers.
//
//nolint:gochecknoglobals // static option definition
var computeEngineWorkerCountOption = option{
	name: "compute engine worker count",
	isSet: func(params *v1alpha1.InstanceConfigurationParameters) bool {
		return params.ComputeEngineWorkerCount != nil
	},
	observe: func(ctx context.Context, clients *instance.InstanceConfigurationClients, observation *v1alpha1.InstanceConfigurationObservation) error {
		count, resp, err := clients.ComputeEngine.WorkerCount(ctx)
		defer helpers.CloseBody(resp)

		if err != nil {
			return err
		}

		if count != nil {
			observation.ComputeEngineWorkerCount = &count.Value
		}

		return nil
	},
	isUpToDate: func(params *v1alpha1.InstanceConfigurationParameters, observation *v1alpha1.InstanceConfigurationObservation) bool {
		return instance.IsInstanceConfigurationOptionUpToDate(params.ComputeEngineWorkerCount, observation.ComputeEngineWorkerCount)
	},
	apply: func(ctx context.Context, clients *instance.InstanceConfigurationClients, params *v1alpha1.InstanceConfigurationParameters) error {
		resp, err := clients.ComputeEngine.SetWorkerCount(ctx, instance.GenerateSetWorkerCountOptions(*params.ComputeEngineWorkerCount))
		defer helpers.CloseBody(resp)

		return err
	},
}

// scimEnabledOption manages whether SCIM provisioning is enabled.
//
//nolint:gochecknoglobals // static option definition
var scimEnabledOption = option{
	name: "SCIM enablement",
	isSet: func(params *v1alpha1.InstanceConfigurationParameters) bool {
		return params.SCIMEnabled != nil
	},
	observe: func(ctx context.Context, clients *instance.InstanceConfigurationClients, observation *v1alpha1.InstanceConfigurationObservation) error {
		status, resp, err := clients.SCIM.Status(ctx)
		defer helpers.CloseBody(resp)

		if err != nil {
			return err
		}

		if status != nil {
			observation.SCIMEnabled = &status.Enabled
		}

		return nil
	},
	isUpToDate: func(params *v1alpha1.InstanceConfigurationParameters, observation *v1alpha1.InstanceConfigurationObservation) bool {
		return instance.IsInstanceConfigurationOptionUpToDate(params.SCIMEnabled, observation.SCIMEnabled)
	},
	apply: func(ctx context.Context, clients *instance.InstanceConfigurationClients, params *v1alpha1.InstanceConfigurationParameters) error {
		toggle := clients.SCIM.Disable
		if *params.SCIMEnabled {
			toggle = clients.SCIM.Enable
		}

		resp, err := toggle(ctx)
		defer helpers.CloseBody(resp)

		return err
	},
}

// scaEnabledOption manages whether Software Composition Analysis is enabled.
//
//nolint:gochecknoglobals,dupl // static option definition, same shape as cleanCodePolicyModeOption but bound to distinct SonarQube API types
var scaEnabledOption = option{
	name: "SCA enablement",
	isSet: func(params *v1alpha1.InstanceConfigurationParameters) bool {
		return params.SCAEnabled != nil
	},
	observe: func(ctx context.Context, clients *instance.InstanceConfigurationClients, observation *v1alpha1.InstanceConfigurationObservation) error {
		enablement, resp, err := clients.SCA.GetEnablement(ctx)
		defer helpers.CloseBody(resp)

		if err != nil {
			return err
		}

		if enablement != nil {
			observation.SCAEnabled = &enablement.Enablement
		}

		return nil
	},
	isUpToDate: func(params *v1alpha1.InstanceConfigurationParameters, observation *v1alpha1.InstanceConfigurationObservation) bool {
		return instance.IsInstanceConfigurationOptionUpToDate(params.SCAEnabled, observation.SCAEnabled)
	},
	apply: func(ctx context.Context, clients *instance.InstanceConfigurationClients, params *v1alpha1.InstanceConfigurationParameters) error {
		_, resp, err := clients.SCA.SetEnablement(ctx, instance.GenerateSetSCAEnablementOptions(*params.SCAEnabled))
		defer helpers.CloseBody(resp)

		return err
	},
}
