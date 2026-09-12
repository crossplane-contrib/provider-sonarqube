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

// Package instance provides clients for managing SonarQube instance resources.
package instance

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// NewCodePeriodsClient is the interface for managing new code periods
// in SonarQube projects.
type NewCodePeriodsClient interface {
	List(ctx context.Context, opt *sonar.NewCodePeriodsListOptions) (*sonar.NewCodePeriodsList, *http.Response, error)
	Set(ctx context.Context, opt *sonar.NewCodePeriodsSetOptions) (*http.Response, error)
	Show(ctx context.Context, opt *sonar.NewCodePeriodsShowOptions) (*sonar.NewCodePeriodsShow, *http.Response, error)
	Unset(ctx context.Context, opt *sonar.NewCodePeriodsUnsetOptions) (*http.Response, error)
}

// NewNewCodePeriodsClient creates a new NewCodePeriodsClient with the provided
// SonarQube client configuration.
func NewNewCodePeriodsClient(clientConfig common.Config) NewCodePeriodsClient {
	newClient := common.NewClient(clientConfig)

	return newClient.NewCodePeriods
}

// GenerateNewCodePeriodsShowOptions generates the options for showing the new
// code period of a SonarQube Project based on the provided project key.
func GenerateNewCodePeriodsShowOptions(projectKey, branch *string) *sonar.NewCodePeriodsShowOptions {
	opts := sonar.NewCodePeriodsShowOptions{}
	helpers.AssignIfNonNil(&opts.Project, projectKey)
	helpers.AssignIfNonNil(&opts.Branch, branch)

	return &opts
}

// GenerateProjectNewCodePeriodsSetOptions generates the options for setting the
// new code period of a SonarQube Project based on the provided
// ProjectParameters.
func GenerateProjectNewCodePeriodsSetOptions(projectKey string, newCodePeriodParameters *v1alpha1.ProjectNewCodePeriodParameters) *sonar.NewCodePeriodsSetOptions {
	opts := sonar.NewCodePeriodsSetOptions{
		Project: projectKey,
	}
	if newCodePeriodParameters != nil {
		opts.Type = newCodePeriodParameters.Type
		helpers.AssignIfNonNil(&opts.Value, newCodePeriodParameters.Value)
	}

	return &opts
}

// GenerateBranchNewCodePeriodsSetOptions generates the options for setting the
// new code period of a SonarQube Project branch based on the provided branch
// name and ProjectParameters.
func GenerateBranchNewCodePeriodsSetOptions(projectKey, branchName string, newCodePeriodParameters *v1alpha1.ProjectNewCodePeriodParameters) *sonar.NewCodePeriodsSetOptions {
	opts := sonar.NewCodePeriodsSetOptions{
		Project: projectKey,
		Branch:  branchName,
	}
	if newCodePeriodParameters != nil {
		opts.Type = newCodePeriodParameters.Type
		helpers.AssignIfNonNil(&opts.Value, newCodePeriodParameters.Value)
	}

	return &opts
}

// GenerateProjectNewCodePeriodsListOptions generates the options for listing
// the new code periods of a SonarQube Project based on the provided
// project key.
func GenerateProjectNewCodePeriodsListOptions(projectKey string) *sonar.NewCodePeriodsListOptions {
	return &sonar.NewCodePeriodsListOptions{
		Project: projectKey,
	}
}

// IsNewCodePeriodUpToDate checks if the observed new code period of a
// SonarQube Project is up to date with the new code definition specified
// in the managed resource.
func IsNewCodePeriodUpToDate(spec *v1alpha1.ProjectNewCodePeriodParameters, observation *v1alpha1.ProjectNewCodePeriodObservation) bool {
	if observation == nil {
		return false
	}

	return spec == nil ||
		spec.Type == observation.Type &&
			helpers.IsComparablePtrEqualComparable(spec.Value, observation.Value)
}

