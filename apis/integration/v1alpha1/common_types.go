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
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// ALMCommonParameters are the common configurable fields for all ALM integrations in SonarQube.
type ALMCommonParameters struct {
	// URL is the URL of the ALM system API to integrate with SonarQube.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	// +kubebuilder:validation:MinLength=1
	URL string `json:"url"`
	// Key is the unique identifier for the ALM integration in SonarQube.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
	// PersonalAccessTokenSecretRef references a Secret containing the personal access token to authenticate with the ALM system.
	// +kubebuilder:validation:Required
	PersonalAccessTokenSecretRef *xpv1.LocalSecretKeySelector `json:"personalAccessTokenSecretRef"`
}

// ALMCommonObservation are the common observable fields for all ALM integrations in SonarQube.
type ALMCommonObservation struct {
	// Key is the unique identifier for the ALM integration in SonarQube.
	Key string `json:"key,omitempty"`
	// URL is the URL of the ALM system to integrate with SonarQube.
	URL string `json:"url,omitempty"`
}
