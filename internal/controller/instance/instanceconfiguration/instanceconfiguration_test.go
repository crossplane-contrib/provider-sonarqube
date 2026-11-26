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

package instanceconfiguration

import (
	"context"
	stderrors "errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakekube "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/fake"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

const (
	// callGetMode records a GetMode call.
	callGetMode = "GetMode"
	// callUpdateMode records an UpdateMode call.
	callUpdateMode = "UpdateMode"
	// callValues records a settings Values call.
	callValues = "Values"
	// callUpdateVisibility records an UpdateDefaultVisibility call.
	callUpdateVisibility = "UpdateDefaultVisibility"
	// callWorkerCount records a WorkerCount call.
	callWorkerCount = "WorkerCount"
	// callSetWorkerCount records a SetWorkerCount call.
	callSetWorkerCount = "SetWorkerCount"
	// callSCIMStatus records a SCIM Status call.
	callSCIMStatus = "SCIMStatus"
	// callSCIMEnable records a SCIM Enable call.
	callSCIMEnable = "SCIMEnable"
	// callSCIMDisable records a SCIM Disable call.
	callSCIMDisable = "SCIMDisable"
	// callGetSCAEnablement records a GetEnablement call.
	callGetSCAEnablement = "GetSCAEnablement"
	// callSetSCAEnablement records a SetEnablement call.
	callSetSCAEnablement = "SetSCAEnablement"
	// visibilityPrivate is the private project visibility.
	visibilityPrivate = "private"
	// visibilityPublic is the public project visibility.
	visibilityPublic = "public"
	// errAPI is the message of the errors returned by failing calls.
	errAPI = "api error"
	// testResourceName is the name of the InstanceConfiguration under test.
	testResourceName = "test-instance-configuration"
	// testResourceNamespace is the namespace of the InstanceConfiguration
	// under test.
	testResourceNamespace = "default"
)

// mockHTTPResponse creates a minimal mock HTTP response for testing.
func mockHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}
}

// errComparer compares error messages for testing.
func errComparer(a, b error) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return a.Error() == b.Error()
}

// fakeSonarQube is an in-memory SonarQube instance backing the mock clients
// of an InstanceConfiguration. It records every call and fails the ones
// listed in failing.
type fakeSonarQube struct {
	mode        string
	visibility  *string
	workerCount int64
	scim        bool
	sca         bool
	failing     map[string]bool
	calls       []string
}

// newFakeSonarQube returns a fakeSonarQube with the default state of a fresh
// SonarQube instance.
func newFakeSonarQube(failing ...string) *fakeSonarQube {
	failingCalls := make(map[string]bool, len(failing))
	for _, call := range failing {
		failingCalls[call] = true
	}

	return &fakeSonarQube{
		mode:        sonar.InstanceModeMQR,
		visibility:  new(visibilityPublic),
		workerCount: 1,
		failing:     failingCalls,
	}
}

// record records a call and returns the error it must fail with, if any.
func (f *fakeSonarQube) record(call string) error {
	f.calls = append(f.calls, call)
	if f.failing[call] {
		return errors.New(errAPI)
	}

	return nil
}

