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

	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

// MockNewCodePeriodsClient is a mock implementation of the NewCodePeriodsClient interface.
type MockNewCodePeriodsClient struct {
	ListFn  func(opt *sonar.NewCodePeriodsListOptions) (*sonar.NewCodePeriodsList, *http.Response, error)
	SetFn   func(opt *sonar.NewCodePeriodsSetOptions) (*http.Response, error)
	ShowFn  func(opt *sonar.NewCodePeriodsShowOptions) (*sonar.NewCodePeriodsShow, *http.Response, error)
	UnsetFn func(opt *sonar.NewCodePeriodsUnsetOptions) (*http.Response, error)
}

// Ensure MockNewCodePeriodsClient implements NewCodePeriodsClient.
var _ instance.NewCodePeriodsClient = &MockNewCodePeriodsClient{}

// List implements NewCodePeriodsClient.List.
func (m *MockNewCodePeriodsClient) List(opt *sonar.NewCodePeriodsListOptions) (*sonar.NewCodePeriodsList, *http.Response, error) {
	if m.ListFn != nil {
		return m.ListFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Set implements NewCodePeriodsClient.Set.
func (m *MockNewCodePeriodsClient) Set(opt *sonar.NewCodePeriodsSetOptions) (*http.Response, error) {
	if m.SetFn != nil {
		return m.SetFn(opt)
	}

	return nil, errNotImplemented
}

// Show implements NewCodePeriodsClient.Show.
func (m *MockNewCodePeriodsClient) Show(opt *sonar.NewCodePeriodsShowOptions) (*sonar.NewCodePeriodsShow, *http.Response, error) {
	if m.ShowFn != nil {
		return m.ShowFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Unset implements NewCodePeriodsClient.Unset.
func (m *MockNewCodePeriodsClient) Unset(opt *sonar.NewCodePeriodsUnsetOptions) (*http.Response, error) {
	if m.UnsetFn != nil {
		return m.UnsetFn(opt)
	}

	return nil, errNotImplemented
}
