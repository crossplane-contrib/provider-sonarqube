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

// MockProjectTagsClient is a mock implementation of
// the ProjectTagsClient interface.
type MockProjectTagsClient struct {
	SearchFn func(opt *sonar.ProjectTagsSearchOptions) (*sonar.ProjectTagsSearch, *http.Response, error)
	SetFn    func(opt *sonar.ProjectTagsSetOptions) (*http.Response, error)
}

// Ensure MockProjectTagsClient implements ProjectTagsClient.
var _ instance.ProjectTagsClient = &MockProjectTagsClient{}

// Search implements ProjectTagsClient.Search.
func (m *MockProjectTagsClient) Search(_ context.Context, opt *sonar.ProjectTagsSearchOptions) (*sonar.ProjectTagsSearch, *http.Response, error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Set implements ProjectTagsClient.Set.
func (m *MockProjectTagsClient) Set(_ context.Context, opt *sonar.ProjectTagsSetOptions) (*http.Response, error) {
	if m.SetFn != nil {
		return m.SetFn(opt)
	}

	return nil, errNotImplemented
}