// clients returns the mock clients backed by the fake instance.
func (f *fakeSonarQube) clients() instance.InstanceConfigurationClients {
	return instance.InstanceConfigurationClients{
		CleanCodePolicy: &fake.MockCleanCodePolicyModeClient{
			GetModeFn: func() (*sonar.CleanCodePolicyMode, *http.Response, error) {
				err := f.record(callGetMode)
				if err != nil {
					return nil, nil, err
				}

				return &sonar.CleanCodePolicyMode{Mode: f.mode}, mockHTTPResponse(), nil
			},
			UpdateModeFn: func(opt *sonar.CleanCodePolicyUpdateModeOptions) (*sonar.CleanCodePolicyMode, *http.Response, error) {
				err := f.record(callUpdateMode)
				if err != nil {
					return nil, nil, err
				}

				f.mode = opt.Mode

				return &sonar.CleanCodePolicyMode{Mode: f.mode, Modified: true}, mockHTTPResponse(), nil
			},
		},
		Projects: &fake.MockDefaultProjectVisibilityClient{
			UpdateDefaultVisibilityFn: func(opt *sonar.ProjectsUpdateDefaultVisibilityOptions) (*http.Response, error) {
				err := f.record(callUpdateVisibility)
				if err != nil {
					return nil, err
				}

				f.visibility = &opt.ProjectVisibility

				return mockHTTPResponse(), nil
			},
		},
		Settings: &fake.MockSettingsClient{
			ValuesFn: func(opt *sonar.SettingsValuesOptions) (*sonar.SettingsValues, *http.Response, error) {
				err := f.record(callValues)
				if err != nil {
					return nil, nil, err
				}

				values := &sonar.SettingsValues{}
				if f.visibility != nil && len(opt.Keys) == 1 && opt.Keys[0] == instance.DefaultProjectVisibilitySettingKey {
					values.Settings = []sonar.SettingValue{{Key: instance.DefaultProjectVisibilitySettingKey, Value: *f.visibility}}
				}

				return values, mockHTTPResponse(), nil
			},
		},
		ComputeEngine: &fake.MockComputeEngineWorkerCountClient{
			WorkerCountFn: func() (*sonar.CeWorkerCount, *http.Response, error) {
				err := f.record(callWorkerCount)
				if err != nil {
					return nil, nil, err
				}

				return &sonar.CeWorkerCount{Value: f.workerCount, CanSetWorkerCount: true}, mockHTTPResponse(), nil
			},
			SetWorkerCountFn: func(opt *sonar.CeSetWorkerCountOptions) (*http.Response, error) {
				err := f.record(callSetWorkerCount)
				if err != nil {
					return nil, err
				}

				f.workerCount = opt.Count

				return mockHTTPResponse(), nil
			},
		},
		SCIM: &fake.MockSCIMManagementClient{
			StatusFn: func() (*sonar.ScimManagementStatus, *http.Response, error) {
				err := f.record(callSCIMStatus)
				if err != nil {
					return nil, nil, err
				}

				return &sonar.ScimManagementStatus{Enabled: f.scim}, mockHTTPResponse(), nil
			},
			EnableFn: func() (*http.Response, error) {
				err := f.record(callSCIMEnable)
				if err != nil {
					return nil, err
				}

				f.scim = true

				return mockHTTPResponse(), nil
			},
			DisableFn: func() (*http.Response, error) {
				err := f.record(callSCIMDisable)
				if err != nil {
					return nil, err
				}

				f.scim = false

				return mockHTTPResponse(), nil
			},
		},
		SCA: &fake.MockSCAEnablementClient{
			GetEnablementFn: func() (*sonar.ScaFeatureEnablement, *http.Response, error) {
				err := f.record(callGetSCAEnablement)
				if err != nil {
					return nil, nil, err
				}

				return &sonar.ScaFeatureEnablement{Enablement: f.sca}, mockHTTPResponse(), nil
			},
			SetEnablementFn: func(opt *sonar.ScaSetEnablementOptions) (*sonar.ScaFeatureEnablement, *http.Response, error) {
				err := f.record(callSetSCAEnablement)
				if err != nil {
					return nil, nil, err
				}

				f.sca = opt.Enablement

				return &sonar.ScaFeatureEnablement{Enablement: f.sca}, mockHTTPResponse(), nil
			},
		},
	}
}

// newExternal returns an external client backed by the fake instance.
func newExternal(sonarQube *fakeSonarQube) *external {
	return &external{clients: sonarQube.clients(), options: options}
}

// newInstanceConfiguration returns an InstanceConfiguration with the given
// parameters.
func newInstanceConfiguration(params v1alpha1.InstanceConfigurationParameters) *v1alpha1.InstanceConfiguration {
	return &v1alpha1.InstanceConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.InstanceConfigurationGroupVersionKind.GroupVersion().String(),
			Kind:       v1alpha1.InstanceConfigurationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testResourceName,
			Namespace: testResourceNamespace,
			UID:       types.UID(testResourceName + "-uid"),
		},
		Spec: v1alpha1.InstanceConfigurationSpec{
			ForProvider: params,
		},
	}
}

