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
	"crypto/tls"
	"net/http"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/provider-sonarqube/apis/v1alpha1"
)

const (
	// clusterProviderConfigKind is the kind name for
	// ClusterProviderConfig resources.
	clusterProviderConfigKind = "ClusterProviderConfig"

	// maxIdleConnsPerHost bounds the idle connections kept per SonarQube
	// instance. The transport default follows GOMAXPROCS, which is too low
	// for a provider that keeps several reconciles of concurrent requests in
	// flight against the same host on a small CPU limit.
	maxIdleConnsPerHost = 32

	// idleConnTimeout closes idle connections before the server does. Tomcat
	// and the usual proxies in front of it drop them after 60 to 75 seconds,
	// and a POST written to a connection the peer is closing fails instead of
	// being retried, as the transport cannot know whether it was processed.
	idleConnTimeout = 45 * time.Second

	// http2HealthCheckTimeout is how long an HTTP/2 connection may stay silent
	// before it is pinged. All requests to a host share one such connection,
	// so without this check a dead peer stalls them until the kernel gives up
	// on it, as a cancelled request does not close the connection.
	http2HealthCheckTimeout = 30 * time.Second

	// DefaultResponseHeaderTimeout bounds the wait for a response once a
	// request is written, so a reused connection to a vanished peer fails the
	// request instead of holding a worker until the reconcile deadline. It
	// matches the timeout of the HTTP client the SonarQube SDK would otherwise
	// use.
	DefaultResponseHeaderTimeout = 30 * time.Second

	// maxRequestAttempts is the number of times an idempotent request is
	// sent before its failure is reported to the reconciler, which then
	// requeues it with its own backoff.
	maxRequestAttempts = 3
	// initialRetryDelay is the base of the exponential backoff between
	// attempts. Retries stay short so that they fit in the reconcile timeout.
	initialRetryDelay = 500 * time.Millisecond
	// maxRetryDelay caps the backoff between attempts.
	maxRetryDelay = 5 * time.Second
)

var (
	// pooledHTTPClient is shared by every SonarQube client that verifies TLS
	// certificates. A client is built on each reconcile, so connections are
	// only reused when the transport outlives it. Sharing the transport is
	// safe because credentials are set on each request, never on a
	// connection.
	pooledHTTPClient = newPooledHTTPClient(false, DefaultResponseHeaderTimeout)
	// insecurePooledHTTPClient is shared by every SonarQube client that skips
	// TLS verification. It is kept apart from pooledHTTPClient so that a
	// connection opened without verification is never handed to a client
	// that requires it.
	insecurePooledHTTPClient = newPooledHTTPClient(true, DefaultResponseHeaderTimeout)

	// requestRetryOptions rides out short SonarQube unavailability, such as a
	// restart behind a proxy. Only idempotent methods are retried: SonarQube
	// mutations are POST requests, and resending one could repeat its effect.
	requestRetryOptions = sonar.RetryOptions{
		MaxAttempts:  maxRequestAttempts,
		InitialDelay: initialRetryDelay,
		MaxDelay:     maxRetryDelay,
		RetryableStatusCodes: []int{
			http.StatusTooManyRequests,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		},
	}
)

// BasicAuthArgs is the expected struct that can be passed in the Config.Token
// field to add support for BasicAuth AuthMethod.
type BasicAuthArgs struct {
	// Username is the username for Basic Authentication.
	// It is expected to be stored in a Kubernetes Secret and
	// referenced in the ProviderConfig.
	Username string `json:"username"`
	// Password is the password for Basic Authentication.
	// It is expected to be stored in a Kubernetes Secret and
	// referenced in the ProviderConfig.
	Password string `json:"password"`
}

// Config provides SonarQube configurations for the SonarQube client.
type Config struct {
	// AuthType is the SonarQube authentication type to use
	// (e.g., BasicAuth, PersonalAccessToken)
	AuthType AuthType
	// BasicAuth contains the Basic authentication credentials for the
	// SonarQube instance
	BasicAuth *BasicAuthArgs
	// Token is the Personal access token for the SonarQube instance
	Token string
	// BaseURL is the URL of the SonarQube instance (trailing slash is optional)
	BaseURL string
	// InsecureSkipVerify indicates whether to skip TLS certificate verification
	// (for self-signed certificates)
	InsecureSkipVerify bool
}

