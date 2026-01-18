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

	sonargo "github.com/boxboxjason/sonarqube-client-go/sonar"
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BasicAuthArgs is the expected struct that can be passed in the Config.Token field to add support for BasicAuth AuthMethod
type BasicAuthArgs struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Config provides SonarQube configurations for the SonarQube client
type Config struct {
	// AuthType is the SonarQube authentication type to use (e.g., BasicAuth, PersonalAccessToken)
	AuthType AuthType
	// BasicAuth contains the Basic authentication credentials for the SonarQube instance
	BasicAuth *BasicAuthArgs
	// Token is the Personal access token for the SonarQube instance
	Token string
	// BaseURL is the URL of the SonarQube instance (trailing slash is optional)
	BaseURL string
	// InsecureSkipVerify indicates whether to skip TLS certificate verification (for self-signed certificates)
	InsecureSkipVerify bool
}

// NewClient creates new SonarQube Client with provided SonarQube Configurations/Credentials.
func NewClient(clientConfig Config) *sonargo.Client {
	var client *sonargo.Client

	switch clientConfig.AuthType {
	case BasicAuth:
		if clientConfig.BasicAuth == nil {
			panic(errors.New("BasicAuth configuration is required for BasicAuth"))
		}
		// Create SonarQube client with Basic Auth
		sonarClient, err := sonargo.NewClient(clientConfig.BaseURL, clientConfig.BasicAuth.Username, clientConfig.BasicAuth.Password)
		if err != nil {
			panic(err)
		}
		client = sonarClient
	case PersonalAccessToken:
		// Create SonarQube client with Personal Access Token
		sonarClient, err := sonargo.NewClientWithToken(clientConfig.BaseURL, clientConfig.Token)
		if err != nil {
			panic(err)
		}
		client = sonarClient
	default:
		panic(errors.New("unsupported authentication type"))
	}

	httpClient := cleanhttp.DefaultClient()

	// Configure TLS settings if InsecureSkipVerify is set to true
	if clientConfig.InsecureSkipVerify {
		transport := cleanhttp.DefaultPooledTransport()
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
		httpClient.Transport = transport
	}
	client.SetHTTPClient(httpClient)

	return client
}

// GetConfig constructs a Config that can be used to authenticate to SonarQube's
// API by the SonarQube Go client
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

// UseProviderConfig uses the given ProviderConfig reference to construct a Config
// that can be used to authenticate to SonarQube's API by the SonarQube Go client
func UseProviderConfig(ctx context.Context, kubeClient client.Client, managedResource resource.ModernManaged) (*Config, error) {
	providerConfigRef := managedResource.GetProviderConfigReference()

	switch providerConfigRef.Kind {
	case "ClusterProviderConfig":
		cpc := &v1alpha1.ClusterProviderConfig{}
		if err := kubeClient.Get(ctx, types.NamespacedName{Name: providerConfigRef.Name}, cpc); err != nil {
			return nil, errors.Wrap(err, "cannot get referenced ClusterProviderConfig")
		}
		return buildConfigFromSpec(ctx, kubeClient, managedResource, cpc.Spec)
	default: // "ProviderConfig" or empty (default)
		pc := &v1alpha1.ProviderConfig{}
		if err := kubeClient.Get(ctx, types.NamespacedName{Name: providerConfigRef.Name, Namespace: managedResource.GetNamespace()}, pc); err != nil {
			return nil, errors.Wrap(err, "cannot get referenced ProviderConfig")
		}
		return buildConfigFromSpec(ctx, kubeClient, managedResource, pc.Spec)
	}
}

// buildConfigFromSpec builds a Config from the given ProviderConfigSpec
func buildConfigFromSpec(ctx context.Context, kubeClient client.Client, managedResource resource.ModernManaged, spec v1alpha1.ProviderConfigSpec) (*Config, error) {
	t := resource.NewProviderConfigUsageTracker(kubeClient, &v1alpha1.ProviderConfigUsage{})
	if err := t.Track(ctx, managedResource); err != nil {
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

	switch authType {
	case PersonalAccessToken:
		token, err := GetTokenValueFromSecret(ctx, kubeClient, managedResource, spec.Token.SecretRef)
		if err != nil {
			return nil, errors.Wrap(err, "cannot get token from secret")
		}
		config.Token = *token
	case BasicAuth:
		username, err := GetTokenValueFromSecret(ctx, kubeClient, managedResource, spec.Username.SecretRef)
		if err != nil {
			return nil, errors.Wrap(err, "cannot get username from secret")
		}
		password, err := GetTokenValueFromSecret(ctx, kubeClient, managedResource, spec.Password.SecretRef)
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

// determineAuthType determines the AuthType based on the provided ProviderConfigSpec
// It populates the AuthType and BasicAuth fields in the Config struct accordingly
// It returns an error if no valid authentication method is found
func determineAuthType(spec v1alpha1.ProviderConfigSpec) (AuthType, error) {
	// Check if Token is provided for Personal Access Token authentication
	if spec.Token != nil {
		switch spec.Token.Source {
		case xpv1.CredentialsSourceSecret:
			if spec.Token.SecretRef == nil {
				return "", errors.New("secretRef must be provided for token")
			}
			return PersonalAccessToken, nil
		default:
			return "", errors.Errorf("credentials source %s for token is not currently supported", spec.Token.Source)
		}
	} else if spec.Username != nil && spec.Password != nil {
		// Check if Username and Password are provided for Basic Authentication
		switch spec.Username.Source {
		case xpv1.CredentialsSourceSecret:
			if spec.Username.SecretRef == nil {
				return "", errors.New("secretRef must be provided for username")
			}
			switch spec.Password.Source {
			case xpv1.CredentialsSourceSecret:
				if spec.Password.SecretRef == nil {
					return "", errors.New("secretRef must be provided for password")
				}
				return BasicAuth, nil
			default:
				return "", errors.Errorf("credentials source %s for password is not currently supported", spec.Password.Source)
			}
		default:
			return "", errors.Errorf("credentials source %s for username is not currently supported", spec.Username.Source)
		}
	}

	return "", errors.New("no valid authentication method found in ProviderConfigSpec")
}
