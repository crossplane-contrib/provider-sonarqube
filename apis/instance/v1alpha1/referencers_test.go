// Copyright 2026 The Crossplane Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"context"
	"strings"
	"testing"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/test"
)

// newTestProject creates a test Project resource.
func newTestProject(namespace string) *Project {
	return &Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project",
			Namespace: namespace,
		},
		Spec: ProjectSpec{
			ForProvider: ProjectParameters{
				Key:  "test-key",
				Name: "Test Project",
			},
		},
	}
}

// newTestQualityProfile creates a test QualityProfile resource.
func newTestQualityProfile(namespace string) *QualityProfile {
	return &QualityProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-qualityprofile",
			Namespace: namespace,
		},
		Spec: QualityProfileSpec{
			ForProvider: QualityProfileParameters{
				Name:     "Go Quality Profile",
				Language: "go",
			},
		},
	}
}

// setExtName sets the external name on a Kubernetes object.
func setExtName(obj metav1.Object, name string) {
	meta.SetExternalName(obj, name)
}

// testNamespace is the test namespace for resources.
const testNamespace = "default"

// TestQualityProfileResolveReferences tests quality profile
// reference resolution.
func TestQualityProfileResolveReferences(t *testing.T) { //nolint:gocognit,maintidx // Exhaustive table-driven test intentionally covers many reference-resolution scenarios.
	t.Parallel()

	notFoundRule := kerrors.NewNotFound(
		schema.GroupResource{Group: "instance.sonarqube.crossplane.io", Resource: "rules"},
		"wrong-rule-name",
	)

	cases := map[string]struct {
		reason    string
		profile   *QualityProfile
		client    client.Reader
		wantRules map[int]string
		errSubstr string
	}{
		"NoRules_NoOp": {
			reason:  "A quality profile with no rules resolves with no error.",
			profile: newTestQualityProfile(testNamespace),
			client:  test.NewMockClient(),
		},
		"RuleAlreadySet_NoRefOrSelector_Preserved": {
			reason: "When Rule is already set and no ref/selector exists, it is preserved.",
			profile: func() *QualityProfile {
				qp := newTestQualityProfile(testNamespace)
				qp.Spec.ForProvider.Rules = []QualityProfileRuleParameters{{Rule: ptr.To("go:S100")}}

				return qp
			}(),
			client: test.NewMockClient(),
			wantRules: map[int]string{
				0: "go:S100",
			},
		},
		"RuleRef_ResolvesToExternalName": {
			reason: "RuleRef.Name uses the Kubernetes object name and resolves to external-name.",
			profile: func() *QualityProfile {
				qp := newTestQualityProfile(testNamespace)
				qp.Spec.ForProvider.Rules = []QualityProfileRuleParameters{{
					RuleRef: &xpv1.NamespacedReference{Name: "example-rule"},
				}}

				return qp
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, "go:S4784")

					return nil
				}),
			},
			wantRules: map[int]string{
				0: "go:S4784",
			},
		},
		"RuleRef_WithExplicitNamespace": {
			reason: "An explicit namespace in RuleRef is respected.",
			profile: func() *QualityProfile {
				qp := newTestQualityProfile(testNamespace)
				qp.Spec.ForProvider.Rules = []QualityProfileRuleParameters{{
					RuleRef: &xpv1.NamespacedReference{Name: "example-rule", Namespace: testNamespace},
				}}

				return qp
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, "java:S2386")

					return nil
				}),
			},
			wantRules: map[int]string{
				0: "java:S2386",
			},
		},
		"RuleRef_WrongName_ReturnsError": {
			reason: "Using a non-existing Kubernetes rule object name returns not found.",
			profile: func() *QualityProfile {
				qp := newTestQualityProfile(testNamespace)
				qp.Spec.ForProvider.Rules = []QualityProfileRuleParameters{{
					RuleRef: &xpv1.NamespacedReference{Name: "sonarqube-rule-key-instead-of-k8s-name"},
				}}

				return qp
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(notFoundRule),
			},
			errSubstr: "spec.forProvider.rules.rule",
		},
		"RuleSelector_ResolvesToExternalName": {
			reason: "A selector lists matching Rules and resolves the external-name.",
			profile: func() *QualityProfile {
				qp := newTestQualityProfile(testNamespace)
				qp.Spec.ForProvider.Rules = []QualityProfileRuleParameters{{
					RuleSelector: &xpv1.NamespacedSelector{MatchLabels: map[string]string{"language": "go"}},
				}}

				return qp
			}(),
			client: &test.MockClient{
				MockList: test.NewMockListFn(nil, func(obj client.ObjectList) error {
					ruleList, ok := obj.(*RuleList)
					if !ok {
						return nil
					}

					r := Rule{ObjectMeta: metav1.ObjectMeta{Name: "example-rule", Namespace: testNamespace}}
					setExtName(&r, "py:S101")
					ruleList.Items = []Rule{r}

					return nil
				}),
			},
			wantRules: map[int]string{
				0: "py:S101",
			},
		},
		"RuleRef_StaleCachedK8sName_IsOverwrittenByAnnotation": {
			reason: "A stale cached Rule value (K8s name) must be overwritten when RuleRef is set.",
			profile: func() *QualityProfile {
				qp := newTestQualityProfile(testNamespace)
				qp.Spec.ForProvider.Rules = []QualityProfileRuleParameters{{
					Rule:    ptr.To("example-rule"),
					RuleRef: &xpv1.NamespacedReference{Name: "example-rule"},
				}}

				return qp
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, "ts:S1128")

					return nil
				}),
			},
			wantRules: map[int]string{
				0: "ts:S1128",
			},
		},
		"MultipleRules_MixedInputs_AllResolvedIndependently": {
			reason: "Direct, ref and selector based rules are each resolved and retained independently.",
			profile: func() *QualityProfile {
				qp := newTestQualityProfile(testNamespace)
				qp.Spec.ForProvider.Rules = []QualityProfileRuleParameters{
					{Rule: ptr.To("go:S100")},
					{RuleRef: &xpv1.NamespacedReference{Name: "rule-ref-java"}},
					{RuleSelector: &xpv1.NamespacedSelector{MatchLabels: map[string]string{"team": "sec"}}},
				}

				return qp
			}(),
			client: &test.MockClient{
				MockGet: func(_ context.Context, key client.ObjectKey, obj client.Object) error {
					if key.Name == "rule-ref-java" {
						setExtName(obj, "java:S2095")

						return nil
					}

					return kerrors.NewNotFound(schema.GroupResource{}, key.Name)
				},
				MockList: test.NewMockListFn(nil, func(obj client.ObjectList) error {
					ruleList, ok := obj.(*RuleList)
					if !ok {
						return nil
					}

					r := Rule{ObjectMeta: metav1.ObjectMeta{Name: "selector-rule", Namespace: testNamespace}}
					setExtName(&r, "java:S1118")
					ruleList.Items = []Rule{r}

					return nil
				}),
			},
			wantRules: map[int]string{
				0: "go:S100",
				1: "java:S2095",
				2: "java:S1118",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tc.profile.ResolveReferences(context.Background(), tc.client)

			if tc.errSubstr != "" {
				if err == nil {
					t.Errorf("[%s] (%s) expected error containing %q, got nil", name, tc.reason, tc.errSubstr)

					return
				}

				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("[%s] (%s) error %q does not contain %q", name, tc.reason, err.Error(), tc.errSubstr)
				}

				return
			}

			if err != nil {
				t.Errorf("[%s] (%s) unexpected error: %v", name, tc.reason, err)

				return
			}

			for idx, wantRule := range tc.wantRules {
				if idx >= len(tc.profile.Spec.ForProvider.Rules) {
					t.Errorf("[%s] (%s) expected rule index %d but only %d rules are present", name, tc.reason, idx, len(tc.profile.Spec.ForProvider.Rules))

					continue
				}

				if got := *tc.profile.Spec.ForProvider.Rules[idx].Rule; got != wantRule {
					t.Errorf("[%s] (%s) Rules[%d].Rule: want %q, got %q", name, tc.reason, idx, wantRule, got)
				}
			}
		})
	}

	t.Run("ErrorOnFirstRule_StopsResolution", func(t *testing.T) {
		t.Parallel()

		qp := newTestQualityProfile(testNamespace)
		qp.Spec.ForProvider.Rules = []QualityProfileRuleParameters{
			{RuleRef: &xpv1.NamespacedReference{Name: "missing-rule"}},
			{RuleRef: &xpv1.NamespacedReference{Name: "second-rule"}},
		}

		callCount := 0
		c := &test.MockClient{
			MockGet: func(_ context.Context, _ client.ObjectKey, _ client.Object) error {
				callCount++

				return notFoundRule
			},
		}

		err := qp.ResolveReferences(context.Background(), c)
		if err == nil {
			t.Fatal("expected an error but got nil")
		}

		if !strings.Contains(err.Error(), "spec.forProvider.rules.rule") {
			t.Fatalf("expected wrapped error path, got %q", err.Error())
		}

		if callCount != 1 {
			t.Errorf("expected resolver to stop on first failing rule, got %d calls", callCount)
		}
	})
}