// NewClient creates new SonarQube Client with provided SonarQube
// Configurations/Credentials.
func NewClient(clientConfig Config) *sonar.Client {
	var sonarClient *sonar.Client

	// The HTTP client must be given at construction: the retry wrapper is
	// built around it there, and replacing it afterwards would drop retries.
	httpClient := sharedHTTPClient(clientConfig.InsecureSkipVerify)

	switch clientConfig.AuthType {
	case BasicAuth:
		if clientConfig.BasicAuth == nil {
			panic(errors.New("BasicAuth configuration is required for BasicAuth"))
		}
		// Create SonarQube client with Basic Auth
		basicAuthClient, err := sonar.NewClient(
			&sonar.ClientCreateOptions{
				URL:        &clientConfig.BaseURL,
				Username:   &clientConfig.BasicAuth.Username,
				Password:   &clientConfig.BasicAuth.Password,
				HttpClient: httpClient,
			},
			sonar.WithRetry(requestRetryOptions))
		if err != nil {
			panic(err)
		}

		sonarClient = basicAuthClient
	case PersonalAccessToken:
		// Create SonarQube client with Personal Access Token
		tokenClient, err := sonar.NewClient(
			&sonar.ClientCreateOptions{
				URL:        &clientConfig.BaseURL,
				Token:      &clientConfig.Token,
				HttpClient: httpClient,
			},
			sonar.WithRetry(requestRetryOptions))
		if err != nil {
			panic(err)
		}

		sonarClient = tokenClient
	default:
		panic(errors.New("unsupported authentication type"))
	}

	return sonarClient
}

// ConfigureHTTPClients sets how long the shared HTTP clients wait for response
// headers once a request is written. Zero disables the timeout. It replaces the
// shared clients, so it must be called before any SonarQube client is built.
func ConfigureHTTPClients(responseHeaderTimeout time.Duration) error {
	if responseHeaderTimeout < 0 {
		return errors.Errorf("response header timeout must not be negative, got %s", responseHeaderTimeout)
	}

	pooledHTTPClient = newPooledHTTPClient(false, responseHeaderTimeout)
	insecurePooledHTTPClient = newPooledHTTPClient(true, responseHeaderTimeout)

	return nil
}

// newPooledHTTPClient returns an HTTP client that keeps connections alive,
// optionally skipping TLS certificate verification.
func newPooledHTTPClient(insecureSkipVerify bool, responseHeaderTimeout time.Duration) *http.Client {
	transport := cleanhttp.DefaultPooledTransport()
	transport.MaxIdleConnsPerHost = maxIdleConnsPerHost
	transport.IdleConnTimeout = idleConnTimeout
	transport.HTTP2 = &http.HTTP2Config{SendPingTimeout: http2HealthCheckTimeout}
	transport.ResponseHeaderTimeout = responseHeaderTimeout

	if insecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true, //nolint:gosec // opted into by the ProviderConfig, for self-signed certificates
		}
	}

	return &http.Client{Transport: transport}
}

// sharedHTTPClient returns the shared HTTP client matching the TLS
// verification setting.
func sharedHTTPClient(insecureSkipVerify bool) *http.Client {
	if insecureSkipVerify {
		return insecurePooledHTTPClient
	}

	return pooledHTTPClient
}

// GetConfig constructs a Config that can be used to authenticate to SonarQube's
// API by the SonarQube Go client.
func GetConfig(ctx context.Context, kubeClient client.Client, managedResource resource.Managed) (*Config, error) {
	switch managedResourceCast := managedResource.(type) {
	case resource.ModernManaged:
		switch {
		case managedResourceCast.GetProviderConfigReference() != nil:
			return UseProviderConfig(ctx, kubeClient, managedResourceCast)
		default:
			return nil, errors.New("providerConfigRef is not given")
		}
	default:
		return nil, errors.New("unknown managed resource type")
	}
}

// UseProviderConfig uses the given ProviderConfig reference to construct a
// Config that can be used to authenticate to SonarQube's API by the
// SonarQube Go client.
func UseProviderConfig(ctx context.Context, kubeClient client.Client, managedResource resource.ModernManaged) (*Config, error) {
	providerConfigRef := managedResource.GetProviderConfigReference()

	switch providerConfigRef.Kind {
	case clusterProviderConfigKind:
		cpc := &v1alpha1.ClusterProviderConfig{}

		err := kubeClient.Get(ctx, types.NamespacedName{Name: providerConfigRef.Name}, cpc)
		if err != nil {
			return nil, errors.Wrap(err, "cannot get referenced ClusterProviderConfig")
		}

		// ClusterProviderConfigs are cluster-scoped and operator-owned, so
		// their credentials may live in any namespace.
		return buildConfigFromSpec(ctx, kubeClient, managedResource, cpc.Spec, nil)
	default: // "ProviderConfig" or empty (default)
		providerConfig := &v1alpha1.ProviderConfig{}

		err := kubeClient.Get(ctx, types.NamespacedName{Name: providerConfigRef.Name, Namespace: managedResource.GetNamespace()}, providerConfig)
		if err != nil {
			return nil, errors.Wrap(err, "cannot get referenced ProviderConfig")
		}

		return buildConfigFromSpec(ctx, kubeClient, managedResource, providerConfig.Spec, new(providerConfig.GetNamespace()))
	}
}

