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

	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
)

// MockFixSuggestionsClient is a mock implementation of the
// FixSuggestionsClient interface.
type MockFixSuggestionsClient struct {
	GetEnablementFn func() (*sonar.FixSuggestionsFeatureEnablement, *http.Response, error)
	SetEnablementFn func(opt *integration.AICodeFixEnablementOptions) (*http.Response, error)
}

// Ensure MockFixSuggestionsClient implements FixSuggestionsClient.
var _ integration.FixSuggestionsClient = &MockFixSuggestionsClient{}

// GetEnablement implements FixSuggestionsClient.GetEnablement.
func (m *MockFixSuggestionsClient) GetEnablement(_ context.Context) (*sonar.FixSuggestionsFeatureEnablement, *http.Response, error) {
	if m.GetEnablementFn != nil {
		return m.GetEnablementFn()
	}

	return nil, nil, errNotImplemented
}

// SetEnablement implements FixSuggestionsClient.SetEnablement.
func (m *MockFixSuggestionsClient) SetEnablement(_ context.Context, opt *integration.AICodeFixEnablementOptions) (*http.Response, error) {
	if m.SetEnablementFn != nil {
		return m.SetEnablementFn(opt)
	}

	return nil, errNotImplemented
}
