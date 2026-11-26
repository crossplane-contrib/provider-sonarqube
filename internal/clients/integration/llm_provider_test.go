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

package integration

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"k8s.io/utils/ptr"

	"github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// testLLMProviderID is the provider ID used in tests.
const testLLMProviderID = "llm-provider-1"

// testOpenAIDefinition returns a definition with a secret, a required and an
// optional string field, but no HTTP headers.
func testOpenAIDefinition() *sonar.LlmConnectivityLlmProviderDefinitionResponse {
	return &sonar.LlmConnectivityLlmProviderDefinitionResponse{
		Provider: "OPENAI",
		Fields: []sonar.LlmConnectivityLlmProviderFieldResponse{
			{Key: "apiKey", Type: "STRING", Required: true, Secret: true},
			{Key: "endpoint", Type: "STRING", Required: true},
			{Key: "organization", Type: "STRING"},
		},
	}
}

// testCustomProxyDefinition returns a definition with required HTTP headers.
func testCustomProxyDefinition() *sonar.LlmConnectivityLlmProviderDefinitionResponse {
	return &sonar.LlmConnectivityLlmProviderDefinitionResponse{
		Provider: "CUSTOM_PROXY",
		Fields: []sonar.LlmConnectivityLlmProviderFieldResponse{
			{Key: "url", Type: "STRING", Required: true},
			{Key: "headers", Type: "HTTP_HEADERS", Required: true},
		},
	}
}

// testSecretRef returns a LocalSecretKeySelector.
func testSecretRef(name, key string) xpv1.LocalSecretKeySelector {
	return xpv1.LocalSecretKeySelector{LocalSecretReference: xpv1.LocalSecretReference{Name: name}, Key: key}
}

// testOpenAISpec returns a valid spec for testOpenAIDefinition.
func testOpenAISpec() *v1alpha1.LLMProviderParameters {
	return &v1alpha1.LLMProviderParameters{
		Provider:                "OPENAI",
		Label:                   "OpenAI",
		Configuration:           map[string]string{"endpoint": "https://api.openai.com"},
		ConfigurationSecretRefs: map[string]xpv1.LocalSecretKeySelector{"apiKey": testSecretRef("openai", "apiKey")},
		ModelMappings: []v1alpha1.LLMProviderModelMapping{
			{AICapability: "AI_CODEFIX", ModelIdentifier: "gpt-4o"},
		},
	}
}

// testCustomProxySpec returns a valid spec for testCustomProxyDefinition,
// with a plain and a secret header.
func testCustomProxySpec() *v1alpha1.LLMProviderParameters {
	return &v1alpha1.LLMProviderParameters{
		Provider:      "CUSTOM_PROXY",
		Label:         "Proxy",
		Configuration: map[string]string{"url": "https://proxy.example.com"},
		HTTPHeaders: []v1alpha1.LLMProviderHTTPHeader{
			{Name: "X-Team", Value: new("platform")},
			{Name: "Authorization", ValueSecretRef: new(testSecretRef("proxy", "auth"))},
		},
	}
}

// TestNewLLMConnectivityClient tests the client constructor.
func TestNewLLMConnectivityClient(t *testing.T) {
	t.Parallel()

	got := NewLLMConnectivityClient(common.Config{AuthType: common.PersonalAccessToken, Token: "token", BaseURL: "http://localhost:9000"})
	if got == nil {
		t.Fatal("NewLLMConnectivityClient() returned nil")
	}
}

// TestFindLLMProviderByID tests looking a provider up by ID.
func TestFindLLMProviderByID(t *testing.T) {
	t.Parallel()

	providers := &sonar.LlmConnectivityLlmProvidersResponse{Providers: []sonar.LlmConnectivityLlmProviderResponse{{Id: "a"}, {Id: "b"}}}

	if got := FindLLMProviderByID(providers, "b"); got == nil || got.Id != "b" {
		t.Errorf("FindLLMProviderByID(b) = %v, want b", got)
	}

	if got := FindLLMProviderByID(providers, "c"); got != nil {
		t.Errorf("FindLLMProviderByID(c) = %v, want nil", got)
	}

	if got := FindLLMProviderByID(nil, "a"); got != nil {
		t.Errorf("FindLLMProviderByID(nil) = %v, want nil", got)
	}
}

