# provider-sonarqube

## Overview

`provider-sonarqube` is the Crossplane infrastructure provider for
[SonarQube](https://www.sonarqube.org/). The provider that is built from the source code
in this repository can be installed into a Crossplane control plane and adds the
following new functionality:

* Custom Resource Definitions (CRDs) that model SonarQube resources
* Controllers to provision these resources in SonarQube based on the users desired
  state captured in CRDs they create
* Implementations of Crossplane's portable resource
  abstractions, enabling
  SonarQube resources to fulfill a user's general need for SonarQube configurations

## Getting Started and Documentation

Create a [User Token](https://docs.sonarsource.com/sonarqube-server/user-guide/managing-tokens) on your SonarQube instance and fill in the corresponding Kubernetes secret:

```bash
kubectl create secret generic example-provider-secret -n default --from-literal=credentials="<USER_TOKEN>"
```

Configure a `ProviderConfig` with a baseURL pointing to your SonarQube instance (you can use either token-based authentication or basic auth):

```yaml
apiVersion: sonarqube.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: example
  namespace: default
spec:
  baseURL: http://sonarqube.example.com/api
  token:
    source: Secret
    secretRef:
      namespace: default
      name: example-provider-secret
      key: token
```

```bash
kubectl apply -f examples/providerconfig.yaml
```

## Developing

1. Clone the repository using: `git clone https://github.com/crossplane-contrib/provider-sonarqube.git`
2. Run `make submodules` to initialize the "build" Make submodule we use for CI/CD.

### Adding a new type

Add your new type by running the following command:

```shell
  export provider_name=SonarQube # Camel case, e.g. GitHub
  export group=instance # lower case e.g. core, cache, database, storage, etc.
  export type=QualityGate # Camel casee.g. Bucket, Database, CacheCluster, etc.
  make provider.addtype provider=${provider_name} group=${group} kind=${type}
```

1. Register your new type into `SetupGated` function in `internal/controller/register.go`
2. Run `make reviewable` to run code generation, linters, and tests.
3. Run `make build` to build the provider.

## Contributing

provider-sonarqube is a community driven project and we welcome contributions.

Refer to Crossplane's [CONTRIBUTING.md] file for more information on how the
Crossplane community prefers to work. The [Provider Development][provider-dev]
guide may also be of use.

[CONTRIBUTING.md]: https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md
[provider-dev]: https://github.com/crossplane/crossplane/blob/master/contributing/guide-provider-development.md

## Report a Bug

For filing bugs, suggesting improvements, or requesting new features, please
open an [issue](https://github.com/crossplane-contrib/provider-sonarqube/issues).

## Contact

Please use the following to reach members of the community:

* Slack: Join our [slack channel](https://slack.crossplane.io)
* Forums:
  [crossplane-dev](https://groups.google.com/forum/#!forum/crossplane-dev)
* Twitter: [@crossplane_io](https://twitter.com/crossplane_io)
* Email: [info@crossplane.io](mailto:info@crossplane.io)

## Governance and Owners

provider-sonarqube is run according to the same
[Governance](https://github.com/crossplane/crossplane/blob/master/GOVERNANCE.md)
and [Ownership](https://github.com/crossplane/crossplane/blob/master/OWNERS.md)
structure as the core Crossplane project.

## Code of Conduct

provider-sonarqube adheres to the same [Code of
Conduct](https://github.com/crossplane/crossplane/blob/master/CODE_OF_CONDUCT.md)
as the core Crossplane project.

## Licensing

provider-sonarqube is under the Apache 2.0 license.

[![FOSSA
Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fcrossplane-contrib%2Fprovider-sonarqube.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fcrossplane-contrib%2Fprovider-sonarqube?ref=badge_large)