// allParameters returns parameters setting every option.
func allParameters() v1alpha1.InstanceConfigurationParameters {
	return v1alpha1.InstanceConfigurationParameters{
		CleanCodePolicyMode:      new(sonar.InstanceModeStandardExperience),
		DefaultProjectVisibility: new(visibilityPrivate),
		ComputeEngineWorkerCount: new(int64(4)),
		SCIMEnabled:              new(true),
		SCAEnabled:               new(true),
	}
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	type want struct {
		o          managed.ExternalObservation
		err        error
		atProvider v1alpha1.InstanceConfigurationObservation
		calls      []string
	}

	cases := map[string]struct {
		sonarQube *fakeSonarQube
		params    v1alpha1.InstanceConfigurationParameters
		deleting  bool
		want      want
	}{
		"DeletingReturnsNotExists": {
			sonarQube: newFakeSonarQube(),
			params:    allParameters(),
			deleting:  true,
			want: want{
				o: managed.ExternalObservation{ResourceExists: false},
			},
		},
		"EmptySpecObservesNothing": {
			sonarQube: newFakeSonarQube(),
			want: want{
				o: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
			},
		},
		"OnlySetOptionsAreObserved": {
			sonarQube: newFakeSonarQube(),
			params: v1alpha1.InstanceConfigurationParameters{
				CleanCodePolicyMode:      new(sonar.InstanceModeMQR),
				DefaultProjectVisibility: new(visibilityPublic),
			},
			want: want{
				o: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
				atProvider: v1alpha1.InstanceConfigurationObservation{
					CleanCodePolicyMode:      new(sonar.InstanceModeMQR),
					DefaultProjectVisibility: new(visibilityPublic),
				},
				calls: []string{callGetMode, callValues},
			},
		},
		"AllOptionsUpToDate": {
			sonarQube: &fakeSonarQube{
				mode:        sonar.InstanceModeStandardExperience,
				visibility:  new(visibilityPrivate),
				workerCount: 4,
				scim:        true,
				sca:         true,
			},
			params: allParameters(),
			want: want{
				o: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
				atProvider: v1alpha1.InstanceConfigurationObservation{
					CleanCodePolicyMode:      new(sonar.InstanceModeStandardExperience),
					DefaultProjectVisibility: new(visibilityPrivate),
					ComputeEngineWorkerCount: new(int64(4)),
					SCIMEnabled:              new(true),
					SCAEnabled:               new(true),
				},
				calls: []string{callGetMode, callValues, callWorkerCount, callSCIMStatus, callGetSCAEnablement},
			},
		},
		"AllOptionsOutdated": {
			sonarQube: newFakeSonarQube(),
			params:    allParameters(),
			want: want{
				o: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false},
				atProvider: v1alpha1.InstanceConfigurationObservation{
					CleanCodePolicyMode:      new(sonar.InstanceModeMQR),
					DefaultProjectVisibility: new(visibilityPublic),
					ComputeEngineWorkerCount: new(int64(1)),
					SCIMEnabled:              new(false),
					SCAEnabled:               new(false),
				},
				calls: []string{callGetMode, callValues, callWorkerCount, callSCIMStatus, callGetSCAEnablement},
			},
		},
		"OneOptionOutdated": {
			sonarQube: newFakeSonarQube(),
			params: v1alpha1.InstanceConfigurationParameters{
				CleanCodePolicyMode: new(sonar.InstanceModeMQR),
				SCIMEnabled:         new(true),
			},
			want: want{
				o: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false},
				atProvider: v1alpha1.InstanceConfigurationObservation{
					CleanCodePolicyMode: new(sonar.InstanceModeMQR),
					SCIMEnabled:         new(false),
				},
				calls: []string{callGetMode, callSCIMStatus},
			},
		},
		"MissingVisibilitySettingIsOutdated": {
			sonarQube: &fakeSonarQube{},
			params: v1alpha1.InstanceConfigurationParameters{
				DefaultProjectVisibility: new(visibilityPublic),
			},
			want: want{
				o:     managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false},
				calls: []string{callValues},
			},
		},
		"GetModeFails": {
			sonarQube: newFakeSonarQube(callGetMode),
			params:    allParameters(),
			want: want{
				err:   errors.Wrapf(errors.New(errAPI), errObserveOption, cleanCodePolicyModeOption.name),
				calls: []string{callGetMode},
			},
		},
		"ValuesFails": {
			sonarQube: newFakeSonarQube(callValues),
			params:    allParameters(),
			want: want{
				err:   errors.Wrapf(errors.New(errAPI), errObserveOption, defaultProjectVisibilityOption.name),
				calls: []string{callGetMode, callValues},
			},
		},
		"WorkerCountFails": {
			sonarQube: newFakeSonarQube(callWorkerCount),
			params:    allParameters(),
			want: want{
				err:   errors.Wrapf(errors.New(errAPI), errObserveOption, computeEngineWorkerCountOption.name),
				calls: []string{callGetMode, callValues, callWorkerCount},
			},
		},
		"SCIMStatusFails": {
			sonarQube: newFakeSonarQube(callSCIMStatus),
			params:    allParameters(),
			want: want{
				err:   errors.Wrapf(errors.New(errAPI), errObserveOption, scimEnabledOption.name),
				calls: []string{callGetMode, callValues, callWorkerCount, callSCIMStatus},
			},
		},
		"GetSCAEnablementFails": {
			sonarQube: newFakeSonarQube(callGetSCAEnablement),
			params:    allParameters(),
			want: want{
				err:   errors.Wrapf(errors.New(errAPI), errObserveOption, scaEnabledOption.name),
				calls: []string{callGetMode, callValues, callWorkerCount, callSCIMStatus, callGetSCAEnablement},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mg := newInstanceConfiguration(tc.params)
			if tc.deleting {
				mg.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			}

			got, err := newExternal(tc.sonarQube).Observe(context.Background(), mg)

			if !errComparer(tc.want.err, err) {
				t.Errorf("Observe() error = %v, want %v", err, tc.want.err)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Observe() observation mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.calls, tc.sonarQube.calls); diff != "" {
				t.Errorf("Observe() calls mismatch (-want +got):\n%s", diff)
			}

			if tc.want.err != nil || tc.deleting {
				return
			}

			if diff := cmp.Diff(tc.want.atProvider, mg.Status.AtProvider); diff != "" {
				t.Errorf("Observe() atProvider mismatch (-want +got):\n%s", diff)
			}

			if mg.GetCondition(xpv1.TypeReady).Reason != xpv1.ReasonAvailable {
				t.Errorf("Observe() Ready reason = %q, want %q", mg.GetCondition(xpv1.TypeReady).Reason, xpv1.ReasonAvailable)
			}
		})
	}
}

