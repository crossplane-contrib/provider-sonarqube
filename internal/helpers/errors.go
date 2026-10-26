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
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// redactedURL replaces a request URL that cannot be parsed for redaction.
const redactedURL = "<redacted>"

// redactedError overrides the message of an error whose wrappers captured
// the unredacted URL, while keeping the original chain for [errors.Is] and
// [errors.As].
type redactedError struct {
	msg string
	err error
}

// Error returns the redacted message.
func (e *redactedError) Error() string { return e.msg }

// Unwrap returns the original error.
func (e *redactedError) Unwrap() error { return e.err }

// RedactURLError strips the query string, fragment and userinfo from the
// first [url.Error] in err's chain.
//
// The SonarQube v1 API sends request parameters, including secrets such as
// webhook HMAC keys, ALM credentials and license keys, in the request URL
// query. When the request fails at the transport layer, net/http returns a
// [url.Error] whose message contains that full URL. Redacting it keeps those
// parameters out of status conditions, events, change logs and logs.
func RedactURLError(err error) error {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) || urlErr == nil {
		return err
	}

	originalURL := urlErr.URL

	redacted := redactURL(originalURL)
	if redacted == originalURL {
		return err
	}

	urlErr.URL = redacted

	// Wrappers such as fmt.Errorf compute their message eagerly, so it may
	// still hold the original URL even though the url.Error is redacted.
	// url.Error formats its URL with %q, so replace the quoted form first in
	// case quoting escaped any of its characters.
	msg := err.Error()

	redactedMsg := strings.ReplaceAll(msg, strconv.Quote(originalURL), strconv.Quote(redacted))
	redactedMsg = strings.ReplaceAll(redactedMsg, originalURL, redacted)

	if redactedMsg != msg {
		return &redactedError{msg: redactedMsg, err: err}
	}

	return err
}

// redactURL returns rawURL without its query string, fragment and userinfo.
func redactURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return redactedURL
	}

	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	parsed.RawFragment = ""

	return parsed.String()
}
