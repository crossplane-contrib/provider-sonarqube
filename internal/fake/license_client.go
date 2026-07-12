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

	instance "github.com/crossplane/provider-sonarqube/internal/clients/instance"
)

// MockLicensesClient is a mock implementation of the LicensesClient interface.
type MockLicensesClient struct {
	// GetFn backs the Get method.
	GetFn func() (*sonar.LicenseGet, *http.Response, error)
	// SetFn backs the Set method.
	SetFn func(opt *sonar.LicenseSetOptions) (*http.Response, error)
	// UnsetLicenseFn backs the UnsetLicense method.
	UnsetLicenseFn func() (*http.Response, error)
}

// Ensure MockLicensesClient implements LicensesClient.
var _ instance.LicensesClient = &MockLicensesClient{}

// Get implements LicensesClient.Get.
func (m *MockLicensesClient) Get(_ context.Context) (*sonar.LicenseGet, *http.Response, error) {
	if m.GetFn != nil {
		return m.GetFn()
	}

	return &sonar.LicenseGet{}, &http.Response{StatusCode: http.StatusOK}, nil
}

// Set implements LicensesClient.Set.
func (m *MockLicensesClient) Set(_ context.Context, opt *sonar.LicenseSetOptions) (*http.Response, error) {
	if m.SetFn != nil {
		return m.SetFn(opt)
	}

	return &http.Response{StatusCode: http.StatusOK}, nil
}

// UnsetLicense implements LicensesClient.UnsetLicense.
func (m *MockLicensesClient) UnsetLicense(_ context.Context) (*http.Response, error) {
	if m.UnsetLicenseFn != nil {
		return m.UnsetLicenseFn()
	}

	return &http.Response{StatusCode: http.StatusOK}, nil
}
