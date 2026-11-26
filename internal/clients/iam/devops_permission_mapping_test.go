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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/common/cache/cachetest"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// recordedRequest is a request received by the test SonarQube server.
type recordedRequest struct {
	Method      string
	Path        string
	ContentType string
	Body        string
}

// newMappingsTestClient returns a client of a server answering every request
// with status and response, and the requests the server received.
func newMappingsTestClient(t *testing.T, status int, response string) (DevOpsPermissionMappingsClient, *[]recordedRequest) {
	t.Helper()

	requests := &[]recordedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		*requests = append(*requests, recordedRequest{
			Method:      request.Method,
			Path:        request.URL.EscapedPath(),
			ContentType: request.Header.Get("Content-Type"),
			Body:        strings.TrimSpace(string(body)),
		})

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = io.WriteString(writer, response)
	}))
	t.Cleanup(server.Close)

	return NewDevOpsPermissionMappingsClient(common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    "token",
		BaseURL:  server.URL + "/api",
	}), requests
}

// TestDevOpsPermissionMappingsClientRequests tests that each operation reaches
// the endpoint of its platform, with the role escaped once.
func TestDevOpsPermissionMappingsClientRequests(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	params := &v1alpha1.DevOpsPermissionMappingParameters{Role: "Security Reviewer", Permissions: []string{"user", "scan"}}
	permissions := `{"user":true,"codeViewer":false,"issueAdmin":false,"securityHotspotAdmin":false,"admin":false,"scan":true}`

	cases := map[string]struct {
		call func(DevOpsPermissionMappingsClient, v1alpha1.DevOpsPlatform) error
		want func(platform string) recordedRequest
	}{
		"Search": {
			call: func(c DevOpsPermissionMappingsClient, p v1alpha1.DevOpsPlatform) error {
				_, resp, err := c.Search(ctx, p) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: func(platform string) recordedRequest {
				return recordedRequest{Method: http.MethodGet, Path: "/api/v2/dop-translation/" + platform + "-permission-mappings"}
			},
		},
		"Create": {
			call: func(c DevOpsPermissionMappingsClient, p v1alpha1.DevOpsPlatform) error {
				_, resp, err := c.Create(ctx, p, GenerateCreateDevOpsPermissionMappingOptions(params)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: func(platform string) recordedRequest {
				return recordedRequest{
					Method:      http.MethodPost,
					Path:        "/api/v2/dop-translation/" + platform + "-permission-mappings",
					ContentType: "application/json",
					Body:        `{"role":"Security Reviewer","permissions":` + permissions + "}",
				}
			},
		},
		"Update": {
			call: func(c DevOpsPermissionMappingsClient, p v1alpha1.DevOpsPlatform) error {
				_, resp, err := c.Update(ctx, p, params.Role, GenerateUpdateDevOpsPermissionMappingOptions(params)) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: func(platform string) recordedRequest {
				return recordedRequest{
					Method:      http.MethodPatch,
					Path:        "/api/v2/dop-translation/" + platform + "-permission-mappings/Security%20Reviewer",
					ContentType: "application/merge-patch+json",
					Body:        `{"permissions":` + permissions + "}",
				}
			},
		},
		"Delete": {
			call: func(c DevOpsPermissionMappingsClient, p v1alpha1.DevOpsPlatform) error {
				resp, err := c.Delete(ctx, p, params.Role) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				return err
			},
			want: func(platform string) recordedRequest {
				return recordedRequest{Method: http.MethodDelete, Path: "/api/v2/dop-translation/" + platform + "-permission-mappings/Security%20Reviewer"}
			},
		},
	}

	for name, tc := range cases {
		for _, platform := range []v1alpha1.DevOpsPlatform{v1alpha1.DevOpsPlatformGitHub, v1alpha1.DevOpsPlatformGitLab} {
			t.Run(name+"/"+string(platform), func(t *testing.T) {
				t.Parallel()

				client, requests := newMappingsTestClient(t, http.StatusOK, "{}")

				err := tc.call(client, platform)
				if err != nil {
					t.Fatalf("%s() error = %v", name, err)
				}

				got := *requests
				want := tc.want(string(platform))

				// The content type of requests without a body is irrelevant.
				if len(got) == 1 && want.ContentType == "" {
					got[0].ContentType = ""
				}

				if diff := cmp.Diff([]recordedRequest{want}, got); diff != "" {
					t.Errorf("%s() requests (-want +got):\n%s", name, diff)
				}
			})
		}

		t.Run(name+"/UnsupportedPlatform", func(t *testing.T) {
			t.Parallel()

			client, requests := newMappingsTestClient(t, http.StatusOK, "{}")

			err := tc.call(client, "bitbucket")
			if err == nil || len(*requests) != 0 {
				t.Errorf("%s() = error %v after %d requests, want an error and no request", name, err, len(*requests))
			}
		})
	}
}

// TestDevOpsPermissionMappingsClientErrors tests invalid role names and API
// errors.
func TestDevOpsPermissionMappingsClientErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, requests := newMappingsTestClient(t, http.StatusOK, "{}")

	resp, err := client.Delete(ctx, v1alpha1.DevOpsPlatformGitHub, "") //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err == nil || len(*requests) != 0 {
		t.Errorf("Delete(\"\") = error %v after %d requests, want an error and no request", err, len(*requests))
	}

	client, _ = newMappingsTestClient(t, http.StatusNotFound, "{}")

	_, resp, err = client.Update(ctx, v1alpha1.DevOpsPlatformGitHub, "role", &sonar.DopTranslationPermissionMappingUpdateOptions{}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err == nil || !common.IsResponseNotFound(resp) {
		t.Errorf("Update() = response %v, error %v, want a not found error", resp, err)
	}
}

// TestFindDevOpsPermissionMapping tests the lookup of a role.
func TestFindDevOpsPermissionMapping(t *testing.T) {
	t.Parallel()

	response := `{"permissionMappings":[{"id":"1","role":"read","baseRole":true},{"id":"2","role":"Reviewer"}]}`

	cases := map[string]struct {
		status    int
		role      string
		wantID    string
		wantFound bool
		wantErr   bool
	}{
		"Found":           {status: http.StatusOK, role: "Reviewer", wantID: "2", wantFound: true},
		"CaseSensitive":   {status: http.StatusOK, role: "reviewer"},
		"Missing":         {status: http.StatusOK, role: "write"},
		"SearchFails":     {status: http.StatusForbidden, role: "read", wantErr: true},
		"EditionNotFound": {status: http.StatusNotFound, role: "read", wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client, _ := newMappingsTestClient(t, tc.status, response)

			got, found, err := FindDevOpsPermissionMapping(context.Background(), client, v1alpha1.DevOpsPlatformGitHub, tc.role)
			if (err != nil) != tc.wantErr {
				t.Fatalf("FindDevOpsPermissionMapping() error = %v, wantErr %t", err, tc.wantErr)
			}

			if found != tc.wantFound || got.Id != tc.wantID {
				t.Errorf("FindDevOpsPermissionMapping() = %q, %t, want %q, %t", got.Id, found, tc.wantID, tc.wantFound)
			}
		})
	}
}

// TestGenerateDevOpsPermissionMappingObservation tests the conversion of a
// mapping to its observation.
func TestGenerateDevOpsPermissionMappingObservation(t *testing.T) {
	t.Parallel()

	got := GenerateDevOpsPermissionMappingObservation(sonar.DopTranslationPermissionMapping{
		Id:       "1",
		Role:     "admin",
		BaseRole: true,
		Permissions: sonar.DopTranslationPermissions{
			User: true, CodeViewer: true, IssueAdmin: true, SecurityHotspotAdmin: true, Admin: true, Scan: true,
		},
	})
	want := v1alpha1.DevOpsPermissionMappingObservation{
		ID:          "1",
		Role:        "admin",
		BaseRole:    true,
		Permissions: []string{"user", "codeviewer", "issueadmin", "securityhotspotadmin", "admin", "scan"},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GenerateDevOpsPermissionMappingObservation() (-want +got):\n%s", diff)
	}
}

// TestIsDevOpsPermissionMappingUpToDate tests that the desired permissions are
// exhaustive.
func TestIsDevOpsPermissionMappingUpToDate(t *testing.T) {
	t.Parallel()

	observed := sonar.DopTranslationPermissionMapping{Permissions: sonar.DopTranslationPermissions{User: true, Scan: true}}

	cases := map[string]struct {
		permissions []string
		want        bool
	}{
		"SameSetAnyOrder": {permissions: []string{"scan", "user"}, want: true},
		"Missing":         {permissions: []string{"scan", "user", "admin"}},
		"Extra":           {permissions: []string{"scan"}},
		"Empty":           {permissions: []string{}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			params := &v1alpha1.DevOpsPermissionMappingParameters{Permissions: tc.permissions}
			if got := IsDevOpsPermissionMappingUpToDate(params, observed); got != tc.want {
				t.Errorf("IsDevOpsPermissionMappingUpToDate() = %t, want %t", got, tc.want)
			}
		})
	}
}

// TestCachedDevOpsPermissionMappingsClient tests that Search is cached per
// platform and invalidated by every write.
func TestCachedDevOpsPermissionMappingsClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	inner, requests := newMappingsTestClient(t, http.StatusOK, "{}")

	if got := newCachedDevOpsPermissionMappingsClient(inner, cachetest.Disabled()); got != inner {
		t.Errorf("newCachedDevOpsPermissionMappingsClient() with a noop store = %T, want the raw client", got)
	}

	client := newCachedDevOpsPermissionMappingsClient(inner, cachetest.NewScoped(t))

	search := func() {
		for _, platform := range []v1alpha1.DevOpsPlatform{v1alpha1.DevOpsPlatformGitHub, v1alpha1.DevOpsPlatformGitLab} {
			for range 3 {
				_, resp, err := client.Search(ctx, platform) //nolint:bodyclose // closed via helpers.CloseBody
				helpers.CloseBody(resp)

				if err != nil {
					t.Fatalf("Search() error = %v", err)
				}
			}
		}
	}

	writes := []func() error{
		func() error {
			_, resp, err := client.Create(ctx, v1alpha1.DevOpsPlatformGitHub, &sonar.DopTranslationPermissionMappingCreateOptions{Role: "role"}) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			return err
		},
		func() error {
			_, resp, err := client.Update(ctx, v1alpha1.DevOpsPlatformGitHub, "role", &sonar.DopTranslationPermissionMappingUpdateOptions{}) //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			return err
		},
		func() error {
			resp, err := client.Delete(ctx, v1alpha1.DevOpsPlatformGitHub, "role") //nolint:bodyclose // closed via helpers.CloseBody
			helpers.CloseBody(resp)

			return err
		},
	}

	search()

	if len(*requests) != 2 {
		t.Fatalf("requests after cached searches = %d, want 1 per platform", len(*requests))
	}

	for _, write := range writes {
		before := len(*requests)

		err := write()
		if err != nil {
			t.Fatalf("write error = %v", err)
		}

		search()

		// One write, then one fresh search per platform.
		if got := len(*requests) - before; got != 3 {
			t.Errorf("requests after a write = %d, want 3", got)
		}
	}
}
