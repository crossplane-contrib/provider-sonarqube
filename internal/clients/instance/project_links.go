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

package instance

import (
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/sonar"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

type ProjectLinksClient interface {
	Create(opt *sonar.ProjectLinksCreateOption) (*sonar.ProjectLinksCreate, *http.Response, error)
	Delete(opt *sonar.ProjectLinksDeleteOption) (*http.Response, error)
	Search(opt *sonar.ProjectLinksSearchOption) (*sonar.ProjectLinksSearch, *http.Response, error)
}

// NewProjectLinksClient creates a new ProjectLinksClient with the provided SonarQube client configuration.
func NewProjectLinksClient(clientConfig common.Config) ProjectLinksClient {
	newClient := common.NewClient(clientConfig)

	return newClient.ProjectLinks
}

// GenerateProjectLinksCreateOptions generates the options for creating a project link based on the provided project ID and ProjectLinkParameters.
func GenerateProjectLinksCreateOptions(projectId string, link v1alpha1.ProjectLinkParameters) *sonar.ProjectLinksCreateOption {
	opts := &sonar.ProjectLinksCreateOption{
		ProjectID: projectId,
		Name:      link.Name,
		URL:       link.URL,
	}

	return opts
}

// GenerateProjectLinksDeleteOptions generates the options for deleting a project link based on the provided link ID.
func GenerateProjectLinksDeleteOptions(linkId string) *sonar.ProjectLinksDeleteOption {
	return &sonar.ProjectLinksDeleteOption{
		ID: linkId,
	}
}

// GenerateProjectLinksSearchOptions generates the options for searching project links based on the provided project ID.
func GenerateProjectLinksSearchOptions(projectId string) *sonar.ProjectLinksSearchOption {
	return &sonar.ProjectLinksSearchOption{
		ProjectID: projectId,
	}
}

// LateInitializeProjectLinks performs late initialization of the ProjectLinkParameters in the ProjectParameters based on the observed project links from SonarQube.
// It updates the ProjectParameters with any missing information from the observed project links.
func LateInitializeProjectLinks(spec *v1alpha1.ProjectParameters, observation map[string]v1alpha1.ProjectLinkObservation) {
	// Update the ProjectLinks that do not have an ID in the spec
	for i, link := range spec.Links {
		if link.ID == nil {
			for _, observedLink := range observation {
				if link.Name == observedLink.Name && link.URL == observedLink.URL {
					spec.Links[i].ID = &observedLink.ID

					break
				}
			}
		}
	}
}

// GenerateProjectLinksObservations generates the observations for the links of a SonarQube Project based on the provided list of links.
func GenerateProjectLinksObservations(links sonar.ProjectLinksSearch) map[string]v1alpha1.ProjectLinkObservation {
	observations := make(map[string]v1alpha1.ProjectLinkObservation)
	for _, link := range links.Links {
		observations[link.Name] = v1alpha1.ProjectLinkObservation{
			ID:  link.ID,
			URL: link.URL,
		}
	}

	return observations
}

// AreProjectLinksUpToDate compares the desired state of the project links in the spec with the observed state of the project links from SonarQube and returns true if they are up to date, or false if they are not.
func AreProjectLinksUpToDate(specLinks []v1alpha1.ProjectLinkParameters, observedLinks map[string]v1alpha1.ProjectLinkObservation) bool {
	if len(specLinks) != len(observedLinks) {
		return false
	}

	for _, specLink := range specLinks {
		observedLink, ok := observedLinks[specLink.Name]
		if !ok || !IsProjectLinkUpToDate(specLink, &observedLink) {
			return false
		}
	}

	return true
}

// IsProjectLinkUpToDate checks if the desired state of a project link in the spec is up to date with the observed state of the project link from SonarQube and returns true if it is up to date, or false if it is not.
func IsProjectLinkUpToDate(specLink v1alpha1.ProjectLinkParameters, observedLink *v1alpha1.ProjectLinkObservation) bool {
	if observedLink == nil {
		return false
	}

	return specLink.Name == observedLink.Name &&
		specLink.URL == observedLink.URL &&
		helpers.IsComparablePtrEqualComparable(specLink.ID, observedLink.ID)
}
