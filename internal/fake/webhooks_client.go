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

	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
)

// MockWebhooksClient is a mock implementation of the WebhooksClient interface.
type MockWebhooksClient struct {
	ListFn   func(opt *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error)
	CreateFn func(opt *sonar.WebhooksCreateOptions) (*sonar.WebhooksCreate, *http.Response, error)
	UpdateFn func(opt *sonar.WebhooksUpdateOptions) (*http.Response, error)
	DeleteFn func(opt *sonar.WebhooksDeleteOptions) (*http.Response, error)
}

// Ensure MockWebhooksClient implements WebhooksClient.
var _ integration.WebhooksClient = &MockWebhooksClient{}

// List implements WebhooksClient.List.
func (m *MockWebhooksClient) List(opt *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error) {
	if m.ListFn != nil {
		return m.ListFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Create implements WebhooksClient.Create.
func (m *MockWebhooksClient) Create(opt *sonar.WebhooksCreateOptions) (*sonar.WebhooksCreate, *http.Response, error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Update implements WebhooksClient.Update.
func (m *MockWebhooksClient) Update(opt *sonar.WebhooksUpdateOptions) (*http.Response, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(opt)
	}

	return nil, errNotImplemented
}

// Delete implements WebhooksClient.Delete.
func (m *MockWebhooksClient) Delete(opt *sonar.WebhooksDeleteOptions) (*http.Response, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errNotImplemented
}
