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
	// GitHubProvisioningTypeJIT provisions users and groups just in time,
	// when users log in.
	GitHubProvisioningTypeJIT = "JIT"
	// GitHubProvisioningTypeAuto provisions users, groups and permissions
	// automatically from GitHub.
	GitHubProvisioningTypeAuto = "AUTO_PROVISIONING"
)

// GitHubConfigurationParameters are the configurable fields of a
// GitHubConfiguration.
type GitHubConfigurationParameters struct {
	// Enabled enables GitHub authentication. Defaults to true.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`
	// ApplicationID is the ID of the GitHub App, found on the GitHub App's
	// page under Settings > Developer Settings > GitHub Apps.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ApplicationID string `json:"applicationId"`
	// ClientID is the client ID of the GitHub App.
	// SonarQube does not return it, so the provider compares it with the
	// value it last applied, saved to writeConnectionSecretToRef.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ClientID string `json:"clientId"`
	// ClientSecretRef references a Secret containing the client secret of
	// the GitHub App.
	// +kubebuilder:validation:Required
	ClientSecretRef *xpv1.LocalSecretKeySelector `json:"clientSecretRef"`
	// PrivateKeyRef references a Secret containing the private key of the
	// GitHub App, the whole content of its .pem file.
	// +kubebuilder:validation:Required
	PrivateKeyRef *xpv1.LocalSecretKeySelector `json:"privateKeyRef"`
	// APIURL is the API URL of the GitHub instance, for example
	// https://api.github.com/ for github.com, or
	// https://github.company.com/api/v3/ for GitHub Enterprise.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	// +kubebuilder:validation:MinLength=1
	APIURL string `json:"apiUrl"`
	// WebURL is the web URL of the GitHub instance, for example
	// https://github.com/ for github.com, or https://github.company.com/
	// for GitHub Enterprise.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	// +kubebuilder:validation:MinLength=1
	WebURL string `json:"webUrl"`
	// AllowedOrganizations restricts authentication (and provisioning) to
	// the members of these GitHub organizations. When empty, users of every
	// organization the GitHub App is installed in can log in.
	// Late-initialized from SonarQube when unset.
	// +kubebuilder:validation:Optional
	// +listType=set
	AllowedOrganizations []string `json:"allowedOrganizations,omitempty"`
	// ProvisioningType is how users and groups are provisioned: JIT (just
	// in time, when users log in) or AUTO_PROVISIONING (users, groups and
	// permissions synchronized from GitHub). Defaults to JIT.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=JIT
	// +kubebuilder:validation:Enum=JIT;AUTO_PROVISIONING
	ProvisioningType string `json:"provisioningType,omitempty"`
	// SynchronizeGroups synchronizes GitHub team memberships with the
	// SonarQube groups of the same name when users log in.
	// Late-initialized from SonarQube when unset.
	// +kubebuilder:validation:Optional
	SynchronizeGroups *bool `json:"synchronizeGroups,omitempty"`
	// AllowUsersToSignUp lets users who do not have a SonarQube account yet
	// sign up through GitHub. Only applies to JIT provisioning.
	// Late-initialized from SonarQube when unset.
	// +kubebuilder:validation:Optional
	AllowUsersToSignUp *bool `json:"allowUsersToSignUp,omitempty"`
	// ProjectVisibility synchronizes the visibility of provisioned projects
	// with the visibility of their GitHub repository. When false, every
	// provisioned project is private. Only applies to AUTO_PROVISIONING.
	// Late-initialized from SonarQube when unset.
	// +kubebuilder:validation:Optional
	ProjectVisibility *bool `json:"projectVisibility,omitempty"`
	// UserConsentRequiredAfterUpgrade reports whether an administrator must
	// consent again to the synchronization of permissions from GitHub,
	// which SonarQube requires after some upgrades. Setting it to false
	// gives that consent. Late-initialized from SonarQube when unset.
	// +kubebuilder:validation:Optional
	UserConsentRequiredAfterUpgrade *bool `json:"userConsentRequiredAfterUpgrade,omitempty"`
}