// TestFindLLMProviderDefinition tests looking a definition up by type.
func TestFindLLMProviderDefinition(t *testing.T) {
	t.Parallel()

	definitions := &sonar.LlmConnectivityLlmProviderDefinitionsResponse{
		ProviderDefinitions: []sonar.LlmConnectivityLlmProviderDefinitionResponse{*testOpenAIDefinition(), *testCustomProxyDefinition()},
	}

	if got := FindLLMProviderDefinition(definitions, "CUSTOM_PROXY"); got == nil || got.Provider != "CUSTOM_PROXY" {
		t.Errorf("FindLLMProviderDefinition(CUSTOM_PROXY) = %v", got)
	}

	if got := FindLLMProviderDefinition(definitions, "ANTHROPIC"); got != nil {
		t.Errorf("FindLLMProviderDefinition(ANTHROPIC) = %v, want nil", got)
	}

	if got := FindLLMProviderDefinition(nil, "OPENAI"); got != nil {
		t.Errorf("FindLLMProviderDefinition(nil) = %v, want nil", got)
	}
}

// TestFindLLMProviderMappings tests filtering and sorting mappings.
func TestFindLLMProviderMappings(t *testing.T) {
	t.Parallel()

	mappings := &sonar.LlmConnectivityProviderMappingsResponse{ProviderMappings: []sonar.LlmConnectivityProviderMappingResponse{
		{AiCapability: "REMEDIATION_AGENT", LlmProviderId: testLLMProviderID},
		{AiCapability: "HUNTER_AGENT", LlmProviderId: "other"},
		{AiCapability: "AI_CODEFIX", LlmProviderId: testLLMProviderID},
	}}

	got := FindLLMProviderMappings(mappings, testLLMProviderID)
	if len(got) != 2 || got[0].AiCapability != "AI_CODEFIX" || got[1].AiCapability != "REMEDIATION_AGENT" {
		t.Errorf("FindLLMProviderMappings() = %v, want AI_CODEFIX then REMEDIATION_AGENT", got)
	}

	if got := FindLLMProviderMappings(nil, testLLMProviderID); got != nil {
		t.Errorf("FindLLMProviderMappings(nil) = %v, want nil", got)
	}
}

