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
	// GitLabProvisioningTypeJIT provisions users and groups just in time,
	// when they first log in.
	GitLabProvisioningTypeJIT = "JIT"
	// GitLabProvisioningTypeAuto provisions users and groups automatically
	// from GitLab, using the provisioning token.
	GitLabProvisioningTypeAuto = "AUTO_PROVISIONING"
)

// GitLabConfigurationParameters are the configurable fields of a
// GitLabConfiguration.
//
// +kubebuilder:validation:XValidation:rule="!has(self.provisioningType) || self.provisioningType != 'AUTO_PROVISIONING' || has(self.provisioningTokenSecretRef)",message="provisioningTokenSecretRef is required when provisioningType is AUTO_PROVISIONING"
// +kubebuilder:validation:XValidation:rule="!has(self.provisioningType) || self.provisioningType != 'AUTO_PROVISIONING' || (has(self.allowAllGroups) && self.allowAllGroups) || (has(self.allowedGroups) && size(self.allowedGroups) > 0)",message="allowedGroups must not be empty when provisioningType is AUTO_PROVISIONING, unless allowAllGroups is true"
type GitLabConfigurationParameters struct {
	// Enabled enables GitLab authentication.
	// Defaults to true.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`

	// ApplicationID is the ID of the GitLab OAuth application.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ApplicationID string `json:"applicationId"`

	// URL is the URL of the GitLab instance, for example https://gitlab.com.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	// +kubebuilder:validation:MinLength=1
	URL string `json:"url"`

	// ApplicationSecretRef references the Secret key holding the secret of
	// the GitLab OAuth application. The Secret is read from the
	// GitLabConfiguration's namespace.
	// +kubebuilder:validation:Required
	ApplicationSecretRef *xpv1.LocalSecretKeySelector `json:"applicationSecretRef"`

	// SynchronizeGroups synchronizes the GitLab groups of a user with
	// SonarQube groups on login.
	// Late-initialized from SonarQube if unset (false on creation).
	// +kubebuilder:validation:Optional
	SynchronizeGroups *bool `json:"synchronizeGroups,omitempty"`

	// AllowedGroups are the GitLab groups whose members may authenticate.
	// Subgroups are included. With AUTO_PROVISIONING, members of these groups
	// are provisioned in SonarQube, and the list must not be empty unless
	// AllowAllGroups is true.
	// Late-initialized from SonarQube if unset.
	// +kubebuilder:validation:Optional
	// +listType=set
	AllowedGroups []string `json:"allowedGroups,omitempty"`

	// AllowAllGroups provisions every group visible to the provisioning
	// token and ignores AllowedGroups. It only applies to AUTO_PROVISIONING,
	// and SonarQube rejects it for gitlab.com.
	// Late-initialized from SonarQube if unset.
	// +kubebuilder:validation:Optional
	AllowAllGroups *bool `json:"allowAllGroups,omitempty"`

	// ProvisioningType is how users and groups are provisioned: JIT (just in
	// time, on first login) or AUTO_PROVISIONING (synchronized from GitLab,
	// requires ProvisioningTokenSecretRef).
	// Defaults to JIT.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=JIT
	// +kubebuilder:validation:Enum=JIT;AUTO_PROVISIONING
	ProvisioningType string `json:"provisioningType,omitempty"`

	// ProvisioningTokenSecretRef references the Secret key holding the GitLab
	// token used for AUTO_PROVISIONING. The Secret is read from the
	// GitLabConfiguration's namespace.
	// Removing this reference does not remove the token from SonarQube.
	// +kubebuilder:validation:Optional
	ProvisioningTokenSecretRef *xpv1.LocalSecretKeySelector `json:"provisioningTokenSecretRef,omitempty"`

	// AllowUsersToSignUp lets users who are not yet in SonarQube sign up
	// with GitLab. With JIT, this creates their SonarQube account on first
	// login.
	// Late-initialized from SonarQube if unset.
	// +kubebuilder:validation:Optional
	AllowUsersToSignUp *bool `json:"allowUsersToSignUp,omitempty"`
}

