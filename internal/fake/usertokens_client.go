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

//nolint:dupl // Mock clients intentionally share this three-method structure.
package fake

import (
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
)

// MockUserTokensClient is a mock implementation of the
// iam.UserTokensClient interface.
type MockUserTokensClient struct {
	GenerateFn func(opt *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error)
	RevokeFn   func(opt *sonar.UserTokensRevokeOptions) (*http.Response, error)
	SearchFn   func(opt *sonar.UserTokensSearchOptions) (*sonar.UserTokensSearch, *http.Response, error)
}

// Ensure MockUserTokensClient implements UserTokensClient.
var _ iam.UserTokensClient = &MockUserTokensClient{}

// Generate implements UserTokensClient.Generate.
func (m *MockUserTokensClient) Generate(opt *sonar.UserTokensGenerateOptions) (*sonar.UserTokensGenerate, *http.Response, error) {
	if m.GenerateFn != nil {
		return m.GenerateFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Revoke implements UserTokensClient.Revoke.
func (m *MockUserTokensClient) Revoke(opt *sonar.UserTokensRevokeOptions) (*http.Response, error) {
	if m.RevokeFn != nil {
		return m.RevokeFn(opt)
	}

	return nil, errNotImplemented
}

// Search implements UserTokensClient.Search.
func (m *MockUserTokensClient) Search(opt *sonar.UserTokensSearchOptions) (*sonar.UserTokensSearch, *http.Response, error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}

	return nil, nil, errNotImplemented
}
