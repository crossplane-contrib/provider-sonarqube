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
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
)

// MockUsersClient is a mock implementation of the iam.UsersClient interface.
type MockUsersClient struct {
	CreateFn     func(opt *sonar.UsersCreateOptionsV2) (*sonar.UserV2, *http.Response, error)
	DeactivateFn func(opt *sonar.UsersDeactivateOptionsV2) (*http.Response, error)
	FetchFn      func(userID string) (*sonar.UserV2, *http.Response, error)
	SearchFn     func(opt *sonar.UsersSearchOptionV2) (*sonar.UsersSearchV2, *http.Response, error)
	UpdateFn     func(userID string, opt *sonar.UsersUpdateOptionsV2) (*sonar.UserV2, *http.Response, error)
}

// Ensure MockUsersClient implements UsersClient.
var _ iam.UsersClient = &MockUsersClient{}

// Create implements UsersClient.Create.
func (m *MockUsersClient) Create(opt *sonar.UsersCreateOptionsV2) (*sonar.UserV2, *http.Response, error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Deactivate implements UsersClient.Deactivate.
func (m *MockUsersClient) Deactivate(opt *sonar.UsersDeactivateOptionsV2) (*http.Response, error) {
	if m.DeactivateFn != nil {
		return m.DeactivateFn(opt)
	}

	return nil, errNotImplemented
}

// Fetch implements UsersClient.Fetch.
func (m *MockUsersClient) Fetch(userID string) (*sonar.UserV2, *http.Response, error) {
	if m.FetchFn != nil {
		return m.FetchFn(userID)
	}

	return nil, nil, errNotImplemented
}

// Search implements UsersClient.Search.
func (m *MockUsersClient) Search(opt *sonar.UsersSearchOptionV2) (*sonar.UsersSearchV2, *http.Response, error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Update implements UsersClient.Update.
func (m *MockUsersClient) Update(userID string, opt *sonar.UsersUpdateOptionsV2) (*sonar.UserV2, *http.Response, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(userID, opt)
	}

	return nil, nil, errNotImplemented
}
