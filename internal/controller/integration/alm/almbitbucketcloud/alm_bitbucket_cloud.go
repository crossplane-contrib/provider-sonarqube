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

// Package almbitbucketcloud provides a controller for ALMBitbucketCloud
// resources.
package almbitbucketcloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"k8s.io/utils/ptr"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
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
	// errNotALMBitbucketCloud indicates the managed resource is not
	// ALMBitbucketCloud.
	errNotALMBitbucketCloud = "managed resource is not a ALMBitbucketCloud custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"

	// errExternalNameNotSet indicates the external name is not set.
	errExternalNameNotSet = "external name is not set for ALMBitbucketCloud resource %s"

	// connectionDetailClientSecretKey is the client secret key.
	connectionDetailClientSecretKey = "clientSecret"
)

// SetupGated adds a controller that reconciles ALMBitbucketCloud managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup ALMBitbucketCloud controller"))
		}
	}, v1alpha1.ALMBitbucketCloudGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles ALMBitbucketCloud managed resources.
func Setup(mgr ctrl.Manager, options controller.Options) error {
	name := managed.ControllerName(v1alpha1.ALMBitbucketCloudGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
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
			mgr.GetClient(), options.Logger, options.MetricOptions.MRStateMetrics, &v1alpha1.ALMBitbucketCloudList{}, options.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.ALMBitbucketCloudList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.ALMBitbucketCloudGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.ALMBitbucketCloud{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	almBitbucketCloud, isValid := managedResource.(*v1alpha1.ALMBitbucketCloud)
	if !isValid {
		return nil, errors.New(errNotALMBitbucketCloud)
	}

	err := c.usage.Track(ctx, almBitbucketCloud)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	// Switch to ModernManaged resource to get ProviderConfigRef
	modernManaged, isValid := managedResource.(resource.ModernManaged)
	if !isValid {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	config, err := common.GetConfig(ctx, c.kube, modernManaged)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	if config == nil {
		return nil, errors.New(errGetPC)
	}

	return &external{
		kubeClient:         c.kube,
		integrationsClient: integration.NewALMIntegrationsBitbucketCloudClient(*config),
		settingsClient:     integration.NewALMSettingsBitbucketCloudClient(*config),
	}, nil
}

// external is used to observe and manipulate the external
// ALMBitbucketCloud resource in the SonarQube API. It implements
// the managed.ExternalClient interface.
type external struct {
	// kubeClient is the Kubernetes API client that can be used to get and update the managed resource. This is expected to be initialized by the connector when the external client is created.
	kubeClient client.Client
	// integrationsClient can be used to observe and manipulate ALMBitbucketCloud resources in SonarQube API. This is expected to be initialized by the connector when the external client is created.
	integrationsClient integration.ALMIntegrationsBitbucketCloudClient
	// settingsClient can be used to observe and manipulate SonarQube settings related to ALMBitbucketCloud integration. This is expected to be initialized by the connector when the external client is created.
	settingsClient integration.ALMSettingsBitbucketCloudClient
}

// Observe observes the ALMBitbucketCloud corresponding external resource
// using the SonarQube API, and returns an ExternalObservation.
// It is used to determine if the resource exists,
// is up to date, and late initialize the spec if needed.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	almBitbucketCloud, ok := mg.(*v1alpha1.ALMBitbucketCloud)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotALMBitbucketCloud)
	}

	// Use the external name as the identifier to get the resource
	externalName := meta.GetExternalName(almBitbucketCloud)
	if externalName == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	clientSecret, err := c.getRequiredClientSecret(ctx, almBitbucketCloud)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	savedClientSecret, err := c.getSavedClientSecret(ctx, almBitbucketCloud)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	almDefinitions, resp, err := c.settingsClient.ListDefinitions(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot list ALM settings definitions from SonarQube API")
	}

	definition := integration.FindBitbucketCloudALMDefinitionByKey(&almDefinitions.BitbucketCloud, externalName)
	if definition == nil {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	almBitbucketCloud.Status.AtProvider = integration.GenerateALMBitbucketCloudObservation(definition)

	// Late initialize the ALMBitbucketCloud spec from the API response to determine if any fields need to be defaulted.
	former := almBitbucketCloud.Spec.ForProvider.DeepCopy()
	integration.LateInitializeALMBitbucketCloud(&almBitbucketCloud.Spec.ForProvider, &almBitbucketCloud.Status.AtProvider)

	almBitbucketCloud.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        integration.IsALMBitbucketCloudUpToDate(&almBitbucketCloud.Spec.ForProvider, &almBitbucketCloud.Status.AtProvider, clientSecret, savedClientSecret),
		ResourceLateInitialized: integration.IsALMBitbucketCloudLateInitialized(former, &almBitbucketCloud.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the ALMBitbucketCloud external resource using the SonarQube
// API, based on the desired state of the managed resource.
// It returns an ExternalCreation that may include connection details to be
// stored in a secret.
// It also sets the external name of the managed resource to the identifier of
// the created external resource if it is not already set.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	almBitbucketCloud, ok := mg.(*v1alpha1.ALMBitbucketCloud)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotALMBitbucketCloud)
	}

	clientSecret, err := c.getRequiredClientSecret(ctx, almBitbucketCloud)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	createOptions := integration.GenerateALMBitbucketCloudCreateOptions(&almBitbucketCloud.Spec.ForProvider, clientSecret)

	resp, err := c.settingsClient.CreateBitbucketCloud(ctx, createOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create ALMBitbucketCloud resource in SonarQube API")
	}

	meta.SetExternalName(almBitbucketCloud, createOptions.Key)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			connectionDetailClientSecretKey: []byte(clientSecret),
		},
	}, nil
}

