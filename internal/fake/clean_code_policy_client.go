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

//nolint:dupl // Same shape as MockSCAEnablementClient, but bound to distinct SonarQube API types.
package fake

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

// MockCleanCodePolicyModeClient is a mock implementation of the
// CleanCodePolicyModeClient interface.
type MockCleanCodePolicyModeClient struct {
	GetModeFn    func() (*sonar.CleanCodePolicyMode, *http.Response, error)
	UpdateModeFn func(opt *sonar.CleanCodePolicyUpdateModeOptions) (*sonar.CleanCodePolicyMode, *http.Response, error)
}

// Ensure MockCleanCodePolicyModeClient implements CleanCodePolicyModeClient.
var _ instance.CleanCodePolicyModeClient = &MockCleanCodePolicyModeClient{}

// GetMode implements CleanCodePolicyModeClient.GetMode.
func (m *MockCleanCodePolicyModeClient) GetMode(_ context.Context) (*sonar.CleanCodePolicyMode, *http.Response, error) {
	if m.GetModeFn != nil {
		return m.GetModeFn()
	}

	return nil, nil, errNotImplemented
}

// UpdateMode implements CleanCodePolicyModeClient.UpdateMode.
func (m *MockCleanCodePolicyModeClient) UpdateMode(_ context.Context, opt *sonar.CleanCodePolicyUpdateModeOptions) (*sonar.CleanCodePolicyMode, *http.Response, error) {
	if m.UpdateModeFn != nil {
		return m.UpdateModeFn(opt)
	}

	return nil, nil, errNotImplemented
}
