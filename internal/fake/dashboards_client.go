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

// MockDashboardsClient is a mock implementation of the DashboardsClient
// interface.
type MockDashboardsClient struct {
	CreateFn func(opt *sonar.DashboardsCreateOptions) (*sonar.DashboardsDashboardResponse, *http.Response, error)
	DeleteFn func(dashboardID string) (*http.Response, error)
	GetFn    func(dashboardID string) (*sonar.DashboardsDashboardResponse, *http.Response, error)
	ListFn   func(opt *sonar.DashboardsListOptions) (*sonar.DashboardsDashboardsResponse, *http.Response, error)
	UpdateFn func(dashboardID string, opt *sonar.DashboardsUpdateOptions) (*sonar.DashboardsDashboardResponse, *http.Response, error)
}

// Ensure MockDashboardsClient implements DashboardsClient.
var _ instance.DashboardsClient = &MockDashboardsClient{}

// Create implements DashboardsClient.Create.
func (m *MockDashboardsClient) Create(_ context.Context, opt *sonar.DashboardsCreateOptions) (*sonar.DashboardsDashboardResponse, *http.Response, error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Delete implements DashboardsClient.Delete.
func (m *MockDashboardsClient) Delete(_ context.Context, dashboardID string) (*http.Response, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(dashboardID)
	}

	return nil, errNotImplemented
}

// Get implements DashboardsClient.Get.
func (m *MockDashboardsClient) Get(_ context.Context, dashboardID string) (*sonar.DashboardsDashboardResponse, *http.Response, error) {
	if m.GetFn != nil {
		return m.GetFn(dashboardID)
	}

	return nil, nil, errNotImplemented
}

// List implements DashboardsClient.List.
func (m *MockDashboardsClient) List(_ context.Context, opt *sonar.DashboardsListOptions) (*sonar.DashboardsDashboardsResponse, *http.Response, error) {
	if m.ListFn != nil {
		return m.ListFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Update implements DashboardsClient.Update.
func (m *MockDashboardsClient) Update(_ context.Context, dashboardID string, opt *sonar.DashboardsUpdateOptions) (*sonar.DashboardsDashboardResponse, *http.Response, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(dashboardID, opt)
	}

	return nil, nil, errNotImplemented
}
