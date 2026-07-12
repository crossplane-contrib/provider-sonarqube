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

// Package rule provides a controller for Rule resources.
package rule

import (
	"context"
	"net/http"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

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

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-sonarqube/apis/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	"github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

const (
	// errNotRule indicates managed resource is not a Rule.
	errNotRule = "managed resource is not a Rule custom resource"
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
)

// SetupGated adds a controller that reconciles Rule managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup Rule controller"))
		}
	}, v1alpha1.RuleGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles Rule managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error { //nolint:varnamelen // consistent with other controllers
	name := managed.ControllerName(v1alpha1.RuleGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewRulesClient}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	if o.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(o.ChangeLogOptions.ChangeLogger))
	}

	if o.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(o.MetricOptions.MRMetrics))
	}

	if o.MetricOptions != nil && o.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &v1alpha1.RuleList{}, o.MetricOptions.PollStateMetricInterval,
		)

		err := mgr.Add(stateMetricsRecorder)
		if err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.RuleList")
		}
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.RuleGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Rule{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) instance.RulesClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, managedResource resource.Managed) (managed.ExternalClient, error) {
	rule, isValid := managedResource.(*v1alpha1.Rule)
	if !isValid {
		return nil, errors.New(errNotRule)
	}

	err := c.usage.Track(ctx, rule)
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

	return &external{rulesClient: svc}, nil
}

// external implements the managed.ExternalClient interface for Rule resources.
type external struct {
	// rulesClient is used to interact with SonarQube Rules API
	rulesClient instance.RulesClient
}

// Observe observes the external resource and returns an ExternalObservation.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	rule, ok := mg.(*v1alpha1.Rule)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRule)
	}

	// Retrieve the external name, which is the identifier of the SonarQube Rule
	externalName := meta.GetExternalName(rule)
	if externalName == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Retrieve the SonarQube Rule using the external name
	ruleShow, resp, err := c.rulesClient.Show(ctx, instance.GenerateRuleGetOptions(externalName)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		if resp == nil {
			return managed.ExternalObservation{}, errors.Wrap(err, "failed to get SonarQube Rule: no response received")
		}

		switch resp.StatusCode {
		// Not found means the resource does not exist, and bad request likely means the external name is invalid, so we can treat both as non-existent resource
		case http.StatusNotFound, http.StatusBadRequest:
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		default:
			return managed.ExternalObservation{}, errors.Wrap(err, "failed to get SonarQube Rule")
		}
	}

	// SonarQube may still return a rule entry after deletion with status REMOVED.
	// Treat it as non-existent so Crossplane can finalize deletion.
	if ruleShow != nil && ruleShow.Rule.Status == sonar.RuleStatusRemoved {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	rule.Status.AtProvider = instance.GenerateRuleObservation(ruleShow)
	rule.Status.SetConditions(xpv1.Available())

	before := rule.Spec.ForProvider.DeepCopy()
	instance.LateInitializeRule(&rule.Spec.ForProvider, &rule.Status.AtProvider)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        instance.IsRuleUpToDate(&rule.Spec.ForProvider, &rule.Status.AtProvider),
		ResourceLateInitialized: instance.IsRuleLateInitialized(before, &rule.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the external resource based on the desired state of
// the managed resource.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	rule, ok := mg.(*v1alpha1.Rule)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRule)
	}

	rule.Status.SetConditions(xpv1.Creating())

	ruleCreateOption := instance.GenerateRuleCreateOptions(&rule.Spec.ForProvider)

	createdRule, resp, err := c.rulesClient.Create(ctx, ruleCreateOption) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to create SonarQube Rule")
	}

	meta.SetExternalName(rule, createdRule.Rule.Key)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// Update updates the external resource to reflect the desired state of
// the managed resource.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	rule, ok := mg.(*v1alpha1.Rule)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRule)
	}

	// Use external name as the identifier to update the resource
	externalName := meta.GetExternalName(rule)
	if externalName == "" {
		return managed.ExternalUpdate{}, errors.New("cannot update SonarQube Rule without external name")
	}

	ruleUpdateOptions := instance.GenerateRuleUpdateOptions(externalName, &rule.Spec.ForProvider)

	_, resp, err := c.rulesClient.Update(ctx, ruleUpdateOptions) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update SonarQube Rule")
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// Delete deletes the external resource if it exists.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	rule, ok := mg.(*v1alpha1.Rule)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRule)
	}

	rule.Status.SetConditions(xpv1.Deleting())

	// Use external name as the identifier to delete the resource
	externalName := meta.GetExternalName(rule)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.rulesClient.Delete(ctx, instance.GenerateRuleDeleteOptions(externalName)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		if resp == nil {
			return managed.ExternalDelete{}, errors.Wrap(err, "failed to delete SonarQube Rule: no response received")
		}

		switch resp.StatusCode {
		// Not found means the resource does not exist, and bad request likely means the external name is invalid, so we can treat both as non-existent resource and consider it deleted
		case http.StatusNotFound, http.StatusBadRequest:
			meta.SetExternalName(rule, "")

			return managed.ExternalDelete{}, nil
		default:
			return managed.ExternalDelete{}, errors.Wrap(err, "failed to delete SonarQube Rule")
		}
	}

	meta.SetExternalName(rule, "")

	return managed.ExternalDelete{}, nil
}

// Disconnect closes the external client connection.
func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
