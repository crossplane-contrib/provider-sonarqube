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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

const (
	// EmailAuthMethodBasic authenticates to the SMTP server with a username
	// and password.
	EmailAuthMethodBasic = "BASIC"
	// EmailAuthMethodOAuth authenticates to the SMTP server with OAuth, which
	// SonarQube only supports for Microsoft Exchange.
	EmailAuthMethodOAuth = "OAUTH"
)

// EmailConfigurationParameters are the configurable fields of an
// EmailConfiguration.
//
// +kubebuilder:validation:XValidation:rule="self.authMethod != 'BASIC' || has(self.basicPasswordSecretRef)",message="basicPasswordSecretRef is required when authMethod is BASIC"
// +kubebuilder:validation:XValidation:rule="self.authMethod != 'OAUTH' || has(self.oauthAuthenticationHost)",message="oauthAuthenticationHost is required when authMethod is OAUTH"
// +kubebuilder:validation:XValidation:rule="self.authMethod != 'OAUTH' || has(self.oauthTenant)",message="oauthTenant is required when authMethod is OAUTH"
// +kubebuilder:validation:XValidation:rule="self.authMethod != 'OAUTH' || has(self.oauthClientIdSecretRef)",message="oauthClientIdSecretRef is required when authMethod is OAUTH"
// +kubebuilder:validation:XValidation:rule="self.authMethod != 'OAUTH' || has(self.oauthClientSecretSecretRef)",message="oauthClientSecretSecretRef is required when authMethod is OAUTH"
type EmailConfigurationParameters struct {
	// Host is the host of the SMTP server.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Host string `json:"host"`

	// Port is the port of the SMTP server, usually 25, 465 or 587.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// SecurityProtocol is the protocol used to secure the connection to the
	// SMTP server: NONE, SSLTLS or STARTTLS.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=NONE;SSLTLS;STARTTLS
	SecurityProtocol string `json:"securityProtocol"`

	// FromAddress is the address emails are sent from.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	FromAddress string `json:"fromAddress"`

	// FromName is the name emails are sent from, usually SonarQube.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	FromName string `json:"fromName"`

	// SubjectPrefix is the prefix added to email subjects, usually
	// [SonarQube].
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	SubjectPrefix string `json:"subjectPrefix"`

	// AuthMethod is how SonarQube authenticates to the SMTP server: BASIC
	// (username and password) or OAUTH (Microsoft Exchange only).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=BASIC;OAUTH
	AuthMethod string `json:"authMethod"`

	// Username is the user authenticating to the SMTP server, with both
	// authentication methods.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Username string `json:"username"`

	// BasicPasswordSecretRef references the Secret key holding the SMTP
	// password. The Secret is read from the EmailConfiguration's namespace.
	// Required when AuthMethod is BASIC, ignored otherwise.
	// +kubebuilder:validation:Optional
	BasicPasswordSecretRef *xpv1.LocalSecretKeySelector `json:"basicPasswordSecretRef,omitempty"`

	// OAuthAuthenticationHost is the host of the identity provider issuing
	// access tokens, for example https://login.microsoftonline.com.
	// Required when AuthMethod is OAUTH, ignored otherwise.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	OAuthAuthenticationHost *string `json:"oauthAuthenticationHost,omitempty"`

	// OAuthTenant is the Microsoft tenant.
	// Required when AuthMethod is OAUTH, ignored otherwise.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	OAuthTenant *string `json:"oauthTenant,omitempty"`

	// OAuthClientIDSecretRef references the Secret key holding the OAuth
	// client ID. The Secret is read from the EmailConfiguration's namespace.
	// Required when AuthMethod is OAUTH, ignored otherwise.
	// +kubebuilder:validation:Optional
	OAuthClientIDSecretRef *xpv1.LocalSecretKeySelector `json:"oauthClientIdSecretRef,omitempty"`

	// OAuthClientSecretSecretRef references the Secret key holding the OAuth
	// client secret. The Secret is read from the EmailConfiguration's
	// namespace.
	// Required when AuthMethod is OAUTH, ignored otherwise.
	// +kubebuilder:validation:Optional
	OAuthClientSecretSecretRef *xpv1.LocalSecretKeySelector `json:"oauthClientSecretSecretRef,omitempty"`

	// OAuthScope is the scope of the OAuth token request, used when
	// AuthMethod is OAUTH. SonarQube defaults it to
	// https://outlook.office365.com/.default. Requires SonarQube 2026.5 or
	// later.
	// Late-initialized from SonarQube if unset.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	OAuthScope *string `json:"oauthScope,omitempty"`
}

