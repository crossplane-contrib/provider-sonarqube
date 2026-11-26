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
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
)

// basicEmailSpec returns a spec using BASIC authentication.
func basicEmailSpec() *v1alpha1.EmailConfigurationParameters {
	return &v1alpha1.EmailConfigurationParameters{
		Host:             "smtp.example.com",
		Port:             587,
		SecurityProtocol: "STARTTLS",
		FromAddress:      "sonarqube@example.com",
		FromName:         "SonarQube",
		SubjectPrefix:    "[SonarQube]",
		AuthMethod:       v1alpha1.EmailAuthMethodBasic,
		Username:         "user",
	}
}

// oauthEmailSpec returns a spec using OAUTH authentication.
func oauthEmailSpec() *v1alpha1.EmailConfigurationParameters {
	spec := basicEmailSpec()
	spec.AuthMethod = v1alpha1.EmailAuthMethodOAuth
	spec.OAuthAuthenticationHost = new("https://login.example.com")
	spec.OAuthTenant = new("tenant")
	spec.OAuthScope = new("scope")

	return spec
}

// observedEmailConfiguration returns the SonarQube configuration matching
// the spec, with every credential set.
func observedEmailConfiguration(spec *v1alpha1.EmailConfigurationParameters) *sonar.SystemEmailConfiguration {
	return &sonar.SystemEmailConfiguration{
		Id:                      "id",
		Host:                    spec.Host,
		Port:                    "587",
		SecurityProtocol:        spec.SecurityProtocol,
		FromAddress:             spec.FromAddress,
		FromName:                spec.FromName,
		SubjectPrefix:           spec.SubjectPrefix,
		AuthMethod:              spec.AuthMethod,
		Username:                new(spec.Username),
		IsBasicPasswordSet:      true,
		OauthAuthenticationHost: spec.OAuthAuthenticationHost,
		OauthTenant:             spec.OAuthTenant,
		OauthScope:              spec.OAuthScope,
		IsOauthClientIdSet:      true,
		IsOauthClientSecretSet:  true,
	}
}

// TestLateInitializeEmailConfiguration tests late initialization.
func TestLateInitializeEmailConfiguration(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec        *v1alpha1.EmailConfigurationParameters
		observation *v1alpha1.EmailConfigurationObservation
		wantScope   *string
		wantChanged bool
	}{
		"NilObservation": {spec: oauthEmailSpec(), wantScope: new("scope")},
		"BasicIgnoresScope": {
			spec:        basicEmailSpec(),
			observation: &v1alpha1.EmailConfigurationObservation{OAuthScope: "observed"},
		},
		"OAuthFillsScope": {
			spec: func() *v1alpha1.EmailConfigurationParameters {
				spec := oauthEmailSpec()
				spec.OAuthScope = nil

				return spec
			}(),
			observation: &v1alpha1.EmailConfigurationObservation{OAuthScope: "observed"},
			wantScope:   new("observed"),
			wantChanged: true,
		},
		"OAuthKeepsScope": {
			spec:        oauthEmailSpec(),
			observation: &v1alpha1.EmailConfigurationObservation{OAuthScope: "observed"},
			wantScope:   new("scope"),
		},
		"OAuthNoObservedScope": {
			spec: func() *v1alpha1.EmailConfigurationParameters {
				spec := oauthEmailSpec()
				spec.OAuthScope = nil

				return spec
			}(),
			observation: &v1alpha1.EmailConfigurationObservation{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			former := tc.spec.DeepCopy()

			LateInitializeEmailConfiguration(tc.spec, tc.observation)

			if diff := cmp.Diff(tc.wantScope, tc.spec.OAuthScope); diff != "" {
				t.Errorf("LateInitializeEmailConfiguration() scope -want, +got:\n%s", diff)
			}

			if got := IsEmailConfigurationLateInitialized(former, tc.spec); got != tc.wantChanged {
				t.Errorf("IsEmailConfigurationLateInitialized() = %t, want %t", got, tc.wantChanged)
			}
		})
	}

	if IsEmailConfigurationLateInitialized(nil, nil) {
		t.Error("IsEmailConfigurationLateInitialized(nil, nil) = true, want false")
	}
}

