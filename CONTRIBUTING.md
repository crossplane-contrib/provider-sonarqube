# Contributing

This repository follows the standard Crossplane contribution model, with a small amount of project-specific guidance below.

## Code Of Conduct And DCO

Please follow the Crossplane Code of Conduct and sign your commits with the Developer Certificate of Origin.

```text
Signed-off-by: John Doe <john.doe@example.org>
```

Use `git commit -s` to add the sign-off automatically.

## Getting Started

1. Fork the repository and create a branch for your work.
2. Clone the repository locally.
3. Run `make submodules` once so the shared build submodule is initialized.

## Working In This Repository

Keep changes focused and prefer small, reviewable commits. When adding or changing provider resources, make sure the corresponding controller, API registration, and tests are updated together.

## Makefile Overview

The most commonly used commands are:

* `make submodules`: initialize or refresh the shared build submodule.
* `make generate`: run code generation.
* `make lint`: run linters.
* `make test`: run unit tests.
* `make reviewable`: run generation, linting, and tests.
* `make build`: build the provider artifacts.
* `make dev`: start a local kind-based development loop.
* `make dev-clean`: delete the local development kind cluster.

## Adding A New Type

Use the helper target to scaffold a new resource type:

```shell
export provider_name=SonarQube
export group=instance
export type=QualityGate
make provider.addtype provider=${provider_name} group=${group} kind=${type}
```

After scaffolding, register the new type in `internal/controller/register.go`, then run
`make reviewable`.

## Local Validation

Before opening a pull request, run `make reviewable`. If your change affects behavior, also run the most relevant focused tests for the area you changed.

## E2E Testing Flow

The integration flow is driven by `make test-integration` or `make e2e.run`. It creates a kind cluster, installs Crossplane and the provider, then bootstraps an in-cluster SonarQube instance via `cluster/local/sonarqube_setup.sh` (Deployment + Service in the `default` namespace, plus a `Secret` and a `ClusterProviderConfig` named `e2e` wired to it). Provider pods reach SonarQube at `http://sonarqube.default.svc.cluster.local:9000/api`.

While the integration script is running you can inspect the in-cluster instance with:

```shell
kubectl logs deployment/sonarqube
kubectl port-forward svc/sonarqube 9000:9000   # then browse http://localhost:9000
```

The setup script is idempotent and can be re-run against a live cluster. Cluster teardown (driven by the script's exit trap) destroys the SonarQube Pod along with everything else.

For ad-hoc local exploration outside KIND, the standalone `make sonarqube.start` / `make sonarqube.stop` targets bring up SonarQube directly on the host via Docker or Podman.

For controller development, `make dev` starts a local kind cluster and runs the provider against it. Use `make dev-clean` to remove that cluster when you are done.

## CI

The CI workflow currently runs these high-level checks:

* linting
* a diff check to keep generated artifacts in sync
* unit tests with coverage publication
* artifact publishing for tagged and branch builds
