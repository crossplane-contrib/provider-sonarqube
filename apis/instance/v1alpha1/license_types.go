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

// LicenseEndpoint configures an HTTP(S) endpoint the license key is fetched
// from, instead of a static Kubernetes secret. This is useful for
// vendor-provided license portals that always serve the currently valid
// license, so that renewals are picked up automatically.
//
// If the endpoint is temporarily unreachable, the provider falls back to
// the license key last saved to writeConnectionSecretToRef instead of
// failing the reconcile, so a transient outage does not disrupt the
// SonarQube instance's license.
//
// +kubebuilder:validation:XValidation:rule="!has(self.basicAuthUsername) || has(self.basicAuthPasswordSecretRef)",message="basicAuthPasswordSecretRef is required when basicAuthUsername is set"
// +kubebuilder:validation:XValidation:rule="!has(self.basicAuthPasswordSecretRef) || has(self.basicAuthUsername)",message="basicAuthUsername is required when basicAuthPasswordSecretRef is set"
// +kubebuilder:validation:XValidation:rule="!(has(self.basicAuthUsername) && has(self.bearerTokenSecretRef))",message="basicAuthUsername and bearerTokenSecretRef are mutually exclusive"
type LicenseEndpoint struct {
	// URL is the HTTP(S) endpoint the license key is fetched from. The
	// response body, trimmed of surrounding whitespace, is used verbatim as
	// the license key.
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// BasicAuthUsername is the username sent using HTTP Basic authentication.
	// Requires BasicAuthPasswordSecretRef to also be set. Mutually exclusive
	// with BearerTokenSecretRef.
	// +optional
	BasicAuthUsername *string `json:"basicAuthUsername,omitempty"`

	// BasicAuthPasswordSecretRef references the secret key holding the
	// password sent using HTTP Basic authentication.
	// Requires BasicAuthUsername to also be set.
	// +optional
	BasicAuthPasswordSecretRef *xpv1.SecretKeySelector `json:"basicAuthPasswordSecretRef,omitempty"`

	// BearerTokenSecretRef references the secret key holding a bearer token
	// sent in the endpoint request's Authorization header. Mutually
	// exclusive with BasicAuthUsername.
	// +optional
	BearerTokenSecretRef *xpv1.SecretKeySelector `json:"bearerTokenSecretRef,omitempty"`
}

// LicenseParameters are the configurable fields of a License.
//
// Exactly one of LicenseKeySecretRef or Endpoint must be set.
//
// +kubebuilder:validation:XValidation:rule="has(self.licenseKeySecretRef) != has(self.endpoint)",message="exactly one of licenseKeySecretRef or endpoint must be set"
type LicenseParameters struct {
	// LicenseKeySecretRef references the secret key holding the license key
	// to apply directly to the SonarQube instance.
	// Exactly one of LicenseKeySecretRef or Endpoint must be set.
	// +optional
	LicenseKeySecretRef *xpv1.SecretKeySelector `json:"licenseKeySecretRef,omitempty"`

	// Endpoint fetches the license key from an HTTP(S) endpoint instead of a
	// static secret, optionally using authentication. Useful when a vendor
	// license portal exposes a stable URL that always serves the current
	// license, e.g. to automatically pick up renewals.
	// Exactly one of LicenseKeySecretRef or Endpoint must be set.
	// +optional
	Endpoint *LicenseEndpoint `json:"endpoint,omitempty"`
}

// LicenseObservation are the observable fields of a License.
type LicenseObservation struct {
	ContactEmail           string `json:"contactEmail"`
	ExpiresAt              string `json:"expiresAt"`
	IsExpired              bool   `json:"isExpired"`
	IsOfficialDistribution bool   `json:"isOfficialDistribution"`
	IsSupported            bool   `json:"isSupported"`
	IsValidEdition         bool   `json:"isValidEdition"`
	IsValidServerId        bool   `json:"isValidServerId"`
	LOCsMax                int64  `json:"locsMax"`
	LOCsRemaining          int64  `json:"locsRemaining"`
	Organization           string `json:"organization"`
	ProductEdition         string `json:"productEdition"`
	ServerId               string `json:"serverId"`
	Type                   string `json:"type"`
}

// A LicenseSpec defines the desired state of a License.
//
// writeConnectionSecretToRef is required: without it the provider has no
// record of the license key it last applied, so it cannot tell whether the
// SonarQube instance is still in sync and will re-apply (and, if Endpoint is
// configured, re-fetch) the license on every reconcile even when nothing
// has actually changed -- a permanent reconcile loop.
// +kubebuilder:validation:XValidation:rule="has(self.writeConnectionSecretToRef) && size(self.writeConnectionSecretToRef.name) > 0",message="writeConnectionSecretToRef with a non-empty name is required"
type LicenseSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider LicenseParameters `json:"forProvider"`
}

// A LicenseStatus represents the observed state of a License.
type LicenseStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider LicenseObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A License manages a SonarQube instance license.
//
// WARNING: spec.writeConnectionSecretToRef is required. Without it the
// provider cannot remember the license key it last applied, so it will
// re-apply (and, if spec.forProvider.endpoint is configured, re-fetch) the
// license on every reconcile even when nothing has changed -- a permanent
// reconcile loop.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type License struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LicenseSpec   `json:"spec"`
	Status LicenseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LicenseList contains a list of License.
type LicenseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []License `json:"items"`
}

var (
	// LicenseKind is the kind name of a License.
	LicenseKind = reflect.TypeFor[License]().Name()
	// LicenseGroupKind is the group-kind name of a License.
	LicenseGroupKind = schema.GroupKind{Group: Group, Kind: LicenseKind}.String()
	// LicenseKindAPIVersion is the kind and API version of a License.
	LicenseKindAPIVersion = LicenseKind + "." + SchemeGroupVersion.String()
	// LicenseGroupVersionKind is the GroupVersionKind of a License.
	LicenseGroupVersionKind = SchemeGroupVersion.WithKind(LicenseKind)
)

// init registers the License and LicenseList types with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&License{}, &LicenseList{})
}
