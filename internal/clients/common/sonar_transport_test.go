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
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
)

// ping sends a request through the SonarQube client and consumes the response.
func ping(t *testing.T, client *sonar.Client) error {
	t.Helper()

	_, resp, err := client.System.Ping(context.Background())
	if resp != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	return err
}

// TestNewClientReusesConnections checks that clients built for successive
// reconciles share connections, and that each still sends its own credentials.
func TestNewClientReusesConnections(t *testing.T) {
	t.Parallel()

	var (
		connections atomic.Int32
		mu          sync.Mutex
		authHeaders []string
	)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		authHeaders = append(authHeaders, r.Header.Get("Authorization"))

		_, _ = w.Write([]byte("pong"))
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.Start()
	t.Cleanup(server.Close)

	tokens := []string{"first-token", "second-token", "third-token"}
	for _, token := range tokens {
		client := NewClient(Config{AuthType: PersonalAccessToken, Token: token, BaseURL: server.URL + "/api/"})

		err := ping(t, client)
		if err != nil {
			t.Fatalf("Ping() unexpected error: %v", err)
		}
	}

	if got := connections.Load(); got != 1 {
		t.Errorf("server accepted %d connections, want 1", got)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(authHeaders) != len(tokens) {
		t.Fatalf("server received %d requests, want %d", len(authHeaders), len(tokens))
	}

	sorted := slices.Sorted(slices.Values(authHeaders))
	if len(slices.Compact(sorted)) != len(tokens) {
		t.Errorf("clients sharing a connection sent the same credentials: %v", authHeaders)
	}
}

// TestNewClientKeepsTLSVerificationApart checks that a connection opened
// without certificate verification is not reused by a client that verifies
// certificates.
func TestNewClientKeepsTLSVerificationApart(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	}))
	t.Cleanup(server.Close)

	config := Config{AuthType: PersonalAccessToken, Token: "token", BaseURL: server.URL + "/api/"}

	insecureConfig := config
	insecureConfig.InsecureSkipVerify = true

	err := ping(t, NewClient(insecureConfig))
	if err != nil {
		t.Fatalf("Ping() with InsecureSkipVerify unexpected error: %v", err)
	}

	err = ping(t, NewClient(config))
	if err == nil {
		t.Fatal("Ping() succeeded against an untrusted certificate, want a verification error")
	}
}
