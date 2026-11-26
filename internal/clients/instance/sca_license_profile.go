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
	"context"
	"net/http"
	"slices"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// scaLicenseProfileProjectsPageSize is the page size used to list the
// projects assigned to a license profile, the largest the API accepts.
const scaLicenseProfileProjectsPageSize = 500

// SCALicenseProfilesClient is the interface for interacting with the
// SonarQube SCA license profiles V2 API (Enterprise Edition only).
//
//nolint:interfacebloat // This interface wraps the SonarQube SCA license profiles API which has many methods
type SCALicenseProfilesClient interface {
	AssignLicenseProfileProject(ctx context.Context, body *sonar.ScaLicenseProfileAssignmentRequest) (*sonar.ScaLicenseProfile, *http.Response, error)
	CreateLicenseProfile(ctx context.Context, body *sonar.ScaLicenseProfileCreateRequest) (*sonar.ScaLicenseProfile, *http.Response, error)
	DeleteLicenseProfile(ctx context.Context, opt *sonar.ScaLicenseProfileGetOptions) (*http.Response, error)
	DeleteLicenseProfileAssignment(ctx context.Context, opt *sonar.ScaLicenseProfileAssignmentDeleteOptions) (*http.Response, error)
	GetLicenseProfile(ctx context.Context, opt *sonar.ScaLicenseProfileGetOptions) (*sonar.ScaLicenseProfileDetails, *http.Response, error)
	ListLicenseProfileAssignableProjects(ctx context.Context, opt *sonar.ScaLicenseProfileAssignableProjectsOptions) (*sonar.ScaLicenseProfileAssignableProjectsResponse, *http.Response, error)
	ListLicenseProfiles(ctx context.Context, opt *sonar.ScaLicenseProfileListOptions) (*sonar.ScaLicenseProfileIndex, *http.Response, error)
	UpdateLicenseProfile(ctx context.Context, key string, body *sonar.ScaLicenseProfileUpdateRequest) (*sonar.ScaLicenseProfile, *http.Response, error)
	UpdateLicenseProfileCategory(ctx context.Context, opt *sonar.ScaLicenseProfileCategoryOptions, body *sonar.ScaLicenseProfileCategoryUpdateRequest) (*sonar.ScaLicenseProfileCategory, *http.Response, error)
	UpdateLicenseProfileLicense(ctx context.Context, opt *sonar.ScaLicenseProfileLicenseOptions, body *sonar.ScaLicensePolicyLicenseUpdateRequest) (*sonar.ScaLicensePolicyLicense, *http.Response, error)
}

// NewSCALicenseProfilesClient creates a new SCALicenseProfilesClient using
// the provided SonarQube client configuration.
func NewSCALicenseProfilesClient(clientConfig common.Config) SCALicenseProfilesClient {
	newClient := common.NewClient(clientConfig)

	return newClient.V2.Sca
}

// SCALicenseProfileLicenseUpdate is a license policy update to apply to a
// license profile. The API identifies the license by the identifier of its
// policy in the profile, not by its SPDX identifier.
type SCALicenseProfileLicenseUpdate struct {
	// ID is the identifier of the license policy in the license profile.
	ID string
	// SPDXLicenseID is the SPDX identifier of the license.
	SPDXLicenseID string
	// Policy is the desired policy of the license.
	Policy string
}

// GenerateSCALicenseProfileCreateOptions generates the request for creating
// a SonarQube SCA license profile from the desired state.
func GenerateSCALicenseProfileCreateOptions(spec *v1alpha1.SCALicenseProfileParameters) *sonar.ScaLicenseProfileCreateRequest {
	return &sonar.ScaLicenseProfileCreateRequest{
		Name:         spec.Name,
		Organization: spec.Organization,
		Default:      ptr.Deref(spec.Default, false),
	}
}

// GenerateSCALicenseProfileUpdateOptions generates the request for updating
// the name and default flag of a SonarQube SCA license profile. The default
// flag is only sent when it is managed.
func GenerateSCALicenseProfileUpdateOptions(spec *v1alpha1.SCALicenseProfileParameters) *sonar.ScaLicenseProfileUpdateRequest {
	return &sonar.ScaLicenseProfileUpdateRequest{
		Name:    spec.Name,
		Default: spec.Default,
	}
}

