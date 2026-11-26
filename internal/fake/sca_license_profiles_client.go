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

package fake

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

// MockSCALicenseProfilesClient is a mock implementation of the
// SCALicenseProfilesClient interface.
type MockSCALicenseProfilesClient struct {
	AssignLicenseProfileProjectFn          func(body *sonar.ScaLicenseProfileAssignmentRequest) (*sonar.ScaLicenseProfile, *http.Response, error)
	CreateLicenseProfileFn                 func(body *sonar.ScaLicenseProfileCreateRequest) (*sonar.ScaLicenseProfile, *http.Response, error)
	DeleteLicenseProfileFn                 func(opt *sonar.ScaLicenseProfileGetOptions) (*http.Response, error)
	DeleteLicenseProfileAssignmentFn       func(opt *sonar.ScaLicenseProfileAssignmentDeleteOptions) (*http.Response, error)
	GetLicenseProfileFn                    func(opt *sonar.ScaLicenseProfileGetOptions) (*sonar.ScaLicenseProfileDetails, *http.Response, error)
	ListLicenseProfileAssignableProjectsFn func(opt *sonar.ScaLicenseProfileAssignableProjectsOptions) (*sonar.ScaLicenseProfileAssignableProjectsResponse, *http.Response, error)
	ListLicenseProfilesFn                  func(opt *sonar.ScaLicenseProfileListOptions) (*sonar.ScaLicenseProfileIndex, *http.Response, error)
	UpdateLicenseProfileFn                 func(key string, body *sonar.ScaLicenseProfileUpdateRequest) (*sonar.ScaLicenseProfile, *http.Response, error)
	UpdateLicenseProfileCategoryFn         func(opt *sonar.ScaLicenseProfileCategoryOptions, body *sonar.ScaLicenseProfileCategoryUpdateRequest) (*sonar.ScaLicenseProfileCategory, *http.Response, error)
	UpdateLicenseProfileLicenseFn          func(opt *sonar.ScaLicenseProfileLicenseOptions, body *sonar.ScaLicensePolicyLicenseUpdateRequest) (*sonar.ScaLicensePolicyLicense, *http.Response, error)
}

// Ensure MockSCALicenseProfilesClient implements SCALicenseProfilesClient.
var _ instance.SCALicenseProfilesClient = &MockSCALicenseProfilesClient{}

// AssignLicenseProfileProject implements the SCALicenseProfilesClient
// method of the same name.
func (m *MockSCALicenseProfilesClient) AssignLicenseProfileProject(_ context.Context, body *sonar.ScaLicenseProfileAssignmentRequest) (*sonar.ScaLicenseProfile, *http.Response, error) {
	if m.AssignLicenseProfileProjectFn != nil {
		return m.AssignLicenseProfileProjectFn(body)
	}

	return nil, nil, errNotImplemented
}

// CreateLicenseProfile implements the SCALicenseProfilesClient
// method of the same name.
func (m *MockSCALicenseProfilesClient) CreateLicenseProfile(_ context.Context, body *sonar.ScaLicenseProfileCreateRequest) (*sonar.ScaLicenseProfile, *http.Response, error) {
	if m.CreateLicenseProfileFn != nil {
		return m.CreateLicenseProfileFn(body)
	}

	return nil, nil, errNotImplemented
}

// DeleteLicenseProfile implements the SCALicenseProfilesClient
// method of the same name.
func (m *MockSCALicenseProfilesClient) DeleteLicenseProfile(_ context.Context, opt *sonar.ScaLicenseProfileGetOptions) (*http.Response, error) {
	if m.DeleteLicenseProfileFn != nil {
		return m.DeleteLicenseProfileFn(opt)
	}

	return nil, errNotImplemented
}

// DeleteLicenseProfileAssignment implements the SCALicenseProfilesClient
// method of the same name.
func (m *MockSCALicenseProfilesClient) DeleteLicenseProfileAssignment(_ context.Context, opt *sonar.ScaLicenseProfileAssignmentDeleteOptions) (*http.Response, error) {
	if m.DeleteLicenseProfileAssignmentFn != nil {
		return m.DeleteLicenseProfileAssignmentFn(opt)
	}

	return nil, errNotImplemented
}

// GetLicenseProfile implements SCALicenseProfilesClient.GetLicenseProfile.
func (m *MockSCALicenseProfilesClient) GetLicenseProfile(_ context.Context, opt *sonar.ScaLicenseProfileGetOptions) (*sonar.ScaLicenseProfileDetails, *http.Response, error) {
	if m.GetLicenseProfileFn != nil {
		return m.GetLicenseProfileFn(opt)
	}

	return nil, nil, errNotImplemented
}

// ListLicenseProfileAssignableProjects implements the SCALicenseProfilesClient
// method of the same name.
func (m *MockSCALicenseProfilesClient) ListLicenseProfileAssignableProjects(_ context.Context, opt *sonar.ScaLicenseProfileAssignableProjectsOptions) (*sonar.ScaLicenseProfileAssignableProjectsResponse, *http.Response, error) {
	if m.ListLicenseProfileAssignableProjectsFn != nil {
		return m.ListLicenseProfileAssignableProjectsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// ListLicenseProfiles implements the SCALicenseProfilesClient
// method of the same name.
func (m *MockSCALicenseProfilesClient) ListLicenseProfiles(_ context.Context, opt *sonar.ScaLicenseProfileListOptions) (*sonar.ScaLicenseProfileIndex, *http.Response, error) {
	if m.ListLicenseProfilesFn != nil {
		return m.ListLicenseProfilesFn(opt)
	}

	return nil, nil, errNotImplemented
}

// UpdateLicenseProfile implements the SCALicenseProfilesClient
// method of the same name.
func (m *MockSCALicenseProfilesClient) UpdateLicenseProfile(_ context.Context, key string, body *sonar.ScaLicenseProfileUpdateRequest) (*sonar.ScaLicenseProfile, *http.Response, error) {
	if m.UpdateLicenseProfileFn != nil {
		return m.UpdateLicenseProfileFn(key, body)
	}

	return nil, nil, errNotImplemented
}

// UpdateLicenseProfileCategory implements the SCALicenseProfilesClient
// method of the same name.
func (m *MockSCALicenseProfilesClient) UpdateLicenseProfileCategory(_ context.Context, opt *sonar.ScaLicenseProfileCategoryOptions, body *sonar.ScaLicenseProfileCategoryUpdateRequest) (*sonar.ScaLicenseProfileCategory, *http.Response, error) {
	if m.UpdateLicenseProfileCategoryFn != nil {
		return m.UpdateLicenseProfileCategoryFn(opt, body)
	}

	return nil, nil, errNotImplemented
}

// UpdateLicenseProfileLicense implements the SCALicenseProfilesClient
// method of the same name.
func (m *MockSCALicenseProfilesClient) UpdateLicenseProfileLicense(_ context.Context, opt *sonar.ScaLicenseProfileLicenseOptions, body *sonar.ScaLicensePolicyLicenseUpdateRequest) (*sonar.ScaLicensePolicyLicense, *http.Response, error) {
	if m.UpdateLicenseProfileLicenseFn != nil {
		return m.UpdateLicenseProfileLicenseFn(opt, body)
	}

	return nil, nil, errNotImplemented
}