// TestResolveReferences tests reference resolution across resources.
func TestResolveReferences(t *testing.T) { //nolint:gocognit,maintidx // TestResolveReferences is a complex function with many cases; we allow it to be cognitively complex without breaking it up into smaller functions.
	t.Parallel()

	const (
		qgUUID = "qg-uuid-123"
		qpUUID = "qp-uuid-456"
		qgK8s  = "example-qualitygate"
		qpK8s  = "example-qualityprofile-go"
	)

	notFoundQG := kerrors.NewNotFound(
		schema.GroupResource{Group: "instance.sonarqube.crossplane.io", Resource: "qualitygates"},
		"wrong-name",
	)
	notFoundQP := kerrors.NewNotFound(
		schema.GroupResource{Group: "instance.sonarqube.crossplane.io", Resource: "qualityprofiles"},
		"wrong-name",
	)

	cases := map[string]struct {
		reason    string
		project   *Project
		client    client.Reader
		wantQGN   string
		wantQPID  string
		errSubstr string
	}{
		"NoRefsNoSelectors_NoOp": {
			reason:  "A project with no refs or selectors resolves without error.",
			project: newTestProject(testNamespace),
			client:  test.NewMockClient(),
		},
		"QualityGateNameAlreadySet_NoRef": {
			reason: "When QualityGateName is already set and no ref exists, it is preserved.",
			project: func() *Project {
				p := newTestProject(testNamespace)
				p.Spec.ForProvider.QualityGateName = ptr.To("Sonar way")

				return p
			}(),
			client:  test.NewMockClient(),
			wantQGN: "Sonar way",
		},
		// Regression: Crossplane's NameAsExternalName initializer defaults the
		// crossplane.io/external-name annotation to the K8s metadata.name before
		// the controller runs and sets the real SonarQube value.  If
		// ResolveReferences passed CurrentValue to the resolver, the non-empty
		// stale K8s name would be treated as a cache hit and re-resolution would
		// be skipped forever — even after the referenced resource was reconciled
		// and its annotation updated.
		// Fix: when a ref or selector is present, always pass CurrentValue="" so
		// the resolver re-reads the referenced resource's annotation on every call.
		"QualityGateNameRef_StaleCachedK8sName_IsOverwrittenByAnnotation": {
			reason: "A stale qualityGateName (K8s name) must be overwritten when a ref is present; the live annotation value wins.",
			project: func() *Project {
				p := newTestProject(testNamespace)
				// Stale cached value: set to the K8s object name by a previous
				// reconcile before the QualityGate controller had run.
				p.Spec.ForProvider.QualityGateName = ptr.To(qgK8s)
				// Ref still points to the same resource.
				p.Spec.ForProvider.QualityGateNameRef = &xpv1.NamespacedReference{Name: qgK8s}

				return p
			}(),
			client: &test.MockClient{
				// The QualityGate resource now has its real annotation: the SonarQube name.
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, "TestQualityGate")

					return nil
				}),
			},
			wantQGN: "TestQualityGate",
		},
		"QualityProfileIdRef_StaleCachedK8sName_IsOverwrittenByAnnotation": {
			reason: "A stale qualityProfile.id (K8s name) must be overwritten when a ref is present; the live annotation value wins.",
			project: func() *Project {
				p := newTestProject(testNamespace)
				p.Spec.ForProvider.QualityProfiles = map[string]ProjectQualityProfileReference{
					"go": {
						// Stale: was set to K8s name before the profile controller ran.
						Id:    ptr.To(qpK8s),
						IdRef: &xpv1.NamespacedReference{Name: qpK8s},
					},
				}

				return p
			}(),
			client: &test.MockClient{
				// The QualityProfile resource now has its real annotation: the UUID key.
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, "e58153d8-075a-4968-ace2-d7afd38c867b")

					return nil
				}),
			},
			wantQPID: "e58153d8-075a-4968-ace2-d7afd38c867b",
		},
		"QualityGateNameRef_ResolvesToExternalName": {
			reason: "QualityGateNameRef.Name is the K8s object name; ExternalName annotation is resolved.",
			project: func() *Project {
				p := newTestProject(testNamespace)
				p.Spec.ForProvider.QualityGateNameRef = &xpv1.NamespacedReference{Name: qgK8s}

				return p
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, qgUUID)

					return nil
				}),
			},
			wantQGN: qgUUID,
		},
		"QualityGateNameRef_WithExplicitNamespace": {
			reason: "An explicit namespace in QualityGateNameRef is respected.",
			project: func() *Project {
				p := newTestProject(testNamespace)
				p.Spec.ForProvider.QualityGateNameRef = &xpv1.NamespacedReference{Name: qgK8s, Namespace: testNamespace}

				return p
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, qgUUID)

					return nil
				}),
			},
			wantQGN: qgUUID,
		},
		// Regression: using the SonarQube display name instead of the Kubernetes object name
		// causes "spec.forProvider.qualityGateName: qualitygates "wrong-name" not found".
		"QualityGateNameRef_WrongName_ReturnsError": {
			reason: "Using the SonarQube name instead of the K8s object name returns a not-found error.",
			project: func() *Project {
				p := newTestProject(testNamespace)
				p.Spec.ForProvider.QualityGateNameRef = &xpv1.NamespacedReference{Name: "sonarqube-display-name"}

				return p
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(notFoundQG),
			},
			errSubstr: "not found",
		},
		"QualityGateNameSelector_ResolvesToExternalName": {
			reason: "A selector lists matching QualityGates and resolves the ExternalName.",
			project: func() *Project {
				p := newTestProject(testNamespace)
				p.Spec.ForProvider.QualityGateNameSelector = &xpv1.NamespacedSelector{
					MatchLabels: map[string]string{"app": "sonarqube"},
				}

				return p
			}(),
			client: &test.MockClient{
				MockList: test.NewMockListFn(nil, func(obj client.ObjectList) error {
					qgList, ok := obj.(*QualityGateList)
					if !ok {
						return nil
					}

					qg := QualityGate{ObjectMeta: metav1.ObjectMeta{Name: qgK8s, Namespace: testNamespace}}
					setExtName(&qg, qgUUID)
					qgList.Items = []QualityGate{qg}

					return nil
				}),
			},
			wantQGN: qgUUID,
		},
		"QualityProfileIdRef_ResolvesToExternalName": {
			reason: "IdRef.Name must be the K8s object name; ExternalName returns the UUID.",
			project: func() *Project {
				p := newTestProject(testNamespace)
				p.Spec.ForProvider.QualityProfiles = map[string]ProjectQualityProfileReference{
					"go": {IdRef: &xpv1.NamespacedReference{Name: qpK8s}},
				}

				return p
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, qpUUID)

					return nil
				}),
			},
			wantQPID: qpUUID,
		},
		"QualityProfileIdRef_WithExplicitNamespace": {
			reason: "An explicit namespace in IdRef is respected.",
			project: func() *Project {
				p := newTestProject(testNamespace)
				p.Spec.ForProvider.QualityProfiles = map[string]ProjectQualityProfileReference{
					"go": {IdRef: &xpv1.NamespacedReference{Name: qpK8s, Namespace: testNamespace}},
				}

				return p
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					setExtName(obj, qpUUID)

					return nil
				}),
			},
			wantQPID: qpUUID,
		},
		// Regression: the original bug reported as
		//   "cannot resolve references: spec.forProvider.qualityProfileId:
		//    cannot get referenced resource: QualityProfile ... \"example-go-profile\" not found"
		// Root cause: idRef.name contained spec.forProvider.name ("example-go-profile") instead
		// of the Kubernetes object name ("example-qualityprofile-go").
		"QualityProfileIdRef_WrongName_SonarQubeNameInsteadOfK8sName": {
			reason: "Using spec.forProvider.name as idRef.name fails; K8s object name must be used.",
			project: func() *Project {
				p := newTestProject(testNamespace)
				p.Spec.ForProvider.QualityProfiles = map[string]ProjectQualityProfileReference{
					// WRONG: "example-go-profile" is the SonarQube name, not the K8s metadata.name.
					// Correct value would be "example-qualityprofile-go".
					"go": {IdRef: &xpv1.NamespacedReference{Name: "example-go-profile"}},
				}

				return p
			}(),
			client: &test.MockClient{
				MockGet: test.NewMockGetFn(notFoundQP),
			},
			errSubstr: "spec.forProvider.qualityProfileId",
		},
		"QualityProfileIdAlreadySet_NoRef": {
			reason: "When Id is already set and no ref/selector exists, it is preserved.",
			project: func() *Project {
				p := newTestProject(testNamespace)
				p.Spec.ForProvider.QualityProfiles = map[string]ProjectQualityProfileReference{
					"go": {Id: ptr.To(qpUUID)},
				}

				return p
			}(),
			client:   test.NewMockClient(),
			wantQPID: qpUUID,
		},
		"QualityProfileIdSelector_ResolvesToExternalName": {
			reason: "IdSelector lists matching QualityProfiles and resolves ExternalName.",
			project: func() *Project {
				p := newTestProject(testNamespace)
				p.Spec.ForProvider.QualityProfiles = map[string]ProjectQualityProfileReference{
					"go": {IdSelector: &xpv1.NamespacedSelector{
						MatchLabels: map[string]string{"language": "go"},
					}},
				}

				return p
			}(),
			client: &test.MockClient{
				MockList: test.NewMockListFn(nil, func(obj client.ObjectList) error {
					qpList, ok := obj.(*QualityProfileList)
					if !ok {
						return nil
					}

					qp := QualityProfile{ObjectMeta: metav1.ObjectMeta{Name: qpK8s, Namespace: testNamespace}}
					setExtName(&qp, qpUUID)
					qpList.Items = []QualityProfile{qp}

					return nil
				}),
			},
			wantQPID: qpUUID,
		},
	}

	// -------------------------------------------------------------------------
	// Regression: pointer aliasing when both QualityGateNameRef and a profile
	// IdRef are present.
	//
	// Root cause: ResolveReferences stored &response.ResolvedValue directly into
	// project.Spec.ForProvider.QualityGateName.  The `response` local variable
	// was then reused (reassigned in-place) for each quality-profile resolution,
	// overwriting the memory the gate pointer was pointing at.  The result was
	// that QualityGateName silently received the last profile's resolved UUID.
	//
	// Fix: use ptr.To(response.ResolvedValue) which allocates an independent copy
	// for each resolved string, making every stored pointer stable.
	// -------------------------------------------------------------------------

	t.Run("BothQualityGateRefAndProfileRef_GateNameNotOverwrittenByProfileId", func(t *testing.T) {
		t.Parallel()

		const (
			gateExtName    = "Sonar way"         // what the quality gate resolves to
			profileExtName = "qp-uuid-different" // what the quality profile resolves to
		)

		p := newTestProject(testNamespace)
		p.Spec.ForProvider.QualityGateNameRef = &xpv1.NamespacedReference{Name: qgK8s}
		p.Spec.ForProvider.QualityProfiles = map[string]ProjectQualityProfileReference{
			"go": {IdRef: &xpv1.NamespacedReference{Name: qpK8s}},
		}

		callCount := 0
		c := &test.MockClient{
			MockGet: func(_ context.Context, _ client.ObjectKey, obj client.Object) error {
				callCount++
				if callCount == 1 {
					// First call: quality gate lookup
					setExtName(obj, gateExtName)
				} else {
					// Second call: quality profile lookup
					setExtName(obj, profileExtName)
				}

				return nil
			},
		}

		err := p.ResolveReferences(context.Background(), c)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		gotGate := ptr.Deref(p.Spec.ForProvider.QualityGateName, "")
		if gotGate != gateExtName {
			t.Errorf("QualityGateName: want %q, got %q — pointer aliasing bug: profile resolution overwrote gate value", gateExtName, gotGate)
		}

		gotProfile := ptr.Deref(p.Spec.ForProvider.QualityProfiles["go"].Id, "")
		if gotProfile != profileExtName {
			t.Errorf("QualityProfiles[go].Id: want %q, got %q", profileExtName, gotProfile)
		}

		if callCount != 2 {
			t.Errorf("expected 2 Get calls (one per ref), got %d", callCount)
		}
	})

	t.Run("MultipleQualityProfiles_EachLanguageKeepsItsOwnId", func(t *testing.T) {
		t.Parallel()

		p := newTestProject(testNamespace)
		p.Spec.ForProvider.QualityProfiles = map[string]ProjectQualityProfileReference{
			"go":   {IdRef: &xpv1.NamespacedReference{Name: "qp-go"}},
			"java": {IdRef: &xpv1.NamespacedReference{Name: "qp-java"}},
			"py":   {IdRef: &xpv1.NamespacedReference{Name: "qp-py"}},
		}

		c := &test.MockClient{
			MockGet: func(_ context.Context, key client.ObjectKey, obj client.Object) error {
				switch key.Name {
				case "qp-go":
					setExtName(obj, "uuid-go")
				case "qp-java":
					setExtName(obj, "uuid-java")
				case "qp-py":
					setExtName(obj, "uuid-py")
				default:
					return kerrors.NewNotFound(schema.GroupResource{}, key.Name)
				}

				return nil
			},
		}

		err := p.ResolveReferences(context.Background(), c)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := map[string]string{"go": "uuid-go", "java": "uuid-java", "py": "uuid-py"}
		for lang, wantID := range want {
			prof, ok := p.Spec.ForProvider.QualityProfiles[lang]
			if !ok {
				t.Errorf("QualityProfiles[%s] missing after resolution", lang)

				continue
			}

			got := ptr.Deref(prof.Id, "")
			if got != wantID {
				t.Errorf("QualityProfiles[%s].Id: want %q, got %q — pointer aliasing: loop iterations overwrite each other", lang, wantID, got)
			}
		}
	})

	// Regression: exact scenario reported by user.
	// The project had qualityGateName="example-qualitygate" and
	// qualityProfiles.go.id="example-qualityprofile-go" — both are the K8s
	// object names that Crossplane's NameAsExternalName initializer sets before
	// the individual controllers run.  After those controllers create the
	// resources in SonarQube and update the annotations, the project must pick
	// up the real values on the next reconcile regardless of the previously
	// cached (wrong) values.
	t.Run("BothRefs_StaleCachedK8sNames_OverwrittenByRealAnnotations", func(t *testing.T) {
		t.Parallel()

		p := newTestProject(testNamespace)
		// Pre-set stale cached values (what Crossplane's initializer writes).
		p.Spec.ForProvider.QualityGateName = ptr.To("example-qualitygate")
		p.Spec.ForProvider.QualityGateNameRef = &xpv1.NamespacedReference{Name: "example-qualitygate"}
		p.Spec.ForProvider.QualityProfiles = map[string]ProjectQualityProfileReference{
			"go": {
				Id:    ptr.To("example-qualityprofile-go"),
				IdRef: &xpv1.NamespacedReference{Name: "example-qualityprofile-go"},
			},
		}

		callCount := 0
		c := &test.MockClient{
			MockGet: func(_ context.Context, key client.ObjectKey, obj client.Object) error {
				callCount++

				switch key.Name {
				case "example-qualitygate":
					// Real SonarQube quality gate name, set by qualitygate controller.
					setExtName(obj, "TestQualityGate")
				case "example-qualityprofile-go":
					// Real SonarQube profile key (UUID), set by qualityprofile controller.
					setExtName(obj, "e58153d8-075a-4968-ace2-d7afd38c867b")
				default:
					t.Errorf("unexpected Get for key %q", key.Name)
				}

				return nil
			},
		}

		err := p.ResolveReferences(context.Background(), c)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := ptr.Deref(p.Spec.ForProvider.QualityGateName, ""); got != "TestQualityGate" {
			t.Errorf("QualityGateName: want \"TestQualityGate\", got %q — stale K8s name was not overwritten", got)
		}

		if got := ptr.Deref(p.Spec.ForProvider.QualityProfiles["go"].Id, ""); got != "e58153d8-075a-4968-ace2-d7afd38c867b" {
			t.Errorf("QualityProfiles[go].Id: want UUID, got %q — stale K8s name was not overwritten", got)
		}

		if callCount != 2 {
			t.Errorf("expected 2 Get calls, got %d", callCount)
		}
	})

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tc.project.ResolveReferences(context.Background(), tc.client)

			if tc.errSubstr != "" {
				if err == nil {
					t.Errorf("[%s] expected error containing %q, got nil", name, tc.errSubstr)

					return
				}

				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("[%s] error %q does not contain %q", name, err.Error(), tc.errSubstr)
				}

				return
			}

			if err != nil {
				t.Errorf("[%s] unexpected error: %v", name, err)

				return
			}

			if tc.wantQGN != "" {
				got := ptr.Deref(tc.project.Spec.ForProvider.QualityGateName, "")
				if got != tc.wantQGN {
					t.Errorf("[%s] QualityGateName: want %q, got %q", name, tc.wantQGN, got)
				}
			}

			if tc.wantQPID != "" {
				prof, ok := tc.project.Spec.ForProvider.QualityProfiles["go"]
				if !ok {
					t.Errorf("[%s] QualityProfiles[go] not found", name)
				} else if got := ptr.Deref(prof.Id, ""); got != tc.wantQPID {
					t.Errorf("[%s] QualityProfiles[go].Id: want %q, got %q", name, tc.wantQPID, got)
				}
			}
		})
	}
}
