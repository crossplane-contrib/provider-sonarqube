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
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// errDecodeJiraBinding indicates a Jira binding returned by SonarQube could
// not be decoded.
const errDecodeJiraBinding = "cannot decode Jira binding"

// JiraClient is the interface for interacting with the binding endpoints of
// the SonarQube Jira V2 API. They are only available in Enterprise Edition and
// above, and SonarQube marks them as internal.
type JiraClient interface {
	GetOrganizationBinding(ctx context.Context, opt *sonar.JiraOrganizationBindingOptions) (*sonar.JiraOrganizationBinding, *http.Response, error)
	UpdateOrganizationBinding(ctx context.Context, body sonar.JiraOrganizationBinding) (*sonar.JiraOrganizationBinding, *http.Response, error)
	DeleteOrganizationBinding(ctx context.Context, opt *sonar.JiraOrganizationBindingOptions) (*http.Response, error)
	GetProjectBinding(ctx context.Context, opt *sonar.JiraProjectBindingOptions) (*sonar.JiraProjectBinding, *http.Response, error)
	CreateProjectBinding(ctx context.Context, body sonar.JiraProjectBinding) (*sonar.JiraProjectBinding, *http.Response, error)
	DeleteProjectBinding(ctx context.Context, opt *sonar.JiraProjectBindingOptions) (*http.Response, error)
}

// NewJiraClient creates a new JiraClient with the provided SonarQube client
// configuration.
func NewJiraClient(clientConfig common.Config) JiraClient {
	return common.NewClient(clientConfig).V2.Jira
}

// JiraOrganizationBinding is the typed form of the organization binding, which
// the SonarQube client only exposes as a map.
type JiraOrganizationBinding struct {
	ID                    string `json:"id"`
	SonarOrganizationUUID string `json:"sonarOrganizationUuid"`
	JiraInstanceURL       string `json:"jiraInstanceUrl"`
	CreatedBy             string `json:"createdBy"`
	// CreatedAt is a Unix timestamp in milliseconds.
	CreatedAt     int64 `json:"createdAt"`
	IsTokenShared bool  `json:"isTokenShared"`
}

// JiraProjectBinding is the typed form of the project binding, which the
// SonarQube client only exposes as a map.
type JiraProjectBinding struct {
	ID                 string `json:"id"`
	SonarProjectID     string `json:"sonarProjectId"`
	JiraOrganizationID string `json:"jiraOrganizationId"`
	JiraProjectKey     string `json:"jiraProjectKey"`
}

// ParseJiraOrganizationBinding converts the response of the organization
// binding endpoints. It returns nil when the response holds no binding.
func ParseJiraOrganizationBinding(raw *sonar.JiraOrganizationBinding) (*JiraOrganizationBinding, error) {
	binding := &JiraOrganizationBinding{}

	err := decodeJiraBinding(raw, binding)
	if err != nil || binding.ID == "" {
		return nil, err
	}

	return binding, nil
}

// ParseJiraProjectBinding converts the response of the project binding
// endpoints. It returns nil when the response holds no binding.
func ParseJiraProjectBinding(raw *sonar.JiraProjectBinding) (*JiraProjectBinding, error) {
	binding := &JiraProjectBinding{}

	err := decodeJiraBinding(raw, binding)
	if err != nil || binding.ID == "" {
		return nil, err
	}

	return binding, nil
}

// decodeJiraBinding decodes the loosely typed binding raw into out.
func decodeJiraBinding(raw, out any) error {
	data, err := json.Marshal(raw)
	if err != nil {
		return errors.Wrap(err, errDecodeJiraBinding)
	}

	return errors.Wrap(json.Unmarshal(data, out), errDecodeJiraBinding)
}

// GenerateJiraOrganizationBindingOptions generates the options identifying the
// organization binding of the given SonarQube organization.
func GenerateJiraOrganizationBindingOptions(spec *v1alpha1.JiraOrganizationBindingParameters) *sonar.JiraOrganizationBindingOptions {
	return &sonar.JiraOrganizationBindingOptions{SonarOrganizationUuid: spec.SonarOrganizationUUID}
}

