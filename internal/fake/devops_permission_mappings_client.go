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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
)

// MockDevOpsPermissionMappingsClient is a mock implementation of the
// DevOpsPermissionMappingsClient interface.
type MockDevOpsPermissionMappingsClient struct {
	// SearchFn backs the Search method.
	SearchFn func(platform v1alpha1.DevOpsPlatform) (*sonar.DopTranslationPermissionMappingsSearch, *http.Response, error)
	// CreateFn backs the Create method.
	CreateFn func(platform v1alpha1.DevOpsPlatform, opt *sonar.DopTranslationPermissionMappingCreateOptions) (*sonar.DopTranslationPermissionMapping, *http.Response, error)
	// UpdateFn backs the Update method.
	UpdateFn func(platform v1alpha1.DevOpsPlatform, role string, opt *sonar.DopTranslationPermissionMappingUpdateOptions) (*sonar.DopTranslationPermissionMapping, *http.Response, error)
	// DeleteFn backs the Delete method.
	DeleteFn func(platform v1alpha1.DevOpsPlatform, role string) (*http.Response, error)
}

// Ensure MockDevOpsPermissionMappingsClient implements
// DevOpsPermissionMappingsClient.
var _ iam.DevOpsPermissionMappingsClient = &MockDevOpsPermissionMappingsClient{}

// Search implements DevOpsPermissionMappingsClient.Search.
func (m *MockDevOpsPermissionMappingsClient) Search(_ context.Context, platform v1alpha1.DevOpsPlatform) (*sonar.DopTranslationPermissionMappingsSearch, *http.Response, error) {
	if m.SearchFn != nil {
		return m.SearchFn(platform)
	}

	return &sonar.DopTranslationPermissionMappingsSearch{}, &http.Response{StatusCode: http.StatusOK}, nil
}

// Create implements DevOpsPermissionMappingsClient.Create.
func (m *MockDevOpsPermissionMappingsClient) Create(_ context.Context, platform v1alpha1.DevOpsPlatform, opt *sonar.DopTranslationPermissionMappingCreateOptions) (*sonar.DopTranslationPermissionMapping, *http.Response, error) {
	if m.CreateFn != nil {
		return m.CreateFn(platform, opt)
	}

	return &sonar.DopTranslationPermissionMapping{}, &http.Response{StatusCode: http.StatusOK}, nil
}

// Update implements DevOpsPermissionMappingsClient.Update.
func (m *MockDevOpsPermissionMappingsClient) Update(_ context.Context, platform v1alpha1.DevOpsPlatform, role string, opt *sonar.DopTranslationPermissionMappingUpdateOptions) (*sonar.DopTranslationPermissionMapping, *http.Response, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(platform, role, opt)
	}

	return &sonar.DopTranslationPermissionMapping{}, &http.Response{StatusCode: http.StatusOK}, nil
}

// Delete implements DevOpsPermissionMappingsClient.Delete.
func (m *MockDevOpsPermissionMappingsClient) Delete(_ context.Context, platform v1alpha1.DevOpsPlatform, role string) (*http.Response, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(platform, role)
	}

	return &http.Response{StatusCode: http.StatusNoContent}, nil
}
