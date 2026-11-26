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

// Package portfolio provides a controller for Portfolio resources.
package portfolio

import (
	"context"
	"net/http"
	"strings"

	"github.com/boxboxjason/sonarqube-client-go/v2/sonar"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
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
	// errTrackPCUsage indicates ProviderConfig usage tracking failed.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC indicates ProviderConfig retrieval failed.
	errGetPC = "cannot get ProviderConfig"
	// errExternalNameNotSet indicates the external name annotation is missing.
	errExternalNameNotSet = "external name is not set for Portfolio resource %s"

	// errObservePortfolio indicates portfolio observation from SonarQube failed.
	errObservePortfolio = "cannot observe SonarQube Portfolio"
	// errCreatePortfolio indicates portfolio creation in SonarQube failed.
	errCreatePortfolio = "cannot create SonarQube Portfolio"
	// errUpdatePortfolio indicates portfolio update in SonarQube failed.
	errUpdatePortfolio = "cannot update SonarQube Portfolio"
	// errDeletePortfolio indicates portfolio deletion in SonarQube failed.
	errDeletePortfolio = "cannot delete SonarQube Portfolio"
	// errSetSelectionMode indicates selection mode configuration failed.
	errSetSelectionMode = "cannot set selection mode for SonarQube Portfolio"
	// errObserveParent indicates the parent portfolio observation failed.
	errObserveParent = "cannot observe parent SonarQube Portfolio %s"
	// errMovePortfolio indicates moving the portfolio under its parent failed.
	errMovePortfolio = "cannot move SonarQube Portfolio under parent %s"
	// errAddProject indicates adding a project to the portfolio failed.
	errAddProject = "cannot add project %s to SonarQube Portfolio"
	// errRemoveProject indicates removing a project from the portfolio failed.
	errRemoveProject = "cannot remove project %s from SonarQube Portfolio"
	// errAddProjectBranch indicates selecting a project branch failed.
	errAddProjectBranch = "cannot add branch %s of project %s to SonarQube Portfolio"
	// errRemoveProjectBranch indicates deselecting a project branch failed.
	errRemoveProjectBranch = "cannot remove branch %s of project %s from SonarQube Portfolio"
	// errAddApplication indicates adding an application failed.
	errAddApplication = "cannot add application %s to SonarQube Portfolio"
	// errRemoveApplication indicates removing an application failed.
	errRemoveApplication = "cannot remove application %s from SonarQube Portfolio"
	// errAddApplicationBranch indicates selecting an application branch failed.
	errAddApplicationBranch = "cannot add branch %s of application %s to SonarQube Portfolio"
	// errRemoveApplicationBranch indicates deselecting an app branch failed.
	errRemoveApplicationBranch = "cannot remove branch %s of application %s from SonarQube Portfolio"
	// errAddSubPortfolio indicates referencing a portfolio failed.
	errAddSubPortfolio = "cannot add portfolio %s to SonarQube Portfolio"
	// errRemoveSubPortfolio indicates removing a referenced portfolio failed.
	errRemoveSubPortfolio = "cannot remove portfolio %s from SonarQube Portfolio"

	// selectionModeNone is the default portfolio selection mode.
	selectionModeNone = "NONE"
	// selectionModeManual is the manual project selection mode.
	selectionModeManual = instance.SelectionModeManual
	// selectionModeRegexp is the regexp-based project selection mode.
	selectionModeRegexp = "REGEXP"
	// selectionModeRemaining is the remaining-projects selection mode.
	selectionModeRemaining = "REMAINING"
	// selectionModeTags is the tag-based project selection mode.
	selectionModeTags = "TAGS"
)

// SetupGated adds a controller that reconciles Portfolio managed
// resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o helpers.Options) error {
	o.Gate.Register(func() {
		err := Setup(mgr, o)
		if err != nil {
			panic(errors.Wrap(err, "cannot setup Portfolio controller"))
		}
	}, v1alpha1.PortfolioGroupVersionKind)

	return nil
}

