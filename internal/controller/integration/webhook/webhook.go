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

// Package webhook provides a controller for Webhook resources.
package webhook

import (
	"context"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/integration/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/integration"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errNotWebhook indicates managed resource is not a Webhook.
	errNotWebhook = "managed resource is not a Webhook custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errListWebhooks indicates webhook list API call failed.
	errListWebhooks = "cannot list webhooks"
	// errCreateWebhook indicates webhook create API call failed.
	errCreateWebhook = "cannot create webhook"
	// errUpdateWebhook indicates webhook update API call failed.
	errUpdateWebhook = "cannot update webhook"
	// errDeleteWebhook indicates webhook delete API call failed.
	errDeleteWebhook = "cannot delete webhook"
	// errGetSecret indicates HMAC secret retrieval failed.
	errGetSecret = "cannot get webhook secret"

	// errExternalNameNotSet indicates external name is not set.
	errExternalNameNotSet = "external name is not set for Webhook resource %s"

	// connectionDetailSecretKey is the connection detail key for the HMAC
	// signing secret.
	connectionDetailSecretKey = "secret"
)

// SetupGated adds a controller that reconciles Webhook managed resources
// with safe-start support.
func SetupGated(mgr ctrl.Manager, options controller.Options) error {
	options.Gate.Register(func() {
		err := Setup(mgr, options)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup Webhook controller"))
		}
	}, v1alpha1.WebhookGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles Webhook managed resources.
func Setup(mgr ctrl.Manager, options controller.Options) error {
	name := managed.ControllerName(v1alpha1.WebhookGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: integration.NewWebhooksClient,
		}),
		managed.WithLogger(options.Logger.WithValues("controller", name)),
		managed.WithPollInterval(options.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name) /*nolint:staticcheck // GetEventRecorderFor is marked as deprecated but is not yet replaced with an alternative in controller-runtime, and the APIRecorder is still required for recording events.*/)),
	}

	if options.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	if options.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(options.ChangeLogOptions.ChangeLogger))
	}

	if options.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(options.MetricOptions.MRMetrics))
	}

	if options.MetricOptions != nil && options.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), options.Logger, options.MetricOptions.MRStateMetrics, &v1alpha1.WebhookList{}, options.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.WebhookList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.WebhookGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Webhook{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect
// method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) integration.WebhooksClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	webhook, isValid := managedResource.(*v1alpha1.Webhook)
	if !isValid {
		return nil, errors.New(errNotWebhook)
	}

	err := c.usage.Track(ctx, webhook)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	modernManaged, isValid := managedResource.(resource.ModernManaged)
	if !isValid {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	config, err := common.GetConfig(ctx, c.kube, modernManaged)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	svc := c.newServiceFn(*config)

	return &external{kubeClient: c.kube, client: svc}, nil
}

// external implements the ExternalClient interface for Webhook resources.
type external struct {
	kubeClient client.Client
	client     integration.WebhooksClient
}

// Observe checks whether the Webhook exists in SonarQube and whether
// it is up to date.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	webhookCR, ok := mg.(*v1alpha1.Webhook)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotWebhook)
	}

	externalName := meta.GetExternalName(webhookCR)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	projectKey := ptr.Deref(webhookCR.Spec.ForProvider.ProjectKey, "")

	list, resp, err := c.client.List(ctx, &sonar.WebhooksListOptions{Project: projectKey}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errListWebhooks)
	}

	found := integration.FindWebhookByKey(list.Webhooks, externalName)
	if found == nil {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	former := webhookCR.Spec.ForProvider.DeepCopy()
	webhookCR.Status.AtProvider = integration.GenerateWebhookObservation(found)
	integration.LateInitializeWebhook(&webhookCR.Spec.ForProvider, &webhookCR.Status.AtProvider)

	webhookCR.SetConditions(xpv1.Available())

	currentSecret, err := c.readSecret(ctx, webhookCR)
	if err != nil {
		return managed.ExternalObservation{
			ResourceExists: true,
		}, err
	}

	savedSecret, err := c.getSavedSecret(ctx, webhookCR)
	if err != nil {
		return managed.ExternalObservation{
			ResourceExists: true,
		}, err
	}

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        integration.IsWebhookUpToDate(&webhookCR.Spec.ForProvider, &webhookCR.Status.AtProvider, currentSecret, savedSecret),
		ResourceLateInitialized: integration.IsWebhookLateInitialized(former, &webhookCR.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the Webhook in SonarQube and sets the external name
// to the key assigned by the API.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	webhookCR, ok := mg.(*v1alpha1.Webhook)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotWebhook)
	}

	secret, err := c.readSecret(ctx, webhookCR)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	projectKey := ptr.Deref(webhookCR.Spec.ForProvider.ProjectKey, "")

	createOptions := integration.GenerateWebhookCreateOptions(&webhookCR.Spec.ForProvider, projectKey, secret)

	result, resp, err := c.client.Create(ctx, createOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateWebhook)
	}

	if result == nil || result.Webhook.Key == "" {
		return managed.ExternalCreation{}, errors.New("API response missing webhook key")
	}

	meta.SetExternalName(webhookCR, result.Webhook.Key)

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{connectionDetailSecretKey: []byte(secret)}}, nil
}

// Update updates the Webhook in SonarQube to match the desired state.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	webhookCR, ok := mg.(*v1alpha1.Webhook)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotWebhook)
	}

	externalName := meta.GetExternalName(webhookCR)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf(errExternalNameNotSet, webhookCR.Name)
	}

	secret, err := c.readSecret(ctx, webhookCR)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	updateOptions := integration.GenerateWebhookUpdateOptions(externalName, &webhookCR.Spec.ForProvider, secret)

	resp, err := c.client.Update(ctx, updateOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateWebhook)
	}

	return managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{connectionDetailSecretKey: []byte(secret)}}, nil
}

// Delete removes the Webhook from SonarQube.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	webhookCR, ok := mg.(*v1alpha1.Webhook)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotWebhook)
	}

	webhookCR.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(webhookCR)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.Delete(ctx, &sonar.WebhooksDeleteOptions{Webhook: externalName}) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteWebhook)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op for the Webhook controller.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// readSecret fetches the HMAC secret value from the referenced Kubernetes
// Secret, returning an empty string when SecretRef is not set.
func (c *external) readSecret(ctx context.Context, webhookCR *v1alpha1.Webhook) (string, error) {
	if webhookCR.Spec.ForProvider.SecretRef == nil {
		return "", nil
	}

	secretValue, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, webhookCR, webhookCR.Spec.ForProvider.SecretRef)
	if err != nil {
		return "", errors.Wrap(err, errGetSecret)
	}

	return *secretValue, nil
}

// getSavedSecret reads the last-applied secret value from the connection
// secret referenced in writeConnectionSecretToRef. Returns an empty
// string (not an error) when the secret or the key is absent.
func (c *external) getSavedSecret(ctx context.Context, webhookCR *v1alpha1.Webhook) (string, error) {
	ref := webhookCR.GetWriteConnectionSecretToReference()
	if ref == nil || ref.Name == "" {
		return "", nil
	}

	val, err := common.GetTokenValueFromLocalSecretReference(ctx, c.kubeClient, webhookCR, ref, connectionDetailSecretKey)
	if err != nil {
		if strings.Contains(err.Error(), common.ErrSecretNotFound) || strings.Contains(err.Error(), common.ErrSecretKeyNotFound) {
			return "", nil
		}

		return "", errors.Wrap(err, "cannot get saved secret")
	}

	return ptr.Deref(val, ""), nil
}
