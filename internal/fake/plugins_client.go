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

// MockPluginsClient is a mock implementation of the PluginsClient interface.
type MockPluginsClient struct {
	// InstallFn backs the Install method.
	InstallFn func(opt *sonar.PluginsInstallOptions) (resp *http.Response, err error)
	// InstalledFn backs the Installed method.
	InstalledFn func(opt *sonar.PluginsInstalledOptions) (v *sonar.PluginsInstalled, resp *http.Response, err error)
	// UninstallFn backs the Uninstall method.
	UninstallFn func(opt *sonar.PluginsUninstallOptions) (resp *http.Response, err error)
	// UpdateFn backs the Update method.
	UpdateFn func(opt *sonar.PluginsUpdateOptions) (resp *http.Response, err error)
	// UpdatesFn backs the Updates method.
	UpdatesFn func() (v *sonar.PluginsUpdates, resp *http.Response, err error)
}

// Ensure MockPluginsClient implements PluginsClient.
var _ instance.PluginsClient = &MockPluginsClient{}

// Install implements PluginsClient.Install.
func (m *MockPluginsClient) Install(opt *sonar.PluginsInstallOptions) (resp *http.Response, err error) {
	if m.InstallFn != nil {
		return m.InstallFn(opt)
	}

	return nil, errNotImplemented
}

// Installed implements PluginsClient.Installed.
func (m *MockPluginsClient) Installed(opt *sonar.PluginsInstalledOptions) (v *sonar.PluginsInstalled, resp *http.Response, err error) {
	if m.InstalledFn != nil {
		return m.InstalledFn(opt)
	}

	return nil, nil, errNotImplemented
}

// Uninstall implements PluginsClient.Uninstall.
func (m *MockPluginsClient) Uninstall(opt *sonar.PluginsUninstallOptions) (resp *http.Response, err error) {
	if m.UninstallFn != nil {
		return m.UninstallFn(opt)
	}

	return nil, errNotImplemented
}

// Update implements PluginsClient.Update.
func (m *MockPluginsClient) Update(opt *sonar.PluginsUpdateOptions) (resp *http.Response, err error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(opt)
	}

	return nil, errNotImplemented
}

// Updates implements PluginsClient.Updates.
func (m *MockPluginsClient) Updates() (v *sonar.PluginsUpdates, resp *http.Response, err error) {
	if m.UpdatesFn != nil {
		return m.UpdatesFn()
	}

	return nil, nil, errNotImplemented
}
