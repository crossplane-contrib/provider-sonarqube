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

package almgitlab

import (
	"context"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"k8s.io/utils/ptr"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
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
	errNotALMGitLab = "managed resource is not a ALMGitLab custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"

	errExternalNameNotSet = "external name is not set for ALMGitLab resource %s"

	connectionDetailTokenKey = "token"
)

// SetupGated adds a controller that reconciles ALMGitLab managed resources
//
//	with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup ALMGitLab controller"))
		}
	}, v1alpha1.ALMGitLabGroupVersionKind)

	return nil
}

func Setup(mgr ctrl.Manager, options controller.Options) error {
	name := managed.ControllerName(v1alpha1.ALMGitLabGroupKind)

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
			mgr.GetClient(), options.Logger, options.MetricOptions.MRStateMetrics, &v1alpha1.ALMGitLabList{}, options.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.ALMGitLabList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.ALMGitLabGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.ALMGitLab{}).
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
	almgitlab, isValid := managedResource.(*v1alpha1.ALMGitLab)
	if !isValid {
		return nil, errors.New(errNotALMGitLab)
	}

	err := c.usage.Track(ctx, almgitlab)
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
		integrationsClient: integration.NewALMIntegrationsGitLabClient(*config),
		settingsClient:     integration.NewALMSettingsGitLabClient(*config),
	}, nil
}

// external is used to observe and manipulate the external resource (ALMGitLab)
// in SonarQube API. It implements the managed.ExternalClient interface.
type external struct {
	// kubeClient is the Kubernetes API client that can be used to get and update the managed resource. This is expected to be initialized by the connector when the external client is created.
	kubeClient client.Client
	// integration client that can be used to observe and manipulate ALMGitLab resources in SonarQube API. This is expected to be initialized by the connector when the external client is created.
	integrationsClient integration.ALMIntegrationsGitLabClient
	// settingsClient can be used to observe and manipulate SonarQube settings related to ALMGitLab integration. This is expected to be initialized by the connector when the external client is created.
	settingsClient integration.ALMSettingsGitLabClient
}

// Observe observes the ALMGitLab corresponding external resource using the
// SonarQube API, and returns an ExternalObservation.
// It is used to determine if the resource exists,
// is up to date, and late initialize the spec if needed.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	almGitlab, ok := mg.(*v1alpha1.ALMGitLab)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotALMGitLab)
	}

	// Use external name as the identifier to get the resource
	externalName := meta.GetExternalName(almGitlab)
	if externalName == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	pat, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, almGitlab, almGitlab.Spec.ForProvider.PersonalAccessTokenSecretRef)
	switch {
	case err != nil:
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get personal access token from secret reference")
	case pat == nil:
		return managed.ExternalObservation{}, errors.New("personal access token is nil")
	case *pat == "":
		return managed.ExternalObservation{}, errors.New("personal access token is empty")
	}

	savedAPIToken, err := c.getSavedAPIToken(ctx, almGitlab)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	almDefinitions, resp, err := c.settingsClient.ListDefinitions() //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot list ALM settings definitions from SonarQube API")
	}

	definition := integration.FindGitLabALMDefinitionByKey(&almDefinitions.Gitlab, externalName)

	if definition == nil {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	almGitlab.Status.AtProvider = integration.GenerateALMGitLabObservation(definition)

	// Late initialize the ALMGitLab spec with the API response to determine if there are any fields that can be defaulted.
	former := almGitlab.Spec.ForProvider.DeepCopy()
	integration.LateInitializeALMGitLab(&almGitlab.Spec.ForProvider, &almGitlab.Status.AtProvider)

	almGitlab.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        integration.IsALMGitLabUpToDate(&almGitlab.Spec.ForProvider, *pat, &almGitlab.Status.AtProvider, savedAPIToken),
		ResourceLateInitialized: integration.IsALMGitLabLateInitialized(former, &almGitlab.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the ALMGitLab external resource using the SonarQube API,
// based on the desired state of the managed resource.
// It returns an ExternalCreation which may include connection details to be
// stored in a secret.
// It also sets the external name of the managed resource to the identifier
// of the created external resource if it is not already set.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	almGitlab, ok := mg.(*v1alpha1.ALMGitLab)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotALMGitLab)
	}

	pat, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, almGitlab, almGitlab.Spec.ForProvider.PersonalAccessTokenSecretRef)
	switch {
	case err != nil:
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot get personal access token from secret reference")
	case pat == nil:
		return managed.ExternalCreation{}, errors.New("personal access token is nil")
	case *pat == "":
		return managed.ExternalCreation{}, errors.New("personal access token is empty")
	}

	createOptions := integration.GenerateALMGitLabCreateOptions(&almGitlab.Spec.ForProvider, *pat)

	resp, err := c.settingsClient.CreateGitlab(createOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create ALMGitLab resource in SonarQube API")
	}

	// Set the external name to the Name of the created ALMGitLab resource if it is not already set. This assumes that the Name of the ALMGitLab resource in SonarQube API is used as the identifier for the resource, and that it is returned in the API response after creation.
	meta.SetExternalName(almGitlab, createOptions.Key)

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{
			connectionDetailTokenKey: []byte(*pat),
		},
	}, nil
}

