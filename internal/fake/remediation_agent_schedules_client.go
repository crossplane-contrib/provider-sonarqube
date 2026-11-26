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

	instance "github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

// MockRemediationAgentSchedulesClient is a mock implementation of the
// RemediationAgentSchedulesClient interface.
type MockRemediationAgentSchedulesClient struct {
	// GetScheduledAgentConfigFn backs the GetScheduledAgentConfig method.
	GetScheduledAgentConfigFn func(id string) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error)
	// CreateScheduledAgentConfigFn backs the CreateScheduledAgentConfig method.
	CreateScheduledAgentConfigFn func(opt *sonar.RemediationAgentCreateScheduledAgentConfigOptions) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error)
	// UpdateScheduledAgentConfigFn backs the UpdateScheduledAgentConfig method.
	UpdateScheduledAgentConfigFn func(id string, opt *sonar.RemediationAgentUpdateScheduledAgentConfigOptions) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error)
	// DeleteScheduledAgentConfigFn backs the DeleteScheduledAgentConfig method.
	DeleteScheduledAgentConfigFn func(id string) (*http.Response, error)
}

// Ensure MockRemediationAgentSchedulesClient implements
// RemediationAgentSchedulesClient.
var _ instance.RemediationAgentSchedulesClient = &MockRemediationAgentSchedulesClient{}

// GetScheduledAgentConfig implements
// RemediationAgentSchedulesClient.GetScheduledAgentConfig.
func (m *MockRemediationAgentSchedulesClient) GetScheduledAgentConfig(_ context.Context, id string) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error) {
	if m.GetScheduledAgentConfigFn != nil {
		return m.GetScheduledAgentConfigFn(id)
	}

	return &sonar.RemediationAgentScheduledAgentConfigResource{Id: id}, &http.Response{StatusCode: http.StatusOK}, nil
}

// CreateScheduledAgentConfig implements
// RemediationAgentSchedulesClient.CreateScheduledAgentConfig.
func (m *MockRemediationAgentSchedulesClient) CreateScheduledAgentConfig(_ context.Context, opt *sonar.RemediationAgentCreateScheduledAgentConfigOptions) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error) {
	if m.CreateScheduledAgentConfigFn != nil {
		return m.CreateScheduledAgentConfigFn(opt)
	}

	return &sonar.RemediationAgentScheduledAgentConfigResource{}, &http.Response{StatusCode: http.StatusCreated}, nil
}

// UpdateScheduledAgentConfig implements
// RemediationAgentSchedulesClient.UpdateScheduledAgentConfig.
func (m *MockRemediationAgentSchedulesClient) UpdateScheduledAgentConfig(_ context.Context, id string, opt *sonar.RemediationAgentUpdateScheduledAgentConfigOptions) (*sonar.RemediationAgentScheduledAgentConfigResource, *http.Response, error) {
	if m.UpdateScheduledAgentConfigFn != nil {
		return m.UpdateScheduledAgentConfigFn(id, opt)
	}

	return &sonar.RemediationAgentScheduledAgentConfigResource{Id: id}, &http.Response{StatusCode: http.StatusOK}, nil
}

// DeleteScheduledAgentConfig implements
// RemediationAgentSchedulesClient.DeleteScheduledAgentConfig.
func (m *MockRemediationAgentSchedulesClient) DeleteScheduledAgentConfig(_ context.Context, id string) (*http.Response, error) {
	if m.DeleteScheduledAgentConfigFn != nil {
		return m.DeleteScheduledAgentConfigFn(id)
	}

	return &http.Response{StatusCode: http.StatusNoContent}, nil
}
