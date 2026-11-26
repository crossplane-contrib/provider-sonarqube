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
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// DashboardResourceTypeProject is the resource type of a project
	// dashboard.
	DashboardResourceTypeProject = "project"
	// DashboardResourceTypePortfolio is the resource type of a portfolio
	// dashboard.
	DashboardResourceTypePortfolio = "portfolio"

	// dashboardsPageSize is the page size used to list the dashboards of a
	// project or portfolio.
	dashboardsPageSize = 500

	// layoutIDMember is the object member SonarQube may add to a layout to
	// identify its elements.
	layoutIDMember = "id"
)

// DashboardsClient is the interface for interacting with the SonarQube V2
// custom dashboards API (Enterprise Edition only).
type DashboardsClient interface {
	Create(ctx context.Context, opt *sonar.DashboardsCreateOptions) (*sonar.DashboardsDashboardResponse, *http.Response, error)
	Delete(ctx context.Context, dashboardID string) (*http.Response, error)
	Get(ctx context.Context, dashboardID string) (*sonar.DashboardsDashboardResponse, *http.Response, error)
	List(ctx context.Context, opt *sonar.DashboardsListOptions) (*sonar.DashboardsDashboardsResponse, *http.Response, error)
	Update(ctx context.Context, dashboardID string, opt *sonar.DashboardsUpdateOptions) (*sonar.DashboardsDashboardResponse, *http.Response, error)
}

// ComponentNavigationClient is the interface for looking up the identifier of
// a SonarQube component (project, portfolio...) from its key.
type ComponentNavigationClient interface {
	Component(ctx context.Context, opt *sonar.NavigationComponentOptions) (*sonar.NavigationComponent, *http.Response, error)
}

// NewDashboardsClient creates a new DashboardsClient with the provided
// SonarQube client configuration.
func NewDashboardsClient(clientConfig common.Config) DashboardsClient {
	return common.NewClient(clientConfig).V2.Dashboards
}

// NewComponentNavigationClient creates a new ComponentNavigationClient with
// the provided SonarQube client configuration.
func NewComponentNavigationClient(clientConfig common.Config) ComponentNavigationClient {
	return common.NewClient(clientConfig).Navigation
}

// DashboardTarget returns the resource type and component key of the project
// or portfolio the dashboard belongs to. The key is empty when the spec holds
// none.
func DashboardTarget(spec *v1alpha1.DashboardParameters) (resourceType, key string) {
	if projectKey := ptr.Deref(spec.ProjectKey, ""); projectKey != "" {
		return DashboardResourceTypeProject, projectKey
	}

	if portfolioKey := ptr.Deref(spec.PortfolioKey, ""); portfolioKey != "" {
		return DashboardResourceTypePortfolio, portfolioKey
	}

	return "", ""
}

// GenerateDashboardCreateOptions generates the options for creating a
// dashboard of the component with the given type and identifier.
func GenerateDashboardCreateOptions(spec *v1alpha1.DashboardParameters, resourceType, resourceID string) *sonar.DashboardsCreateOptions {
	return &sonar.DashboardsCreateOptions{
		Name:         spec.Name,
		Description:  ptr.Deref(spec.Description, ""),
		Layout:       spec.Layout,
		ResourceType: resourceType,
		ResourceId:   resourceID,
	}
}

// GenerateDashboardUpdateOptions generates the options for updating a
// dashboard. A nil description leaves the current one untouched.
func GenerateDashboardUpdateOptions(spec *v1alpha1.DashboardParameters) *sonar.DashboardsUpdateOptions {
	return &sonar.DashboardsUpdateOptions{
		Name:        &spec.Name,
		Description: spec.Description,
		Layout:      &spec.Layout,
	}
}

// DashboardsFetcher returns a PageFetcher over the dashboards of the
// component with the given type and identifier.
func DashboardsFetcher(client DashboardsClient, resourceType, resourceID string) common.PageFetcher[sonar.DashboardsDashboardItem] {
	return func(ctx context.Context, page sonar.PaginationArgs) ([]sonar.DashboardsDashboardItem, sonar.Paging, error) {
		search := func(ctx context.Context, page sonar.PaginationArgs) (*sonar.DashboardsDashboardsResponse, *http.Response, error) {
			return client.List(ctx, &sonar.DashboardsListOptions{
				ResourceType: resourceType,
				ResourceId:   resourceID,
				PageIndex:    int32(page.Page),     //nolint:gosec // bounded by the number of dashboards
				PageSize:     int32(page.PageSize), //nolint:gosec // always dashboardsPageSize
			})
		}

		return common.SearchPages(search, func(result *sonar.DashboardsDashboardsResponse) ([]sonar.DashboardsDashboardItem, sonar.Paging) {
			return result.Dashboards, sonar.Paging{
				PageIndex: int64(result.Page.PageIndex),
				PageSize:  int64(result.Page.PageSize),
				Total:     int64(result.Page.Total),
			}
		})(ctx, page)
	}
}

