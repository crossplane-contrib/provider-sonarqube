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
	"slices"
	"sync"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

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

// TestUpdateBranches tests that only the drifted settings of existing branches
// are updated.
func TestUpdateBranches(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec           map[string]*v1alpha1.ProjectBranchParameters
		observation    map[string]v1alpha1.ProjectBranchObservation
		defaultBranch  *string
		apiErr         error
		wantProtection []sonar.ProjectBranchesSetAutomaticDeletionProtectionOptions
		wantPeriods    []sonar.NewCodePeriodsSetOptions
		wantErrs       []string
	}{
		"UpToDate": {
			spec:        map[string]*v1alpha1.ProjectBranchParameters{"release": {ExcludedFromPurge: new(true)}},
			observation: map[string]v1alpha1.ProjectBranchObservation{"release": {ExcludedFromPurge: true}},
		},
		"MissingBranchIsSkipped": {
			spec:        map[string]*v1alpha1.ProjectBranchParameters{"release": {ExcludedFromPurge: new(true)}, "stale": {ExcludedFromPurge: new(true)}},
			observation: map[string]v1alpha1.ProjectBranchObservation{"stale": {}},
			wantProtection: []sonar.ProjectBranchesSetAutomaticDeletionProtectionOptions{
				{Project: "key", Branch: "stale", Value: true},
			},
		},
		"ProtectionOnly": {
			spec:        map[string]*v1alpha1.ProjectBranchParameters{"release": {ExcludedFromPurge: new(false)}},
			observation: map[string]v1alpha1.ProjectBranchObservation{"release": {ExcludedFromPurge: true}},
			wantProtection: []sonar.ProjectBranchesSetAutomaticDeletionProtectionOptions{
				{Project: "key", Branch: "release", Value: false},
			},
		},
		"MainBranchIsNeverUnprotected": {
			spec: map[string]*v1alpha1.ProjectBranchParameters{"main": {Type: new("PREVIOUS_VERSION"), ExcludedFromPurge: new(false)}},
			observation: map[string]v1alpha1.ProjectBranchObservation{
				"main": {IsMain: true, ExcludedFromPurge: true},
			},
			wantPeriods: []sonar.NewCodePeriodsSetOptions{{Project: "key", Branch: "main", Type: "PREVIOUS_VERSION"}},
		},
		"PromotedBranchIsNotUnprotected": {
			spec:          map[string]*v1alpha1.ProjectBranchParameters{"release": {ExcludedFromPurge: new(false)}},
			observation:   map[string]v1alpha1.ProjectBranchObservation{"release": {ExcludedFromPurge: true}},
			defaultBranch: new("release"),
		},
		"PromotedBranchIsProtected": {
			spec:          map[string]*v1alpha1.ProjectBranchParameters{"release": {ExcludedFromPurge: new(true)}},
			observation:   map[string]v1alpha1.ProjectBranchObservation{"release": {}},
			defaultBranch: new("release"),
			wantProtection: []sonar.ProjectBranchesSetAutomaticDeletionProtectionOptions{
				{Project: "key", Branch: "release", Value: true},
			},
		},
		"NewCodePeriodOnly": {
			spec:        map[string]*v1alpha1.ProjectBranchParameters{"release": {Type: new("NUMBER_OF_DAYS"), Value: new("30")}},
			observation: map[string]v1alpha1.ProjectBranchObservation{"release": {ExcludedFromPurge: true}},
			wantPeriods: []sonar.NewCodePeriodsSetOptions{{Project: "key", Branch: "release", Type: "NUMBER_OF_DAYS", Value: "30"}},
		},
		"BothFail": {
			spec:        map[string]*v1alpha1.ProjectBranchParameters{"release": {Type: new("PREVIOUS_VERSION"), ExcludedFromPurge: new(true)}},
			observation: map[string]v1alpha1.ProjectBranchObservation{"release": {}},
			apiErr:      errors.New("boom"),
			wantProtection: []sonar.ProjectBranchesSetAutomaticDeletionProtectionOptions{
				{Project: "key", Branch: "release", Value: true},
			},
			wantPeriods: []sonar.NewCodePeriodsSetOptions{{Project: "key", Branch: "release", Type: "PREVIOUS_VERSION"}},
			wantErrs: []string{
				"cannot update automatic deletion protection for branch release: boom",
				"cannot update new code period for branch release: boom",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			project := &v1alpha1.Project{}
			project.Spec.ForProvider.Branches = tc.spec
			project.Status.AtProvider.Branches = tc.observation
			project.Spec.ForProvider.DefaultBranch = tc.defaultBranch
			project.Status.AtProvider.DefaultBranch = "main"

			var (
				mutex      sync.Mutex
				protection []sonar.ProjectBranchesSetAutomaticDeletionProtectionOptions
				periods    []sonar.NewCodePeriodsSetOptions
			)

			ext := &external{
				projectBranchesClient: &fake.MockProjectBranchesClient{
					SetAutomaticDeletionProtectionFn: func(opt *sonar.ProjectBranchesSetAutomaticDeletionProtectionOptions) (*http.Response, error) {
						mutex.Lock()
						defer mutex.Unlock()

						protection = append(protection, *opt)

						return nil, tc.apiErr
					},
				},
				projectNewCodePeriodsClient: &fake.MockNewCodePeriodsClient{
					SetFn: func(opt *sonar.NewCodePeriodsSetOptions) (*http.Response, error) {
						mutex.Lock()
						defer mutex.Unlock()

						periods = append(periods, *opt)

						return nil, tc.apiErr
					},
				},
			}

			errChan := make(chan error, 2*len(tc.spec))
			ext.updateBranches(context.Background(), project, "key", errChan)
			close(errChan)

			var gotErrs []string
			for err := range errChan {
				gotErrs = append(gotErrs, err.Error())
			}

			slices.Sort(gotErrs)

			if diff := cmp.Diff(tc.wantProtection, protection); diff != "" {
				t.Errorf("protection calls mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantPeriods, periods); diff != "" {
				t.Errorf("new code period calls mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantErrs, gotErrs); diff != "" {
				t.Errorf("errors mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