// TestIsEmailConfigurationUpToDate tests drift detection.
func TestIsEmailConfigurationUpToDate(t *testing.T) {
	t.Parallel()

	basic := EmailConfigurationCredentials{BasicPassword: "password"}
	oauth := EmailConfigurationCredentials{OAuthClientID: "id", OAuthClientSecret: "secret"}

	type observationOption = func(*sonar.SystemEmailConfiguration)

	cases := map[string]struct {
		spec    *v1alpha1.EmailConfigurationParameters
		mutate  observationOption
		current EmailConfigurationCredentials
		saved   EmailConfigurationCredentials
		want    bool
	}{
		"BasicUpToDate": {spec: basicEmailSpec(), current: basic, saved: basic, want: true},
		"BasicIgnoresOAuthFields": {
			spec: basicEmailSpec(), current: basic, saved: basic, want: true,
			mutate: func(c *sonar.SystemEmailConfiguration) { c.OauthTenant = new("other"); c.IsOauthClientIdSet = false },
		},
		"PasswordRotated":    {spec: basicEmailSpec(), current: basic, saved: EmailConfigurationCredentials{BasicPassword: "old"}},
		"PasswordNeverSaved": {spec: basicEmailSpec(), current: basic},
		"PasswordUnset": {
			spec: basicEmailSpec(), current: basic, saved: basic,
			mutate: func(c *sonar.SystemEmailConfiguration) { c.IsBasicPasswordSet = false },
		},
		"OAuthUpToDate": {spec: oauthEmailSpec(), current: oauth, saved: oauth, want: true},
		"OAuthUnsetScopeMatchesAny": {
			spec: func() *v1alpha1.EmailConfigurationParameters {
				spec := oauthEmailSpec()
				spec.OAuthScope = nil

				return spec
			}(),
			current: oauth, saved: oauth, want: true,
		},
		"ClientSecretRotated": {
			spec: oauthEmailSpec(), current: oauth,
			saved: EmailConfigurationCredentials{OAuthClientID: "id", OAuthClientSecret: "old"},
		},
		"ClientIDUnset": {
			spec: oauthEmailSpec(), current: oauth, saved: oauth,
			mutate: func(c *sonar.SystemEmailConfiguration) { c.IsOauthClientIdSet = false },
		},
		"ClientSecretUnset": {
			spec: oauthEmailSpec(), current: oauth, saved: oauth,
			mutate: func(c *sonar.SystemEmailConfiguration) { c.IsOauthClientSecretSet = false },
		},
		"TenantDrift": {
			spec: oauthEmailSpec(), current: oauth, saved: oauth,
			mutate: func(c *sonar.SystemEmailConfiguration) { c.OauthTenant = new("other") },
		},
		"AuthenticationHostDrift": {
			spec: oauthEmailSpec(), current: oauth, saved: oauth,
			mutate: func(c *sonar.SystemEmailConfiguration) { c.OauthAuthenticationHost = nil },
		},
		"ScopeDrift": {
			spec: oauthEmailSpec(), current: oauth, saved: oauth,
			mutate: func(c *sonar.SystemEmailConfiguration) { c.OauthScope = new("other") },
		},
	}

	fieldDrifts := map[string]observationOption{
		"HostDrift":             func(c *sonar.SystemEmailConfiguration) { c.Host = "other" },
		"PortDrift":             func(c *sonar.SystemEmailConfiguration) { c.Port = "25" },
		"SecurityProtocolDrift": func(c *sonar.SystemEmailConfiguration) { c.SecurityProtocol = "NONE" },
		"FromAddressDrift":      func(c *sonar.SystemEmailConfiguration) { c.FromAddress = "other" },
		"FromNameDrift":         func(c *sonar.SystemEmailConfiguration) { c.FromName = "other" },
		"SubjectPrefixDrift":    func(c *sonar.SystemEmailConfiguration) { c.SubjectPrefix = "other" },
		"AuthMethodDrift":       func(c *sonar.SystemEmailConfiguration) { c.AuthMethod = v1alpha1.EmailAuthMethodOAuth },
		"UsernameDrift":         func(c *sonar.SystemEmailConfiguration) { c.Username = nil },
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			configuration := observedEmailConfiguration(tc.spec)
			if tc.mutate != nil {
				tc.mutate(configuration)
			}

			observation := GenerateEmailConfigurationObservation(configuration)

			if got := IsEmailConfigurationUpToDate(tc.spec, &observation, tc.current, tc.saved); got != tc.want {
				t.Errorf("IsEmailConfigurationUpToDate() = %t, want %t", got, tc.want)
			}
		})
	}

	for name, mutate := range fieldDrifts {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			configuration := observedEmailConfiguration(basicEmailSpec())
			mutate(configuration)
			observation := GenerateEmailConfigurationObservation(configuration)

			if IsEmailConfigurationUpToDate(basicEmailSpec(), &observation, basic, basic) {
				t.Error("IsEmailConfigurationUpToDate() = true, want false")
			}
		})
	}

	if !IsEmailConfigurationUpToDate(nil, nil, basic, basic) {
		t.Error("IsEmailConfigurationUpToDate() with a nil spec = false, want true")
	}

	if IsEmailConfigurationUpToDate(basicEmailSpec(), nil, basic, basic) {
		t.Error("IsEmailConfigurationUpToDate() with a nil observation = true, want false")
	}
}

