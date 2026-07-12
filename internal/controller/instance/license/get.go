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

package license

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"

	v1alpha1 "github.com/crossplane/provider-sonarqube/apis/instance/v1alpha1"
	"github.com/crossplane/provider-sonarqube/internal/clients/common"
	instance "github.com/crossplane/provider-sonarqube/internal/clients/instance"
	"github.com/crossplane/provider-sonarqube/internal/helpers"
)

// Observe observes the License corresponding external resource using the
// SonarQube API, and returns an ExternalObservation.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	license, ok := mg.(*v1alpha1.License)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotLicense)
	}

	licenseGet, resp, err := c.client.Get(ctx) //nolint:bodyclose // closed via helpers.CloseBody
	defer helpers.CloseBody(resp)

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetLicense)
	}

	license.Status.AtProvider = instance.GenerateLicenseObservation(&licenseGet.License)

	if licenseGet.License.ProductEdition == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	desiredLicenseKey, err := c.resolveLicenseKey(ctx, license)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetLicenseKey)
	}

	savedLicenseKey, err := c.getSavedLicenseKey(ctx, license)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get saved license key")
	}

	former := license.Spec.ForProvider.DeepCopy()
	instance.LateInitializeLicense(&license.Spec.ForProvider, &license.Status.AtProvider)

	license.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        instance.IsLicenseUpToDate(ptr.Deref(desiredLicenseKey, ""), savedLicenseKey),
		ResourceLateInitialized: instance.IsLicenseLateInitialized(former, &license.Spec.ForProvider),
	}, nil
}

// getSavedLicenseKey retrieves the license key that was last applied to
// SonarQube from the connection secret referenced in the License spec.
// Returns an empty string (not an error) if the connection secret or the
// key is missing.
func (c *external) getSavedLicenseKey(ctx context.Context, license *v1alpha1.License) (string, error) {
	ref := license.GetWriteConnectionSecretToReference()
	if ref == nil || ref.Name == "" {
		return "", nil
	}

	val, err := common.GetTokenValueFromLocalSecretReference(ctx, c.kube, license, ref, connectionDetailLicenseKeyKey)
	if err != nil {
		if strings.Contains(err.Error(), common.ErrSecretNotFound) || strings.Contains(err.Error(), common.ErrSecretKeyNotFound) {
			return "", nil
		}

		return "", errors.Wrap(err, "cannot get saved license key")
	}

	return ptr.Deref(val, ""), nil
}

// resolveLicenseKey resolves the desired license key from its configured
// source: either a direct secret reference, or an HTTP(S) endpoint.
// If neither is set, it returns an error, as a license key is required for
// creating or updating a License resource.
//
// If the source is an endpoint and it is temporarily unreachable, the
// endpoint is likely just down for maintenance or a vendor renewal is in
// progress, rather than a permanent misconfiguration. In that case,
// resolveLicenseKey falls back to the license key that was last saved to
// the connection secret, instead of failing the reconcile, so a transient
// endpoint outage does not put the resource into an error-retry loop or
// unset the SonarQube license. The endpoint error is only returned if no
// saved license key is available to fall back to (e.g. on first Create).
func (c *external) resolveLicenseKey(ctx context.Context, license *v1alpha1.License) (*string, error) {
	forProvider := license.Spec.ForProvider

	switch {
	case forProvider.LicenseKeySecretRef != nil:
		licenseKey, err := common.GetTokenValueFromSecret(ctx, c.kube, license, forProvider.LicenseKeySecretRef)
		if err != nil {
			return nil, errors.Wrap(err, "cannot get license key from secret")
		}

		return licenseKey, nil
	case forProvider.Endpoint != nil:
		licenseKey, err := c.licenseFromEndpoint(ctx, license, forProvider.Endpoint)
		if err == nil {
			return &licenseKey, nil
		}

		savedLicenseKey, savedErr := c.getSavedLicenseKey(ctx, license)
		if savedErr != nil || savedLicenseKey == "" {
			return nil, errors.Wrap(err, "cannot get license key from endpoint")
		}

		return &savedLicenseKey, nil
	default:
		return nil, errors.New("licenseKeySecretRef or endpoint must be set")
	}
}

// licenseFromEndpoint fetches the license key from the HTTP(S) endpoint
// configured in the given LicenseEndpoint, optionally authenticating the
// request.
func (c *external) licenseFromEndpoint(ctx context.Context, license *v1alpha1.License, endpoint *v1alpha1.LicenseEndpoint) (string, error) {
	auth, err := c.endpointAuth(ctx, license, endpoint)
	if err != nil {
		return "", err
	}

	licenseKey, err := common.FetchFromEndpoint(ctx, endpoint.URL, auth)
	if err != nil {
		return "", errors.Wrap(err, "cannot fetch license from endpoint")
	}

	return licenseKey, nil
}

// endpointAuth builds the authentication used to fetch the license from the
// endpoint, resolving any referenced secrets. It returns nil if the
// endpoint is unauthenticated.
func (c *external) endpointAuth(ctx context.Context, license *v1alpha1.License, endpoint *v1alpha1.LicenseEndpoint) (*common.EndpointAuth, error) {
	switch {
	case endpoint.BearerTokenSecretRef != nil:
		token, err := common.GetTokenValueFromSecret(ctx, c.kube, license, endpoint.BearerTokenSecretRef)
		if err != nil {
			return nil, errors.Wrap(err, "cannot get bearer token from secret")
		}

		return &common.EndpointAuth{BearerToken: ptr.Deref(token, "")}, nil
	case endpoint.BasicAuthUsername != nil:
		password, err := common.GetTokenValueFromSecret(ctx, c.kube, license, endpoint.BasicAuthPasswordSecretRef)
		if err != nil {
			return nil, errors.Wrap(err, "cannot get basic auth password from secret")
		}

		return &common.EndpointAuth{
			BasicAuthUsername: ptr.Deref(endpoint.BasicAuthUsername, ""),
			BasicAuthPassword: ptr.Deref(password, ""),
		}, nil
	default:
		return nil, nil //nolint:nilnil // nil auth, nil error means the endpoint is unauthenticated, not an error condition.
	}
}