// GenerateProjectNewCodePeriodObservation generates the observation for the new
// code period of a SonarQube Project based on the provided
// NewCodePeriodsShow response.
func GenerateProjectNewCodePeriodObservation(obs *sonar.NewCodePeriodsShow) v1alpha1.ProjectNewCodePeriodObservation {
	return v1alpha1.ProjectNewCodePeriodObservation{
		Type:      obs.Type,
		Value:     obs.Value,
		Inherited: obs.Inherited,
	}
}

// GenerateBranchNewCodePeriodObservation generates the observation for the
// new code period of a SonarQube Project branch based on the provided
// NewCodePeriod response.
func GenerateBranchNewCodePeriodObservation(obs *sonar.NewCodePeriod) v1alpha1.ProjectNewCodePeriodObservation {
	return v1alpha1.ProjectNewCodePeriodObservation{
		Type:           obs.Type,
		Value:          obs.Value,
		Inherited:      obs.Inherited,
		EffectiveValue: obs.EffectiveValue,
	}
}

// LateInitializeProjectNewCodePeriod performs late initialization of the
// ProjectNewCodePeriodParameters in the ProjectParameters based on the observed
// new code period from SonarQube.
func LateInitializeProjectNewCodePeriod(spec *v1alpha1.ProjectNewCodePeriodParameters, observation *v1alpha1.ProjectNewCodePeriodObservation) {
	if spec == nil || observation == nil {
		return
	}

	helpers.AssignIfNil(&spec.Value, observation.Value)
}

// GenerateInstanceNewCodePeriodsShowOptions generates the options for showing
// the instance-wide default new code period.
func GenerateInstanceNewCodePeriodsShowOptions() *sonar.NewCodePeriodsShowOptions {
	return &sonar.NewCodePeriodsShowOptions{}
}

// GenerateInstanceNewCodePeriodsSetOptions generates the options for setting
// the instance-wide default new code period. No project or branch is
// provided, which makes SonarQube update the global level.
func GenerateInstanceNewCodePeriodsSetOptions(params *v1alpha1.NewCodePeriodParameters) *sonar.NewCodePeriodsSetOptions {
	opts := sonar.NewCodePeriodsSetOptions{}
	if params != nil {
		opts.Type = params.Type
		helpers.AssignIfNonNil(&opts.Value, params.Value)
	}

	return &opts
}

// GenerateInstanceNewCodePeriodsUnsetOptions generates the options for
// unsetting the instance-wide default new code period.
func GenerateInstanceNewCodePeriodsUnsetOptions() *sonar.NewCodePeriodsUnsetOptions {
	return &sonar.NewCodePeriodsUnsetOptions{}
}

// GenerateInstanceNewCodePeriodObservation generates the observation for the
// instance-wide default new code period from a Show response.
func GenerateInstanceNewCodePeriodObservation(obs *sonar.NewCodePeriodsShow) v1alpha1.NewCodePeriodObservation {
	if obs == nil {
		return v1alpha1.NewCodePeriodObservation{}
	}

	return v1alpha1.NewCodePeriodObservation{
		Type:      obs.Type,
		Value:     obs.Value,
		Inherited: obs.Inherited,
		UpdatedAt: obs.UpdatedAt,
	}
}

// AreInstanceNewCodePeriodsUpToDate checks whether the observed instance-wide
// default new code period matches the desired one.
func AreInstanceNewCodePeriodsUpToDate(spec *v1alpha1.NewCodePeriodParameters, observation *v1alpha1.NewCodePeriodObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	if spec.Type != observation.Type {
		return false
	}

	if spec.Value == nil {
		return observation.Value == ""
	}

	return *spec.Value == observation.Value
}

// LateInitializeInstanceNewCodePeriod fills the empty fields of the desired
// instance-wide default new code period with the observed ones.
func LateInitializeInstanceNewCodePeriod(spec *v1alpha1.NewCodePeriodParameters, observation *v1alpha1.NewCodePeriodObservation) {
	if spec == nil || observation == nil {
		return
	}

	if spec.Type == "" {
		spec.Type = observation.Type
	}

	helpers.AssignIfNil(&spec.Value, observation.Value)
}