// TestGenerateEmailConfigurationOptions tests the create and update options.
func TestGenerateEmailConfigurationOptions(t *testing.T) {
	t.Parallel()

	credentials := EmailConfigurationCredentials{BasicPassword: "password", OAuthClientID: "id", OAuthClientSecret: "secret"}

	cases := map[string]struct {
		spec *v1alpha1.EmailConfigurationParameters
		want sonar.SystemEmailConfigurationCreateOptions
	}{
		"Basic": {
			spec: func() *v1alpha1.EmailConfigurationParameters {
				spec := basicEmailSpec()
				spec.OAuthTenant = new("ignored")

				return spec
			}(),
			want: sonar.SystemEmailConfigurationCreateOptions{
				Host: "smtp.example.com", Port: "587", SecurityProtocol: "STARTTLS",
				FromAddress: "sonarqube@example.com", FromName: "SonarQube", SubjectPrefix: "[SonarQube]",
				AuthMethod: "BASIC", Username: "user", BasicPassword: "password",
			},
		},
		"OAuth": {
			spec: oauthEmailSpec(),
			want: sonar.SystemEmailConfigurationCreateOptions{
				Host: "smtp.example.com", Port: "587", SecurityProtocol: "STARTTLS",
				FromAddress: "sonarqube@example.com", FromName: "SonarQube", SubjectPrefix: "[SonarQube]",
				AuthMethod: "OAUTH", Username: "user", BasicPassword: "password",
				OauthAuthenticationHost: "https://login.example.com", OauthTenant: "tenant", OauthScope: "scope",
				OauthClientId: "id", OauthClientSecret: "secret",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(&tc.want, GenerateEmailConfigurationCreateOptions(tc.spec, credentials)); diff != "" {
				t.Errorf("GenerateEmailConfigurationCreateOptions() -want, +got:\n%s", diff)
			}

			want := sonar.SystemEmailConfigurationUpdateOptions(tc.want)
			if diff := cmp.Diff(&want, GenerateEmailConfigurationUpdateOptions(tc.spec, credentials)); diff != "" {
				t.Errorf("GenerateEmailConfigurationUpdateOptions() -want, +got:\n%s", diff)
			}
		})
	}

	if GenerateEmailConfigurationCreateOptions(nil, credentials) != nil {
		t.Error("GenerateEmailConfigurationCreateOptions(nil) != nil")
	}

	if GenerateEmailConfigurationUpdateOptions(nil, credentials) != nil {
		t.Error("GenerateEmailConfigurationUpdateOptions(nil) != nil")
	}
}

// TestGenerateEmailConfigurationObservation tests the observation.
func TestGenerateEmailConfigurationObservation(t *testing.T) {
	t.Parallel()

	if diff := cmp.Diff(v1alpha1.EmailConfigurationObservation{}, GenerateEmailConfigurationObservation(nil)); diff != "" {
		t.Errorf("GenerateEmailConfigurationObservation(nil) -want, +got:\n%s", diff)
	}

	want := v1alpha1.EmailConfigurationObservation{
		ID: "id", Host: "smtp.example.com", Port: "587", SecurityProtocol: "STARTTLS",
		FromAddress: "sonarqube@example.com", FromName: "SonarQube", SubjectPrefix: "[SonarQube]",
		AuthMethod: "OAUTH", Username: "user", IsBasicPasswordSet: true,
		OAuthAuthenticationHost: "https://login.example.com", OAuthTenant: "tenant", OAuthScope: "scope",
		IsOAuthClientIDSet: true, IsOAuthClientSecretSet: true,
	}

	if diff := cmp.Diff(want, GenerateEmailConfigurationObservation(observedEmailConfiguration(oauthEmailSpec()))); diff != "" {
		t.Errorf("GenerateEmailConfigurationObservation() -want, +got:\n%s", diff)
	}
}

// TestNewEmailConfigurationsClient tests the client constructor.
func TestNewEmailConfigurationsClient(t *testing.T) {
	t.Parallel()

	if NewEmailConfigurationsClient(newTestConfig()) == nil {
		t.Error("NewEmailConfigurationsClient() = nil")
	}
}
