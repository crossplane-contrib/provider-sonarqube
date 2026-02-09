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

var errSettingsNotImplemented = errors.New("settings operation not implemented")

// MockSettingsClient is a mock implementation of the SettingsClient interface.
type MockSettingsClient struct {
	SetFn    func(opt *sonar.SettingsSetOption) (*http.Response, error)
	ValuesFn func(opt *sonar.SettingsValuesOption) (*sonar.SettingsValues, *http.Response, error)
	ResetFn  func(opt *sonar.SettingsResetOption) (*http.Response, error)
}

// Ensure MockSettingsClient implements SettingsClient.
var _ instance.SettingsClient = &MockSettingsClient{}

// Set implements SettingsClient.Set.
func (m *MockSettingsClient) Set(opt *sonar.SettingsSetOption) (*http.Response, error) {
	if m.SetFn != nil {
		return m.SetFn(opt)
	}

	return nil, errSettingsNotImplemented
}

// Values implements SettingsClient.Values.
func (m *MockSettingsClient) Values(opt *sonar.SettingsValuesOption) (*sonar.SettingsValues, *http.Response, error) {
	if m.ValuesFn != nil {
		return m.ValuesFn(opt)
	}

	return nil, nil, errSettingsNotImplemented
}

// Reset implements SettingsClient.Reset.
func (m *MockSettingsClient) Reset(opt *sonar.SettingsResetOption) (*http.Response, error) {
	if m.ResetFn != nil {
		return m.ResetFn(opt)
	}

	return nil, errSettingsNotImplemented
}