// TestValidateLLMProviderSpec tests validating a spec against a definition.
func TestValidateLLMProviderSpec(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		spec       func() *v1alpha1.LLMProviderParameters
		definition *sonar.LlmConnectivityLlmProviderDefinitionResponse
		wantErrs   []string
	}{
		"Valid": {
			spec:       testOpenAISpec,
			definition: testOpenAIDefinition(),
		},
		"ValidWithHeaders": {
			spec:       testCustomProxySpec,
			definition: testCustomProxyDefinition(),
		},
		"UnknownType": {
			spec:     testOpenAISpec,
			wantErrs: []string{`does not support LLM provider type "OPENAI"`},
		},
		"UnknownKeys": {
			spec: func() *v1alpha1.LLMProviderParameters {
				spec := testOpenAISpec()
				spec.Configuration["region"] = "eu"
				spec.ConfigurationSecretRefs["token"] = testSecretRef("openai", "token")

				return spec
			},
			definition: testOpenAIDefinition(),
			wantErrs: []string{
				`unknown configuration key "region"`,
				`unknown configurationSecretRefs key "token"`,
				"supported keys: apiKey, endpoint, organization",
			},
		},
		"PlainSecretAndMissingRequired": {
			spec: func() *v1alpha1.LLMProviderParameters {
				spec := testOpenAISpec()
				spec.Configuration = map[string]string{"apiKey": "sk-plain"}
				spec.ConfigurationSecretRefs = nil

				return spec
			},
			definition: testOpenAIDefinition(),
			wantErrs: []string{
				`configuration key "apiKey" is secret and must be set in configurationSecretRefs`,
				`required configuration key "endpoint" is missing`,
			},
		},
		"MissingRequiredHeaders": {
			spec: func() *v1alpha1.LLMProviderParameters {
				spec := testCustomProxySpec()
				spec.HTTPHeaders = nil

				return spec
			},
			definition: testCustomProxyDefinition(),
			wantErrs:   []string{`required field "headers" is missing: set httpHeaders`},
		},
		"HeadersNotAccepted": {
			spec: func() *v1alpha1.LLMProviderParameters {
				spec := testOpenAISpec()
				spec.HTTPHeaders = []v1alpha1.LLMProviderHTTPHeader{{Name: "X-Team", Value: new("platform")}}

				return spec
			},
			definition: testOpenAIDefinition(),
			wantErrs:   []string{`LLM provider type "OPENAI" does not accept httpHeaders`},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := ValidateLLMProviderSpec(tc.spec(), tc.definition)

			if len(tc.wantErrs) == 0 {
				if err != nil {
					t.Fatalf("ValidateLLMProviderSpec() unexpected error: %v", err)
				}

				return
			}

			if err == nil {
				t.Fatalf("ValidateLLMProviderSpec() expected an error containing %q", tc.wantErrs)
			}

			for _, want := range tc.wantErrs {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("ValidateLLMProviderSpec() error = %q, want substring %q", err.Error(), want)
				}
			}
		})
	}
}

