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

package almgithub

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	errNotAlmGithub = "managed resource is not an AlmGithub custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"

	errCreateAlmGithub  = "cannot create SonarQube ALM GitHub integration"
	errUpdateAlmGithub  = "cannot update SonarQube ALM GitHub integration"
	errDeleteAlmGithub  = "cannot delete SonarQube ALM GitHub integration"
	errObserveAlmGithub = "cannot observe SonarQube ALM GitHub integration"
	errResolveSecrets   = "cannot resolve secret references"
)

// SetupGated adds a controller that reconciles AlmGithub managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup AlmGithub controller"))
		}
	}, v1alpha1.AlmGithubGroupVersionKind)

	return nil
}

func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.AlmGithubGroupKind)

	options := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewAlmClient,
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if opts.Features.Enabled(feature.EnableBetaManagementPolicies) {
		options = append(options, managed.WithManagementPolicies())
	}

	if opts.Features.Enabled(feature.EnableAlphaChangeLogs) {
		options = append(options, managed.WithChangeLogger(opts.ChangeLogOptions.ChangeLogger))
	}

	if opts.MetricOptions != nil {
		options = append(options, managed.WithMetricRecorder(opts.MetricOptions.MRMetrics))
	}

	if opts.MetricOptions != nil && opts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.Logger, opts.MetricOptions.MRStateMetrics, &v1alpha1.AlmGithubList{}, opts.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.AlmGithubList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.AlmGithubGroupVersionKind), options...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.AlmGithub{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) instance.AlmClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	almGithub, isValid := managedResource.(*v1alpha1.AlmGithub)
	if !isValid {
		return nil, errors.New(errNotAlmGithub)
	}

	err := c.usage.Track(ctx, almGithub)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	// Switch to ModernManaged resource to get ProviderConfigRef
	modernManaged, isValid := managedResource.(resource.ModernManaged)
	if !isValid {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	config, err := common.GetConfig(ctx, c.kube, modernManaged)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	svc := c.newServiceFn(*config)

	return &external{almClient: svc, kube: c.kube}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// almClient is used to interact with SonarQube ALM Settings API
	almClient instance.AlmClient
	// kube is used to resolve secret references
	kube client.Client
}

// Observe checks if the external resource exists and if it matches the desired state of the managed resource.
func (c *external) Observe(ctx context.Context, managedResource resource.Managed) (managed.ExternalObservation, error) {
	almGithub, isValid := managedResource.(*v1alpha1.AlmGithub)
	if !isValid {
		return managed.ExternalObservation{}, errors.New(errNotAlmGithub)
	}

	// Use external name as the identifier to check if the resource exists
	externalName := meta.GetExternalName(almGithub)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Retrieve the ALM definitions from SonarQube
	definitions, resp, err := c.almClient.ListDefinitions() //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errObserveAlmGithub)
	}

	// Find the GitHub definition matching our key
	definition := instance.FindGithubDefinition(definitions, externalName)
	if definition == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update status with observed state
	almGithub.Status.AtProvider = instance.GenerateAlmGithubObservation(definition)
	almGithub.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instance.IsAlmGithubUpToDate(almGithub.Spec.ForProvider, almGithub.Status.AtProvider),
	}, nil
}

// Create creates the external resource in SonarQube.
func (c *external) Create(ctx context.Context, managedResource resource.Managed) (managed.ExternalCreation, error) {
	almGithub, isValid := managedResource.(*v1alpha1.AlmGithub)
	if !isValid {
		return managed.ExternalCreation{}, errors.New(errNotAlmGithub)
	}

	almGithub.Status.SetConditions(xpv1.Creating())

	secrets, err := c.resolveSecrets(ctx, almGithub)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errResolveSecrets)
	}

	createOptions := instance.GenerateAlmGithubCreateOptions(almGithub.Spec.ForProvider, *secrets)

	resp, err := c.almClient.CreateGithub(createOptions) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateAlmGithub)
	}

	// Set the external name to the key of the created ALM GitHub setting
	meta.SetExternalName(almGithub, almGithub.Spec.ForProvider.Key)

	return managed.ExternalCreation{}, nil
}

// Update updates the external resource to match the desired state of the managed resource.
func (c *external) Update(ctx context.Context, managedResource resource.Managed) (managed.ExternalUpdate, error) {
	almGithub, isValid := managedResource.(*v1alpha1.AlmGithub)
	if !isValid {
		return managed.ExternalUpdate{}, errors.New(errNotAlmGithub)
	}

	secrets, err := c.resolveSecrets(ctx, almGithub)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errResolveSecrets)
	}

	updateOptions := instance.GenerateAlmGithubUpdateOptions(almGithub.Spec.ForProvider, *secrets)

	resp, err := c.almClient.UpdateGithub(updateOptions) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAlmGithub)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the external resource from SonarQube.
func (c *external) Delete(ctx context.Context, managedResource resource.Managed) (managed.ExternalDelete, error) {
	almGithub, isValid := managedResource.(*v1alpha1.AlmGithub)
	if !isValid {
		return managed.ExternalDelete{}, errors.New(errNotAlmGithub)
	}

	almGithub.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(almGithub)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	deleteOptions := instance.GenerateAlmDeleteOptions(externalName)

	resp, err := c.almClient.Delete(deleteOptions) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAlmGithub)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is called when the external resource is disconnected from the provider.
func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

// resolveSecrets resolves all secret references from the AlmGithub spec.
func (c *external) resolveSecrets(ctx context.Context, almGithub *v1alpha1.AlmGithub) (*instance.AlmGithubSecrets, error) {
	clientSecret, err := common.GetTokenValueFromSecret(ctx, c.kube, almGithub, &almGithub.Spec.ForProvider.ClientSecretSecretRef)
	if err != nil {
		return nil, errors.Wrap(err, "cannot resolve clientSecretSecretRef")
	}

	privateKey, err := common.GetTokenValueFromSecret(ctx, c.kube, almGithub, &almGithub.Spec.ForProvider.PrivateKeySecretRef)
	if err != nil {
		return nil, errors.Wrap(err, "cannot resolve privateKeySecretRef")
	}

	secrets := &instance.AlmGithubSecrets{
		ClientSecret: *clientSecret,
		PrivateKey:   *privateKey,
	}

	if almGithub.Spec.ForProvider.WebhookSecretSecretRef != nil {
		webhookSecret, err := common.GetTokenValueFromSecret(ctx, c.kube, almGithub, almGithub.Spec.ForProvider.WebhookSecretSecretRef)
		if err != nil {
			return nil, errors.Wrap(err, "cannot resolve webhookSecretSecretRef")
		}

		secrets.WebhookSecret = webhookSecret
	}

	return secrets, nil
}
