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

// MockGovernanceReportsClient is a mock implementation of the
// GovernanceReportsClient interface.
type MockGovernanceReportsClient struct {
	StatusFn           func(opt *sonar.GovernanceReportsStatusOptions) (*sonar.GovernanceReportsStatus, *http.Response, error)
	UpdateFrequencyFn  func(opt *sonar.GovernanceReportsUpdateFrequencyOptions) (*http.Response, error)
	UpdateRecipientsFn func(opt *sonar.GovernanceReportsUpdateRecipientsOptions) (*http.Response, error)
}

// Ensure MockGovernanceReportsClient implements GovernanceReportsClient.
var _ instance.GovernanceReportsClient = &MockGovernanceReportsClient{}

// Status implements GovernanceReportsClient.Status.
func (m *MockGovernanceReportsClient) Status(_ context.Context, opt *sonar.GovernanceReportsStatusOptions) (*sonar.GovernanceReportsStatus, *http.Response, error) {
	if m.StatusFn != nil {
		return m.StatusFn(opt)
	}

	return nil, nil, errNotImplemented
}

// UpdateFrequency implements GovernanceReportsClient.UpdateFrequency.
func (m *MockGovernanceReportsClient) UpdateFrequency(_ context.Context, opt *sonar.GovernanceReportsUpdateFrequencyOptions) (*http.Response, error) {
	if m.UpdateFrequencyFn != nil {
		return m.UpdateFrequencyFn(opt)
	}

	return nil, errNotImplemented
}

// UpdateRecipients implements GovernanceReportsClient.UpdateRecipients.
func (m *MockGovernanceReportsClient) UpdateRecipients(_ context.Context, opt *sonar.GovernanceReportsUpdateRecipientsOptions) (*http.Response, error) {
	if m.UpdateRecipientsFn != nil {
		return m.UpdateRecipientsFn(opt)
	}

	return nil, errNotImplemented
}
