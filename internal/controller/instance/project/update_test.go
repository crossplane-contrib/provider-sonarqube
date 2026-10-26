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

package project

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/fake"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// TestUpdateProjectLinksBoundsConcurrency tests that a long links list does
// not open one concurrent SonarQube request per item.
func TestUpdateProjectLinksBoundsConcurrency(t *testing.T) {
	t.Parallel()

	const itemCount = 40

	project := &v1alpha1.Project{}
	project.Status.AtProvider.Links = make(map[string]v1alpha1.ProjectLinkObservation, 2*itemCount)

	for i := range itemCount {
		// Spec links whose observed URL differs: deleted, then recreated.
		name := fmt.Sprintf("changed-%d", i)
		project.Spec.ForProvider.Links = append(project.Spec.ForProvider.Links, v1alpha1.ProjectLinkParameters{Name: name, URL: "https://new.example.com"})
		project.Status.AtProvider.Links[name] = v1alpha1.ProjectLinkObservation{ID: "id-" + name, Name: name, URL: "https://old.example.com"}

		// Observed links missing from the spec: deleted.
		stale := fmt.Sprintf("stale-%d", i)
		project.Status.AtProvider.Links[stale] = v1alpha1.ProjectLinkObservation{ID: "id-" + stale, Name: stale, URL: "https://old.example.com"}
	}

	tracker := &fake.InFlightTracker{}
	ext := &external{projectLinksClient: &fake.MockProjectLinksClient{
		CreateFn: func(_ *sonar.ProjectLinksCreateOptions) (*sonar.ProjectLinksCreate, *http.Response, error) {
			tracker.Track()

			return &sonar.ProjectLinksCreate{}, nil, nil
		},
		DeleteFn: func(_ *sonar.ProjectLinksDeleteOptions) (*http.Response, error) {
			tracker.Track()

			return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
		},
	}}

	errChan := make(chan error)
	done := make(chan []error)

	go func() {
		var errs []error
		for err := range errChan {
			errs = append(errs, err)
		}

		done <- errs
	}()

	ext.updateProjectLinks(context.Background(), project, "project-id", errChan)
	close(errChan)

	if errs := <-done; len(errs) != 0 {
		t.Fatalf("updateProjectLinks() unexpected errors: %v", errs)
	}

	// Each changed link is deleted and recreated, each stale link deleted.
	if got := tracker.Total(); got != 3*itemCount {
		t.Errorf("updateProjectLinks() made %d requests, want %d", got, 3*itemCount)
	}

	if got := tracker.Peak(); got > helpers.MaxConcurrentRequests {
		t.Errorf("updateProjectLinks() peak concurrency = %d, want at most %d", got, helpers.MaxConcurrentRequests)
	}
}
