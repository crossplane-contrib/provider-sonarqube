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

// GovernanceReportParameters configure the PDF reports of a Portfolio or an
// Application (Enterprise Edition only). Each setting is only managed when it
// is set.
type GovernanceReportParameters struct {
	// Frequency is how often the report is sent. When unset, the frequency is
	// not managed: removing it keeps the last value applied in SonarQube.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=daily;weekly;monthly
	Frequency *string `json:"frequency,omitempty"`

	// Recipients is the exhaustive list of email addresses the report is sent
	// to, as configured on the component itself. When unset, the recipients are
	// not managed; an empty list removes all of them.
	// SonarQube only documents recipients for portfolios: it may reject them on
	// an application.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:items:MinLength=1
	// +kubebuilder:validation:items:Pattern=`^[^,\s]+$`
	// +listType=set
	Recipients *[]string `json:"recipients,omitempty"`
}

// GovernanceReportObservation is the observed report configuration of a
// Portfolio or an Application. It is only filled when a report setting is
// managed.
type GovernanceReportObservation struct {
	// Frequency is the report frequency set on the component. It is empty when
	// the component uses the global frequency.
	Frequency string `json:"frequency,omitempty"`
	// GlobalFrequency is the report frequency configured globally.
	GlobalFrequency string `json:"globalFrequency,omitempty"`
	// Recipients are the report recipients set on the component.
	Recipients []string `json:"recipients,omitempty"`
}
