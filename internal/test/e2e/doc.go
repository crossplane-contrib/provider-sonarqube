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

// Package e2e provides reusable helpers for end-to-end tests that exercise
// the provider against a live SonarQube instance and a real Kubernetes API
// server.
//
// All files in this package carry the `e2e` build tag so they are excluded
// from regular `go test ./...` runs. To execute the suite use:
//
//	go test -tags=e2e ./internal/test/e2e/...
//
// or, equivalently, `make e2e.test`. Both invocations expect the following
// environment variables to be set, which `cluster/local/integration_tests.sh`
// exports automatically:
//
//	KUBECONFIG       - points at a cluster running the provider
//	SONARQUBE_URL    - base URL of the SonarQube API (e.g. http://localhost:9000/api)
//	SONARQUBE_TOKEN  - admin-scoped SonarQube token used to verify state
//
// The companion ClusterProviderConfig used by the managed resources under
// test is named `e2e` and is created by `cluster/local/sonarqube_setup.sh`.
//
// # Enterprise Edition suite
//
// internal/test/e2e/instance/license_test.go and portfolio_test.go carry an
// additional `enterprise` build tag (`//go:build e2e && enterprise`), since
// the License and Portfolio managed resources they exercise only work
// against a licensed Enterprise (or higher) edition SonarQube instance. Run
// them, together with the rest of the suite, against a second instance:
//
//	go test -tags=e2e,enterprise ./internal/test/e2e/...
//
// or `make e2e.test.enterprise`, pointed at the enterprise instance via the
// same SONARQUBE_URL/SONARQUBE_TOKEN/SONARQUBE_PROVIDERCONFIG environment
// variables (SONARQUBE_PROVIDERCONFIG defaults to `e2e-enterprise` there).
// `cluster/local/integration_tests.sh` stands up both a community and an
// enterprise SonarQube instance in the same kind cluster and runs both
// suites concurrently.
//
// Since the enterprise suite reruns the same test files as the community
// suite, its managed resources would collide by name if created in the same
// namespace. SONARQUBE_E2E_NAMESPACE selects the namespace managed
// resources are created in (default `default`); integration_tests.sh sets
// it to `e2e-enterprise` for the enterprise run.
//
// The license/portfolio tests additionally require a real Enterprise
// Edition license key in SONARQUBE_LICENSE_KEY; they call t.Skip when it is
// unset rather than fail, since a valid license is a paid artifact not
// every environment has.
//
// Conversely, internal/test/e2e/instance/plugin_test.go carries
// `//go:build e2e && !enterprise` and is excluded from the enterprise suite:
// SonarQube's commercial editions disable the marketplace plugin-install WS
// entirely, so TestPluginInstall can only run against Community Edition.
package e2e