// GitHubConfigurationObservation are the observable fields of a
// GitHubConfiguration.
type GitHubConfigurationObservation struct {
	// ID is the identifier of the GitHub configuration in SonarQube.
	ID string `json:"id,omitempty"`
	// Enabled reports whether GitHub authentication is enabled.
	Enabled bool `json:"enabled,omitempty"`
	// ApplicationID is the ID of the GitHub App.
	ApplicationID string `json:"applicationId,omitempty"`
	// APIURL is the API URL of the GitHub instance.
	APIURL string `json:"apiUrl,omitempty"`
	// WebURL is the web URL of the GitHub instance.
	WebURL string `json:"webUrl,omitempty"`
	// AllowedOrganizations are the GitHub organizations allowed to
	// authenticate.
	AllowedOrganizations []string `json:"allowedOrganizations,omitempty"`
	// ProvisioningType is the provisioning type.
	ProvisioningType string `json:"provisioningType,omitempty"`
	// SynchronizeGroups reports whether GitHub teams are synchronized with
	// SonarQube groups.
	SynchronizeGroups bool `json:"synchronizeGroups,omitempty"`
	// AllowUsersToSignUp reports whether new users may sign up through
	// GitHub.
	AllowUsersToSignUp bool `json:"allowUsersToSignUp,omitempty"`
	// ProjectVisibility reports whether project visibility is synchronized
	// with GitHub.
	ProjectVisibility bool `json:"projectVisibility,omitempty"`
	// UserConsentRequiredAfterUpgrade reports whether an administrator must
	// consent again to the synchronization of permissions from GitHub.
	UserConsentRequiredAfterUpgrade bool `json:"userConsentRequiredAfterUpgrade,omitempty"`
	// ErrorMessage is the error SonarQube reports for an incorrect
	// configuration, if any.
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// A GitHubConfigurationSpec defines the desired state of a
// GitHubConfiguration.
// +kubebuilder:validation:XValidation:rule="has(self.writeConnectionSecretToRef) && size(self.writeConnectionSecretToRef.name) > 0",message="writeConnectionSecretToRef with a non-empty name is required"
type GitHubConfigurationSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider GitHubConfigurationParameters `json:"forProvider"`
}

// A GitHubConfigurationStatus represents the observed state of a
// GitHubConfiguration.
type GitHubConfigurationStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider GitHubConfigurationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A GitHubConfiguration is a managed resource that represents the GitHub
// authentication and provisioning configuration of a SonarQube instance
// (Administration > Authentication > GitHub). It differs from ALMGitHub,
// which configures a GitHub DevOps Platform integration used to import and
// analyze repositories.
//
// SonarQube supports a single GitHub configuration per instance. A
// GitHubConfiguration without an external name adopts the existing
// configuration, if there is one, and deleting the GitHubConfiguration
// deletes it, unless the deletion policy is Orphan.
//
// spec.writeConnectionSecretToRef is required: SonarQube does not return
// the client ID, client secret and private key, so the provider saves the
// values it last applied there to detect changes.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type GitHubConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitHubConfigurationSpec   `json:"spec"`
	Status GitHubConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GitHubConfigurationList contains a list of GitHubConfiguration.
type GitHubConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GitHubConfiguration `json:"items"`
}

// GitHubConfiguration type metadata.
var (
	GitHubConfigurationKind             = reflect.TypeFor[GitHubConfiguration]().Name()
	GitHubConfigurationGroupKind        = schema.GroupKind{Group: Group, Kind: GitHubConfigurationKind}.String()
	GitHubConfigurationKindAPIVersion   = GitHubConfigurationKind + "." + SchemeGroupVersion.String()
	GitHubConfigurationGroupVersionKind = SchemeGroupVersion.WithKind(GitHubConfigurationKind)
)

// init registers the GitHubConfiguration resource with the Scheme.
func init() {
	register(&GitHubConfiguration{}, &GitHubConfigurationList{})
}
