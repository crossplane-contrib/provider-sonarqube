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
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
)

const (
	// endpointRequestTimeout bounds how long FetchFromEndpoint waits for a
	// response.
	endpointRequestTimeout = 30 * time.Second
	// endpointMaxResponseSize bounds how much of an endpoint's response body
	// FetchFromEndpoint reads.
	endpointMaxResponseSize = 1 << 20 // 1 MiB
)

// EndpointAuth holds optional authentication used by FetchFromEndpoint.
// A non-empty BearerToken takes precedence over BasicAuthUsername.
type EndpointAuth struct {
	BasicAuthUsername string
	BasicAuthPassword string
	BearerToken       string
}

// FetchFromEndpoint performs an HTTP GET against url, optionally
// authenticating with auth, and returns the response body trimmed of
// surrounding whitespace.
func FetchFromEndpoint(ctx context.Context, url string, auth *EndpointAuth) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, endpointRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", errors.Wrap(err, "cannot build endpoint request")
	}

	applyEndpointAuth(req, auth)

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return "", errors.Wrap(err, "cannot reach endpoint")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", errors.Errorf("endpoint returned unexpected status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, endpointMaxResponseSize+1))
	if err != nil {
		return "", errors.Wrap(err, "cannot read endpoint response")
	}
	if int64(len(body)) > endpointMaxResponseSize {
		return "", errors.Errorf("endpoint response exceeds %d bytes", endpointMaxResponseSize)
	}

	value := strings.TrimSpace(string(body))
	if value == "" {
		return "", errors.New("endpoint returned an empty response")
	}

	return value, nil
}

// applyEndpointAuth sets the request's Authorization header from auth, if
// any. A non-empty bearer token takes precedence over basic auth.
func applyEndpointAuth(req *http.Request, auth *EndpointAuth) {
	if auth == nil {
		return
	}

	if auth.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+auth.BearerToken)

		return
	}

	if auth.BasicAuthUsername != "" {
		creds := base64.StdEncoding.EncodeToString([]byte(auth.BasicAuthUsername + ":" + auth.BasicAuthPassword))
		req.Header.Set("Authorization", "Basic "+creds)
	}
}
