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
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// reportServer is a fake SonarQube server recording the governance report
// requests it receives.
type reportServer struct {
	mu       sync.Mutex
	requests []string
	status   int
	body     string
}

// ServeHTTP records the request and answers with the configured response.
func (s *reportServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests = append(s.requests, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s.status)
	_, _ = w.Write([]byte(s.body))
}

// recorded returns the requests received so far.
func (s *reportServer) recorded() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.requests
}

// newReportClient returns a GovernanceReportsClient talking to a fake server
// answering every request with status and body.
func newReportClient(t *testing.T, status int, body string) (GovernanceReportsClient, *reportServer) {
	t.Helper()

	handler := &reportServer{status: status, body: body}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return NewGovernanceReportsClient(common.Config{
		BaseURL:  server.URL + "/api/",
		AuthType: common.PersonalAccessToken,
		Token:    "token",
	}), handler
}

// TestObserveGovernanceReport verifies the status endpoint is only called for
// managed settings, and its response is fully read.
func TestObserveGovernanceReport(t *testing.T) {
	t.Parallel()

	const statusRequest = "GET /api/governance_reports/status?componentKey=my-key"

	cases := map[string]struct {
		spec         *v1alpha1.GovernanceReportParameters
		status       int
		body         string
		want         *v1alpha1.GovernanceReportObservation
		wantRequests []string
		wantErr      string
	}{
		"NilSpecDoesNotCall": {},
		"EmptySpecDoesNotCall": {
			spec: &v1alpha1.GovernanceReportParameters{},
		},
		"ReadsComponentSettings": {
			spec:   &v1alpha1.GovernanceReportParameters{Frequency: new("daily")},
			status: http.StatusOK,
			body:   `{"componentFrequency":"daily","globalFrequency":"monthly","componentRecipients":["b@example.com","a@example.com"],"globalRecipients":["c@example.com"]}`,
			want: &v1alpha1.GovernanceReportObservation{
				Frequency:       "daily",
				GlobalFrequency: "monthly",
				Recipients:      []string{"a@example.com", "b@example.com"},
			},
			wantRequests: []string{statusRequest},
		},
		"Error": {
			spec:         &v1alpha1.GovernanceReportParameters{Recipients: &[]string{}},
			status:       http.StatusBadRequest,
			body:         `{"errors":[{"msg":"boom"}]}`,
			wantRequests: []string{statusRequest},
			wantErr:      errGetReportStatus,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client, server := newReportClient(t, tc.status, tc.body)

			got, err := ObserveGovernanceReport(context.Background(), client, "my-key", tc.spec)
			assertReportError(t, err, tc.wantErr)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ObserveGovernanceReport() -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantRequests, server.recorded()); diff != "" {
				t.Errorf("requests -want +got:\n%s", diff)
			}
		})
	}
}

// assertReportError fails the test unless err contains want, or is nil when
// want is empty.
func assertReportError(t *testing.T, err error, want string) {
	t.Helper()

	if want == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		return
	}

	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want to contain %q", err, want)
	}
}

// TestGenerateGovernanceReportObservationNil verifies a nil status yields no
// observation.
func TestGenerateGovernanceReportObservationNil(t *testing.T) {
	t.Parallel()

	if got := GenerateGovernanceReportObservation(nil); got != nil {
		t.Errorf("GenerateGovernanceReportObservation(nil) = %v, want nil", got)
	}
}

// TestIsGovernanceReportUpToDate tests the drift detection of report settings,
// directly and through both kinds sharing it.
func TestIsGovernanceReportUpToDate(t *testing.T) {
	t.Parallel()

	observed := &v1alpha1.GovernanceReportObservation{
		Frequency:  "weekly",
		Recipients: []string{"a@example.com", "b@example.com"},
	}

	cases := map[string]struct {
		spec        *v1alpha1.GovernanceReportParameters
		observation *v1alpha1.GovernanceReportObservation
		want        bool
	}{
		"Unmanaged":         {want: true},
		"UnmanagedFields":   {spec: &v1alpha1.GovernanceReportParameters{}, observation: observed, want: true},
		"MissingFrequency":  {spec: &v1alpha1.GovernanceReportParameters{Frequency: new("weekly")}},
		"MissingRecipients": {spec: &v1alpha1.GovernanceReportParameters{Recipients: &[]string{}}},
		"SameFrequency": {
			spec:        &v1alpha1.GovernanceReportParameters{Frequency: new("weekly")},
			observation: observed,
			want:        true,
		},
		"DifferentFrequency": {
			spec:        &v1alpha1.GovernanceReportParameters{Frequency: new("daily")},
			observation: observed,
		},
		"InheritedFrequency": {
			spec:        &v1alpha1.GovernanceReportParameters{Frequency: new("monthly")},
			observation: &v1alpha1.GovernanceReportObservation{GlobalFrequency: "monthly"},
		},
		"ReorderedRecipients": {
			spec:        &v1alpha1.GovernanceReportParameters{Recipients: &[]string{"b@example.com", "a@example.com"}},
			observation: observed,
			want:        true,
		},
		"DifferentRecipients": {
			spec:        &v1alpha1.GovernanceReportParameters{Recipients: &[]string{"a@example.com"}},
			observation: observed,
		},
		"RecipientsToClear": {
			spec:        &v1alpha1.GovernanceReportParameters{Recipients: &[]string{}},
			observation: observed,
		},
		"RecipientsAreCleared": {
			spec:        &v1alpha1.GovernanceReportParameters{Recipients: &[]string{}},
			observation: &v1alpha1.GovernanceReportObservation{},
			want:        true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsGovernanceReportUpToDate(tc.spec, tc.observation); got != tc.want {
				t.Errorf("IsGovernanceReportUpToDate() = %v, want %v", got, tc.want)
			}

			application := IsApplicationUpToDate(
				&v1alpha1.ApplicationParameters{Report: tc.spec},
				&v1alpha1.ApplicationObservation{Report: tc.observation})
			if application != tc.want {
				t.Errorf("IsApplicationUpToDate() = %v, want %v", application, tc.want)
			}

			portfolio := IsPortfolioUpToDate(
				&v1alpha1.PortfolioParameters{Report: tc.spec},
				&v1alpha1.PortfolioObservation{Report: tc.observation})
			if portfolio != tc.want {
				t.Errorf("IsPortfolioUpToDate() = %v, want %v", portfolio, tc.want)
			}
		})
	}
}

