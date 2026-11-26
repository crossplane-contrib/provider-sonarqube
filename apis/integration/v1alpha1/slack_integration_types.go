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

// SlackIntegrationParameters are the configurable fields of a
// SlackIntegration. Their values are found in the settings of the Slack app.
type SlackIntegrationParameters struct {
	// ClientID is the client ID of the Slack app.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ClientID string `json:"clientId"`

	// ClientSecretRef references the Secret key holding the client secret of
	// the Slack app. The Secret is read from the SlackIntegration's
	// namespace.
	// +kubebuilder:validation:Required
	ClientSecretRef *xpv1.LocalSecretKeySelector `json:"clientSecretRef"`

	// SigningSecretRef references the Secret key holding the signing secret
	// of the Slack app, which SonarQube uses to validate the requests Slack
	// sends. The Secret is read from the SlackIntegration's namespace.
	// +kubebuilder:validation:Required
	SigningSecretRef *xpv1.LocalSecretKeySelector `json:"signingSecretRef"`
}

// SlackIntegrationObservation are the observable fields of a
// SlackIntegration.
type SlackIntegrationObservation struct {
	// ID is the identifier of the configuration in SonarQube, used as the
	// external name.
	ID string `json:"id,omitempty"`
	// ClientID is the client ID of the Slack app.
	ClientID string `json:"clientId,omitempty"`
	// AppID is the identifier of the Slack app, as reported by SonarQube.
	AppID string `json:"appId,omitempty"`
}

// A SlackIntegrationSpec defines the desired state of a SlackIntegration.
//
// writeConnectionSecretToRef is required: SonarQube never returns the client
// secret nor the signing secret, so the provider stores the values it last
// applied there to detect their rotation.
// +kubebuilder:validation:XValidation:rule="has(self.writeConnectionSecretToRef) && size(self.writeConnectionSecretToRef.name) > 0",message="writeConnectionSecretToRef with a non-empty name is required"
type SlackIntegrationSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`

	ForProvider SlackIntegrationParameters `json:"forProvider"`
}

// A SlackIntegrationStatus represents the observed state of a
// SlackIntegration.
type SlackIntegrationStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`

	AtProvider SlackIntegrationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A SlackIntegration manages the Slack integration configuration of a
// SonarQube instance, through the
// api/v2/integrations/integration-configurations endpoints. It requires
// SonarQube Enterprise Edition or above.
//
// SonarQube holds a single Slack configuration per instance. The external
// name is the configuration ID assigned by SonarQube. To manage a
// configuration that already exists, set the crossplane.io/external-name
// annotation to its ID.
//
// Deleting a SlackIntegration deletes the configuration along with its
// related data in SonarQube: workspaces, user bindings and subscriptions.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,sonarqube}
type SlackIntegration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SlackIntegrationSpec   `json:"spec"`
	Status SlackIntegrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SlackIntegrationList contains a list of SlackIntegration.
type SlackIntegrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SlackIntegration `json:"items"`
}

// SlackIntegration type metadata.
var (
	SlackIntegrationKind             = reflect.TypeFor[SlackIntegration]().Name()
	SlackIntegrationGroupKind        = schema.GroupKind{Group: Group, Kind: SlackIntegrationKind}.String()
	SlackIntegrationKindAPIVersion   = SlackIntegrationKind + "." + SchemeGroupVersion.String()
	SlackIntegrationGroupVersionKind = SchemeGroupVersion.WithKind(SlackIntegrationKind)
)

// init registers the SlackIntegration resource with the Scheme.
func init() {
	register(&SlackIntegration{}, &SlackIntegrationList{})
}
