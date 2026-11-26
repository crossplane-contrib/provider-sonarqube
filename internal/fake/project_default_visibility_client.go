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

// MockDefaultProjectVisibilityClient is a mock implementation of the
// DefaultProjectVisibilityClient interface.
type MockDefaultProjectVisibilityClient struct {
	UpdateDefaultVisibilityFn func(opt *sonar.ProjectsUpdateDefaultVisibilityOptions) (*http.Response, error)
}

// Ensure MockDefaultProjectVisibilityClient implements
// DefaultProjectVisibilityClient.
var _ instance.DefaultProjectVisibilityClient = &MockDefaultProjectVisibilityClient{}

// UpdateDefaultVisibility implements
// DefaultProjectVisibilityClient.UpdateDefaultVisibility.
func (m *MockDefaultProjectVisibilityClient) UpdateDefaultVisibility(_ context.Context, opt *sonar.ProjectsUpdateDefaultVisibilityOptions) (*http.Response, error) {
	if m.UpdateDefaultVisibilityFn != nil {
		return m.UpdateDefaultVisibilityFn(opt)
	}

	return nil, errNotImplemented
}
