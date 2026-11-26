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

package instance

import (
	"context"
	"net/http"
	"strconv"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// EmailConfigurationsClient is the interface for interacting with the
// SonarQube email configuration API (api/v2/system/email-configurations).
type EmailConfigurationsClient interface {
	SearchEmailConfigurations(ctx context.Context) (*sonar.SystemEmailConfigurationsSearch, *http.Response, error)
	CreateEmailConfiguration(ctx context.Context, opt *sonar.SystemEmailConfigurationCreateOptions) (*sonar.SystemEmailConfiguration, *http.Response, error)
	GetEmailConfiguration(ctx context.Context, id string) (*sonar.SystemEmailConfiguration, *http.Response, error)
	UpdateEmailConfiguration(ctx context.Context, id string, opt *sonar.SystemEmailConfigurationUpdateOptions) (*sonar.SystemEmailConfiguration, *http.Response, error)
	DeleteEmailConfiguration(ctx context.Context, id string) (*http.Response, error)
}

// EmailConfigurationCredentials are the write-only values of an email
// configuration, which SonarQube never returns. Only the values of the
// selected authentication method are set.
type EmailConfigurationCredentials struct {
	// BasicPassword is the SMTP password, for BASIC authentication.
	BasicPassword string
	// OAuthClientID is the OAuth client ID, for OAUTH authentication.
	OAuthClientID string
	// OAuthClientSecret is the OAuth client secret, for OAUTH
	// authentication.
	OAuthClientSecret string
}

// NewEmailConfigurationsClient creates a new EmailConfigurationsClient with
// the provided SonarQube client configuration. SonarQube holds at most one
// email configuration, read by a single managed resource, so its responses
// are not cached.
func NewEmailConfigurationsClient(clientConfig common.Config) EmailConfigurationsClient {
	return common.NewClient(clientConfig).V2.System
}

// LateInitializeEmailConfiguration fills the empty fields in the
// EmailConfiguration spec with the values observed in SonarQube.
func LateInitializeEmailConfiguration(spec *v1alpha1.EmailConfigurationParameters, observation *v1alpha1.EmailConfigurationObservation) {
	if spec == nil || observation == nil {
		return
	}

	if spec.AuthMethod == v1alpha1.EmailAuthMethodOAuth && observation.OAuthScope != "" {
		helpers.AssignIfNil(&spec.OAuthScope, observation.OAuthScope)
	}
}

// IsEmailConfigurationLateInitialized reports whether late initialization
// changed the EmailConfiguration spec.
func IsEmailConfigurationLateInitialized(former, current *v1alpha1.EmailConfigurationParameters) bool {
	if former == nil || current == nil {
		return false
	}

	return !helpers.IsComparablePtrEqualComparablePtr(former.OAuthScope, current.OAuthScope)
}

// IsEmailConfigurationUpToDate reports whether the EmailConfiguration spec
// matches the configuration observed in SonarQube.
//
// SonarQube never returns the SMTP password nor the OAuth client ID and
// secret, so the values read from the spec's secret references (current) are
// compared with those last applied by the provider (saved). Fields of the
// authentication method that is not selected are ignored.
func IsEmailConfigurationUpToDate(spec *v1alpha1.EmailConfigurationParameters, observation *v1alpha1.EmailConfigurationObservation, current, saved EmailConfigurationCredentials) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	if !areEmailConfigurationFieldsUpToDate(spec, observation) || current != saved {
		return false
	}

	if spec.AuthMethod == v1alpha1.EmailAuthMethodOAuth {
		return areEmailConfigurationOAuthFieldsUpToDate(spec, observation)
	}

	return observation.IsBasicPasswordSet
}

// areEmailConfigurationFieldsUpToDate reports whether the readable fields
// shared by both authentication methods match the observation.
func areEmailConfigurationFieldsUpToDate(spec *v1alpha1.EmailConfigurationParameters, observation *v1alpha1.EmailConfigurationObservation) bool {
	return spec.Host == observation.Host &&
		formatEmailPort(spec.Port) == observation.Port &&
		spec.SecurityProtocol == observation.SecurityProtocol &&
		spec.FromAddress == observation.FromAddress &&
		spec.FromName == observation.FromName &&
		spec.SubjectPrefix == observation.SubjectPrefix &&
		spec.AuthMethod == observation.AuthMethod &&
		spec.Username == observation.Username
}