// Update updates the ALMGitLab corresponding external resource to reflect
// the desired state of the managed resource.
// It returns an ExternalUpdate which may include connection details to be
// stored in a secret.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	almGitlab, ok := mg.(*v1alpha1.ALMGitLab)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotALMGitLab)
	}

	externalName := meta.GetExternalName(almGitlab)
	if externalName == "" {
		return managed.ExternalUpdate{}, fmt.Errorf(errExternalNameNotSet, almGitlab.Name)
	}

	pat, err := common.GetTokenValueFromLocalSecret(ctx, c.kubeClient, almGitlab, almGitlab.Spec.ForProvider.PersonalAccessTokenSecretRef)
	switch {
	case err != nil:
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot get personal access token from secret reference")
	case pat == nil:
		return managed.ExternalUpdate{}, errors.New("personal access token is nil")
	case *pat == "":
		return managed.ExternalUpdate{}, errors.New("personal access token is empty")
	}

	updateOptions := integration.GenerateALMGitLabUpdateOptions(externalName, &almGitlab.Spec.ForProvider, *pat)

	resp, err := c.settingsClient.UpdateGitlab(updateOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update ALMGitLab resource in SonarQube API")
	}

	if updateOptions.NewKey != "" {
		// If the Key was updated, also update the external name to the new Key to keep it in sync with the identifier of the ALMGitLab resource in SonarQube API.
		meta.SetExternalName(almGitlab, updateOptions.NewKey)

		kubeErr := c.kubeClient.Update(ctx, almGitlab)
		if kubeErr != nil {
			return managed.ExternalUpdate{}, errors.Wrap(kubeErr, "cannot update external name annotation after key change")
		}
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{
			connectionDetailTokenKey: []byte(*pat),
		},
	}, nil
}

// Delete deletes the ALMGitLab corresponding external resource using the
// SonarQube API. It returns an ExternalDelete which may include
// connection details to be stored in a secret.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	almGitlab, ok := mg.(*v1alpha1.ALMGitLab)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotALMGitLab)
	}

	almGitlab.Status.SetConditions(xpv1.Deleting())

	// Use external name as the identifier to delete the resource
	externalName := meta.GetExternalName(almGitlab)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	deleteOptions := integration.GenerateALMDeleteOptions(externalName)

	resp, err := c.settingsClient.Delete(deleteOptions) //nolint:bodyclose // closed via helpers.CloseBody
	helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete ALMGitLab resource in SonarQube API")
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

// getSavedAPIToken retrieves the saved API token from the connection secret.
func (c *external) getSavedAPIToken(ctx context.Context, almGitlab *v1alpha1.ALMGitLab) (string, error) {
	ref := almGitlab.GetWriteConnectionSecretToReference()
	if ref == nil || ref.Name == "" {
		return "", nil
	}

	pat, err := common.GetTokenValueFromLocalSecretReference(ctx, c.kubeClient, almGitlab, ref, connectionDetailTokenKey)
	if err != nil {
		if strings.Contains(err.Error(), common.ErrSecretNotFound) || strings.Contains(err.Error(), common.ErrSecretKeyNotFound) {
			return "", nil
		}

		return "", errors.Wrap(err, "cannot get saved API token")
	}

	return ptr.Deref(pat, ""), nil
}
