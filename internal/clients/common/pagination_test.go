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
	"errors"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"
)

// fakePages serves items page by page like a SonarQube search endpoint, and
// records the requested pages.
type fakePages struct {
	items         []int
	failOnPage    int64
	zeroPageSize  bool
	requested     []sonar.PaginationArgs
	cancelOnPage  int64
	cancelContext context.CancelFunc
}

// fetch implements PageFetcher.
func (f *fakePages) fetch(_ context.Context, page sonar.PaginationArgs) ([]int, sonar.Paging, error) {
	f.requested = append(f.requested, page)

	if page.Page == f.cancelOnPage {
		f.cancelContext()
	}

	if page.Page == f.failOnPage {
		return nil, sonar.Paging{}, errors.New("boom")
	}

	paging := sonar.Paging{PageIndex: page.Page, PageSize: page.PageSize, Total: int64(len(f.items))}
	if f.zeroPageSize {
		paging.PageSize = 0
	}

	start := min((page.Page-1)*page.PageSize, int64(len(f.items)))
	end := min(start+page.PageSize, int64(len(f.items)))

	return f.items[start:end], paging, nil
}

// sequence returns the integers 0..n-1.
func sequence(n int) []int {
	items := make([]int, n)
	for i := range items {
		items[i] = i
	}

	return items
}

// TestFetchAllPages tests fetching and concatenating every page of a
// dataset.
func TestFetchAllPages(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		items        []int
		pageSize     int64
		failOnPage   int64
		zeroPageSize bool
		want         []int
		wantPages    int
		wantErr      bool
	}{
		"NoItems": {
			items: nil, pageSize: 10, want: []int{}, wantPages: 1,
		},
		"SinglePage": {
			items: sequence(3), pageSize: 10, want: sequence(3), wantPages: 1,
		},
		"ExactlyFullPages": {
			items: sequence(20), pageSize: 10, want: sequence(20), wantPages: 2,
		},
		"TotalNotMultipleOfPageSize": {
			items: sequence(25), pageSize: 10, want: sequence(25), wantPages: 3,
		},
		"ErrorOnPageK": {
			items: sequence(25), pageSize: 10, failOnPage: 2, wantPages: 2, wantErr: true,
		},
		"ZeroPageSizeInResponse": {
			items: sequence(25), pageSize: 10, zeroPageSize: true, wantPages: 1, wantErr: true,
		},
		"ZeroPageSizeRequested": {
			items: sequence(25), pageSize: 0, wantPages: 0, wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pages := &fakePages{items: tc.items, failOnPage: tc.failOnPage, zeroPageSize: tc.zeroPageSize}

			got, err := FetchAllPages(context.Background(), tc.pageSize, pages.fetch)
			if (err != nil) != tc.wantErr {
				t.Fatalf("FetchAllPages() error = %v, wantErr %v", err, tc.wantErr)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FetchAllPages() -want +got:\n%s", diff)
			}

			if len(pages.requested) != tc.wantPages {
				t.Errorf("FetchAllPages() requested %d pages, want %d", len(pages.requested), tc.wantPages)
			}

			for index, page := range pages.requested {
				want := sonar.PaginationArgs{Page: int64(index + 1), PageSize: tc.pageSize}
				if page != want {
					t.Errorf("request %d = %+v, want %+v", index, page, want)
				}
			}
		})
	}
}

// TestFetchAllPagesStopsOnCancelledContext tests that no page is fetched
// once the context is cancelled.
func TestFetchAllPagesStopsOnCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pages := &fakePages{items: sequence(50), cancelOnPage: 2, cancelContext: cancel}

	_, err := FetchAllPages(ctx, 10, pages.fetch)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("FetchAllPages() error = %v, want %v", err, context.Canceled)
	}

	if len(pages.requested) != 2 {
		t.Errorf("FetchAllPages() requested %d pages, want 2", len(pages.requested))
	}
}

// TestFindInPages tests finding an item across pages with an early exit.
func TestFindInPages(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		items      []int
		failOnPage int64
		target     int
		wantFound  bool
		wantPages  int
		wantErr    bool
	}{
		"FoundOnFirstPage": {
			items: sequence(25), target: 3, wantFound: true, wantPages: 1,
		},
		"FoundOnLastPage": {
			items: sequence(25), target: 24, wantFound: true, wantPages: 3,
		},
		"NotFound": {
			items: sequence(25), target: 100, wantFound: false, wantPages: 3,
		},
		"Empty": {
			items: nil, target: 0, wantFound: false, wantPages: 1,
		},
		"ErrorBeforeMatch": {
			items: sequence(25), failOnPage: 2, target: 24, wantPages: 2, wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pages := &fakePages{items: tc.items, failOnPage: tc.failOnPage}

			got, found, err := FindInPages(context.Background(), 10, pages.fetch, func(item int) bool {
				return item == tc.target
			})
			if (err != nil) != tc.wantErr {
				t.Fatalf("FindInPages() error = %v, wantErr %v", err, tc.wantErr)
			}

			if found != tc.wantFound {
				t.Errorf("FindInPages() found = %v, want %v", found, tc.wantFound)
			}

			if found && got != tc.target {
				t.Errorf("FindInPages() = %d, want %d", got, tc.target)
			}

			if len(pages.requested) != tc.wantPages {
				t.Errorf("FindInPages() requested %d pages, want %d", len(pages.requested), tc.wantPages)
			}
		})
	}
}