// areEmailConfigurationOAuthFieldsUpToDate reports whether the OAuth fields
// match the observation. Unset optional fields match any value. SonarQube
// versions before 2026.5 neither store nor return the OAuth scope, so an
// empty observed scope matches any value rather than requesting an update
// that could never converge.
func areEmailConfigurationOAuthFieldsUpToDate(spec *v1alpha1.EmailConfigurationParameters, observation *v1alpha1.EmailConfigurationObservation) bool {
	return observation.IsOAuthClientIDSet &&
		observation.IsOAuthClientSecretSet &&
		helpers.IsComparablePtrEqualComparable(spec.OAuthAuthenticationHost, observation.OAuthAuthenticationHost) &&
		helpers.IsComparablePtrEqualComparable(spec.OAuthTenant, observation.OAuthTenant) &&
		(observation.OAuthScope == "" || helpers.IsComparablePtrEqualComparable(spec.OAuthScope, observation.OAuthScope))
}

// formatEmailPort formats the port as the SonarQube API represents it.
func formatEmailPort(port int32) string {
	return strconv.FormatInt(int64(port), 10)
}

// GenerateEmailConfigurationCreateOptions generates the SonarQube API options
// for creating an email configuration from the spec and the values of its
// secret references.
func GenerateEmailConfigurationCreateOptions(spec *v1alpha1.EmailConfigurationParameters, credentials EmailConfigurationCredentials) *sonar.SystemEmailConfigurationCreateOptions {
	if spec == nil {
		return nil
	}

	options := &sonar.SystemEmailConfigurationCreateOptions{
		Host:             spec.Host,
		Port:             formatEmailPort(spec.Port),
		SecurityProtocol: spec.SecurityProtocol,
		FromAddress:      spec.FromAddress,
		FromName:         spec.FromName,
		SubjectPrefix:    spec.SubjectPrefix,
		AuthMethod:       spec.AuthMethod,
		Username:         spec.Username,
		BasicPassword:    credentials.BasicPassword,
	}

	if spec.AuthMethod == v1alpha1.EmailAuthMethodOAuth {
		options.OauthAuthenticationHost = ptr.Deref(spec.OAuthAuthenticationHost, "")
		options.OauthTenant = ptr.Deref(spec.OAuthTenant, "")
		options.OauthScope = ptr.Deref(spec.OAuthScope, "")
		options.OauthClientId = credentials.OAuthClientID
		options.OauthClientSecret = credentials.OAuthClientSecret
	}

	return options
}

// GenerateEmailConfigurationUpdateOptions generates the SonarQube API options
// for updating an email configuration from the spec and the values of its
// secret references. Empty fields are left unchanged by SonarQube.
func GenerateEmailConfigurationUpdateOptions(spec *v1alpha1.EmailConfigurationParameters, credentials EmailConfigurationCredentials) *sonar.SystemEmailConfigurationUpdateOptions {
	create := GenerateEmailConfigurationCreateOptions(spec, credentials)
	if create == nil {
		return nil
	}

	options := sonar.SystemEmailConfigurationUpdateOptions(*create)

	return &options
}

// GenerateEmailConfigurationObservation generates the
// EmailConfigurationObservation from the configuration returned by SonarQube.
func GenerateEmailConfigurationObservation(configuration *sonar.SystemEmailConfiguration) v1alpha1.EmailConfigurationObservation {
	if configuration == nil {
		return v1alpha1.EmailConfigurationObservation{}
	}

	return v1alpha1.EmailConfigurationObservation{
		ID:                      configuration.Id,
		Host:                    configuration.Host,
		Port:                    configuration.Port,
		SecurityProtocol:        configuration.SecurityProtocol,
		FromAddress:             configuration.FromAddress,
		FromName:                configuration.FromName,
		SubjectPrefix:           configuration.SubjectPrefix,
		AuthMethod:              configuration.AuthMethod,
		Username:                ptr.Deref(configuration.Username, ""),
		IsBasicPasswordSet:      configuration.IsBasicPasswordSet,
		OAuthAuthenticationHost: ptr.Deref(configuration.OauthAuthenticationHost, ""),
		OAuthTenant:             ptr.Deref(configuration.OauthTenant, ""),
		OAuthScope:              ptr.Deref(configuration.OauthScope, ""),
		IsOAuthClientIDSet:      configuration.IsOauthClientIdSet,
		IsOAuthClientSecretSet:  configuration.IsOauthClientSecretSet,
	}
}