// FindDashboardByName returns the dashboard named name among those of the
// component with the given type and identifier. found is false when there is
// none.
func FindDashboardByName(ctx context.Context, client DashboardsClient, resourceType, resourceID, name string) (item sonar.DashboardsDashboardItem, found bool, err error) {
	return common.FindInPages(ctx, dashboardsPageSize, DashboardsFetcher(client, resourceType, resourceID),
		func(dashboard sonar.DashboardsDashboardItem) bool {
			return dashboard.Name == name
		})
}

// GenerateDashboardObservation generates the observation of a dashboard.
func GenerateDashboardObservation(dashboard *sonar.DashboardsDashboardResponse) v1alpha1.DashboardObservation {
	return v1alpha1.DashboardObservation{
		ID:           dashboard.Id,
		Name:         dashboard.Name,
		Description:  dashboard.Description,
		Layout:       dashboard.Layout,
		ResourceType: dashboard.ResourceType,
		ResourceID:   dashboard.ResourceId,
		CreatedAt:    dashboard.CreatedAt,
		UpdatedAt:    dashboard.UpdatedAt,
	}
}

// LateInitializeDashboard fills the unset optional fields of the spec from
// the observation.
func LateInitializeDashboard(spec *v1alpha1.DashboardParameters, observation *v1alpha1.DashboardObservation) {
	if spec == nil || observation == nil || observation.Description == "" {
		return
	}

	helpers.AssignIfNil(&spec.Description, observation.Description)
}

// IsDashboardLateInitialized returns true when the spec changed as a result
// of late initialization.
func IsDashboardLateInitialized(former, current *v1alpha1.DashboardParameters) bool {
	if former == nil || current == nil {
		return false
	}

	return !cmp.Equal(former, current)
}

// IsDashboardUpToDate returns true when the dashboard spec matches the
// observed state. The target is immutable and checked separately.
func IsDashboardUpToDate(spec *v1alpha1.DashboardParameters, observation *v1alpha1.DashboardObservation) bool {
	if spec == nil {
		return true
	}

	if observation == nil {
		return false
	}

	return spec.Name == observation.Name &&
		(spec.Description == nil || *spec.Description == observation.Description) &&
		AreDashboardLayoutsEquivalent(spec.Layout, observation.Layout)
}

// AreDashboardLayoutsEquivalent returns true when the observed layout holds
// the desired one. JSON layouts are compared by content: formatting and key
// order are ignored, as are "id" members that only the observed layout has,
// which SonarQube may have generated. Any other layout is compared verbatim.
func AreDashboardLayoutsEquivalent(desired, observed string) bool {
	if desired == observed {
		return true
	}

	var desiredValue, observedValue any

	if json.Unmarshal([]byte(desired), &desiredValue) != nil || json.Unmarshal([]byte(observed), &observedValue) != nil {
		return false
	}

	return layoutMatches(desiredValue, observedValue)
}

// layoutMatches compares two decoded JSON values, ignoring the "id" members
// of observed objects that the desired ones do not set.
func layoutMatches(desired, observed any) bool {
	switch desiredValue := desired.(type) {
	case map[string]any:
		observedValue, ok := observed.(map[string]any)

		return ok && layoutObjectMatches(desiredValue, observedValue)
	case []any:
		observedValue, ok := observed.([]any)
		if !ok || len(desiredValue) != len(observedValue) {
			return false
		}

		for index := range desiredValue {
			if !layoutMatches(desiredValue[index], observedValue[index]) {
				return false
			}
		}

		return true
	default:
		return reflect.DeepEqual(desired, observed)
	}
}

// layoutObjectMatches is layoutMatches for JSON objects.
func layoutObjectMatches(desired, observed map[string]any) bool {
	for key, desiredMember := range desired {
		observedMember, ok := observed[key]
		if !ok || !layoutMatches(desiredMember, observedMember) {
			return false
		}
	}

	for key := range observed {
		if _, ok := desired[key]; !ok && key != layoutIDMember {
			return false
		}
	}

	return true
}
