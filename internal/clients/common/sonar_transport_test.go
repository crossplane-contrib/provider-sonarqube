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
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"

	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// ping sends a request through the SonarQube client.
func ping(t *testing.T, client *sonar.Client) error {
	t.Helper()

	_, resp, err := client.System.Ping(context.Background()) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	return err
}

// TestNewClientReusesConnections checks that clients built for successive
// reconciles share connections, and that each still sends its own credentials.
func TestNewClientReusesConnections(t *testing.T) {
	t.Parallel()

	var (
		connections atomic.Int32
		mu          sync.Mutex
		usernames   []string
	)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		// A personal access token is sent as the basic auth username.
		username, _, _ := r.BasicAuth()
		usernames = append(usernames, username)

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

	if !slices.Equal(usernames, tokens) {
		t.Errorf("server received credentials %v, want %v", usernames, tokens)
	}
}

// TestNewPooledHTTPClientTransport checks the settings that keep a long-lived
// shared connection from failing requests after its peer closed or vanished.
func TestNewPooledHTTPClientTransport(t *testing.T) {
	t.Parallel()

	for _, insecureSkipVerify := range []bool{false, true} {
		transport, ok := newPooledHTTPClient(insecureSkipVerify, DefaultResponseHeaderTimeout).Transport.(*http.Transport)
		if !ok {
			t.Fatalf("newPooledHTTPClient(%t) transport is not an *http.Transport", insecureSkipVerify)
		}

		if transport.IdleConnTimeout != idleConnTimeout {
			t.Errorf("newPooledHTTPClient(%t) IdleConnTimeout = %s, want %s", insecureSkipVerify, transport.IdleConnTimeout, idleConnTimeout)
		}

		if transport.HTTP2 == nil || transport.HTTP2.SendPingTimeout != http2HealthCheckTimeout {
			t.Errorf("newPooledHTTPClient(%t) HTTP2 = %+v, want SendPingTimeout %s", insecureSkipVerify, transport.HTTP2, http2HealthCheckTimeout)
		}

		if transport.ResponseHeaderTimeout != DefaultResponseHeaderTimeout {
			t.Errorf("newPooledHTTPClient(%t) ResponseHeaderTimeout = %s, want %s", insecureSkipVerify, transport.ResponseHeaderTimeout, DefaultResponseHeaderTimeout)
		}
	}
}

// TestConfigureHTTPClients replaces the package-level shared clients, so it
// must not run in parallel with other tests relying on them.
//
//nolint:paralleltest // mutates package-level state
func TestConfigureHTTPClients(t *testing.T) {
	t.Cleanup(func() {
		err := ConfigureHTTPClients(DefaultResponseHeaderTimeout)
		if err != nil {
			t.Errorf("resetting ConfigureHTTPClients() error = %v", err)
		}
	})

	if ConfigureHTTPClients(-time.Second) == nil {
		t.Fatal("ConfigureHTTPClients() with a negative timeout succeeded")
	}

	const timeout = 5 * time.Second

	err := ConfigureHTTPClients(timeout)
	if err != nil {
		t.Fatalf("ConfigureHTTPClients() error = %v", err)
	}

	for _, client := range []*http.Client{pooledHTTPClient, insecurePooledHTTPClient} {
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatal("shared client transport is not an *http.Transport")
		}

		if transport.ResponseHeaderTimeout != timeout {
			t.Errorf("ResponseHeaderTimeout = %s, want %s", transport.ResponseHeaderTimeout, timeout)
		}
	}
}

// TestResponseHeaderTimeoutFailsStalledRequest checks that a request fails when
// the server accepts it but never answers.
func TestResponseHeaderTimeoutFailsStalledRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)

	client := NewClient(Config{AuthType: PersonalAccessToken, Token: "token", BaseURL: server.URL + "/api/"})
	client.SetHTTPClient(newPooledHTTPClient(false, 50*time.Millisecond))

	err := ping(t, client)
	if err == nil || !strings.Contains(err.Error(), "timeout awaiting response headers") {
		t.Errorf("Ping() error = %v, want a response header timeout", err)
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