// TestObserveNilResponses verifies Observe tolerates empty responses, which
// leave the options unobserved and therefore outdated.
func TestObserveNilResponses(t *testing.T) {
	t.Parallel()

	ext := &external{
		clients: instance.InstanceConfigurationClients{
			CleanCodePolicy: &fake.MockCleanCodePolicyModeClient{
				GetModeFn: func() (*sonar.CleanCodePolicyMode, *http.Response, error) { return nil, nil, nil },
			},
			Settings: &fake.MockSettingsClient{
				ValuesFn: func(*sonar.SettingsValuesOptions) (*sonar.SettingsValues, *http.Response, error) {
					return nil, nil, nil
				},
			},
			ComputeEngine: &fake.MockComputeEngineWorkerCountClient{
				WorkerCountFn: func() (*sonar.CeWorkerCount, *http.Response, error) { return nil, nil, nil },
			},
			SCIM: &fake.MockSCIMManagementClient{
				StatusFn: func() (*sonar.ScimManagementStatus, *http.Response, error) { return nil, nil, nil },
			},
			SCA: &fake.MockSCAEnablementClient{
				GetEnablementFn: func() (*sonar.ScaFeatureEnablement, *http.Response, error) { return nil, nil, nil },
			},
		},
		options: options,
	}

	mg := newInstanceConfiguration(allParameters())

	got, err := ext.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	want := managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Observe() observation mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(v1alpha1.InstanceConfigurationObservation{}, mg.Status.AtProvider); diff != "" {
		t.Errorf("Observe() atProvider mismatch (-want +got):\n%s", diff)
	}
}

