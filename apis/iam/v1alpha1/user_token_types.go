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

// UserTokenParameters are the configurable fields of a UserToken.
//
// Exactly one of ExpirationDate or RenewalPeriodDays may be set.
// If neither is set the token never expires.
//
// +kubebuilder:validation:XValidation:rule="!has(self.expirationDate) || !has(self.renewalPeriodDays)",message="expirationDate and renewalPeriodDays are mutually exclusive"
// +kubebuilder:validation:XValidation:rule="!has(self.projectKey) || self.type == 'PROJECT_ANALYSIS_TOKEN'",message="projectKey is only valid when type is PROJECT_ANALYSIS_TOKEN"
// +kubebuilder:validation:XValidation:rule="self.type != 'PROJECT_ANALYSIS_TOKEN' || has(self.projectKey)",message="projectKey is required when type is PROJECT_ANALYSIS_TOKEN"
// +kubebuilder:validation:XValidation:rule="!has(self.renewBeforeDays) || has(self.renewalPeriodDays)",message="renewBeforeDays is only valid when renewalPeriodDays is set"
// +kubebuilder:validation:XValidation:rule="!has(self.renewBeforeDays) || self.renewBeforeDays < self.renewalPeriodDays",message="renewBeforeDays must be less than renewalPeriodDays"
type UserTokenParameters struct {
	// Name is the token name. Immutable after creation. Used as external name.
	// WARNING: This field is immutable once set.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Name is immutable."
	Name string `json:"name"`

	// Login is the user login the token belongs to.
	// WARNING: This field is immutable once set.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Login is immutable."
	// +crossplane:generate:reference:type=github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1.User
	// +crossplane:generate:reference:extractor=github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1.UserLogin()
	Login *string `json:"login,omitempty"`

	// LoginRef references a User to populate the login field.
	// +kubebuilder:validation:Optional
	LoginRef *xpv1.NamespacedReference `json:"loginRef,omitempty"`

	// LoginSelector selects a User to populate the login field.
	// +kubebuilder:validation:Optional
	LoginSelector *xpv1.NamespacedSelector `json:"loginSelector,omitempty"`

	// Type is the token type.
	// WARNING: This field is immutable once set.
	// +kubebuilder:validation:Enum=USER_TOKEN;GLOBAL_ANALYSIS_TOKEN;PROJECT_ANALYSIS_TOKEN
	// +kubebuilder:default=USER_TOKEN
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Type is immutable."
	Type string `json:"type"`

	// ProjectKey is the key of the project this token grants analysis access to.
	// Required when Type is PROJECT_ANALYSIS_TOKEN. Immutable after creation.
	// WARNING: This field is immutable once set.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ProjectKey is immutable."
	// +kubebuilder:validation:Optional
	ProjectKey *string `json:"projectKey,omitempty"`

	// ExpirationDate is an explicit token expiration date in YYYY-MM-DD format.
	// Mutually exclusive with RenewalPeriodDays.
	// If neither ExpirationDate nor RenewalPeriodDays is set, the token never expires.
	// +kubebuilder:validation:Optional
	ExpirationDate *string `json:"expirationDate,omitempty"`

	// RenewalPeriodDays sets the token lifetime in days.
	// The controller creates the token with expirationDate = now + N days.
	// When the token expires the controller automatically revokes it and
	// issues a new one, refreshing the connection secret.
	// Mutually exclusive with ExpirationDate.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Optional
	RenewalPeriodDays *int64 `json:"renewalPeriodDays,omitempty"`

	// RenewBeforeDays allows the provider to proactively rotate the token before
	// it expires.
	// When set, the provider rotates the token when fewer than RenewBeforeDays
	// remain before expiry. Only valid with RenewalPeriodDays.
	// Must be less than RenewalPeriodDays.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Optional
	RenewBeforeDays *int64 `json:"renewBeforeDays,omitempty"`
}

// UserTokenObservation are the observable fields of a UserToken.
type UserTokenObservation struct {
	// CreatedAt is when the token was generated.
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`
	// ExpirationDate is the token expiration date, if set.
	ExpirationDate *metav1.Time `json:"expirationDate,omitempty"`
	// IsExpired is true when the token has passed its expiration date.
	IsExpired bool `json:"isExpired"`
	// Name is the name of the token.
	Name string `json:"name"`
	// Type is the type of the token (USER_TOKEN, GLOBAL_ANALYSIS_TOKEN, PROJECT_ANALYSIS_TOKEN).
	Type string `json:"type,omitempty"`
	// RenewAt is the computed time at which the provider will renew the token.
	// Only populated for renewal-managed tokens.
	// +kubebuilder:validation:Optional
	RenewAt *metav1.Time `json:"renewAt,omitempty"`
}

// A UserTokenSpec defines the desired state of a UserToken.
// +kubebuilder:validation:XValidation:rule="has(self.writeConnectionSecretToRef) && size(self.writeConnectionSecretToRef.name) > 0",message="writeConnectionSecretToRef with a non-empty name is required"
type UserTokenSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider UserTokenParameters `json:"forProvider"`
}

// A UserTokenStatus represents the observed state of a UserToken.
type UserTokenStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	AtProvider UserTokenObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A UserToken manages a SonarQube user access token.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="TYPE",type="string",JSONPath=".status.atProvider.type"
// +kubebuilder:printcolumn:name="EXPIRATION",type="string",JSONPath=".status.atProvider.expirationDate"
// +kubebuilder:printcolumn:name="RENEW-AT",type="string",JSONPath=".status.atProvider.renewAt"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type UserToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserTokenSpec   `json:"spec"`
	Status UserTokenStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserTokenList contains a list of UserToken.
type UserTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []UserToken `json:"items"`
}

// UserToken type metadata.
var (
	UserTokenKind             = reflect.TypeFor[UserToken]().Name()
	UserTokenGroupKind        = schema.GroupKind{Group: APIGroup, Kind: UserTokenKind}.String()
	UserTokenKindAPIVersion   = UserTokenKind + "." + SchemeGroupVersion.String()
	UserTokenGroupVersionKind = SchemeGroupVersion.WithKind(UserTokenKind)
)

// init registers UserToken types with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&UserToken{}, &UserTokenList{})
}
