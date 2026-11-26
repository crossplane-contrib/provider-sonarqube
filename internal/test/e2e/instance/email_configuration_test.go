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

package instance_test

import (
	"context"
	"testing"
	"time"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	instancev1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
	"github.com/crossplane/provider-sonarqube/internal/test/e2e"
)

// waitForEmailConfiguration polls SonarQube until the email configuration
// with the given ID satisfies check, failing the test after two minutes.
func waitForEmailConfiguration(t *testing.T, f *e2e.Framework, id, desc string, check func(*sonar.SystemEmailConfiguration) bool) {
	t.Helper()

	var last *sonar.SystemEmailConfiguration
	if err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		got, err := f.FetchEmailConfiguration(ctx, id)
		if err != nil {
			return false, err
		}
		last = got
		return got != nil && check(got), nil
	}); err != nil {
		t.Fatalf("waiting for email configuration %q (%s): %v; last observed: %+v", id, desc, err, last)
	}
}

// TestEmailConfigurationLifecycle creates the email configuration, verifies
// SonarQube reports it, updates it from the spec, reverts a change made
// directly in SonarQube, and deletes it.
//
// SonarQube holds a single email configuration per instance, so the whole
// lifecycle runs in one test. SonarQube does not connect to the SMTP server
// when the configuration is saved.
func TestEmailConfigurationLifecycle(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	const (
		crName         = "e2e-email-configuration"
		secretName     = "e2e-email-configuration-credentials"
		secretKey      = "password"
		connSecretName = "e2e-email-configuration-connection"
		smtpHost       = "smtp.example.com"
		smtpPassword   = "e2e-fake-smtp-password"
	)

	credentials := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: f.Namespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{secretKey: smtpPassword},
	}
	if err := f.Kube.Create(context.Background(), credentials); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating SMTP password secret: %v", err)
	}
	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), credentials)
	})

	configuration := &instancev1alpha1.EmailConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: f.Namespace},
		Spec: instancev1alpha1.EmailConfigurationSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				WriteConnectionSecretToReference: &xpv1.LocalSecretReference{Name: connSecretName},
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.EmailConfigurationParameters{
				Host:             smtpHost,
				Port:             587,
				SecurityProtocol: "STARTTLS",
				FromAddress:      "sonarqube@example.com",
				FromName:         "SonarQube",
				SubjectPrefix:    "[SonarQube]",
				AuthMethod:       instancev1alpha1.EmailAuthMethodBasic,
				Username:         "e2e-user",
				BasicPasswordSecretRef: &xpv1.LocalSecretKeySelector{
					LocalSecretReference: xpv1.LocalSecretReference{Name: secretName},
					Key:                  secretKey,
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, configuration, 2*time.Minute)
	e2e.AssertReady(t, configuration)
	e2e.AssertSynced(t, configuration)

	id := meta.GetExternalName(configuration)
	if id == "" {
		t.Fatal("external name is empty, want the SonarQube-assigned configuration id")
	}

	got, err := f.FetchEmailConfiguration(context.Background(), id)
	if err != nil {
		t.Fatalf("fetching email configuration: %v", err)
	}
	if got == nil {
		t.Fatalf("email configuration %q not found in SonarQube", id)
	}
	if got.Host != smtpHost || got.Port != "587" || got.AuthMethod != instancev1alpha1.EmailAuthMethodBasic || !got.IsBasicPasswordSet {
		t.Errorf("email configuration = %+v, want it to match the spec", got)
	}

	// Update the spec and wait for SonarQube to reflect it.
	if err := f.Kube.Get(context.Background(), kubeKey(configuration), configuration); err != nil {
		t.Fatalf("re-fetching EmailConfiguration: %v", err)
	}
	configuration.Spec.ForProvider.Port = 465
	configuration.Spec.ForProvider.SecurityProtocol = "SSLTLS"
	f.Update(t, configuration)

	waitForEmailConfiguration(t, f, id, "after spec update", func(got *sonar.SystemEmailConfiguration) bool {
		return got.Port == "465" && got.SecurityProtocol == "SSLTLS"
	})

	// Change the host directly in SonarQube to simulate drift. SonarQube
	// refuses to change the host unless the password is sent with it.
	_, resp, err := f.Sonar.V2.System.UpdateEmailConfiguration(context.Background(), id, &sonar.SystemEmailConfigurationUpdateOptions{
		Host:          "drifted.example.com",
		BasicPassword: smtpPassword,
	})
	helpers.CloseBody(resp)
	if err != nil {
		t.Fatalf("updating email configuration in SonarQube: %v", err)
	}

	// Touch an annotation to trigger a reconcile rather than waiting for
	// the provider poll interval.
	if err := f.Kube.Get(context.Background(), kubeKey(configuration), configuration); err != nil {
		t.Fatalf("re-fetching EmailConfiguration: %v", err)
	}
	annotations := configuration.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["e2e.sonarqube.crossplane.io/drift"] = time.Now().Format(time.RFC3339Nano)
	configuration.SetAnnotations(annotations)
	f.Update(t, configuration)

	waitForEmailConfiguration(t, f, id, "after drift", func(got *sonar.SystemEmailConfiguration) bool {
		return got.Host == smtpHost
	})

	// Delete the resource and verify SonarQube no longer has a configuration.
	f.Delete(t, configuration)
	if err := f.WaitForDeletion(context.Background(), configuration, e2e.DefaultDeleteTimeout); err != nil {
		t.Fatalf("waiting for EmailConfiguration deletion: %v", err)
	}

	got, err = f.FetchEmailConfiguration(context.Background(), id)
	if err != nil {
		t.Fatalf("fetching email configuration after deletion: %v", err)
	}
	if got != nil {
		t.Errorf("email configuration %q still exists after deletion: %+v", id, got)
	}
}
