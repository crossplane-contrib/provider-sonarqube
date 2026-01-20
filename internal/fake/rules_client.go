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

	sonargo "github.com/boxboxjason/sonarqube-client-go/sonar"

	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

// MockRulesClient is a mock implementation of the RulesClient interface.
type MockRulesClient struct {
	AppFn          func() (v *sonargo.RulesAppObject, resp *http.Response, err error)
	CreateFn       func(opt *sonargo.RulesCreateOption) (v *sonargo.RulesCreateObject, resp *http.Response, err error)
	DeleteFn       func(opt *sonargo.RulesDeleteOption) (resp *http.Response, err error)
	ListFn         func(opt *sonargo.RulesListOption) (v *string, resp *http.Response, err error)
	RepositoriesFn func(opt *sonargo.RulesRepositoriesOption) (v *sonargo.RulesRepositoriesObject, resp *http.Response, err error)
	SearchFn       func(opt *sonargo.RulesSearchOption) (v *sonargo.RulesSearchObject, resp *http.Response, err error)
	ShowFn         func(opt *sonargo.RulesShowOption) (v *sonargo.RulesShowObject, resp *http.Response, err error)
	TagsFn         func(opt *sonargo.RulesTagsOption) (v *sonargo.RulesTagsObject, resp *http.Response, err error)
	UpdateFn       func(opt *sonargo.RulesUpdateOption) (v *sonargo.RulesUpdateObject, resp *http.Response, err error)
}

// Ensure MockRulesClient implements RulesClient
var _ instance.RulesClient = &MockRulesClient{}

// App implements RulesClient.App
func (m *MockRulesClient) App() (v *sonargo.RulesAppObject, resp *http.Response, err error) {
	if m.AppFn != nil {
		return m.AppFn()
	}
	return nil, nil, nil
}

// Create implements RulesClient.Create
func (m *MockRulesClient) Create(opt *sonargo.RulesCreateOption) (v *sonargo.RulesCreateObject, resp *http.Response, err error) {
	if m.CreateFn != nil {
		return m.CreateFn(opt)
	}
	return nil, nil, nil
}

// Delete implements RulesClient.Delete
func (m *MockRulesClient) Delete(opt *sonargo.RulesDeleteOption) (resp *http.Response, err error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(opt)
	}
	return nil, nil
}

// List implements RulesClient.List
func (m *MockRulesClient) List(opt *sonargo.RulesListOption) (v *string, resp *http.Response, err error) {
	if m.ListFn != nil {
		return m.ListFn(opt)
	}
	return nil, nil, nil
}

// Repositories implements RulesClient.Repositories
func (m *MockRulesClient) Repositories(opt *sonargo.RulesRepositoriesOption) (v *sonargo.RulesRepositoriesObject, resp *http.Response, err error) {
	if m.RepositoriesFn != nil {
		return m.RepositoriesFn(opt)
	}
	return nil, nil, nil
}

// Search implements RulesClient.Search
func (m *MockRulesClient) Search(opt *sonargo.RulesSearchOption) (v *sonargo.RulesSearchObject, resp *http.Response, err error) {
	if m.SearchFn != nil {
		return m.SearchFn(opt)
	}
	return nil, nil, nil
}

// Show implements RulesClient.Show
func (m *MockRulesClient) Show(opt *sonargo.RulesShowOption) (v *sonargo.RulesShowObject, resp *http.Response, err error) {
	if m.ShowFn != nil {
		return m.ShowFn(opt)
	}
	return nil, nil, nil
}

// Tags implements RulesClient.Tags
func (m *MockRulesClient) Tags(opt *sonargo.RulesTagsOption) (v *sonargo.RulesTagsObject, resp *http.Response, err error) {
	if m.TagsFn != nil {
		return m.TagsFn(opt)
	}
	return nil, nil, nil
}

// Update implements RulesClient.Update
func (m *MockRulesClient) Update(opt *sonargo.RulesUpdateOption) (v *sonargo.RulesUpdateObject, resp *http.Response, err error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(opt)
	}
	return nil, nil, nil
}