// TestGenerateLLMProviderCreateOptions tests building the create request.
func TestGenerateLLMProviderCreateOptions(t *testing.T) {
	t.Parallel()

	t.Run("SecretConfiguration", func(t *testing.T) {
		t.Parallel()

		secrets := LLMProviderSecrets{Configuration: map[string]string{"apiKey": "sk-secret"}}

		got := GenerateLLMProviderCreateOptions(testOpenAISpec(), testOpenAIDefinition(), secrets)

		want := &sonar.LlmConnectivityCreateProviderOptions{
			Provider:      "OPENAI",
			Label:         "OpenAI",
			Configuration: map[string]any{"endpoint": "https://api.openai.com", "apiKey": "sk-secret"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("GenerateLLMProviderCreateOptions() = %#v, want %#v", got, want)
		}
	})

	t.Run("Headers", func(t *testing.T) {
		t.Parallel()

		secrets := LLMProviderSecrets{HTTPHeaders: map[string]string{"Authorization": "Bearer t"}}

		got := GenerateLLMProviderCreateOptions(testCustomProxySpec(), testCustomProxyDefinition(), secrets)

		wantHeaders := []map[string]any{
			{"name": "X-Team", "value": "platform", "secret": false},
			{"name": "Authorization", "value": "Bearer t", "secret": true},
		}
		if !reflect.DeepEqual(got.Configuration["headers"], wantHeaders) {
			t.Errorf("headers = %#v, want %#v", got.Configuration["headers"], wantHeaders)
		}
	})

	t.Run("NoHeadersOmitted", func(t *testing.T) {
		t.Parallel()

		spec := testCustomProxySpec()
		spec.HTTPHeaders = nil

		got := GenerateLLMProviderCreateOptions(spec, testCustomProxyDefinition(), LLMProviderSecrets{})
		if _, set := got.Configuration["headers"]; set {
			t.Errorf("headers = %#v, want unset", got.Configuration["headers"])
		}
	})
}

// TestGenerateLLMProviderUpdateOptions tests building the update request,
// which clears the headers removed from the spec.
func TestGenerateLLMProviderUpdateOptions(t *testing.T) {
	t.Parallel()

	spec := testCustomProxySpec()
	spec.HTTPHeaders = nil

	got := GenerateLLMProviderUpdateOptions(spec, testCustomProxyDefinition(), LLMProviderSecrets{})

	if ptr.Deref(got.Label, "") != "Proxy" {
		t.Errorf("Label = %v, want Proxy", got.Label)
	}

	headers, isList := got.Configuration["headers"].([]map[string]any)
	if !isList || len(headers) != 0 {
		t.Errorf("headers = %#v, want an empty list", got.Configuration["headers"])
	}

	if got.Configuration["url"] != "https://proxy.example.com" {
		t.Errorf("url = %v, want https://proxy.example.com", got.Configuration["url"])
	}
}

// TestGenerateLLMProviderMappingUpsertOptions tests building a mapping
// request.
func TestGenerateLLMProviderMappingUpsertOptions(t *testing.T) {
	t.Parallel()

	got := GenerateLLMProviderMappingUpsertOptions(testLLMProviderID, &v1alpha1.LLMProviderModelMapping{
		AICapability:             "HUNTER_AGENT",
		ModelIdentifier:          "primary",
		SecondaryModelIdentifier: new("secondary"),
	})

	want := &sonar.LlmConnectivityUpsertProviderMappingOptions{
		AiCapability:             "HUNTER_AGENT",
		LlmProviderId:            testLLMProviderID,
		ModelIdentifier:          "primary",
		SecondaryModelIdentifier: "secondary",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GenerateLLMProviderMappingUpsertOptions() = %#v, want %#v", got, want)
	}
}

// TestGenerateLLMProviderObservation tests that the observation never
// reports secrets.
func TestGenerateLLMProviderObservation(t *testing.T) {
	t.Parallel()

	t.Run("Nil", func(t *testing.T) {
		t.Parallel()

		got := GenerateLLMProviderObservation(nil, nil, nil)
		if !reflect.DeepEqual(got, v1alpha1.LLMProviderObservation{}) {
			t.Errorf("GenerateLLMProviderObservation(nil) = %#v, want empty", got)
		}
	})

	t.Run("StringFields", func(t *testing.T) {
		t.Parallel()

		provider := &sonar.LlmConnectivityLlmProviderResponse{
			Id:       testLLMProviderID,
			Provider: "OPENAI",
			Label:    "OpenAI",
			Configuration: map[string]any{
				"apiKey":       "sk-unmasked",
				"endpoint":     "https://api.openai.com",
				"organization": nil,
				"undeclared":   "x",
			},
		}
		mappings := []sonar.LlmConnectivityProviderMappingResponse{{AiCapability: "AI_CODEFIX", LlmProviderId: testLLMProviderID, ModelIdentifier: "gpt-4o"}}

		got := GenerateLLMProviderObservation(provider, testOpenAIDefinition(), mappings)

		want := v1alpha1.LLMProviderObservation{
			ID:            testLLMProviderID,
			Provider:      "OPENAI",
			Label:         "OpenAI",
			Configuration: map[string]string{"endpoint": "https://api.openai.com"},
			ModelMappings: []v1alpha1.LLMProviderModelMappingObservation{{AICapability: "AI_CODEFIX", ModelIdentifier: "gpt-4o"}},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("GenerateLLMProviderObservation() = %#v, want %#v", got, want)
		}
	})

	t.Run("Headers", func(t *testing.T) {
		t.Parallel()

		provider := &sonar.LlmConnectivityLlmProviderResponse{
			Id:       testLLMProviderID,
			Provider: "CUSTOM_PROXY",
			Configuration: map[string]any{
				"url": "https://proxy.example.com",
				"headers": []any{
					map[string]any{"name": "X-Team", "value": "platform", "secret": false},
					map[string]any{"name": "Authorization", "value": "****", "secret": true},
					"not-an-object",
				},
			},
		}

		got := GenerateLLMProviderObservation(provider, testCustomProxyDefinition(), nil)

		want := []v1alpha1.LLMProviderHTTPHeaderObservation{
			{Name: "X-Team", Value: "platform"},
			{Name: "Authorization", Secret: true},
		}
		if !reflect.DeepEqual(got.HTTPHeaders, want) {
			t.Errorf("HTTPHeaders = %#v, want %#v", got.HTTPHeaders, want)
		}
	})

	t.Run("HeadersNotAList", func(t *testing.T) {
		t.Parallel()

		provider := &sonar.LlmConnectivityLlmProviderResponse{Configuration: map[string]any{"headers": "oops", "url": 42}}

		got := GenerateLLMProviderObservation(provider, testCustomProxyDefinition(), nil)
		if got.HTTPHeaders != nil || got.Configuration["url"] != "42" {
			t.Errorf("GenerateLLMProviderObservation() = %#v, want no headers and url 42", got)
		}
	})

	t.Run("UnknownDefinition", func(t *testing.T) {
		t.Parallel()

		provider := &sonar.LlmConnectivityLlmProviderResponse{Id: testLLMProviderID, Configuration: map[string]any{"apiKey": "sk"}}

		got := GenerateLLMProviderObservation(provider, nil, nil)
		if got.Configuration != nil {
			t.Errorf("Configuration = %#v, want nil without a definition", got.Configuration)
		}
	})
}

// testOpenAIObservation returns the observation matching testOpenAISpec.
func testOpenAIObservation() *v1alpha1.LLMProviderObservation {
	return &v1alpha1.LLMProviderObservation{
		ID:            testLLMProviderID,
		Provider:      "OPENAI",
		Label:         "OpenAI",
		Configuration: map[string]string{"endpoint": "https://api.openai.com"},
		ModelMappings: []v1alpha1.LLMProviderModelMappingObservation{{AICapability: "AI_CODEFIX", ModelIdentifier: "gpt-4o"}},
	}
}

// TestIsLLMProviderUpToDate tests drift detection.
func TestIsLLMProviderUpToDate(t *testing.T) {
	t.Parallel()

	secrets := LLMProviderSecrets{Configuration: map[string]string{"apiKey": "sk-1"}}
	rotated := LLMProviderSecrets{Configuration: map[string]string{"apiKey": "sk-2"}}

	cases := map[string]struct {
		spec        *v1alpha1.LLMProviderParameters
		observation func() *v1alpha1.LLMProviderObservation
		current     LLMProviderSecrets
		want        bool
	}{
		"UpToDate": {
			spec:        testOpenAISpec(),
			observation: testOpenAIObservation,
			current:     secrets,
			want:        true,
		},
		"NilSpec": {
			observation: testOpenAIObservation,
			want:        true,
		},
		"NilObservation": {
			spec:        testOpenAISpec(),
			observation: func() *v1alpha1.LLMProviderObservation { return nil },
			want:        false,
		},
		"ProviderTypeDiffers": {
			spec: testOpenAISpec(),
			observation: func() *v1alpha1.LLMProviderObservation {
				observation := testOpenAIObservation()
				observation.Provider = "ANTHROPIC"

				return observation
			},
			current: secrets,
		},
		"LabelDiffers": {
			spec: testOpenAISpec(),
			observation: func() *v1alpha1.LLMProviderObservation {
				observation := testOpenAIObservation()
				observation.Label = "Old"

				return observation
			},
			current: secrets,
		},
		"ConfigurationDiffers": {
			spec: testOpenAISpec(),
			observation: func() *v1alpha1.LLMProviderObservation {
				observation := testOpenAIObservation()
				observation.Configuration = map[string]string{"endpoint": "https://other"}

				return observation
			},
			current: secrets,
		},
		"ConfigurationMissing": {
			spec: testOpenAISpec(),
			observation: func() *v1alpha1.LLMProviderObservation {
				observation := testOpenAIObservation()
				observation.Configuration = nil

				return observation
			},
			current: secrets,
		},
		"SecretRotated": {
			spec:        testOpenAISpec(),
			observation: testOpenAIObservation,
			current:     rotated,
		},
		"MappingMissing": {
			spec: testOpenAISpec(),
			observation: func() *v1alpha1.LLMProviderObservation {
				observation := testOpenAIObservation()
				observation.ModelMappings = nil

				return observation
			},
			current: secrets,
		},
		"MappingModelDiffers": {
			spec: testOpenAISpec(),
			observation: func() *v1alpha1.LLMProviderObservation {
				observation := testOpenAIObservation()
				observation.ModelMappings[0].ModelIdentifier = "gpt-3.5"

				return observation
			},
			current: secrets,
		},
		"ExtraObservedMappingIgnored": {
			spec: testOpenAISpec(),
			observation: func() *v1alpha1.LLMProviderObservation {
				observation := testOpenAIObservation()
				observation.ModelMappings = append(observation.ModelMappings, v1alpha1.LLMProviderModelMappingObservation{AICapability: "REMEDIATION_AGENT", ModelIdentifier: "m"})

				return observation
			},
			current: secrets,
			want:    true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsLLMProviderUpToDate(tc.spec, tc.observation(), tc.current, secrets); got != tc.want {
				t.Errorf("IsLLMProviderUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsLLMProviderConfigurationUpToDateHeaders tests HTTP header drift
// detection.
func TestIsLLMProviderConfigurationUpToDateHeaders(t *testing.T) {
	t.Parallel()

	saved := LLMProviderSecrets{HTTPHeaders: map[string]string{"Authorization": "Bearer t"}}
	upToDate := func() []v1alpha1.LLMProviderHTTPHeaderObservation {
		return []v1alpha1.LLMProviderHTTPHeaderObservation{
			{Name: "X-Team", Value: "platform"},
			{Name: "Authorization", Secret: true},
		}
	}

	cases := map[string]struct {
		headers func() []v1alpha1.LLMProviderHTTPHeaderObservation
		current LLMProviderSecrets
		want    bool
	}{
		"UpToDate": {headers: upToDate, current: saved, want: true},
		"Missing": {
			headers: func() []v1alpha1.LLMProviderHTTPHeaderObservation { return upToDate()[:1] },
			current: saved,
		},
		"Renamed": {
			headers: func() []v1alpha1.LLMProviderHTTPHeaderObservation {
				headers := upToDate()
				headers[0].Name = "X-Other"

				return headers
			},
			current: saved,
		},
		"PlainValueDiffers": {
			headers: func() []v1alpha1.LLMProviderHTTPHeaderObservation {
				headers := upToDate()
				headers[0].Value = "other"

				return headers
			},
			current: saved,
		},
		"PlainHeaderSecretInSonarQube": {
			headers: func() []v1alpha1.LLMProviderHTTPHeaderObservation {
				headers := upToDate()
				headers[0] = v1alpha1.LLMProviderHTTPHeaderObservation{Name: "X-Team", Secret: true}

				return headers
			},
			current: saved,
		},
		"SecretHeaderPlainInSonarQube": {
			headers: func() []v1alpha1.LLMProviderHTTPHeaderObservation {
				headers := upToDate()
				headers[1] = v1alpha1.LLMProviderHTTPHeaderObservation{Name: "Authorization", Value: "Bearer t"}

				return headers
			},
			current: saved,
		},
		"SecretHeaderRotated": {
			headers: upToDate,
			current: LLMProviderSecrets{HTTPHeaders: map[string]string{"Authorization": "Bearer new"}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			observation := &v1alpha1.LLMProviderObservation{
				Label:         "Proxy",
				Configuration: map[string]string{"url": "https://proxy.example.com"},
				HTTPHeaders:   tc.headers(),
			}

			if got := IsLLMProviderConfigurationUpToDate(testCustomProxySpec(), observation, tc.current, saved); got != tc.want {
				t.Errorf("IsLLMProviderConfigurationUpToDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsLLMProviderConfigurationUpToDateNil tests the nil guards.
func TestIsLLMProviderConfigurationUpToDateNil(t *testing.T) {
	t.Parallel()

	if !IsLLMProviderConfigurationUpToDate(nil, nil, LLMProviderSecrets{}, LLMProviderSecrets{}) {
		t.Error("IsLLMProviderConfigurationUpToDate(nil spec) = false, want true")
	}

	if IsLLMProviderConfigurationUpToDate(testOpenAISpec(), nil, LLMProviderSecrets{}, LLMProviderSecrets{}) {
		t.Error("IsLLMProviderConfigurationUpToDate(nil observation) = true, want false")
	}
}

// TestOutdatedLLMProviderModelMappings tests listing the mappings to apply.
func TestOutdatedLLMProviderModelMappings(t *testing.T) {
	t.Parallel()

	spec := &v1alpha1.LLMProviderParameters{ModelMappings: []v1alpha1.LLMProviderModelMapping{
		{AICapability: "AI_CODEFIX", ModelIdentifier: "gpt-4o"},
		{AICapability: "HUNTER_AGENT", ModelIdentifier: "a", SecondaryModelIdentifier: new("b")},
	}}
	observation := &v1alpha1.LLMProviderObservation{ModelMappings: []v1alpha1.LLMProviderModelMappingObservation{
		{AICapability: "AI_CODEFIX", ModelIdentifier: "gpt-4o"},
		{AICapability: "HUNTER_AGENT", ModelIdentifier: "a", SecondaryModelIdentifier: "c"},
	}}

	got := OutdatedLLMProviderModelMappings(spec, observation)
	if len(got) != 1 || got[0].AICapability != "HUNTER_AGENT" {
		t.Errorf("OutdatedLLMProviderModelMappings() = %v, want HUNTER_AGENT", got)
	}

	if got := OutdatedLLMProviderModelMappings(spec, nil); len(got) != 2 {
		t.Errorf("OutdatedLLMProviderModelMappings(nil observation) = %v, want both", got)
	}

	if got := OutdatedLLMProviderModelMappings(nil, observation); got != nil {
		t.Errorf("OutdatedLLMProviderModelMappings(nil spec) = %v, want nil", got)
	}
}

// TestCheckLLMProviderDeletable tests the mapped provider deletion guard.
func TestCheckLLMProviderDeletable(t *testing.T) {
	t.Parallel()

	mappings := &sonar.LlmConnectivityProviderMappingsResponse{ProviderMappings: []sonar.LlmConnectivityProviderMappingResponse{
		{AiCapability: "AI_CODEFIX", LlmProviderId: testLLMProviderID},
		{AiCapability: "HUNTER_AGENT"},
	}}

	err := CheckLLMProviderDeletable(mappings, testLLMProviderID)
	if !errors.Is(err, ErrLLMProviderMapped) || !strings.Contains(err.Error(), "AI_CODEFIX") {
		t.Errorf("CheckLLMProviderDeletable() = %v, want ErrLLMProviderMapped naming AI_CODEFIX", err)
	}

	err = CheckLLMProviderDeletable(mappings, "other")
	if err != nil {
		t.Errorf("CheckLLMProviderDeletable(other) = %v, want nil", err)
	}
}

// TestLLMProviderSecretsConnectionDetails tests the connection detail keys.
func TestLLMProviderSecretsConnectionDetails(t *testing.T) {
	t.Parallel()

	secrets := LLMProviderSecrets{
		Configuration: map[string]string{"apiKey": "sk"},
		HTTPHeaders:   map[string]string{"Authorization": "Bearer t"},
	}

	got := secrets.ConnectionDetails()
	if string(got["configuration.apiKey"]) != "sk" || string(got["httpHeader.Authorization"]) != "Bearer t" || len(got) != 2 {
		t.Errorf("ConnectionDetails() = %v", got)
	}
}
