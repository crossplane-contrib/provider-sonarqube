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

package common

import (
	"context"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/pkg/errors"
)

// PageFetcher fetches one page of a paginated SonarQube endpoint and returns
// its items along with the paging information of the response.
type PageFetcher[T any] func(ctx context.Context, page sonar.PaginationArgs) (items []T, paging sonar.Paging, err error)

// FetchAllPages calls fetch for pages 1..N, until the paging of the
// responses says the dataset is exhausted, and concatenates their items.
func FetchAllPages[T any](ctx context.Context, pageSize int64, fetch PageFetcher[T]) ([]T, error) {
	var all []T

	err := walkPages(ctx, pageSize, fetch, func(items []T) bool {
		all = append(all, items...)

		return false
	})
	if err != nil {
		return nil, err
	}

	if all == nil {
		all = []T{}
	}

	return all, nil
}

// FindInPages calls fetch for pages 1..N and returns the first item for
// which match returns true, without fetching the following pages. found is
// false when no item of any page matches.
func FindInPages[T any](ctx context.Context, pageSize int64, fetch PageFetcher[T], match func(T) bool) (item T, found bool, err error) {
	err = walkPages(ctx, pageSize, fetch, func(items []T) bool {
		for index := range items {
			if match(items[index]) {
				item = items[index]
				found = true

				return true
			}
		}

		return false
	})
	if err != nil {
		var zero T

		return zero, false, err
	}

	return item, found, nil
}

// walkPages calls fetch for pages 1..N and hands the items of every page to
// visit, until visit returns true or the paging of the responses says the
// dataset is exhausted.
func walkPages[T any](ctx context.Context, pageSize int64, fetch PageFetcher[T], visit func(items []T) (stop bool)) error {
	if pageSize <= 0 {
		return errors.Errorf("page size must be greater than 0, got %d", pageSize)
	}

	for page := int64(1); ; page++ {
		err := ctx.Err()
		if err != nil {
			return errors.Wrapf(err, "cannot fetch page %d", page)
		}

		items, paging, err := fetch(ctx, sonar.PaginationArgs{Page: page, PageSize: pageSize})
		if err != nil {
			return errors.Wrapf(err, "cannot fetch page %d", page)
		}

		if paging.PageSize == 0 {
			return errors.Errorf("received zero PageSize in the paging of page %d from SonarQube", page)
		}

		if visit(items) {
			return nil
		}

		// Rely on the requested page index rather than the returned one, so
		// that a server echoing a wrong index cannot cause an endless loop.
		if paging.Total <= page*paging.PageSize {
			return nil
		}
	}
}
