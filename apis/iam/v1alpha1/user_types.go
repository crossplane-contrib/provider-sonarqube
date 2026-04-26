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

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// UserParameters are the configurable fields of a User.
type UserParameters struct {
	// Login is the unique login of the user in SonarQube.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=100
	Login string `json:"login"`
	// Name is the display name of the user in SonarQube.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=200
	Name string `json:"name"`
	// Email is the user's email address.
	// +kubebuilder:validation:Optional
	Email *string `json:"email,omitempty"`
	// Local indicates whether the user is authenticated locally.
	// +kubebuilder:validation:Optional
	Local *bool `json:"local,omitempty"`
	// PasswordSecretRef points to a secret containing the password for local user creation.
	// WARNING: this field is only used during creation and is not reconciled. Updating the password in the secret after creation will have no effect on the external resource. To change the password after creation, please use SonarQube's Change Password API directly.
	// +kubebuilder:validation:Optional
	PasswordSecretRef *xpv1.SecretKeySelector `json:"passwordSecretRef,omitempty"`
	// ScmAccounts are the SCM accounts associated with the user.
	// +kubebuilder:validation:Optional
	ScmAccounts *[]string `json:"scmAccounts,omitempty"`
	// Groups is the desired set of group ids the user should belong to.
	// +kubebuilder:validation:Optional
	Groups *[]UserGroupsParameters `json:"groups,omitempty"`
	// ExternalId is the user's external identifier.
	// +kubebuilder:validation:Optional
	ExternalId *string `json:"externalId,omitempty"`
	// ExternalLogin is the user's external login.
	// +kubebuilder:validation:Optional
	ExternalLogin *string `json:"externalLogin,omitempty"`
	// ExternalProvider is the user's external provider.
	// +kubebuilder:validation:Optional
	ExternalProvider *string `json:"externalProvider,omitempty"`
	// Anonymize defines the deletion (deactivation) behaviour.
	// If set to true, the user will be anonymized in addition to being deactivated, which means that all personally identifiable information will be removed from the user and replaced with generic values. This is useful for compliance with data protection regulations such as GDPR.
	Anonymize *bool `json:"anonymize,omitempty"`
}

// UserGroupsParameters specifies group membership for a user.
type UserGroupsParameters struct {
	// GroupId is the unique identifier of the group in SonarQube.
	// +kubebuilder:validation:Optional
	GroupId *string `json:"groupId,omitempty"`
	// GroupIdRef references a Group to retrieve its ID from.
	// +kubebuilder:validation:Optional
	GroupIdRef *xpv1.NamespacedReference `json:"groupIdRef,omitempty"`
	// GroupIdSelector selects a Group to retrieve its ID from, based on labels.
	// +kubebuilder:validation:Optional
	GroupIdSelector *xpv1.Selector `json:"groupIdSelector,omitempty"`
}

// UserObservation are the observable fields of a User.
type UserObservation struct {
	// Id is the unique identifier of the user in SonarQube.
	Id string `json:"id,omitempty"`
	// Active indicates whether the user is active.
	Active bool `json:"active,omitempty"`
	// Email is the user's email address.
	Email string `json:"email,omitempty"`
	// ExternalId is the user's external identifier.
	ExternalId string `json:"externalId,omitempty"`
	// ExternalLogin is the user's external login.
	ExternalLogin string `json:"externalLogin,omitempty"`
	// ExternalProvider is the user's external provider.
	ExternalProvider string `json:"externalProvider,omitempty"`
	// Groups is the map of group ids:group membership Id the user belongs to.
	Groups map[string]string `json:"groups,omitempty"`
	// Local indicates whether the user is locally authenticated.
	Local bool `json:"local,omitempty"`
	// Login is the user's login.
	Login string `json:"login,omitempty"`
	// Managed indicates whether the user is managed externally.
	Managed bool `json:"managed,omitempty"`
	// Name is the user's display name.
	Name string `json:"name,omitempty"`
	// ScmAccounts are the SCM accounts associated with the user.
	ScmAccounts []string `json:"scmAccounts,omitempty"`
	// SonarLintLastConnectionDate is the user's last SonarLint connection date.
	SonarLintLastConnectionDate string `json:"sonarLintLastConnectionDate,omitempty"`
	// SonarQubeLastConnectionDate is the user's last SonarQube connection date.
	SonarQubeLastConnectionDate string `json:"sonarQubeLastConnectionDate,omitempty"`
	// Avatar is the URL of the user's avatar.
	Avatar string `json:"avatar,omitempty"`
}

// A UserSpec defines the desired state of a User.
type UserSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider UserParameters `json:"forProvider"`
}

// A UserStatus represents the observed state of a User.
type UserStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	AtProvider UserObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A User is the Schema for the Users API.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec"`
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserList contains a list of User.
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []User `json:"items"`
}

// User type metadata.
var (
	UserKind             = reflect.TypeFor[User]().Name()
	UserGroupKind        = schema.GroupKind{Group: APIGroup, Kind: UserKind}.String()
	UserKindAPIVersion   = UserKind + "." + SchemeGroupVersion.String()
	UserGroupVersionKind = SchemeGroupVersion.WithKind(UserKind)
)

// init registers the User resource with the Scheme.
func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
