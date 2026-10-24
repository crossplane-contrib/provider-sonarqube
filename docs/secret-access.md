# Least-privilege Kubernetes Secret access

The manager normally caches Kubernetes Secrets. Its first named Secret read can
start an informer that lists and watches Secrets across the manager's cache
scope. A ServiceAccount allowed to get only selected Secrets cannot populate
that cache.

Set `--no-enable-secret-cache` or `ENABLE_SECRET_CACHE=false` to send Secret
reads directly to the Kubernetes API. Secret caching defaults to `true` for
compatibility. Other Kubernetes kinds retain their existing caches. This is
independent of `--enable-observe-cache`, which caches SonarQube API responses.
Direct reads still require authorization and increase Kubernetes API traffic;
the option neither grants permissions nor narrows existing RBAC grants.

## Runtime configuration

```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: sonarqube-direct-secrets
spec:
  serviceAccountTemplate:
    metadata:
      name: sonarqube-runtime
  deploymentTemplate:
    spec:
      selector: {}
      template:
        spec:
          containers:
            - name: package-runtime
              args:
                - --no-enable-secret-cache
```

Set `spec.runtimeConfigRef.name: sonarqube-direct-secrets` on the installed
Provider. Preserve any other runtime arguments required by that installation.

## Paths covered

The option configures the standard controller-runtime manager client with
`client.CacheOptions.DisableFor` for `corev1.Secret`. No separate Kubernetes
client or token store is introduced.

| Path | Implementation and behavior |
| --- | --- |
| ProviderConfig / ClusterProviderConfig credentials | `internal/clients/common/sonar.go` calls the Secret helpers in `kubernetes.go` for token or username/password credentials. Referenced Secrets are point-read in their explicitly selected namespaces. ProviderConfig and usage objects retain their existing caching. |
| Other Secret references | The same helpers resolve local-user passwords, ALM credentials, webhook secrets, license keys and license endpoint credentials. Local references resolve in the managed resource's namespace. Saved connection details also use these helpers. |
| Connection Secret creation and republication | `crossplane-runtime` constructs `APISecretPublisher` and `APILocalSecretPublisher` with `mgr.GetClient()`. Their native patching applicator first gets the named Secret, then creates it if absent or patches changed details. All these Secret reads honor the option. Re-publishing identical details is a no-op. |
| UserToken creation and renewal | `internal/controller/iam/usertoken/user_token.go` returns the one-time token in `ConnectionDetails` from Create and renewal Update. The managed reconciler uses the local publisher to create or patch the Secret in the UserToken namespace. Renewal revokes and regenerates the SonarQube token; the option does not change this lifecycle. |
| Connection Secret removal | The native publishers' Unpublish methods are no-ops. Owner references allow Kubernetes garbage collection when the managed resource is deleted. |

SonarQube exposes a token's value only when generating it. This setting does not
recover a lost token value from the SonarQube search API, add a parallel token
store, or make token revocation and Kubernetes publication a cross-system
transaction. A missing connection Secret or failed publication needs operational
attention; successful renewal publishes the new value through the same uncached
path. Existing token renewal configuration and error handling remain unchanged.

## Runtime Secret permissions

For the current crossplane-runtime dependency, the relevant permissions are:

| Use | Secret verbs | Scope |
| --- | --- | --- |
| Read credentials and input references | `get` | Each referenced namespace and Secret name. |
| Read existing connection details | `get` | The named connection Secret. |
| Publish or rotate connection details | `get`, `create`, `patch` | The target namespace; named `get` and `patch` can be restricted with `resourceNames`. |

These paths do not require Secret `list`, `watch`, `update` or `delete` when
caching is disabled. Kubernetes RBAC cannot restrict a collection-level `create`
request using `resourceNames`. Granting Secret creation in a namespace allows
creation under any name in that namespace. Use a dedicated namespace or an
admission policy if name restrictions on creation are required.

For example, this Role grants access to one input credential and one connection
Secret in `team`, with namespace-scoped creation:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: sonarqube-secret-access
  namespace: team
rules:
  - apiGroups: [""]
    resources: [secrets]
    resourceNames: [sonarqube-credentials]
    verbs: [get]
  - apiGroups: [""]
    resources: [secrets]
    resourceNames: [analysis-token]
    verbs: [get, patch]
  - apiGroups: [""]
    resources: [secrets]
    verbs: [create]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: sonarqube-secret-access
  namespace: team
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: sonarqube-secret-access
subjects:
  - kind: ServiceAccount
    name: sonarqube-runtime
    namespace: crossplane-system
```

Adjust namespaces and names to the actual references. This is only the Secret
portion of runtime RBAC: controllers still need their managed resources,
ProviderConfigs, usage records, events, CRD discovery and any enabled leader-election permissions. Review all effective bindings, including generated or
aggregated ClusterRoles; a narrow Role does not cancel broader inherited access.

Installation permissions belong to the Crossplane package manager or installer,
which may need to create CRDs, runtime workloads and RBAC. Those installation
permissions are distinct from the provider Pod's ServiceAccount permissions.
This flag does not alter package installation or generated RBAC policy.

## Validation

The manager-client tests use a synthetic HTTP Kubernetes API, not a fake client.
They exercise credential refresh, native connection Secret creation, token value
replacement, identical-detail republication, forbidden reads/publication and
unchanged caching of other kinds. Secret list/watch requests are denied and
unexpected requests fail the test. Existing UserToken controller tests separately
cover token generation and revoke/regenerate renewal.

A real Kubernetes RBAC/admission and SonarQube end-to-end run remains a separate
qualification; the HTTP fixture is not an API server authorization implementation.

## References

Grafana — Optional Secret cache, upstream PR #701 (open when examined on 24 October 2026).

https://github.com/grafana/crossplane-provider-grafana/pull/701

Crossplane — crossplane-runtime v2.4.0, native connection publishers.

https://github.com/crossplane/crossplane-runtime/blob/v2.4.0/pkg/reconciler/managed/api.go

Kubernetes — RBAC: referring to resources (resourceNames restrictions).

https://kubernetes.io/docs/reference/access-authn-authz/rbac/#referring-to-resources