// buildConfigFromSpec builds a Config from the given ProviderConfigSpec.
// secretNamespace is the namespace every credential secret must live in; nil
// means the spec is cluster-scoped and secrets may live anywhere. A non-nil
// empty namespace is not treated as unrestricted, so the check fails closed.
func buildConfigFromSpec(ctx context.Context, kubeClient client.Client, managedResource resource.ModernManaged, spec v1alpha1.ProviderConfigSpec, secretNamespace *string) (*Config, error) {
	t := resource.NewProviderConfigUsageTracker(kubeClient, &v1alpha1.ProviderConfigUsage{})

	err := t.Track(ctx, managedResource)
	if err != nil {
		return nil, errors.Wrap(err, "cannot track ProviderConfig usage")
	}

	config := &Config{
		BaseURL:            spec.BaseURL,
		InsecureSkipVerify: ptr.Deref(spec.InsecureSkipVerify, false),
	}

	authType, err := determineAuthType(spec)
	if err != nil {
		return nil, errors.Wrap(err, "cannot determine authentication type from ProviderConfigSpec")
	}

	config.AuthType = authType

	readSecret := func(selector *xpv1.SecretKeySelector) (*string, error) {
		if secretNamespace == nil {
			return GetTokenValueFromSecret(ctx, kubeClient, managedResource, selector)
		}

		return GetTokenValueFromSecretInNamespace(ctx, kubeClient, managedResource, *secretNamespace, selector)
	}

	switch authType {
	case PersonalAccessToken:
		token, err := readSecret(spec.Token.SecretRef)
		if err != nil {
			return nil, errors.Wrap(err, "cannot get token from secret")
		}

		config.Token = *token
	case BasicAuth:
		username, err := readSecret(spec.Username.SecretRef)
		if err != nil {
			return nil, errors.Wrap(err, "cannot get username from secret")
		}

		password, err := readSecret(spec.Password.SecretRef)
		if err != nil {
			return nil, errors.Wrap(err, "cannot get password from secret")
		}

		config.BasicAuth = &BasicAuthArgs{
			Username: *username,
			Password: *password,
		}
	}

	return config, nil
}

// determineAuthType determines the AuthType based on the provided
// ProviderConfigSpec.
// It populates the AuthType and BasicAuth fields in the Config struct
// accordingly.
// It returns an error if no valid authentication method is found.
func determineAuthType(spec v1alpha1.ProviderConfigSpec) (AuthType, error) {
	// Check if Token is provided for Personal Access Token authentication
	if spec.Token != nil {
		return validateTokenAuth(spec.Token)
	}

	// Check if Username and Password are provided for Basic Authentication
	if spec.Username != nil && spec.Password != nil {
		return validateBasicAuth(spec.Username, spec.Password)
	}

	return "", errors.New("no valid authentication method found in ProviderConfigSpec")
}

// validateTokenAuth validates token-based authentication configuration.
func validateTokenAuth(token *v1alpha1.ProviderCredentials) (AuthType, error) {
	if token.Source != xpv1.CredentialsSourceSecret {
		return "", errors.Errorf("credentials source %s for token is not currently supported", token.Source)
	}

	if token.SecretRef == nil {
		return "", errors.New("secretRef must be provided for token")
	}

	return PersonalAccessToken, nil
}

// validateBasicAuth validates basic authentication configuration.
func validateBasicAuth(username, password *v1alpha1.ProviderCredentials) (AuthType, error) {
	if username.Source != xpv1.CredentialsSourceSecret {
		return "", errors.Errorf("credentials source %s for username is not currently supported", username.Source)
	}

	if username.SecretRef == nil {
		return "", errors.New("secretRef must be provided for username")
	}

	if password.Source != xpv1.CredentialsSourceSecret {
		return "", errors.Errorf("credentials source %s for password is not currently supported", password.Source)
	}

	if password.SecretRef == nil {
		return "", errors.New("secretRef must be provided for password")
	}

	return BasicAuth, nil
}
