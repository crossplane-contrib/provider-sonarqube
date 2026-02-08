package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A ProviderConfigStatus defines the status of a Provider.
type ProviderConfigStatus struct {
	xpv1.ProviderConfigStatus `json:",inline"`
}

// ProviderCredentials required to authenticate.
type ProviderCredentials struct {
	xpv1.CommonCredentialSelectors `json:",inline"`

	// Source of the provider credentials.
	// +kubebuilder:validation:Enum=None;Secret;InjectedIdentity;Environment;Filesystem
	Source xpv1.CredentialsSource `json:"source"`
}

type ProviderConfigSpec struct {
	// BaseURL of the SonarQube instance.
	// +kubebuilder:validation:Required
	BaseURL string `json:"baseUrl"`

	// InsecureSkipVerify indicates whether to skip TLS certificate verification.
	InsecureSkipVerify *bool `json:"insecureSkipVerify,omitempty"`

	// Token is the User Token required to authenticate with the SonarQube instance.
	// WARNING: This MUST NOT be an Analysis token / project token, it MUST be a User token with appropriate permissions.
	// +kubebuilder:validation:Optional
	Token *ProviderCredentials `json:"token,omitempty"`

	// Username is the username for Basic Authentication to the SonarQube instance.
	// +kubebuilder:validation:Optional
	Username *ProviderCredentials `json:"username,omitempty"`
	// Password is the password for Basic Authentication to the SonarQube instance.
	// +kubebuilder:validation:Optional
	Password *ProviderCredentials `json:"password,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="SECRET-NAME",type="string",JSONPath=".spec.credentials.secretRef.name",priority=1
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,provider,sonarqube}

// ProviderConfig configures a SonarQube provider.
//
//nolint:modernize // omitempty is needed because of kubebuilder's handling of optional fields in status.
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig.
//
//nolint:modernize // omitempty is needed because of kubebuilder's handling of optional fields in status.
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ProviderConfig `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="CONFIG-NAME",type="string",JSONPath=".providerConfigRef.name"
// +kubebuilder:printcolumn:name="RESOURCE-KIND",type="string",JSONPath=".resourceRef.kind"
// +kubebuilder:printcolumn:name="RESOURCE-NAME",type="string",JSONPath=".resourceRef.name"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,provider,sonarqube}

// ProviderConfigUsage indicates that a resource is using a ProviderConfig.
//
//nolint:modernize // omitempty is needed because of kubebuilder's handling of optional fields in status.
type ProviderConfigUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	xpv2.TypedProviderConfigUsage `json:",inline"`
}

// +kubebuilder:object:root=true

// ProviderConfigUsageList contains a list of ProviderConfigUsage.
//
//nolint:modernize // omitempty is needed because of kubebuilder's handling of optional fields in status.
type ProviderConfigUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ProviderConfigUsage `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="SECRET-NAME",type="string",JSONPath=".spec.credentials.secretRef.name",priority=1
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,sonarqube}

// ClusterProviderConfig configures a SonarQube provider.
//
//nolint:modernize // omitempty is needed because of kubebuilder's handling of optional fields in status.
type ClusterProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterProviderConfigList contains a list of ClusterProviderConfig.
//
//nolint:modernize // omitempty is needed because of kubebuilder's handling of optional fields in status.
type ClusterProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ClusterProviderConfig `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="CONFIG-NAME",type="string",JSONPath=".providerConfigRef.name"
// +kubebuilder:printcolumn:name="RESOURCE-KIND",type="string",JSONPath=".resourceRef.kind"
// +kubebuilder:printcolumn:name="RESOURCE-NAME",type="string",JSONPath=".resourceRef.name"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,sonarqube}

// ClusterProviderConfigUsage indicates that a resource is using a ClusterProviderConfig.
//
//nolint:modernize // omitempty is needed because of kubebuilder's handling of optional fields in status.
type ClusterProviderConfigUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	xpv2.TypedProviderConfigUsage `json:",inline"`
}

// +kubebuilder:object:root=true

// ClusterProviderConfigUsageList contains a list of ClusterProviderConfigUsage.
//
//nolint:modernize // omitempty is needed because of kubebuilder's handling of optional fields in status.
type ClusterProviderConfigUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ClusterProviderConfigUsage `json:"items"`
}
