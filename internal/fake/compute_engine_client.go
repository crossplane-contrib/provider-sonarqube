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

// MockComputeEngineWorkerCountClient is a mock implementation of the
// ComputeEngineWorkerCountClient interface.
type MockComputeEngineWorkerCountClient struct {
	WorkerCountFn    func() (*sonar.CeWorkerCount, *http.Response, error)
	SetWorkerCountFn func(opt *sonar.CeSetWorkerCountOptions) (*http.Response, error)
}

// Ensure MockComputeEngineWorkerCountClient implements
// ComputeEngineWorkerCountClient.
var _ instance.ComputeEngineWorkerCountClient = &MockComputeEngineWorkerCountClient{}

// WorkerCount implements ComputeEngineWorkerCountClient.WorkerCount.
func (m *MockComputeEngineWorkerCountClient) WorkerCount(_ context.Context) (*sonar.CeWorkerCount, *http.Response, error) {
	if m.WorkerCountFn != nil {
		return m.WorkerCountFn()
	}

	return nil, nil, errNotImplemented
}

// SetWorkerCount implements ComputeEngineWorkerCountClient.SetWorkerCount.
func (m *MockComputeEngineWorkerCountClient) SetWorkerCount(_ context.Context, opt *sonar.CeSetWorkerCountOptions) (*http.Response, error) {
	if m.SetWorkerCountFn != nil {
		return m.SetWorkerCountFn(opt)
	}

	return nil, errNotImplemented
}