// Setup adds a controller that reconciles Portfolio managed resources.
func Setup(mgr ctrl.Manager, options helpers.Options) error {
	name := managed.ControllerName(v1alpha1.PortfolioGroupKind)

	opts := helpers.ReconcilerOptions(mgr, name, options,
		managed.WithTypedExternalConnector[*v1alpha1.Portfolio](&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: instance.NewPortfoliosClient,
		}),
		helpers.WithSpecExternalName(mgr.GetClient(), func(mg *v1alpha1.Portfolio) string {
			return mg.Spec.ForProvider.Key
		}),
	)

	err := helpers.AddMRStateRecorder(mgr, options, &v1alpha1.PortfolioList{})
	if err != nil {
		return err
	}

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.PortfolioGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(options.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Portfolio{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, options.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its
// Connect method is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(config common.Config) instance.PortfoliosClient
}

// Connect produces an ExternalClient by tracking ProviderConfig usage,
// retrieving credentials, and constructing a SonarQube portfolios client.
func (c *connector) Connect(ctx context.Context, portfolio *v1alpha1.Portfolio) (managed.TypedExternalClient[*v1alpha1.Portfolio], error) {
	err := c.usage.Track(ctx, portfolio)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	config, err := common.GetConfig(ctx, c.kube, portfolio)
	if err != nil || config == nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	svc := c.newServiceFn(*config)

	return &external{client: svc}, nil
}

// external implements the ExternalClient interface for Portfolio resources.
type external struct {
	client instance.PortfoliosClient
}

// Observe checks if the Portfolio exists in SonarQube and whether it
// matches the desired state.
func (c *external) Observe(ctx context.Context, portfolio *v1alpha1.Portfolio) (managed.ExternalObservation, error) {
	externalName := meta.GetExternalName(portfolio)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	result, resp, err := c.client.Show(ctx, &sonar.ViewsShowOptions{Key: externalName}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		if common.IsResponseNotFound(resp) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errObservePortfolio)
	}

	observation := instance.GeneratePortfolioObservation(result)

	observation.Parent, err = c.observeParent(ctx, externalName, portfolio.Spec.ForProvider.Parent)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	portfolio.Status.AtProvider = observation
	portfolio.SetConditions(xpv1.Available())

	former := portfolio.Spec.ForProvider.DeepCopy()
	instance.LateInitializePortfolio(&portfolio.Spec.ForProvider, &portfolio.Status.AtProvider)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        instance.IsPortfolioUpToDate(&portfolio.Spec.ForProvider, &portfolio.Status.AtProvider),
		ResourceLateInitialized: instance.IsPortfolioLateInitialized(former, &portfolio.Spec.ForProvider),
		ConnectionDetails:       managed.ConnectionDetails{},
	}, nil
}

// Create creates the Portfolio in SonarQube, under its parent when set, and
// sets the external name to the portfolio key. Members are reconciled by the
// subsequent Update.
func (c *external) Create(ctx context.Context, portfolio *v1alpha1.Portfolio) (managed.ExternalCreation, error) {
	portfolio.Status.SetConditions(xpv1.Creating())

	resp, err := c.client.Create(ctx, instance.GeneratePortfolioCreateOptions(&portfolio.Spec.ForProvider)) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreatePortfolio)
	}

	err = c.setSelectionMode(ctx, portfolio.Spec.ForProvider.Key, &portfolio.Spec.ForProvider)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errSetSelectionMode)
	}

	meta.SetExternalName(portfolio, portfolio.Spec.ForProvider.Key)

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Update updates the Portfolio name, description, parent and selection
// mode in SonarQube, then reconciles its projects, applications and
// referenced portfolios.
func (c *external) Update(ctx context.Context, portfolio *v1alpha1.Portfolio) (managed.ExternalUpdate, error) {
	externalName := meta.GetExternalName(portfolio)
	if externalName == "" {
		return managed.ExternalUpdate{}, errors.Errorf(errExternalNameNotSet, portfolio.Name)
	}

	spec := &portfolio.Spec.ForProvider
	observation := &portfolio.Status.AtProvider

	resp, err := c.client.Update(ctx, &sonar.ViewsUpdateOptions{ //nolint:bodyclose // closed via helpers.CloseBody
		Key:         externalName,
		Name:        spec.Name,
		Description: spec.Description,
	})
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdatePortfolio)
	}

	err = c.moveUnderParent(ctx, externalName, spec, observation)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	err = c.updateSelectionMode(ctx, externalName, spec, observation)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errSetSelectionMode)
	}

	err = c.updateMembers(ctx, instance.PortfolioProjectsDiff(spec, observation), c.projectCalls(externalName))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	err = c.updateMembers(ctx, instance.PortfolioApplicationsDiff(spec, observation), c.applicationCalls(externalName))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	err = c.updateSubPortfolios(ctx, externalName, instance.PortfolioSubPortfoliosToAdd(spec, observation), instance.PortfolioSubPortfoliosToRemove(spec, observation))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

