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
    echo_error "xpkg not built - run 'make build' first"
fi

echo_step "creating kind controlplane and installing crossplane"
make -C "${projectdir}" controlplane.up

echo_step "loading SonarQube images into kind cluster"
docker pull docker.io/library/sonarqube:community
docker pull docker.io/library/sonarqube:enterprise
"${KIND}" load docker-image docker.io/library/sonarqube:community --name="${KIND_CLUSTER_NAME}"
"${KIND}" load docker-image docker.io/library/sonarqube:enterprise --name="${KIND_CLUSTER_NAME}"

echo_step "deploying ${PACKAGE_NAME} provider package"
# local.xpkg.deploy.provider.<name> patches Crossplane with a dev sidecar,
# extracts each xpkg into the cache under the cache key Crossplane >=2.2
# expects (xpkg.crossplane.internal/dev/<name>@<digest>), kubectl-cps the
# cache into the sidecar, loads the controller image into kind, and
# creates a Provider + DeploymentRuntimeConfig wired to the local image.
# See build/makelib/local.xpkg.mk.
make -C "${projectdir}" "local.xpkg.deploy.provider.${PACKAGE_NAME}"

# Crossplane's ProviderRevision establish reconciler has a known flake with
# packages this size (~20 CRDs): on a fresh install it sometimes establishes
# only a subset of the package's CRDs and never converges on the rest, with
# no error or warning logged (reproduced across crossplane 2.0.0, 2.2.1 and
# 2.3.3, independent of the MRD-conversion feature flag). Deleting and
# recreating the Provider forces a fresh ProviderRevision and has been
# observed to reach full establishment on a later attempt, so poll for full
# establishment and retry a bounded number of times rather than failing
# outright on the first partial establish.
expected_crd_count="$(ls "${projectdir}"/package/crds/*.yaml | wc -l | tr -d ' ')"

wait_for_crds_established() {
    local timeout=$1 elapsed=0 count
    while [ "${elapsed}" -lt "${timeout}" ]; do
        count="$("${KUBECTL}" get crds -o name 2>/dev/null | grep -c 'sonarqube.crossplane.io' || true)"
        if [ "${count}" -ge "${expected_crd_count}" ]; then
            return 0
        fi
        sleep 3
        elapsed=$((elapsed + 3))
    done
    return 1
}

echo_step "waiting for all ${expected_crd_count} provider CRDs to establish"
max_attempts=5
attempt=1
until wait_for_crds_established 90; do
    count="$("${KUBECTL}" get crds -o name 2>/dev/null | grep -c 'sonarqube.crossplane.io' || true)"
    if [ "${attempt}" -ge "${max_attempts}" ]; then
        echo_error "provider only established ${count}/${expected_crd_count} CRDs after ${max_attempts} attempts"
    fi
    echo "only ${count}/${expected_crd_count} CRDs established after attempt ${attempt}/${max_attempts}; recreating Provider to force a fresh ProviderRevision"
    "${KUBECTL}" delete "provider.pkg.crossplane.io/${PACKAGE_NAME}" --wait=true --timeout=60s || true
    make -C "${projectdir}" "local.xpkg.deploy.provider.${PACKAGE_NAME}"
    attempt=$((attempt + 1))
done

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

echo_step "configuring in-cluster SonarQube (community) and ClusterProviderConfig"
KUBECTL="${KUBECTL}" "${projectdir}/cluster/local/sonarqube_setup.sh"

echo_step "configuring in-cluster SonarQube (enterprise) and ClusterProviderConfig"
# Distinct deployment/service/token/secret/providerconfig names so the two
# editions coexist in the same cluster without colliding.
KUBECTL="${KUBECTL}" \
SONARQUBE_MANIFEST="${projectdir}/cluster/local/sonarqube-enterprise.yaml" \
SONARQUBE_DEPLOYMENT="sonarqube-enterprise" \
SONARQUBE_SERVICE="sonarqube-enterprise" \
SONARQUBE_LOCAL_PORT="${E2E_SONARQUBE_ENTERPRISE_PORT:-9001}" \
SONARQUBE_TOKEN_NAME="e2e-enterprise-token" \
SONARQUBE_SECRET_NAME="sonarqube-enterprise-credentials" \
SONARQUBE_PROVIDERCONFIG_NAME="e2e-enterprise" \
"${projectdir}/cluster/local/sonarqube_setup.sh"

echo_step "running e2e Go suites (community + enterprise, in parallel)"
e2e_pf_port="${E2E_SONARQUBE_PORT:-9000}"
e2e_enterprise_pf_port="${E2E_SONARQUBE_ENTERPRISE_PORT:-9001}"

# The community and enterprise suites are the same Go test files (rerun
# against different ClusterProviderConfigs), so their managed resources
# share hardcoded names. Running them concurrently in one namespace would
# collide; put the enterprise suite's resources in their own namespace.
e2e_enterprise_namespace="e2e-enterprise"
"${KUBECTL}" create namespace "${e2e_enterprise_namespace}" --dry-run=client -o yaml | "${KUBECTL}" apply -f -

"${KUBECTL}" port-forward -n default service/sonarqube "${e2e_pf_port}:9000" >/dev/null 2>&1 &
e2e_pf_pid=$!
"${KUBECTL}" port-forward -n default service/sonarqube-enterprise "${e2e_enterprise_pf_port}:9000" >/dev/null 2>&1 &
e2e_enterprise_pf_pid=$!

# Wait for both port-forwards to bind.
for port in "${e2e_pf_port}" "${e2e_enterprise_pf_port}"; do
  for _ in $(seq 1 30); do
    if curl -sf -o /dev/null "http://localhost:${port}/api/system/status"; then
      break
    fi
    sleep 1
  done
done

e2e_token="$("${KUBECTL}" get secret -n default sonarqube-credentials -o jsonpath='{.data.token}' | base64 -d)"
e2e_enterprise_token="$("${KUBECTL}" get secret -n default sonarqube-enterprise-credentials -o jsonpath='{.data.token}' | base64 -d)"

# The two suites are independent (separate ClusterProviderConfigs, separate
# SonarQube instances) so they run concurrently as background jobs rather
# than one after another.
(
  SONARQUBE_URL="http://localhost:${e2e_pf_port}/api" \
  SONARQUBE_TOKEN="${e2e_token}" \
  make -C "${projectdir}" e2e.test
) &
e2e_community_job=$!

(
  SONARQUBE_URL="http://localhost:${e2e_enterprise_pf_port}/api" \
  SONARQUBE_TOKEN="${e2e_enterprise_token}" \
  SONARQUBE_PROVIDERCONFIG="e2e-enterprise" \
  SONARQUBE_E2E_NAMESPACE="${e2e_enterprise_namespace}" \
  SONARQUBE_LICENSE_KEY="${SONARQUBE_LICENSE_KEY:-}" \
  make -C "${projectdir}" e2e.test.enterprise
) &
e2e_enterprise_job=$!

e2e_status=0
wait "${e2e_community_job}" || e2e_status=$?
e2e_enterprise_status=0
wait "${e2e_enterprise_job}" || e2e_enterprise_status=$?

kill "${e2e_pf_pid}" "${e2e_enterprise_pf_pid}" 2>/dev/null || true

if [ "${e2e_status}" -ne 0 ]; then
  echo_error "community e2e Go suite failed (exit ${e2e_status})"
fi
if [ "${e2e_enterprise_status}" -ne 0 ]; then
  echo_error "enterprise e2e Go suite failed (exit ${e2e_enterprise_status})"
fi

echo_success "Integration tests succeeded!"