// TestUpdateGovernanceReport verifies only drifted settings are sent,
// including an empty recipient list.
func TestUpdateGovernanceReport(t *testing.T) {
	t.Parallel()

	const (
		frequencyRequest = "POST /api/governance_reports/update_frequency?componentKey=my-key&frequency=daily"
		clearRequest     = "POST /api/governance_reports/update_recipients?componentKey=my-key&recipients="
		setRequest       = clearRequest + "a%40example.com%2Cb%40example.com"
	)

	observed := &v1alpha1.GovernanceReportObservation{Frequency: "weekly", Recipients: []string{"a@example.com"}}

	cases := map[string]struct {
		spec         *v1alpha1.GovernanceReportParameters
		status       int
		wantRequests []string
		wantErr      string
	}{
		"Unmanaged": {status: http.StatusNoContent},
		"UpToDate": {
			spec:   &v1alpha1.GovernanceReportParameters{Frequency: new("weekly"), Recipients: &[]string{"a@example.com"}},
			status: http.StatusNoContent,
		},
		"SetsBoth": {
			spec:         &v1alpha1.GovernanceReportParameters{Frequency: new("daily"), Recipients: &[]string{"a@example.com", "b@example.com"}},
			status:       http.StatusNoContent,
			wantRequests: []string{frequencyRequest, setRequest},
		},
		"ClearsRecipients": {
			spec:         &v1alpha1.GovernanceReportParameters{Recipients: &[]string{}},
			status:       http.StatusNoContent,
			wantRequests: []string{clearRequest},
		},
		"FrequencyError": {
			spec:         &v1alpha1.GovernanceReportParameters{Frequency: new("daily"), Recipients: &[]string{}},
			status:       http.StatusBadRequest,
			wantRequests: []string{frequencyRequest},
			wantErr:      errUpdateReportFrequency,
		},
		"RecipientsError": {
			spec:         &v1alpha1.GovernanceReportParameters{Recipients: &[]string{}},
			status:       http.StatusBadRequest,
			wantRequests: []string{clearRequest},
			wantErr:      errUpdateReportRecipients,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client, server := newReportClient(t, tc.status, "")

			err := UpdateGovernanceReport(context.Background(), client, "my-key", tc.spec, observed)
			assertReportError(t, err, tc.wantErr)

			if diff := cmp.Diff(tc.wantRequests, server.recorded()); diff != "" {
				t.Errorf("requests -want +got:\n%s", diff)
			}
		})
	}
}

// TestGovernanceReportsClientUpdateRecipientsNilOptions verifies nil options
// are rejected without sending a request.
func TestGovernanceReportsClientUpdateRecipientsNilOptions(t *testing.T) {
	t.Parallel()

	client, server := newReportClient(t, http.StatusNoContent, "")

	resp, err := client.UpdateRecipients(context.Background(), nil) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err == nil {
		t.Error("UpdateRecipients(nil) expected an error")
	}

	if got := server.recorded(); len(got) != 0 {
		t.Errorf("UpdateRecipients(nil) sent %v", got)
	}
}

// TestGovernanceReportParametersSerialization verifies an empty recipient list
// survives serialization, so that it is not mistaken for an unmanaged one.
func TestGovernanceReportParametersSerialization(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(&v1alpha1.GovernanceReportParameters{Recipients: &[]string{}})
	if err != nil {
		t.Fatalf("Marshal() unexpected error: %v", err)
	}

	got := &v1alpha1.GovernanceReportParameters{}

	err = json.Unmarshal(data, got)
	if err != nil {
		t.Fatalf("Unmarshal() unexpected error: %v", err)
	}

	if got.Recipients == nil || len(*got.Recipients) != 0 {
		t.Errorf("round trip of %s: recipients = %v, want an empty list", data, got.Recipients)
	}
}