// Delete deletes the Portfolio from SonarQube.
func (c *external) Delete(ctx context.Context, portfolio *v1alpha1.Portfolio) (managed.ExternalDelete, error) {
	portfolio.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(portfolio)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	resp, err := c.client.Delete(ctx, &sonar.ViewsDeleteOptions{Key: externalName}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	// Deleting a portfolio deletes its sub-portfolios, which may therefore
	// already be gone.
	if err != nil && !common.IsResponseNotFound(resp) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeletePortfolio)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op because the SonarQube client is stateless.
func (c *external) Disconnect(_ context.Context) error {
	return nil
}

// normalizeSelectionMode returns the NONE selection mode for an empty
// selection mode.
func normalizeSelectionMode(mode string) string {
	if mode == "" {
		return selectionModeNone
	}

	return mode
}

// observeParent returns the desired parent key when the portfolio is a
// sub-portfolio of it, or an empty string otherwise. SonarQube does not
// return the parent of a portfolio, so the sub-portfolios of the desired
// parent are looked up instead.
func (c *external) observeParent(ctx context.Context, key string, parent *string) (string, error) {
	if parent == nil {
		return "", nil
	}

	details, resp, err := c.client.Show(ctx, &sonar.ViewsShowOptions{Key: *parent}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		if common.IsResponseNotFound(resp) {
			return "", nil
		}

		return "", errors.Wrapf(err, errObserveParent, *parent)
	}

	if instance.IsPortfolioSubPortfolio(details, key) {
		return *parent, nil
	}

	return "", nil
}

// moveUnderParent moves the portfolio under its desired parent when it is
// not a sub-portfolio of it yet.
func (c *external) moveUnderParent(ctx context.Context, key string, spec *v1alpha1.PortfolioParameters, observation *v1alpha1.PortfolioObservation) error {
	if instance.IsPortfolioParentUpToDate(spec, observation) {
		return nil
	}

	resp, err := c.client.Move(ctx, &sonar.ViewsMoveOptions{Key: key, Destination: *spec.Parent}) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	return errors.Wrapf(err, errMovePortfolio, *spec.Parent)
}

// updateSelectionMode sets the selection mode of the portfolio. Setting the
// MANUAL selection mode again could reset the selected projects, so it is
// only set when the portfolio is in another mode. Other modes are always
// set, since their settings are not observed.
func (c *external) updateSelectionMode(ctx context.Context, key string, spec *v1alpha1.PortfolioParameters, observation *v1alpha1.PortfolioObservation) error {
	if normalizeSelectionMode(spec.SelectionMode) == selectionModeManual && observation.SelectionMode == selectionModeManual {
		return nil
	}

	return c.setSelectionMode(ctx, key, spec)
}

// portfolioMemberCalls are the SonarQube calls, and their error messages,
// reconciling the projects or the applications of a portfolio.
type portfolioMemberCalls struct {
	add          func(ctx context.Context, member string) (*http.Response, error)
	remove       func(ctx context.Context, member string) (*http.Response, error)
	addBranch    func(ctx context.Context, member, branch string) (*http.Response, error)
	removeBranch func(ctx context.Context, member, branch string) (*http.Response, error)

	errAdd          string
	errRemove       string
	errAddBranch    string
	errRemoveBranch string
}

// projectCalls returns the calls reconciling the projects of a portfolio.
func (c *external) projectCalls(key string) portfolioMemberCalls { //nolint:dupl // same shape as applicationCalls, distinct endpoints
	return portfolioMemberCalls{
		add: func(ctx context.Context, project string) (*http.Response, error) {
			return c.client.AddProject(ctx, &sonar.ViewsAddProjectOptions{Key: key, Project: project})
		},
		remove: func(ctx context.Context, project string) (*http.Response, error) {
			return c.client.RemoveProject(ctx, &sonar.ViewsRemoveProjectOptions{Key: key, Project: project})
		},
		addBranch: func(ctx context.Context, project, branch string) (*http.Response, error) {
			return c.client.AddProjectBranch(ctx, &sonar.ViewsAddProjectBranchOptions{Key: key, Project: project, Branch: branch})
		},
		removeBranch: func(ctx context.Context, project, branch string) (*http.Response, error) {
			return c.client.RemoveProjectBranch(ctx, &sonar.ViewsRemoveProjectBranchOptions{Key: key, Project: project, Branch: branch})
		},
		errAdd:          errAddProject,
		errRemove:       errRemoveProject,
		errAddBranch:    errAddProjectBranch,
		errRemoveBranch: errRemoveProjectBranch,
	}
}

// applicationCalls returns the calls reconciling the applications of a
// portfolio.
func (c *external) applicationCalls(key string) portfolioMemberCalls { //nolint:dupl // same shape as projectCalls, distinct endpoints
	return portfolioMemberCalls{
		add: func(ctx context.Context, application string) (*http.Response, error) {
			return c.client.AddApplication(ctx, &sonar.ViewsAddApplicationOptions{Portfolio: key, Application: application})
		},
		remove: func(ctx context.Context, application string) (*http.Response, error) {
			return c.client.RemoveApplication(ctx, &sonar.ViewsRemoveApplicationOptions{Portfolio: key, Application: application})
		},
		addBranch: func(ctx context.Context, application, branch string) (*http.Response, error) {
			return c.client.AddApplicationBranch(ctx, &sonar.ViewsAddApplicationBranchOptions{Key: key, Application: application, Branch: branch})
		},
		removeBranch: func(ctx context.Context, application, branch string) (*http.Response, error) {
			return c.client.RemoveApplicationBranch(ctx, &sonar.ViewsRemoveApplicationBranchOptions{Key: key, Application: application, Branch: branch})
		},
		errAdd:          errAddApplication,
		errRemove:       errRemoveApplication,
		errAddBranch:    errAddApplicationBranch,
		errRemoveBranch: errRemoveApplicationBranch,
	}
}

// updateMembers adds the missing members, selects their missing branches,
// deselects their unwanted branches, then removes the unwanted members of
// the portfolio.
func (c *external) updateMembers(ctx context.Context, diff instance.PortfolioMembersDiff, calls portfolioMemberCalls) error {
	for _, member := range diff.Add {
		resp, err := calls.add(ctx, member) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrapf(err, calls.errAdd, member)
		}
	}

	for _, branch := range diff.AddBranches {
		resp, err := calls.addBranch(ctx, branch.Key, branch.Branch) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrapf(err, calls.errAddBranch, branch.Branch, branch.Key)
		}
	}

	for _, branch := range diff.RemoveBranches {
		resp, err := calls.removeBranch(ctx, branch.Key, branch.Branch) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrapf(err, calls.errRemoveBranch, branch.Branch, branch.Key)
		}
	}

	for _, member := range diff.Remove {
		resp, err := calls.remove(ctx, member) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrapf(err, calls.errRemove, member)
		}
	}

	return nil
}

