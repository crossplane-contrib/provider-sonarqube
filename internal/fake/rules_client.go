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

// MockRulesClient is a mock implementation of the RulesClient interface.
type MockRulesClient struct {
	AppFn          func() (v *sonar.RulesApp, resp *http.Response, err error)
	CreateFn       func(opt *sonar.RulesCreateOptions) (v *sonar.RulesCreate, resp *http.Response, err error)
	DeleteFn       func(opt *sonar.RulesDeleteOptions) (resp *http.Response, err error)
	ListFn         func(opt *sonar.RulesListOptions) (v *string, resp *http.Response, err error)
	RepositoriesFn func(opt *sonar.RulesRepositoriesOptions) (v *sonar.RulesRepositories, resp *http.Response, err error)
	SearchFn       func(opt *sonar.RulesSearchOptions) (v *sonar.RulesSearch, resp *http.Response, err error)
	ShowFn         func(opt *sonar.RulesShowOptions) (v *sonar.RulesShow, resp *http.Response, err error)
	TagsFn         func(opt *sonar.RulesTagsOptions) (v *sonar.RulesTags, resp *http.Response, err error)
	UpdateFn       func(opt *sonar.RulesUpdateOptions) (v *sonar.RulesUpdate, resp *http.Response, err error)
}

// Ensure MockRulesClient implements RulesClient.
var _ instance.RulesClient = &MockRulesClient{}

// App implements RulesClient.App.
func (m *MockRulesClient) App(_ context.Context) (v *sonar.RulesApp, resp *http.Response, err error) {
	if m.AppFn != nil {
		return m.AppFn()
	}

	return nil, nil, errNotImplemented
}

// Create implements RulesClient.Create.
func (m *MockRulesClient) Create(_ context.Context, opt *sonar.RulesCreateOptions) (v *sonar.RulesCreate, resp *http.Response, err error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Delete implements RulesClient.Delete.
func (m *MockRulesClient) Delete(_ context.Context, opt *sonar.RulesDeleteOptions) (resp *http.Response, err error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errNotImplemented
}

// List implements RulesClient.List.
func (m *MockRulesClient) List(_ context.Context, opt *sonar.RulesListOptions) (v *string, resp *http.Response, err error) {
	if m.ListFn != nil {
		return m.ListFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Repositories implements RulesClient.Repositories.
func (m *MockRulesClient) Repositories(_ context.Context, opt *sonar.RulesRepositoriesOptions) (v *sonar.RulesRepositories, resp *http.Response, err error) {
	if m.RepositoriesFn != nil {
		return m.RepositoriesFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Search implements RulesClient.Search.
func (m *MockRulesClient) Search(_ context.Context, opt *sonar.RulesSearchOptions) (v *sonar.RulesSearch, resp *http.Response, err error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Show implements RulesClient.Show.
func (m *MockRulesClient) Show(_ context.Context, opt *sonar.RulesShowOptions) (v *sonar.RulesShow, resp *http.Response, err error) {
	if m.ShowFn != nil {
		return m.ShowFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Tags implements RulesClient.Tags.
func (m *MockRulesClient) Tags(_ context.Context, opt *sonar.RulesTagsOptions) (v *sonar.RulesTags, resp *http.Response, err error) {
	if m.TagsFn != nil {
		return m.TagsFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Update implements RulesClient.Update.
func (m *MockRulesClient) Update(_ context.Context, opt *sonar.RulesUpdateOptions) (v *sonar.RulesUpdate, resp *http.Response, err error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(opt)
	}

	return nil, nil, errNotImplemented
}
