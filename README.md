# provider-sonarqube

## Overview

`provider-sonarqube` brings SonarQube configuration under Crossplane management. Install
it into a control plane and declare SonarQube resources as Kubernetes manifests instead of
managing them through ad hoc scripts or manual clicks.

The provider currently covers the areas teams usually need to standardize first:

* Instance resources such as projects, quality gates, quality profiles, rules, and settings
* IAM resources such as groups and permissions templates
* Integration resources such as GitLab ALM configuration

## Why This Provider

Use `provider-sonarqube` when you want SonarQube configuration to behave like the rest of
your platform:

* GitOps-friendly reconciliation for SonarQube state
* Crossplane compositions and abstractions for reusable platform building blocks
* A single declarative model for shared quality policies and project configuration

## Getting Started and Documentation

1. Create a [User Token](https://docs.sonarsource.com/sonarqube-server/user-guide/managing-tokens) on your SonarQube instance. It should preferably have admin permissions to ensure you can manage all resource types.
2. Store it in a Kubernetes secret:

    ```bash
    kubectl create secret generic example-provider-secret -n default --from-literal=token="<USER_TOKEN>"
    ```

3. Configure a `ProviderConfig` that points at your SonarQube instance. You can use token-based authentication or basic auth.

    ```yaml
    apiVersion: sonarqube.crossplane.io/v1alpha1
    kind: ProviderConfig
    metadata:
      name: example
      namespace: default
    spec:
      baseUrl: http://sonarqube.example.com/api
      token:
        source: Secret
        secretRef:
          namespace: default
          name: example-provider-secret
          key: token
    ```

4. Apply the example manifests in this repository to get started quickly.

    ```bash
    kubectl apply -f examples/providerconfig.yaml
    ```

Available examples live under `examples/` and are a practical starting point for
understanding the resource shapes supported by the provider.

## Configuration

Set flags or environment variables through a `DeploymentRuntimeConfig`. Boolean
flags also accept a `--no-` prefix.

| Flag | Environment variable | Default | Description |
| --- | --- | --- | --- |
| `--debug`, `-d` | `CROSSPLANE_SONARQUBE_PROVIDER_DEBUG` | `false` | Enable debug logging. |
| `--leader-election`, `-l` | `LEADER_ELECTION` | `false` | Use leader election for the controller manager. |
| `--sync`, `-s` | `CROSSPLANE_SONARQUBE_PROVIDER_SYNC` | `1h` | How often all resources are re-checked for drift. |
| `--poll` | `CROSSPLANE_SONARQUBE_PROVIDER_POLL` | `1m` | How often each resource is checked for drift. |
| `--min-poll-interval` | `MIN_POLL_INTERVAL` | `30s` | Shortest interval the `crossplane.io/poll-interval` annotation can request. `0` removes the limit. |
| `--poll-state-metric` | `CROSSPLANE_SONARQUBE_PROVIDER_POLL_STATE_METRIC` | `5s` | State metric recording interval. |
| `--max-reconcile-rate` | `CROSSPLANE_SONARQUBE_PROVIDER_MAX_RECONCILE_RATE` | `10` | Maximum reconciles per second. |
| `--enable-management-policies` | `ENABLE_MANAGEMENT_POLICIES` | `true` | Enable Management Policies. |
| `--enable-changelogs` | `ENABLE_CHANGE_LOGS` | `false` | Capture change logs during reconciliation. |
| `--changelogs-socket-path` | `CHANGELOGS_SOCKET_PATH` | `/var/run/changelogs/changelogs.sock` | Change logs socket path. |
| `--sonarqube-response-header-timeout` | `SONARQUBE_RESPONSE_HEADER_TIMEOUT` | `30s` | Maximum wait for SonarQube's response headers. `0` disables it. |
| `--enable-secret-cache` | `ENABLE_SECRET_CACHE` | `true` | Read Secrets through the informer cache. See [Kubernetes Secret access](#kubernetes-secret-access). |
| `--enable-observe-cache` | `ENABLE_OBSERVE_CACHE` | `false` | Alpha. Cache SonarQube list responses. See [Observe cache](#observe-cache-alpha). |
| `--observe-cache-ttl` | `OBSERVE_CACHE_TTL` | `20s` | Lifetime of a cached response. Must be above `0` and below `30s`. |
| `--observe-cache-max-entries` | `OBSERVE_CACHE_MAX_ENTRIES` | `1000` | Maximum cached responses. Must be above `0`. |

```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: provider-sonarqube
spec:
  deploymentTemplate:
    spec:
      selector: {}
      template:
        spec:
          containers:
            - name: package-runtime
              env:
                - name: ENABLE_OBSERVE_CACHE
                  value: "true"
```

## Kubernetes Secret access

Secret references on namespaced objects resolve only inside that object's
namespace. Managed resource Secret references (user passwords, license keys,
ALM credentials, webhook secrets) take only a `name` and `key`, and the Secret
is always read from the managed resource's namespace. A `ProviderConfig`'s
credentials must be in the `ProviderConfig`'s namespace; a reference to another
namespace is rejected. Only the cluster-scoped `ClusterProviderConfig` can
reference Secrets in any namespace.

By default, Secrets are read through an informer cache, which needs
cluster-wide `list` and `watch` on Secrets. With `--no-enable-secret-cache`,
each read goes to the Kubernetes API, so the provider only needs `get` on the
Secrets it reads, plus `get`, `create` and `patch` on connection Secrets.

* The flag does not change RBAC: restrict the provider's ServiceAccount yourself.
* Each read counts against the client limit of 5 × `--max-reconcile-rate`
  requests per second, which can slow reconciliation at scale.

## Reconciliation frequency

The provider checks every managed resource for drift every `--poll` interval,
one minute by default. Two annotations adjust this for a single resource:

- `crossplane.io/poll-interval` overrides the interval, for example `1h` for a
  rarely changing resource. A value below `--min-poll-interval` is raised to it,
  so that a namespace tenant cannot use up the reconcile budget that
  `--max-reconcile-rate` shares between all tenants.
- `crossplane.io/reconcile-requested-at` triggers a reconcile when its value
  changes. The handled value is reported in `status.lastHandledReconcileAt`.

## Cross-resource references

References and selectors between managed resources (for example
`groupIdRef` or `qualityGateNameSelector`) always resolve in the referencing
resource's own `metadata.namespace`. They have no `namespace` field, so one
tenant cannot read identifiers from another tenant's resources.

When upgrading, a value that was already resolved from another namespace
stays in `spec.forProvider` because the default `IfNotPresent` policy does
not re-resolve it. Clear the resolved field (for example `groupId` or
`projectKey`) so it resolves again in the resource's own namespace. Fields
marked immutable cannot be cleared, so recreate those resources instead.

## Observe cache (alpha)

Shares SonarQube list responses between reconciles for a short time, which
cuts API calls when many resources are managed. Enable it with
`--enable-observe-cache`.

* **Staleness:** drift from outside Crossplane can be detected up to one TTL
  late. The provider's own changes invalidate entries immediately.
* **Scope:** entries are never shared between credentials. Errors are not cached.
* **Memory:** `GOMEMLIMIT` is set to 90% of the container limit, unless you set
  it yourself. `AUTOMEMLIMIT` changes the ratio (`off` disables it).

Supported resources:

| Resource | Cached endpoints | Status |
| --- | --- | --- |
| `Plugin` | `plugins/installed`, `plugins/pending`, `plugins/updates` | Supported |
| `ALMAzure`, `ALMBitbucket`, `ALMBitbucketCloud`, `ALMGitHub`, `ALMGitLab` | `alm_settings/list_definitions`, shared by every ALM kind | Supported |
| `Permissions`, `Group` | `permissions/groups` (unfiltered per project, by name for global permissions), `permissions/users` (by login) | Supported |
| `PermissionsTemplate` | `permissions/search_templates` (indexed once per instance), `permissions/template_groups`, `permissions/template_users` (per template) | Supported |
| `QualityGateUsergroupAssociation`, `QualityProfileUsergroupAssociation` | `qualitygates/search_groups`, `qualitygates/search_users`, `qualityprofiles/search_groups`, `qualityprofiles/search_users` (selected principals, per gate or profile) | Supported |
| `Webhook`, `UserToken` | `webhooks/list` (per project), `user_tokens/search` (per login) | Supported |

When the cache is enabled, the provider exposes the following metrics:

| Metric | Type | Labels | Description |
| --- | --- | --- | --- |
| `provider_sonarqube_observe_cache_requests_total` | Counter | `namespace`, `result` (`hit`, `miss`, `coalesced`) | Cached reads. `coalesced` reads shared an in-flight request made by a concurrent reconcile. |
| `provider_sonarqube_observe_cache_invalidations_total` | Counter | `namespace` | Invalidations triggered by writes. |
| `provider_sonarqube_observe_cache_entries` | Gauge | | Number of cached responses. |

## Documentation

* [CRD documentation](https://marketplace.upbound.io/providers/crossplane-contrib/provider-sonarqube/latest/crds)
* [Contributing guide](CONTRIBUTING.md)
* [Issue tracker](https://github.com/crossplane-contrib/provider-sonarqube/issues)

## Support

If you run into a problem or want to request a feature, please open an issue in the
[provider repository](https://github.com/crossplane-contrib/provider-sonarqube/issues).

## Licensing

`provider-sonarqube` is licensed under Apache 2.0. See [LICENSE](LICENSE) for full license text.

[![FOSSA
Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fcrossplane-contrib%2Fprovider-sonarqube.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fcrossplane-contrib%2Fprovider-sonarqube?ref=badge_large)