// EmailConfigurationObservation are the observable fields of an
// EmailConfiguration.
type EmailConfigurationObservation struct {
	// ID is the identifier of the configuration in SonarQube, used as the
	// external name.
	ID string `json:"id,omitempty"`
	// Host is the host of the SMTP server.
	Host string `json:"host,omitempty"`
	// Port is the port of the SMTP server.
	Port string `json:"port,omitempty"`
	// SecurityProtocol is the protocol securing the SMTP connection.
	SecurityProtocol string `json:"securityProtocol,omitempty"`
	// FromAddress is the address emails are sent from.
	FromAddress string `json:"fromAddress,omitempty"`
	// FromName is the name emails are sent from.
	FromName string `json:"fromName,omitempty"`
	// SubjectPrefix is the prefix added to email subjects.
	SubjectPrefix string `json:"subjectPrefix,omitempty"`
	// AuthMethod is the authentication method.
	AuthMethod string `json:"authMethod,omitempty"`
	// Username is the user authenticating to the SMTP server.
	Username string `json:"username,omitempty"`
	// IsBasicPasswordSet reports whether SonarQube holds an SMTP password.
	IsBasicPasswordSet bool `json:"isBasicPasswordSet,omitempty"`
	// OAuthAuthenticationHost is the host of the identity provider.
	OAuthAuthenticationHost string `json:"oauthAuthenticationHost,omitempty"`
	// OAuthTenant is the Microsoft tenant.
	OAuthTenant string `json:"oauthTenant,omitempty"`
	// OAuthScope is the scope of the OAuth token request.
	OAuthScope string `json:"oauthScope,omitempty"`
	// IsOAuthClientIDSet reports whether SonarQube holds an OAuth client ID.
	IsOAuthClientIDSet bool `json:"isOauthClientIdSet,omitempty"`
	// IsOAuthClientSecretSet reports whether SonarQube holds an OAuth client
	// secret.
	IsOAuthClientSecretSet bool `json:"isOauthClientSecretSet,omitempty"`
}

// An EmailConfigurationSpec defines the desired state of an
// EmailConfiguration.
//
// writeConnectionSecretToRef is required: SonarQube never returns the SMTP
// password nor the OAuth client ID and secret, so the provider stores the
// values it last applied there to detect their rotation.
// +kubebuilder:validation:XValidation:rule="has(self.writeConnectionSecretToRef) && size(self.writeConnectionSecretToRef.name) > 0",message="writeConnectionSecretToRef with a non-empty name is required"
type EmailConfigurationSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider EmailConfigurationParameters `json:"forProvider"`
}

// An EmailConfigurationStatus represents the observed state of an
// EmailConfiguration.
type EmailConfigurationStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider EmailConfigurationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// An EmailConfiguration manages the email (SMTP) configuration of a
// SonarQube instance (Administration > Configuration > General Settings >
// Email), through the api/v2/system/email-configurations endpoints.
//
// SonarQube supports a single email configuration per instance. The external
// name is the configuration ID assigned by SonarQube. To manage a
// configuration that already exists, set the crossplane.io/external-name
// annotation to its ID.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type EmailConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmailConfigurationSpec   `json:"spec"`
	Status EmailConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EmailConfigurationList contains a list of EmailConfiguration.
type EmailConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []EmailConfiguration `json:"items"`
}

// EmailConfiguration type metadata.
var (
	EmailConfigurationKind             = reflect.TypeFor[EmailConfiguration]().Name()
	EmailConfigurationGroupKind        = schema.GroupKind{Group: Group, Kind: EmailConfigurationKind}.String()
	EmailConfigurationKindAPIVersion   = EmailConfigurationKind + "." + SchemeGroupVersion.String()
	EmailConfigurationGroupVersionKind = SchemeGroupVersion.WithKind(EmailConfigurationKind)
)

// init registers the EmailConfiguration resource with the Scheme.
func init() {
	register(&EmailConfiguration{}, &EmailConfigurationList{})
}