// TestUpdate tests the Update method, which applies the options that Observe
// recorded as outdated.
func TestUpdate(t *testing.T) {
	t.Parallel()

	type want struct {
		err   error
		calls []string
	}

	cases := map[string]struct {
		sonarQube  *fakeSonarQube
		params     v1alpha1.InstanceConfigurationParameters
		atProvider v1alpha1.InstanceConfigurationObservation
		want       want
	}{
		"EmptySpecAppliesNothing": {
			sonarQube: newFakeSonarQube(),
		},
		"UpToDateOptionsAreSkipped": {
			sonarQube: newFakeSonarQube(),
			params: v1alpha1.InstanceConfigurationParameters{
				CleanCodePolicyMode:      new(sonar.InstanceModeMQR),
				DefaultProjectVisibility: new(visibilityPrivate),
			},
			atProvider: v1alpha1.InstanceConfigurationObservation{
				CleanCodePolicyMode:      new(sonar.InstanceModeMQR),
				DefaultProjectVisibility: new(visibilityPublic),
			},
			want: want{
				calls: []string{callUpdateVisibility},
			},
		},
		"AllOptionsApplied": {
			sonarQube: newFakeSonarQube(),
			params:    allParameters(),
			want: want{
				calls: []string{callUpdateMode, callUpdateVisibility, callSetWorkerCount, callSCIMEnable, callSetSCAEnablement},
			},
		},
		"SCIMDisabled": {
			sonarQube: newFakeSonarQube(),
			params: v1alpha1.InstanceConfigurationParameters{
				SCIMEnabled: new(false),
			},
			atProvider: v1alpha1.InstanceConfigurationObservation{
				SCIMEnabled: new(true),
			},
			want: want{
				calls: []string{callSCIMDisable},
			},
		},
		"FailuresDoNotStopOtherOptions": {
			sonarQube: newFakeSonarQube(callUpdateMode, callSCIMEnable),
			params:    allParameters(),
			want: want{
				err: stderrors.Join(
					errors.Wrapf(errors.New(errAPI), errApplyOption, cleanCodePolicyModeOption.name),
					errors.Wrapf(errors.New(errAPI), errApplyOption, scimEnabledOption.name),
				),
				calls: []string{callUpdateMode, callUpdateVisibility, callSetWorkerCount, callSCIMEnable, callSetSCAEnablement},
			},
		},
		"EveryApplyFails": {
			sonarQube: newFakeSonarQube(callUpdateVisibility, callSetWorkerCount, callSetSCAEnablement),
			params: v1alpha1.InstanceConfigurationParameters{
				DefaultProjectVisibility: new(visibilityPrivate),
				ComputeEngineWorkerCount: new(int64(2)),
				SCAEnabled:               new(true),
			},
			want: want{
				err: stderrors.Join(
					errors.Wrapf(errors.New(errAPI), errApplyOption, defaultProjectVisibilityOption.name),
					errors.Wrapf(errors.New(errAPI), errApplyOption, computeEngineWorkerCountOption.name),
					errors.Wrapf(errors.New(errAPI), errApplyOption, scaEnabledOption.name),
				),
				calls: []string{callUpdateVisibility, callSetWorkerCount, callSetSCAEnablement},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mg := newInstanceConfiguration(tc.params)
			mg.Status.AtProvider = tc.atProvider

			got, err := newExternal(tc.sonarQube).Update(context.Background(), mg)

			if !errComparer(tc.want.err, err) {
				t.Errorf("Update() error = %v, want %v", err, tc.want.err)
			}

			if diff := cmp.Diff(managed.ExternalUpdate{}, got); diff != "" {
				t.Errorf("Update() result mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want.calls, tc.sonarQube.calls); diff != "" {
				t.Errorf("Update() calls mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestUpdateConverges verifies that an Observe following an Update reports
// the resource as up to date, with the applied values observed.
func TestUpdateConverges(t *testing.T) {
	t.Parallel()

	sonarQube := newFakeSonarQube()
	ext := newExternal(sonarQube)
	mg := newInstanceConfiguration(allParameters())

	first, err := ext.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if first.ResourceUpToDate {
		t.Fatal("Observe() before Update reported up to date, want outdated")
	}

	_, err = ext.Update(context.Background(), mg)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	second, err := ext.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if !second.ResourceUpToDate {
		t.Fatalf("Observe() after Update reported outdated, atProvider = %+v", mg.Status.AtProvider)
	}

	want := v1alpha1.InstanceConfigurationObservation{
		CleanCodePolicyMode:      new(sonar.InstanceModeStandardExperience),
		DefaultProjectVisibility: new(visibilityPrivate),
		ComputeEngineWorkerCount: new(int64(4)),
		SCIMEnabled:              new(true),
		SCAEnabled:               new(true),
	}
	if diff := cmp.Diff(want, mg.Status.AtProvider); diff != "" {
		t.Errorf("Observe() atProvider mismatch (-want +got):\n%s", diff)
	}
}

// TestCreate tests the Create method, which applies the set options.
func TestCreate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		sonarQube *fakeSonarQube
		wantErr   error
		wantCalls []string
	}{
		"Success": {
			sonarQube: newFakeSonarQube(),
			wantCalls: []string{callUpdateMode, callUpdateVisibility, callSetWorkerCount, callSCIMEnable, callSetSCAEnablement},
		},
		"Failure": {
			sonarQube: newFakeSonarQube(callSetWorkerCount),
			wantErr:   errors.Wrapf(errors.New(errAPI), errApplyOption, computeEngineWorkerCountOption.name),
			wantCalls: []string{callUpdateMode, callUpdateVisibility, callSetWorkerCount, callSCIMEnable, callSetSCAEnablement},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mg := newInstanceConfiguration(allParameters())

			got, err := newExternal(tc.sonarQube).Create(context.Background(), mg)

			if !errComparer(tc.wantErr, err) {
				t.Errorf("Create() error = %v, want %v", err, tc.wantErr)
			}

			if diff := cmp.Diff(managed.ExternalCreation{}, got); diff != "" {
				t.Errorf("Create() result mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantCalls, tc.sonarQube.calls); diff != "" {
				t.Errorf("Create() calls mismatch (-want +got):\n%s", diff)
			}

			if mg.GetCondition(xpv1.TypeReady).Reason != xpv1.ReasonCreating {
				t.Errorf("Create() Ready reason = %q, want %q", mg.GetCondition(xpv1.TypeReady).Reason, xpv1.ReasonCreating)
			}
		})
	}
}

// TestDelete verifies Delete leaves every option in place.
func TestDelete(t *testing.T) {
	t.Parallel()

	sonarQube := newFakeSonarQube()
	mg := newInstanceConfiguration(allParameters())

	got, err := newExternal(sonarQube).Delete(context.Background(), mg)
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	if diff := cmp.Diff(managed.ExternalDelete{}, got); diff != "" {
		t.Errorf("Delete() result mismatch (-want +got):\n%s", diff)
	}

	if len(sonarQube.calls) != 0 {
		t.Errorf("Delete() called SonarQube: %v, want no call", sonarQube.calls)
	}

	if mg.GetCondition(xpv1.TypeReady).Reason != xpv1.ReasonDeleting {
		t.Errorf("Delete() Ready reason = %q, want %q", mg.GetCondition(xpv1.TypeReady).Reason, xpv1.ReasonDeleting)
	}
}

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	err := newExternal(newFakeSonarQube()).Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}

// mockGate is a mock implementation of the feature gate interface.
type mockGate struct {
	registered bool
	callback   func()
	gvks       []schema.GroupVersionKind
}

// Register implements the gate interface, capturing the callback and GVKs.
func (m *mockGate) Register(callback func(), gvks ...schema.GroupVersionKind) {
	m.registered = true
	m.callback = callback
	m.gvks = append(m.gvks, gvks...)
}

// Set implements the gate interface as a no-op.
func (m *mockGate) Set(_ schema.GroupVersionKind, _ bool) bool {
	return false
}

// newScheme creates a runtime.Scheme with all required types registered.
func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()

	err := apisv1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(apisv1alpha1) unexpected error: %v", err)
	}

	err = v1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(v1alpha1) unexpected error: %v", err)
	}

	err = corev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme(corev1) unexpected error: %v", err)
	}

	return scheme
}

