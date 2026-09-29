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

package iam

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/pkg/errors"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache"
)

const (
	// SubjectTypeGroup identifies a group principal in an association
	// external name.
	SubjectTypeGroup = "group"
	// SubjectTypeUser identifies a user principal in an association
	// external name.
	SubjectTypeUser = "user"

	// selectedPageSize is the page size of the Quality Gate and Quality
	// Profile search_groups and search_users endpoints, which cap it at 100.
	selectedPageSize int64 = 100
)

// errSelectionTargetNotFound is returned by the page fetchers of
// selectedIndex when the Quality Gate or Quality Profile does not exist.
var errSelectionTargetNotFound = errors.New("quality gate or quality profile not found")

// selectedIndex returns the groups or users selected on one Quality Gate or
// Quality Profile (that is, allowed to edit it), indexed by keyOf, and
// whether the target exists.
//
// The selected set is read without name filter, so one fetch per target
// and TTL serves every association resource on that target. It is usually
// small: only the principals granted edit rights are listed. A missing
// target is reported as targetFound false, and is never cached.
//
// The returned map is shared with other callers and must not be mutated.
func selectedIndex[T any](ctx context.Context, scoped cache.Scoped, key cache.Key, fetchPage common.PageFetcher[T], keyOf func(T) string) (index map[string]T, targetFound bool, err error) {
	index, err = cache.Fetch(ctx, scoped.Store, key, func(ctx context.Context) (map[string]T, error) {
		items, fetchErr := common.FetchAllPages(ctx, selectedPageSize, fetchPage)
		if fetchErr != nil {
			return nil, fetchErr
		}

		selected := make(map[string]T, len(items))
		for _, item := range items {
			selected[keyOf(item)] = item
		}

		return selected, nil
	})

	switch {
	case errors.Is(err, errSelectionTargetNotFound):
		return nil, false, nil
	case err != nil:
		return nil, false, err
	default:
		return index, true, nil
	}
}

// selectedPages adapts a search_groups or search_users SDK call into the
// PageFetcher of selectedIndex: items extracts the entries and paging of a
// response, and a 404 is reported as errSelectionTargetNotFound.
func selectedPages[R, T any](search common.PageSearch[R], items func(result *R) ([]T, sonar.Paging)) common.PageFetcher[T] {
	return common.SearchPages(func(ctx context.Context, page sonar.PaginationArgs) (*R, *http.Response, error) {
		result, resp, err := search(ctx, page)
		if common.IsResponseNotFound(resp) {
			return nil, resp, errSelectionTargetNotFound
		}

		return result, resp, err
	}, items)
}
