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

// Package usertoken provides a controller for UserToken resources.
package usertoken

import (
	"context"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/iam/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/iam"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errNotUserToken indicates the managed resource is not a UserToken.
	errNotUserToken = "managed resource is not a UserToken custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errObserveToken indicates UserToken observation failed.
	errObserveToken = "cannot observe SonarQube UserToken"
	// errCreateToken indicates UserToken creation failed.
	errCreateToken = "cannot create SonarQube UserToken"
	// errDeleteToken indicates UserToken deletion failed.
	errDeleteToken = "cannot delete SonarQube UserToken"
	// errRenewToken indicates UserToken renewal failed.
	errRenewToken = "cannot renew SonarQube UserToken"

	// connectionSecretTokenKey is the key used in the connection secret.
	connectionSecretTokenKey = "token"
)

// SetupGated adds a controller that reconciles UserToken managed resources
// with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup UserToken controller"))
		}
	}, v1alpha1.UserTokenGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles UserToken managed resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.UserTokenGroupKind)

	options := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: iam.NewUserTokensClient,
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name) /*nolint:staticcheck // GetEventRecorderFor is marked as deprecated but is not yet replaced with an alternative in controller-runtime, and the APIRecorder is still required for recording events.*/)),
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
			mgr.GetClient(), opts.Logger, opts.MetricOptions.MRStateMetrics, &v1alpha1.UserTokenList{}, opts.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.UserTokenList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.UserTokenGroupVersionKind), options...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.UserToken{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// A connector produces an ExternalClient when its Connect method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) iam.UserTokensClient
}

// Connect produces an ExternalClient by tracking ProviderConfig usage,
// retrieving credentials, and constructing a SonarQube user tokens client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	usertoken, isUserToken := managedResource.(*v1alpha1.UserToken)
	if !isUserToken {
		return nil, errors.New(errNotUserToken)
	}

	err := c.usage.Track(ctx, usertoken)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	modernManaged, isModernManaged := managedResource.(resource.ModernManaged)
	if !isModernManaged {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	config, err := common.GetConfig(ctx, c.kube, modernManaged)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	return &external{client: c.newServiceFn(*config)}, nil
}

// external implements the ExternalClient interface for UserToken resources.
type external struct {
	client iam.UserTokensClient
}

// Observe checks whether the token exists in SonarQube and populates
// status.atProvider.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	usertoken, ok := mg.(*v1alpha1.UserToken)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUserToken)
	}

	externalName := meta.GetExternalName(usertoken)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	result, resp, err := c.client.Search(iam.GenerateUserTokenSearchOptions(&usertoken.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errObserveToken)
	}

	found := iam.FindUserToken(result, externalName)
	if found == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	usertoken.Status.AtProvider = iam.GenerateUserTokenObservation(found)
	usertoken.Status.AtProvider.RenewAt = iam.ComputeRenewAt(
		usertoken.Status.AtProvider.ExpirationDate,
		usertoken.Spec.ForProvider.RenewBeforeDays,
	)

	former := usertoken.Spec.ForProvider.DeepCopy()
	iam.LateInitializeUserToken(&usertoken.Spec.ForProvider, &usertoken.Status.AtProvider)
	usertoken.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        iam.IsUserTokenUpToDate(&usertoken.Spec.ForProvider, &usertoken.Status.AtProvider),
		ResourceLateInitialized: iam.IsUserTokenLateInitialized(former, &usertoken.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create generates a new token in SonarQube and stores the token value in
// the connection secret.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	usertoken, ok := mg.(*v1alpha1.UserToken)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUserToken)
	}

	usertoken.Status.SetConditions(xpv1.Creating())

	generated, resp, err := c.client.Generate(iam.GenerateUserTokenCreateOptions(&usertoken.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateToken)
	}

	meta.SetExternalName(usertoken, generated.Name)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			connectionSecretTokenKey: []byte(generated.Token),
		},
	}, nil
}

// Update is only triggered for RenewalPeriodDays mode when the token has
// expired. It revokes the old token, which forces SonarQube to generate a
// new token.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	usertoken, ok := mg.(*v1alpha1.UserToken)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUserToken)
	}

	externalName := meta.GetExternalName(usertoken)
	if externalName == "" {
		return managed.ExternalUpdate{}, errors.New("cannot renew UserToken without external name")
	}

	revokeResp, err := c.client.Revoke(iam.GenerateUserTokenRevokeOptions(externalName, &usertoken.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(revokeResp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errRenewToken)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete revokes the token in SonarQube.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	usertoken, ok := mg.(*v1alpha1.UserToken)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUserToken)
	}

	usertoken.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(usertoken)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.Revoke(iam.GenerateUserTokenRevokeOptions(externalName, &usertoken.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteToken)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op because the SonarQube client is stateless.
func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