// TestConnectTrackUsageError verifies Connect returns an error when
// usage tracking fails.
func TestConnectTrackUsageError(t *testing.T) {
	t.Parallel()

	kubeClient := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
	mg := newInstanceConfiguration(allParameters())

	c := &connector{
		kube:  kubeClient,
		usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
	}

	_, err := c.Connect(context.Background(), mg)
	if err == nil {
		t.Fatal("Connect() expected error, got nil")
	}

	if !strings.Contains(err.Error(), errTrackPCUsage) {
		t.Fatalf("Connect() error = %q, want to contain %q", err.Error(), errTrackPCUsage)
	}
}

// TestConnectGetConfigError verifies Connect returns an error when
// the ProviderConfig is missing.
func TestConnectGetConfigError(t *testing.T) {
	t.Parallel()

	kubeClient := fakekube.NewClientBuilder().WithScheme(newScheme(t)).Build()
	mg := newInstanceConfiguration(allParameters())
	mg.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "missing-pc", Kind: "ProviderConfig"})

	c := &connector{
		kube:  kubeClient,
		usage: resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
	}

	_, err := c.Connect(context.Background(), mg)
	if err == nil {
		t.Fatal("Connect() expected error, got nil")
	}

	if !strings.Contains(err.Error(), errGetPC) {
		t.Fatalf("Connect() error = %q, want to contain %q", err.Error(), errGetPC)
	}
}

