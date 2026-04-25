#!/usr/bin/env bash
set -e

# setting up colors
BLU='\033[0;34m'
GRN='\033[0;32m'
RED='\033[0;31m'
NOC='\033[0m' # No Color
echo_step(){
    printf "\n${BLU}>>>>>>> %s${NOC}\n" "$1"
}
echo_success(){
    printf "\n${GRN}%s${NOC}\n" "$1"
}
echo_error(){
    printf "\n${RED}%s${NOC}\n" "$1"
    exit 1
}

PACKAGE_NAME="provider-sonarqube"
projectdir="$( cd "$( dirname "${BASH_SOURCE[0]}")"/../.. && pwd )"

# Pull tool paths and project metadata from the build submodule.
eval "$(make --no-print-directory -C "${projectdir}" build.vars)"

# build/makelib/local.xpkg.mk and controlplane.mk default to KIND_CLUSTER_NAME=local-dev.
# Use a build-id-tagged name so concurrent runs (CI parallelism) do not collide.
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-${BUILD_REGISTRY:-build}-inttests}"
export KIND_CLUSTER_NAME

# cleanup on exit unless skipcleanup is set.
if [ "${skipcleanup}" != "true" ]; then
  cleanup() {
    echo_step "cleaning up controlplane"
    "${KIND}" delete cluster --name="${KIND_CLUSTER_NAME}" || true
  }
  trap cleanup EXIT
fi

if [ ! -f "${OUTPUT_DIR}/xpkg/${PLATFORM}/${PACKAGE_NAME}-${VERSION}.xpkg" ]; then
    echo_error "xpkg not built — run 'make build' first"
fi

echo_step "creating kind controlplane and installing crossplane"
make -C "${projectdir}" controlplane.up

echo_step "loading SonarQube image into kind cluster"
docker pull docker.io/library/sonarqube:community
"${KIND}" load docker-image docker.io/library/sonarqube:community --name="${KIND_CLUSTER_NAME}"

echo_step "deploying ${PACKAGE_NAME} provider package"
# local.xpkg.deploy.provider.<name> patches Crossplane with a dev sidecar,
# extracts each xpkg into the cache under the cache key Crossplane >=2.2
# expects (xpkg.crossplane.internal/dev/<name>@<digest>), kubectl-cps the
# cache into the sidecar, loads the controller image into kind, and
# creates a Provider + DeploymentRuntimeConfig wired to the local image.
# See build/makelib/local.xpkg.mk.
make -C "${projectdir}" "local.xpkg.deploy.provider.${PACKAGE_NAME}"

echo_step "granting provider service account permission to watch CRDs"
# The package declares the `safe-start` capability, which makes each
# controller wait for its CRD before starting. Crossplane's auto-generated
# system ClusterRole for the provider does not include CRD list/watch, so
# the safe-start gate would otherwise crash-loop the provider with a
# Forbidden error from controller-runtime's CRD informer. Bind the
# auto-created provider ServiceAccount to a small extra ClusterRole that
# closes that gap.
echo "waiting for provider service account to be created"
provider_sa=""
for _ in $(seq 1 60); do
    provider_sa="$("${KUBECTL}" get sa -n crossplane-system -o name 2>/dev/null | grep "${PACKAGE_NAME}" | head -1 | sed 's|^serviceaccount/||')"
    if [ -n "${provider_sa}" ]; then
        break
    fi
    sleep 2
done
if [ -z "${provider_sa}" ]; then
    echo_error "provider service account did not appear within 120s"
fi
echo "binding ServiceAccount ${provider_sa} to CRD watch role"
"${KUBECTL}" apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ${PACKAGE_NAME}-crd-watcher
rules:
  - apiGroups: ["apiextensions.k8s.io"]
    resources: ["customresourcedefinitions"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ${PACKAGE_NAME}-crd-watcher
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ${PACKAGE_NAME}-crd-watcher
subjects:
  - kind: ServiceAccount
    name: ${provider_sa}
    namespace: crossplane-system
EOF

echo_step "waiting for provider to become healthy"
"${KUBECTL}" wait "provider.pkg.crossplane.io/${PACKAGE_NAME}" --for=condition=healthy --timeout=300s

echo_step "configuring in-cluster SonarQube and ClusterProviderConfig"
KUBECTL="${KUBECTL}" "${projectdir}/cluster/local/sonarqube_setup.sh"

echo_step "running e2e Go suite"
e2e_pf_port="${E2E_SONARQUBE_PORT:-9000}"
"${KUBECTL}" port-forward -n default service/sonarqube "${e2e_pf_port}:9000" >/dev/null 2>&1 &
e2e_pf_pid=$!
# Wait for the port-forward to bind.
for _ in $(seq 1 30); do
  if curl -sf -o /dev/null "http://localhost:${e2e_pf_port}/api/system/status"; then
    break
  fi
  sleep 1
done
e2e_token="$("${KUBECTL}" get secret -n default sonarqube-credentials -o jsonpath='{.data.token}' | base64 -d)"
e2e_status=0
SONARQUBE_URL="http://localhost:${e2e_pf_port}/api" \
SONARQUBE_TOKEN="${e2e_token}" \
make -C "${projectdir}" e2e.test || e2e_status=$?
kill "${e2e_pf_pid}" 2>/dev/null || true
if [ "${e2e_status}" -ne 0 ]; then
  echo_error "e2e Go suite failed (exit ${e2e_status})"
fi

echo_success "Integration tests succeeded!"
