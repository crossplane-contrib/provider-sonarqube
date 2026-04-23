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

//go:build e2e

package instance_test

import (
	"strings"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// TestRuleCRUD creates a custom rule based on the java:S124 template
// (comment-regexp matcher — stock in SonarJava), waits for Ready, and
// verifies the rule exists in SonarQube with the expected name. SonarQube
// prefixes the custom rule key with the template's language+repo, so we
// look up by key with a java: prefix.
func TestRuleCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName      = "e2e-rule-crud"
		ruleKey     = "e2e_rule_crud"
		ruleName    = "E2E Rule CRUD"
		templateKey = "java:S124"
		markdown    = "E2E-managed rule used by the framework test suite."
	)

	rule := &instancev1alpha1.Rule{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: "default"},
		Spec: instancev1alpha1.RuleSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.RuleParameters{
				Key:                 ruleKey,
				Name:                ruleName,
				TemplateKey:         templateKey,
				MarkdownDescription: markdown,
				Status:              stringPtr("READY"),
				Type:                stringPtr("CODE_SMELL"),
			},
		},
	}

	f.CreateAndWaitForReady(t, rule, 2*time.Minute)
	e2e.AssertReady(t, rule)
	e2e.AssertSynced(t, rule)

	// The provider stores the fully-qualified key (e.g. "java:e2e_rule_crud")
	// in the external-name annotation. Use it to query SonarQube directly.
	externalName := rule.GetAnnotations()["crossplane.io/external-name"]
	if externalName == "" {
		t.Fatalf("expected external-name to be populated after Ready")
	}
	if !strings.HasSuffix(externalName, ":"+ruleKey) {
		t.Errorf("external-name = %q, want it to end with :%s", externalName, ruleKey)
	}

	got, err := f.FetchRule(externalName)
	if err != nil {
		t.Fatalf("fetching rule: %v", err)
	}
	if got == nil {
		t.Fatalf("rule %q not found in SonarQube", externalName)
	}
	if got.Name != ruleName {
		t.Errorf("rule name = %q, want %q", got.Name, ruleName)
	}
	if got.TemplateKey != templateKey {
		t.Errorf("rule templateKey = %q, want %q", got.TemplateKey, templateKey)
	}
}