// TestConnectSuccess verifies Connect returns a valid ExternalClient
// managing every option on success.
func TestConnectSuccess(t *testing.T) {
	t.Parallel()

	providerConfig := &apisv1alpha1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "pc", Namespace: testResourceNamespace},
		Spec: apisv1alpha1.ProviderConfigSpec{
			BaseURL: "http://localhost:9000",
			Token: &apisv1alpha1.ProviderCredentials{
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Name:      "sonar-secret",
							Namespace: testResourceNamespace,
						},
						Key: "token",
					},
				},
				Source: xpv1.CredentialsSourceSecret,
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "sonar-secret", Namespace: testResourceNamespace},
		Data:       map[string][]byte{"token": []byte("my-token")},
	}

	kubeClient := fakekube.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(providerConfig, secret).Build()
	mg := newInstanceConfiguration(allParameters())
	mg.SetProviderConfigReference(&xpv1.ProviderConfigReference{Name: "pc", Kind: "ProviderConfig"})

	c := &connector{
		kube:         kubeClient,
		usage:        resource.NewProviderConfigUsageTracker(kubeClient, &apisv1alpha1.ProviderConfigUsage{}),
		newClientsFn: instance.NewInstanceConfigurationClients,
	}

	got, err := c.Connect(context.Background(), mg)
	if err != nil {
		t.Fatalf("Connect() unexpected error: %v", err)
	}

	ext, ok := got.(*external)
	if !ok {
		t.Fatalf("Connect() returned %T, want *external", got)
	}

	if len(ext.options) != len(options) {
		t.Errorf("Connect() external manages %d options, want %d", len(ext.options), len(options))
	}
}

// TestSetupGatedRegistersInstanceConfigurationGVK verifies SetupGated
// registers the InstanceConfiguration GVK.
func TestSetupGatedRegistersInstanceConfigurationGVK(t *testing.T) {
	t.Parallel()

	g := &mockGate{}
	o := helpers.Options{Options: controller.DefaultOptions()}
	o.Gate = g

	err := SetupGated(nil, o)
	if err != nil {
		t.Fatalf("SetupGated() unexpected error: %v", err)
	}

	if !g.registered {
		t.Fatal("SetupGated() expected Gate.Register to be called")
	}

	if g.callback == nil {
		t.Fatal("SetupGated() expected a non-nil callback")
	}

	if len(g.gvks) != 1 {
		t.Fatalf("SetupGated() registered %d GVKs, want 1", len(g.gvks))
	}

	if diff := cmp.Diff(v1alpha1.InstanceConfigurationGroupVersionKind, g.gvks[0]); diff != "" {
		t.Fatalf("SetupGated() GVK mismatch (-want +got):\n%s", diff)
	}
}

// TestSetupGatedCallbackPanicsWhenSetupFails tests that the SetupGated
// callback panics when the underlying Setup call fails.
func TestSetupGatedCallbackPanicsWhenSetupFails(t *testing.T) {
	t.Parallel()

	g := &mockGate{}
	o := helpers.Options{Options: controller.DefaultOptions()}
	o.Gate = g

	err := SetupGated(nil, o)
	if err != nil {
		t.Fatalf("SetupGated() unexpected error: %v", err)
	}

	if g.callback == nil {
		t.Fatal("SetupGated() expected callback to be registered")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected callback to panic when setup fails")
		}
	}()

	g.callback()
}
