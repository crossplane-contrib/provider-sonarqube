//go:build e2e

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

package e2e

import (
	"os"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/sonar"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/crossplane/provider-sonarqube/apis"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// Environment variable names recognised by the framework.
const (
	// EnvSonarQubeURL is the base URL of the SonarQube API used for verification.
	EnvSonarQubeURL = "SONARQUBE_URL"
	// EnvSonarQubeToken is an admin-scoped token used by the framework's Sonar client.
	EnvSonarQubeToken = "SONARQUBE_TOKEN"
	// EnvProviderConfig is the ClusterProviderConfig name that managed resources should reference.
	EnvProviderConfig = "SONARQUBE_PROVIDERCONFIG"
)

// DefaultProviderConfigName is used when EnvProviderConfig is unset; it matches
// the ClusterProviderConfig provisioned by cluster/local/sonarqube_setup.sh.
const DefaultProviderConfigName = "e2e"

// Framework groups the clients every e2e test needs: a controller-runtime
// client wired to the project's schemes, and a SonarQube SDK client
// authenticated with an admin token. Construct one per test with New.
type Framework struct {
	// Kube is a controller-runtime client with the provider's APIs registered.
	Kube client.Client
	// Sonar talks to the SonarQube REST API directly, used to verify the
	// state the provider reconciled into SonarQube.
	Sonar *sonar.Client
	// ProviderConfigName is the ClusterProviderConfig managed resources should reference.
	ProviderConfigName string
}

// New constructs a Framework, failing the test immediately on missing config
// or unreachable cluster - there is no value in deferring those errors.
func New(t *testing.T) *Framework {
	t.Helper()

	url := mustEnv(t, EnvSonarQubeURL)
	token := mustEnv(t, EnvSonarQubeToken)

	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("loading kubeconfig: %v", err)
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("registering core scheme: %v", err)
	}
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatalf("registering provider schemes: %v", err)
	}

	kc, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("building kube client: %v", err)
	}

	sc := common.NewClient(common.Config{
		AuthType: common.PersonalAccessToken,
		Token:    token,
		BaseURL:  url,
	})

	pc := os.Getenv(EnvProviderConfig)
	if pc == "" {
		pc = DefaultProviderConfigName
	}

	return &Framework{
		Kube:               kc,
		Sonar:              sc,
		ProviderConfigName: pc,
	}
}

func mustEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Fatalf("required environment variable %s is not set", key)
	}
	return v
}
