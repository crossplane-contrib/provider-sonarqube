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

package helpers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// testSecretValue is a dummy secret that must never appear in redacted errors.
const testSecretValue = "DUMMYSECRET0001"

// TestRedactURLError tests that request parameters are removed from
// [url.Error].
func TestRedactURLError(t *testing.T) {
	t.Parallel()

	plainErr := errors.New("plain error")

	tests := map[string]struct {
		err  error
		want string
	}{
		"NilErrorReturnsNil": {
			err:  nil,
			want: "",
		},
		"NonURLErrorUnchanged": {
			err:  plainErr,
			want: "plain error",
		},
		"QueryStripped": {
			err: &url.Error{
				Op:  "Post",
				URL: "http://sonar.example.com/api/webhooks/create?name=hook&secret=" + testSecretValue,
				Err: errors.New("connection refused"),
			},
			want: `Post "http://sonar.example.com/api/webhooks/create": connection refused`,
		},
		"UserinfoAndFragmentStripped": {
			err: &url.Error{
				Op:  "Post",
				URL: "https://admin:" + testSecretValue + "@sonar.example.com/api/editions/set_license?license=" + testSecretValue + "#" + testSecretValue,
				Err: errors.New("EOF"),
			},
			want: `Post "https://sonar.example.com/api/editions/set_license": EOF`,
		},
		"WrappedURLErrorRedacted": {
			err: fmt.Errorf("cannot create webhook: %w", &url.Error{
				Op:  "Post",
				URL: "http://sonar.example.com/api/webhooks/create?secret=" + testSecretValue,
				Err: errors.New("connection refused"),
			}),
			want: `cannot create webhook: Post "http://sonar.example.com/api/webhooks/create": connection refused`,
		},
		"WrappedQuoteEscapedURLRedacted": {
			err: fmt.Errorf("cannot create webhook: %w", &url.Error{
				Op:  "parse",
				URL: "http://sonar.example.com/api/webhooks/create?secret=" + testSecretValue + "\x7f",
				Err: errors.New("invalid control character in URL"),
			}),
			want: `cannot create webhook: parse "` + redactedURL + `": invalid control character in URL`,
		},
		"NoQueryUnchanged": {
			err: &url.Error{
				Op:  "Post",
				URL: "http://sonar.example.com/api/webhooks/create",
				Err: errors.New("connection refused"),
			},
			want: `Post "http://sonar.example.com/api/webhooks/create": connection refused`,
		},
		"UnparsableURLReplaced": {
			err: &url.Error{
				Op:  "Post",
				URL: "http://[::1%zz/?secret=" + testSecretValue,
				Err: errors.New("EOF"),
			},
			want: `Post "` + redactedURL + `": EOF`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := RedactURLError(tc.err)
			if tc.err == nil {
				if got != nil {
					t.Errorf("RedactURLError(nil) = %v, want nil", got)
				}

				return
			}

			if got.Error() != tc.want {
				t.Errorf("RedactURLError() = %q, want %q", got.Error(), tc.want)
			}

			if strings.Contains(got.Error(), testSecretValue) {
				t.Errorf("RedactURLError() leaked secret: %q", got.Error())
			}

			var urlErr *url.Error
			if errors.As(tc.err, &urlErr) && !errors.As(got, &urlErr) {
				t.Errorf("RedactURLError() dropped the url.Error from the chain")
			}
		})
	}
}

// TestRedactURLErrorTransportFailure tests redaction of a real net/http
// transport error for a POST carrying a secret in its query string.
func TestRedactURLErrorTransportFailure(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://127.0.0.1:1/api/webhooks/create?secret="+testSecretValue, http.NoBody)
	if err != nil {
		t.Fatalf("NewRequestWithContext() unexpected error: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		CloseBody(resp)
	}

	if err == nil {
		t.Fatal("Do() expected a transport error, got nil")
	}

	if !strings.Contains(err.Error(), testSecretValue) {
		t.Fatalf("precondition: transport error should contain the query, got %q", err.Error())
	}

	redacted := RedactURLError(err)
	if strings.Contains(redacted.Error(), testSecretValue) {
		t.Errorf("RedactURLError() leaked secret: %q", redacted.Error())
	}

	if !strings.Contains(redacted.Error(), "http://127.0.0.1:1/api/webhooks/create") {
		t.Errorf("RedactURLError() = %q, want the method, host and path kept", redacted.Error())
	}
}
