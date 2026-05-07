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
package e2e