// updateSubPortfolios references the missing portfolios, then removes the
// unwanted referenced portfolios.
func (c *external) updateSubPortfolios(ctx context.Context, key string, toAdd, toRemove []string) error {
	for _, reference := range toAdd {
		resp, err := c.client.AddPortfolio(ctx, &sonar.ViewsAddPortfolioOptions{Portfolio: key, Reference: reference}) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrapf(err, errAddSubPortfolio, reference)
		}
	}

	for _, reference := range toRemove {
		resp, err := c.client.RemovePortfolio(ctx, &sonar.ViewsRemovePortfolioOptions{Portfolio: key, Reference: reference}) //nolint:bodyclose // closed via helpers.CloseBody
		helpers.CloseBody(resp)

		if err != nil {
			return errors.Wrapf(err, errRemoveSubPortfolio, reference)
		}
	}

	return nil
}

// setSelectionMode sets the project selection mode for the portfolio
// in SonarQube.
func (c *external) setSelectionMode(ctx context.Context, key string, spec *v1alpha1.PortfolioParameters) error {
	switch normalizeSelectionMode(spec.SelectionMode) {
	case selectionModeNone:
		resp, err := c.client.SetNoneMode(ctx, &sonar.ViewsSetNoneModeOptions{Portfolio: key}) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(resp)

		return errors.Wrap(err, "cannot set NONE selection mode")
	case selectionModeManual:
		resp, err := c.client.SetManualMode(ctx, &sonar.ViewsSetManualModeOptions{Portfolio: key}) //nolint:bodyclose // closed via helpers.CloseBody
		defer helpers.CloseBody(resp)

		return errors.Wrap(err, "cannot set MANUAL selection mode")
	case selectionModeRegexp:
		resp, err := c.client.SetRegexpMode(ctx, &sonar.ViewsSetRegexpModeOptions{ //nolint:bodyclose // closed via helpers.CloseBody
			Portfolio: key,
			Regexp:    spec.Regexp,
			Branch:    spec.Branch,
		})
		defer helpers.CloseBody(resp)

		return errors.Wrap(err, "cannot set REGEXP selection mode")
	case selectionModeRemaining:
		resp, err := c.client.SetRemainingProjectsMode(ctx, &sonar.ViewsSetRemainingProjectsModeOptions{ //nolint:bodyclose // closed via helpers.CloseBody
			Portfolio: key,
			Branch:    spec.Branch,
		})
		defer helpers.CloseBody(resp)

		return errors.Wrap(err, "cannot set REMAINING selection mode")
	case selectionModeTags:
		resp, err := c.client.SetTagsMode(ctx, &sonar.ViewsSetTagsModeOptions{ //nolint:bodyclose // closed via helpers.CloseBody
			Portfolio: key,
			Tags:      strings.Split(spec.Tags, ","),
			Branch:    spec.Branch,
		})
		defer helpers.CloseBody(resp)

		return errors.Wrap(err, "cannot set TAGS selection mode")
	default:
		return nil
	}
}
