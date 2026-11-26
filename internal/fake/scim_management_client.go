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

// MockSCIMManagementClient is a mock implementation of the
// SCIMManagementClient interface.
type MockSCIMManagementClient struct {
	StatusFn  func() (*sonar.ScimManagementStatus, *http.Response, error)
	EnableFn  func() (*http.Response, error)
	DisableFn func() (*http.Response, error)
}

// Ensure MockSCIMManagementClient implements SCIMManagementClient.
var _ instance.SCIMManagementClient = &MockSCIMManagementClient{}

// Status implements SCIMManagementClient.Status.
func (m *MockSCIMManagementClient) Status(_ context.Context) (*sonar.ScimManagementStatus, *http.Response, error) {
	if m.StatusFn != nil {
		return m.StatusFn()
	}

	return nil, nil, errNotImplemented
}

// Enable implements SCIMManagementClient.Enable.
func (m *MockSCIMManagementClient) Enable(_ context.Context) (*http.Response, error) {
	if m.EnableFn != nil {
		return m.EnableFn()
	}

	return nil, errNotImplemented
}

// Disable implements SCIMManagementClient.Disable.
func (m *MockSCIMManagementClient) Disable(_ context.Context) (*http.Response, error) {
	if m.DisableFn != nil {
		return m.DisableFn()
	}

	return nil, errNotImplemented
}