// GitLabConfigurationObservation are the observable fields of a
// GitLabConfiguration.
type GitLabConfigurationObservation struct {
	// ID is the identifier of the configuration in SonarQube, used as the
	// external name.
	ID string `json:"id,omitempty"`
	// Enabled reports whether GitLab authentication is enabled.
	Enabled bool `json:"enabled,omitempty"`
	// ApplicationID is the ID of the GitLab OAuth application.
	ApplicationID string `json:"applicationId,omitempty"`
	// URL is the URL of the GitLab instance.
	URL string `json:"url,omitempty"`
	// SynchronizeGroups reports whether GitLab groups are synchronized.
	SynchronizeGroups bool `json:"synchronizeGroups,omitempty"`
	// AllowedGroups are the GitLab groups whose members may authenticate.
	AllowedGroups []string `json:"allowedGroups,omitempty"`
	// AllowAllGroups reports whether every group visible to the provisioning
	// token is provisioned.
	AllowAllGroups bool `json:"allowAllGroups,omitempty"`
	// AllowUsersToSignUp reports whether new users may sign up.
	AllowUsersToSignUp bool `json:"allowUsersToSignUp,omitempty"`
	// ProvisioningType is the provisioning type.
	ProvisioningType string `json:"provisioningType,omitempty"`
	// IsProvisioningTokenSet reports whether SonarQube holds a provisioning
	// token.
	IsProvisioningTokenSet bool `json:"isProvisioningTokenSet,omitempty"`
	// ErrorMessage is the error SonarQube reports when validating the
	// configuration against GitLab, if any.
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// A GitLabConfigurationSpec defines the desired state of a
// GitLabConfiguration.
//
// writeConnectionSecretToRef is required: SonarQube never returns the
// application secret nor the provisioning token, so the provider stores the
// values it last applied there to detect their rotation.
// +kubebuilder:validation:XValidation:rule="has(self.writeConnectionSecretToRef) && size(self.writeConnectionSecretToRef.name) > 0",message="writeConnectionSecretToRef with a non-empty name is required"
type GitLabConfigurationSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider GitLabConfigurationParameters `json:"forProvider"`
}

// A GitLabConfigurationStatus represents the observed state of a
// GitLabConfiguration.
type GitLabConfigurationStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider GitLabConfigurationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A GitLabConfiguration manages the GitLab authentication and provisioning
// configuration of a SonarQube instance (Administration > Authentication >
// GitLab), through the api/v2/dop-translation/gitlab-configurations
// endpoints.
//
// It is not the GitLab ALM integration managed by ALMGitLab, which
// configures DevOps platform settings used for project import and merge
// request decoration.
//
// SonarQube supports a single GitLab configuration per instance. The
// external name is the configuration ID assigned by SonarQube. To manage a
// configuration that already exists, set the crossplane.io/external-name
// annotation to its ID.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type GitLabConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitLabConfigurationSpec   `json:"spec"`
	Status GitLabConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GitLabConfigurationList contains a list of GitLabConfiguration.
type GitLabConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GitLabConfiguration `json:"items"`
}

// GitLabConfiguration type metadata.
var (
	GitLabConfigurationKind             = reflect.TypeFor[GitLabConfiguration]().Name()
	GitLabConfigurationGroupKind        = schema.GroupKind{Group: Group, Kind: GitLabConfigurationKind}.String()
	GitLabConfigurationKindAPIVersion   = GitLabConfigurationKind + "." + SchemeGroupVersion.String()
	GitLabConfigurationGroupVersionKind = SchemeGroupVersion.WithKind(GitLabConfigurationKind)
)

// init registers the GitLabConfiguration resource with the Scheme.
func init() {
	register(&GitLabConfiguration{}, &GitLabConfigurationList{})
}
