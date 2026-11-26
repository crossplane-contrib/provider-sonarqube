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

//nolint:dupl // Same shape as MockCleanCodePolicyModeClient, but bound to distinct SonarQube API types.
package fake

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

// MockSCAEnablementClient is a mock implementation of the
// SCAEnablementClient interface.
type MockSCAEnablementClient struct {
	GetEnablementFn func() (*sonar.ScaFeatureEnablement, *http.Response, error)
	SetEnablementFn func(opt *sonar.ScaSetEnablementOptions) (*sonar.ScaFeatureEnablement, *http.Response, error)
}

// Ensure MockSCAEnablementClient implements SCAEnablementClient.
var _ instance.SCAEnablementClient = &MockSCAEnablementClient{}

// GetEnablement implements SCAEnablementClient.GetEnablement.
func (m *MockSCAEnablementClient) GetEnablement(_ context.Context) (*sonar.ScaFeatureEnablement, *http.Response, error) {
	if m.GetEnablementFn != nil {
		return m.GetEnablementFn()
	}

	return nil, nil, errNotImplemented
}

// SetEnablement implements SCAEnablementClient.SetEnablement.
func (m *MockSCAEnablementClient) SetEnablement(_ context.Context, opt *sonar.ScaSetEnablementOptions) (*sonar.ScaFeatureEnablement, *http.Response, error) {
	if m.SetEnablementFn != nil {
		return m.SetEnablementFn(opt)
	}

	return nil, nil, errNotImplemented
}