// GenerateJiraOrganizationBindingUpdate generates the body updating the
// editable properties of the organization binding. IsTokenShared must be set.
func GenerateJiraOrganizationBindingUpdate(spec *v1alpha1.JiraOrganizationBindingParameters) sonar.JiraOrganizationBinding {
	return sonar.JiraOrganizationBinding{
		"sonarOrganizationUuid": spec.SonarOrganizationUUID,
		"isTokenShared":         ptr.Deref(spec.IsTokenShared, false),
	}
}

// GenerateJiraOrganizationBindingObservation generates the observation of the
// given organization binding.
func GenerateJiraOrganizationBindingObservation(binding *JiraOrganizationBinding) v1alpha1.JiraOrganizationBindingObservation {
	observation := v1alpha1.JiraOrganizationBindingObservation{
		ID:                    binding.ID,
		SonarOrganizationUUID: binding.SonarOrganizationUUID,
		JiraInstanceURL:       binding.JiraInstanceURL,
		CreatedBy:             binding.CreatedBy,
		IsTokenShared:         binding.IsTokenShared,
	}

	if binding.CreatedAt > 0 {
		observation.CreatedAt = new(metav1.NewTime(time.UnixMilli(binding.CreatedAt)))
	}

	return observation
}

// LateInitializeJiraOrganizationBinding fills the unset fields of spec from
// the observation, and reports whether it changed spec.
func LateInitializeJiraOrganizationBinding(spec *v1alpha1.JiraOrganizationBindingParameters, observation *v1alpha1.JiraOrganizationBindingObservation) bool {
	if spec.IsTokenShared != nil {
		return false
	}

	spec.IsTokenShared = new(observation.IsTokenShared)

	return true
}

// IsJiraOrganizationBindingUpToDate reports whether the observed organization
// binding matches the desired state.
func IsJiraOrganizationBindingUpToDate(spec *v1alpha1.JiraOrganizationBindingParameters, observation *v1alpha1.JiraOrganizationBindingObservation) bool {
	return spec.IsTokenShared == nil || *spec.IsTokenShared == observation.IsTokenShared
}

// GenerateJiraProjectBindingOptions generates the options identifying the
// binding of the SonarQube project with the given ID.
func GenerateJiraProjectBindingOptions(projectID string) *sonar.JiraProjectBindingOptions {
	return &sonar.JiraProjectBindingOptions{SonarProjectId: projectID}
}

// GenerateJiraProjectBindingBody generates the body creating or updating the
// binding of the SonarQube project with the given ID.
func GenerateJiraProjectBindingBody(projectID string, spec *v1alpha1.JiraProjectBindingParameters) sonar.JiraProjectBinding {
	return sonar.JiraProjectBinding{
		"sonarProjectId":     projectID,
		"jiraOrganizationId": ptr.Deref(spec.JiraOrganizationBindingID, ""),
		"jiraProjectKey":     spec.JiraProjectKey,
	}
}

// GenerateJiraProjectBindingObservation generates the observation of the given
// project binding.
func GenerateJiraProjectBindingObservation(binding *JiraProjectBinding) v1alpha1.JiraProjectBindingObservation {
	return v1alpha1.JiraProjectBindingObservation{
		ID:                        binding.ID,
		SonarProjectID:            binding.SonarProjectID,
		JiraOrganizationBindingID: binding.JiraOrganizationID,
		JiraProjectKey:            binding.JiraProjectKey,
	}
}

// IsJiraProjectBindingUpToDate reports whether the observed project binding
// matches the desired state.
func IsJiraProjectBindingUpToDate(spec *v1alpha1.JiraProjectBindingParameters, observation *v1alpha1.JiraProjectBindingObservation) bool {
	return spec.JiraProjectKey == observation.JiraProjectKey &&
		(spec.JiraOrganizationBindingID == nil || *spec.JiraOrganizationBindingID == observation.JiraOrganizationBindingID)
}
