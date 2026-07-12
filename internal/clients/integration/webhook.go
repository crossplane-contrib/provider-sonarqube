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

package integration

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// WebhooksClient is the interface for interacting with
// the SonarQube Webhook API.
type WebhooksClient interface {
	List(ctx context.Context, opt *sonar.WebhooksListOptions) (*sonar.WebhooksList, *http.Response, error)
	Create(ctx context.Context, opt *sonar.WebhooksCreateOptions) (*sonar.WebhooksCreate, *http.Response, error)
	Update(ctx context.Context, opt *sonar.WebhooksUpdateOptions) (*http.Response, error)
	Delete(ctx context.Context, opt *sonar.WebhooksDeleteOptions) (*http.Response, error)
}

// NewWebhooksClient creates a new WebhooksClient using the provided
// SonarQube client configuration.
func NewWebhooksClient(clientConfig common.Config) WebhooksClient {
	return common.NewClient(clientConfig).Webhooks
}

// LateInitializeWebhook is a no-op: all Webhook fields are either
// user-specified or immutable, so there is nothing to back-fill from
// the observed state.
func LateInitializeWebhook(_ *v1alpha1.WebhookParameters, _ *v1alpha1.WebhookObservation) {}

// IsWebhookLateInitialized reports whether LateInitializeWebhook
// changed the spec.
// Because LateInitializeWebhook is a no-op, this always returns false.
func IsWebhookLateInitialized(_, _ *v1alpha1.WebhookParameters) bool {
	return false
}

// IsWebhookUpToDate reports whether the desired spec matches the observed
// state. currentSecret is the value read from the user's SecretRef, and
// savedSecret is the value last written to the connection secret. When a
// secretRef is configured, both values are compared to detect rotation.
func IsWebhookUpToDate(spec *v1alpha1.WebhookParameters, observation *v1alpha1.WebhookObservation, currentSecret, savedSecret string) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	if spec.Name != observation.Name {
		return false
	}

	if spec.URL != observation.URL {
		return false
	}

	if spec.SecretRef == nil {
		// Secret was removed from spec; update if SonarQube still has one.
		return !observation.HasSecret
	}

	// Secret is configured: SonarQube must also have it, and the value must
	// match what was last applied (savedSecret tracks the last-written value).
	if !observation.HasSecret {
		return false
	}

	return currentSecret == savedSecret
}

// FindWebhookByKey returns the first webhook with the given key,
// or nil if no match is found.
func FindWebhookByKey(webhooks []sonar.WebhooksDefinition, key string) *sonar.WebhooksDefinition {
	for index := range webhooks {
		if webhooks[index].Key == key {
			return &webhooks[index]
		}
	}

	return nil
}

// GenerateWebhookObservation converts a SonarQube Webhook response
// into the Crossplane observation type.
func GenerateWebhookObservation(sonarWebhook *sonar.WebhooksDefinition) v1alpha1.WebhookObservation {
	if sonarWebhook == nil {
		return v1alpha1.WebhookObservation{}
	}

	return v1alpha1.WebhookObservation{
		Key:       sonarWebhook.Key,
		Name:      sonarWebhook.Name,
		URL:       sonarWebhook.URL,
		HasSecret: sonarWebhook.HasSecret,
	}
}

// GenerateWebhookCreateOptions builds the options struct for the
// Create API call.
func GenerateWebhookCreateOptions(spec *v1alpha1.WebhookParameters, projectKey, secret string) *sonar.WebhooksCreateOptions {
	return &sonar.WebhooksCreateOptions{
		Name:    spec.Name,
		URL:     spec.URL,
		Project: projectKey,
		Secret:  secret,
	}
}

// GenerateWebhookUpdateOptions builds the options struct for the
// Update API call.
func GenerateWebhookUpdateOptions(webhookKey string, spec *v1alpha1.WebhookParameters, secret string) *sonar.WebhooksUpdateOptions {
	return &sonar.WebhooksUpdateOptions{
		Webhook: webhookKey,
		Name:    spec.Name,
		URL:     spec.URL,
		Secret:  secret,
	}
}