// FindSCALicenseProfile returns the license profile with the given
// identifier, if any.
func FindSCALicenseProfile(profiles []sonar.ScaLicenseProfile, id string) (*sonar.ScaLicenseProfile, bool) {
	for i := range profiles {
		if profiles[i].Id == id {
			return &profiles[i], true
		}
	}

	return nil, false
}

// FetchSCALicenseProfileProjects returns the sorted keys of the projects
// assigned to the license profile with the given identifier.
func FetchSCALicenseProfileProjects(ctx context.Context, client SCALicenseProfilesClient, profileID string) ([]string, error) {
	fetch := func(ctx context.Context, page sonar.PaginationArgs) ([]string, sonar.Paging, error) {
		result, resp, err := client.ListLicenseProfileAssignableProjects(ctx, &sonar.ScaLicenseProfileAssignableProjectsOptions{ //nolint:bodyclose // closed via helpers.CloseBody
			PaginationParamsV2: sonar.PaginationParamsV2{
				PageIndex: int32(page.Page),     //nolint:gosec // page indexes stay far below the int32 range
				PageSize:  int32(page.PageSize), //nolint:gosec // the page size is a small constant
			},
			AssignedToLicenseProfile: new(true),
			LicenseProfileId:         profileID,
		})
		helpers.CloseBody(resp)

		if err != nil {
			return nil, sonar.Paging{}, err
		}

		if result == nil {
			return nil, sonar.Paging{}, errors.New("received a nil assignable projects result from SonarQube")
		}

		keys := make([]string, 0, len(result.AssignableProjects))
		for _, project := range result.AssignableProjects {
			// The filter is applied by SonarQube; checking it again keeps a
			// server ignoring it from reporting every project as assigned.
			if project.AssignedToLicenseProfile {
				keys = append(keys, project.ProjectKey)
			}
		}

		return keys, sonar.Paging{
			PageIndex: int64(result.Page.PageIndex),
			PageSize:  int64(result.Page.PageSize),
			Total:     int64(result.Page.Total),
		}, nil
	}

	projects, err := common.FetchAllPages(ctx, scaLicenseProfileProjectsPageSize, fetch)
	if err != nil {
		return nil, err
	}

	slices.Sort(projects)

	return projects, nil
}

// GenerateSCALicenseProfileObservation converts the SonarQube license
// profile details and assigned projects to a SCALicenseProfileObservation.
// Only the licenses listed in specLicenses are reported, since a profile
// holds a policy for every known license.
func GenerateSCALicenseProfileObservation(details *sonar.ScaLicenseProfileDetails, specLicenses []v1alpha1.SCALicenseProfileLicensePolicy, projects []string) v1alpha1.SCALicenseProfileObservation {
	if details == nil {
		return v1alpha1.SCALicenseProfileObservation{}
	}

	var categories []v1alpha1.SCALicenseProfileCategoryObservation
	for _, category := range details.Categories {
		categories = append(categories, v1alpha1.SCALicenseProfileCategoryObservation{
			Category: category.Key,
			Policy:   category.Policy,
		})
	}

	wanted := make(map[string]struct{}, len(specLicenses))
	for _, license := range specLicenses {
		wanted[license.SPDXLicenseID] = struct{}{}
	}

	var licenses []v1alpha1.SCALicenseProfileLicenseObservation

	for _, license := range details.Licenses {
		if _, found := wanted[license.SpdxLicenseId]; !found {
			continue
		}

		licenses = append(licenses, v1alpha1.SCALicenseProfileLicenseObservation{
			ID:            license.Id,
			SPDXLicenseID: license.SpdxLicenseId,
			Name:          license.Name,
			Category:      license.Category,
			Policy:        license.Policy,
		})
	}

	if len(projects) == 0 {
		projects = nil
	}

	return v1alpha1.SCALicenseProfileObservation{
		ID:         details.Profile.Id,
		Name:       details.Profile.Name,
		Default:    details.Profile.Default,
		UpdatedAt:  details.Profile.UpdatedAt,
		Categories: categories,
		Licenses:   licenses,
		Projects:   projects,
	}
}

