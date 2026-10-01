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

## Observe cache (alpha)

Many managed resources of the same kind usually read the same SonarQube list
endpoint during `Observe` (for example, every `Plugin` reads the list of
installed plugins). The observe cache shares those responses between
reconciles for a short time, which cuts the number of API calls sent to
SonarQube when many resources are managed.

The cache is **disabled by default**. Enable it with the following flags, for
example through a `DeploymentRuntimeConfig`:

| Flag | Environment variable | Default | Description |
| --- | --- | --- | --- |
| `--enable-observe-cache` | `ENABLE_OBSERVE_CACHE` | `false` | Enables the cache. |
| `--observe-cache-ttl` | `OBSERVE_CACHE_TTL` | `20s` | Lifetime of a cached response. Must be greater than `0` and lower than `30s` (Crossplane's creation grace period). |
| `--observe-cache-max-entries` | `OBSERVE_CACHE_MAX_ENTRIES` | `1000` | Maximum number of cached responses. The least recently used are evicted first. Must be greater than `0`. |

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

Things to know before enabling it:

* **Staleness:** changes made by the provider itself invalidate the affected
  cached responses immediately. Drift introduced outside of Crossplane (in the
  SonarQube UI, by another tool, ...) can however be detected up to one TTL
  later than without the cache.
* **Scope:** cached responses are scoped per SonarQube connection, identified by
  a hash of the base URL, authentication type and credentials of the
  `ProviderConfig`. Resources using different credentials never share cached
  data, even against the same instance. Raw credentials are never kept in the
  cache.
* **Errors are never cached.**
* **Memory:** the cache is bounded by entry count. When it is enabled, the
  provider also sets `GOMEMLIMIT` to 90% of the container memory limit read
  from the pod's cgroup (using
  [automemlimit](https://github.com/KimMachineGun/automemlimit)), so that the
  Go garbage collector works harder before the pod gets OOM-killed. Set
  `GOMEMLIMIT` yourself to override it, or `AUTOMEMLIMIT` to change the ratio
  (`off` disables it). Set the pod memory limit with the provider's own needs
  and the cached responses in mind.

Supported resources:

| Resource | Cached endpoints | Status |
| --- | --- | --- |
| `Plugin` | `plugins/installed`, `plugins/pending`, `plugins/updates` | Supported |
| `ALMAzure`, `ALMBitbucket`, `ALMBitbucketCloud`, `ALMGitHub`, `ALMGitLab` | `alm_settings/list_definitions` | Planned ([#123](https://github.com/crossplane-contrib/provider-sonarqube/issues/123)) |
| `Permissions`, `Group` | permissions search | Planned ([#124](https://github.com/crossplane-contrib/provider-sonarqube/issues/124)) |
| `PermissionsTemplate` | permission template search | Planned ([#125](https://github.com/crossplane-contrib/provider-sonarqube/issues/125)) |
| Quality Gate & Quality Profile usergroup associations | `search_groups`, `search_users` | Planned ([#126](https://github.com/crossplane-contrib/provider-sonarqube/issues/126)) |
| `Webhook`, `UserToken` | scoped list endpoints | Planned ([#127](https://github.com/crossplane-contrib/provider-sonarqube/issues/127)) |

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
