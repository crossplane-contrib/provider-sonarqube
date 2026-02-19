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

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// ProjectTagsClient is the interface for interacting with SonarQube Project Tags API
// It handles all the operations related to Project Tags in SonarQube, such as searching and setting project tags.
type ProjectTagsClient interface {
	Search(opt *sonar.ProjectTagsSearchOption) (*sonar.ProjectTagsSearch, *http.Response, error)
	Set(opt *sonar.ProjectTagsSetOption) (*http.Response, error)
}

// NewProjectTagsClient creates a new ProjectTagsClient with the provided SonarQube client configuration.
func NewProjectTagsClient(clientConfig common.Config) ProjectTagsClient {
	newClient := common.NewClient(clientConfig)

	return newClient.ProjectTags
}

// GenerateProjectTagsSetOptions generates the options for setting project tags based on the provided project key and list of tags.
func GenerateProjectTagsSetOptions(projectKey string, tags []string) *sonar.ProjectTagsSetOption {
	return &sonar.ProjectTagsSetOption{
		Project: projectKey,
		Tags:    tags,
	}
}