// LateInitializeSCALicenseProfile fills the unset Default field of the spec
// from the observation.
func LateInitializeSCALicenseProfile(spec *v1alpha1.SCALicenseProfileParameters, observation *v1alpha1.SCALicenseProfileObservation) {
	if spec == nil || observation == nil {
		return
	}

	helpers.AssignIfNil(&spec.Default, observation.Default)
}

// IsSCALicenseProfileLateInitialized returns true if the spec changed
// during late initialization.
func IsSCALicenseProfileLateInitialized(former, current *v1alpha1.SCALicenseProfileParameters) bool {
	if former == nil || current == nil {
		return false
	}

	return !helpers.IsComparablePtrEqualComparablePtr(former.Default, current.Default)
}

// IsSCALicenseProfileSettingsUpToDate returns true when the name and the
// managed default flag match the observed state.
func IsSCALicenseProfileSettingsUpToDate(spec *v1alpha1.SCALicenseProfileParameters, observation *v1alpha1.SCALicenseProfileObservation) bool {
	return spec.Name == observation.Name && (spec.Default == nil || *spec.Default == observation.Default)
}

// IsSCALicenseProfileUpToDate returns true when the license profile spec
// matches the observed state.
func IsSCALicenseProfileUpToDate(spec *v1alpha1.SCALicenseProfileParameters, observation *v1alpha1.SCALicenseProfileObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	if !IsSCALicenseProfileSettingsUpToDate(spec, observation) {
		return false
	}

	if len(SCALicenseProfileCategoriesToUpdate(spec.Categories, observation.Categories)) > 0 {
		return false
	}

	updates, missing := SCALicenseProfileLicensesToUpdate(spec.Licenses, observation.Licenses)
	if len(updates) > 0 || len(missing) > 0 {
		return false
	}

	return helpers.AreStringSlicesEqualDeDuped(spec.Projects, observation.Projects)
}

// SCALicenseProfileCategoriesToUpdate returns the desired category policies
// that differ from the observed ones. A category that is not observed is
// returned too, so that its policy gets set.
func SCALicenseProfileCategoriesToUpdate(spec []v1alpha1.SCALicenseProfileCategoryPolicy, observation []v1alpha1.SCALicenseProfileCategoryObservation) []v1alpha1.SCALicenseProfileCategoryPolicy {
	observed := make(map[string]string, len(observation))
	for _, category := range observation {
		observed[category.Category] = category.Policy
	}

	var updates []v1alpha1.SCALicenseProfileCategoryPolicy

	for _, category := range spec {
		policy, found := observed[category.Category]
		if !found || policy != category.Policy {
			updates = append(updates, category)
		}
	}

	return updates
}

// SCALicenseProfileLicensesToUpdate returns the desired license policies
// that differ from the observed ones, and the SPDX identifiers of the
// desired licenses that the license profile does not know.
func SCALicenseProfileLicensesToUpdate(spec []v1alpha1.SCALicenseProfileLicensePolicy, observation []v1alpha1.SCALicenseProfileLicenseObservation) (updates []SCALicenseProfileLicenseUpdate, missing []string) {
	observed := make(map[string]*v1alpha1.SCALicenseProfileLicenseObservation, len(observation))
	for i := range observation {
		observed[observation[i].SPDXLicenseID] = &observation[i]
	}

	for _, license := range spec {
		current, found := observed[license.SPDXLicenseID]
		if !found {
			missing = append(missing, license.SPDXLicenseID)

			continue
		}

		if current.Policy != license.Policy {
			updates = append(updates, SCALicenseProfileLicenseUpdate{
				ID:            current.ID,
				SPDXLicenseID: license.SPDXLicenseID,
				Policy:        license.Policy,
			})
		}
	}

	return updates, missing
}

// SCALicenseProfileProjectsToAssign returns the sorted keys of the desired
// projects that are not yet assigned to the license profile.
func SCALicenseProfileProjectsToAssign(spec, observation []string) []string {
	return helpers.StringSetDifference(spec, observation)
}

// SCALicenseProfileProjectsToUnassign returns the sorted keys of the
// projects assigned to the license profile that are not in the desired
// state.
func SCALicenseProfileProjectsToUnassign(spec, observation []string) []string {
	return helpers.StringSetDifference(observation, spec)
}