// Update updates the ALMBitbucketCloud corresponding external resource to
// reflect the desired state of the managed resource.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	almBitbucketCloud, ok := mg.(*v1alpha1.ALMBitbucketCloud)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotALMBitbucketCloud)
	}

	externalName := meta.GetExternalName(almBitbucketCloud)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf(errExternalNameNotSet, almBitbucketCloud.Name)
	}

	clientSecret, err := c.getRequiredClientSecret(ctx, almBitbucketCloud)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	updateOptions := integration.GenerateALMBitbucketCloudUpdateOptions(externalName, &almBitbucketCloud.Spec.ForProvider, clientSecret)

	resp, err := c.settingsClient.UpdateBitbucketCloud(ctx, updateOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update ALMBitbucketCloud resource in SonarQube API")
	}

	if updateOptions.NewKey != "" {
		// If the Key was updated, also update the external name to the new Key to keep it in sync with the identifier of the ALMBitbucketCloud resource in SonarQube API.
		meta.SetExternalName(almBitbucketCloud, updateOptions.NewKey)

		kubeErr := c.kubeClient.Update(ctx, almBitbucketCloud)
		if kubeErr != nil {
			return managed.ExternalUpdate{}, errors.Wrap(kubeErr, "cannot update external name annotation after key change")
		}
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{
			connectionDetailClientSecretKey: []byte(clientSecret),
		},
	}, nil
}

// Delete deletes the ALMBitbucketCloud corresponding external resource using
// the SonarQube API. It returns an ExternalDelete that may include connection
// details to be stored in a secret.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	almBitbucketCloud, ok := mg.(*v1alpha1.ALMBitbucketCloud)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotALMBitbucketCloud)
	}

	almBitbucketCloud.Status.SetConditions(xpv1.Deleting())

	// Use the external name as the identifier to delete the resource
	externalName := meta.GetExternalName(almBitbucketCloud)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	deleteOptions := integration.GenerateALMDeleteOptions(externalName)

	resp, err := c.settingsClient.Delete(ctx, deleteOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete ALMBitbucketCloud resource in SonarQube API")
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect closes the external client connection.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// getSavedClientSecret retrieves the saved client secret from the connection
// secret referenced in the ALMBitbucketCloud spec.
// Returns an empty string (not an error) if the connection secret or the key
// is missing.
func (c *external) getSavedClientSecret(ctx context.Context, almBitbucketCloud *v1alpha1.ALMBitbucketCloud) (string, error) {
	ref := almBitbucketCloud.GetWriteConnectionSecretToReference()
	if ref == nil || ref.Name == "" {
		return "", nil
	}

	clientSecret, err := common.GetTokenValueFromLocalSecretReference(ctx, c.kubeClient, almBitbucketCloud, ref, connectionDetailClientSecretKey)
	if err != nil {
		if strings.Contains(err.Error(), common.ErrSecretNotFound) || strings.Contains(err.Error(), common.ErrSecretKeyNotFound) {
			return "", nil
		}

		return "", errors.Wrap(err, "cannot get saved client secret")
	}

	return ptr.Deref(clientSecret, ""), nil
}

// getRequiredClientSecret retrieves the client secret from the secret
// reference declared in the ALMBitbucketCloud spec.
func (c *external) getRequiredClientSecret(ctx context.Context, almBitbucketCloud *v1alpha1.ALMBitbucketCloud) (string, error) {
	clientSecret, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, almBitbucketCloud, almBitbucketCloud.Spec.ForProvider.ClientSecretRef)

	switch {
	case err != nil:
		return "", errors.Wrap(err, "cannot get client secret from secret reference")
	case ptr.Deref(clientSecret, "") == "":
		return "", errors.New("client secret is empty or not provided")
	}

	return *clientSecret, nil
}
