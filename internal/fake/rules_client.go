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
	"errors"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

var errRulesNotImplemented = errors.New("not implemented")

// MockRulesClient is a mock implementation of the RulesClient interface.
type MockRulesClient struct {
	AppFn          func() (v *sonar.RulesApp, resp *http.Response, err error)
	CreateFn       func(opt *sonar.RulesCreateOption) (v *sonar.RulesCreate, resp *http.Response, err error)
	DeleteFn       func(opt *sonar.RulesDeleteOption) (resp *http.Response, err error)
	ListFn         func(opt *sonar.RulesListOption) (v *string, resp *http.Response, err error)
	RepositoriesFn func(opt *sonar.RulesRepositoriesOption) (v *sonar.RulesRepositories, resp *http.Response, err error)
	SearchFn       func(opt *sonar.RulesSearchOption) (v *sonar.RulesSearch, resp *http.Response, err error)
	ShowFn         func(opt *sonar.RulesShowOption) (v *sonar.RulesShow, resp *http.Response, err error)
	TagsFn         func(opt *sonar.RulesTagsOption) (v *sonar.RulesTags, resp *http.Response, err error)
	UpdateFn       func(opt *sonar.RulesUpdateOption) (v *sonar.RulesUpdate, resp *http.Response, err error)
}

// Ensure MockRulesClient implements RulesClient.
var _ instance.RulesClient = &MockRulesClient{}

// App implements RulesClient.App.
func (m *MockRulesClient) App() (v *sonar.RulesApp, resp *http.Response, err error) {
	if m.AppFn != nil {
		return m.AppFn()
	}

	return nil, nil, errRulesNotImplemented
}

// Create implements RulesClient.Create.
func (m *MockRulesClient) Create(opt *sonar.RulesCreateOption) (v *sonar.RulesCreate, resp *http.Response, err error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}

	return nil, nil, errRulesNotImplemented
}

// Delete implements RulesClient.Delete.
func (m *MockRulesClient) Delete(opt *sonar.RulesDeleteOption) (resp *http.Response, err error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}

	return nil, errRulesNotImplemented
}

// List implements RulesClient.List.
func (m *MockRulesClient) List(opt *sonar.RulesListOption) (v *string, resp *http.Response, err error) {
	if m.ListFn != nil {
		return m.ListFn(opt)
	}

	return nil, nil, errRulesNotImplemented
}

// Repositories implements RulesClient.Repositories.
func (m *MockRulesClient) Repositories(opt *sonar.RulesRepositoriesOption) (v *sonar.RulesRepositories, resp *http.Response, err error) {
	if m.RepositoriesFn != nil {
		return m.RepositoriesFn(opt)
	}

	return nil, nil, errRulesNotImplemented
}

// Search implements RulesClient.Search.
func (m *MockRulesClient) Search(opt *sonar.RulesSearchOption) (v *sonar.RulesSearch, resp *http.Response, err error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}

	return nil, nil, errRulesNotImplemented
}

// Show implements RulesClient.Show.
func (m *MockRulesClient) Show(opt *sonar.RulesShowOption) (v *sonar.RulesShow, resp *http.Response, err error) {
	if m.ShowFn != nil {
		return m.ShowFn(opt)
	}

	return nil, nil, errRulesNotImplemented
}

// Tags implements RulesClient.Tags.
func (m *MockRulesClient) Tags(opt *sonar.RulesTagsOption) (v *sonar.RulesTags, resp *http.Response, err error) {
	if m.TagsFn != nil {
		return m.TagsFn(opt)
	}

	return nil, nil, errRulesNotImplemented
}

// Update implements RulesClient.Update.
func (m *MockRulesClient) Update(opt *sonar.RulesUpdateOption) (v *sonar.RulesUpdate, resp *http.Response, err error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(opt)
	}

	return nil, nil, errRulesNotImplemented
}
