#!/usr/bin/env bash
# Idempotent setup for an in-cluster SonarQube instance used by e2e tests.
#
# Steps:
#   1. apply Deployment + Service from sonarqube.yaml
#   2. wait for the Deployment to become Available
#   3. open a port-forward and wait for SonarQube to report status=UP
#   4. set the admin password (tolerates already-changed state)
#   5. accept the plugin risk consent (required before any plugin install)
#   6. (re)generate a deterministic API token
#   7. write the token to a Kubernetes Secret
#   8. apply a ClusterProviderConfig wired to the in-cluster Service
#
# Required environment:
#   KUBECTL (path to kubectl, falls back to "kubectl")
#
# Optional environment overrides have sensible defaults below.
set -euo pipefail

KUBECTL="${KUBECTL:-kubectl}"

SONARQUBE_NAMESPACE="${SONARQUBE_NAMESPACE:-default}"
SONARQUBE_DEPLOYMENT="${SONARQUBE_DEPLOYMENT:-sonarqube}"
SONARQUBE_SERVICE="${SONARQUBE_SERVICE:-sonarqube}"
SONARQUBE_ADMIN_PASSWORD="${SONARQUBE_ADMIN_PASSWORD:-adminPassword123!}"
SONARQUBE_LOCAL_PORT="${SONARQUBE_LOCAL_PORT:-9000}"
SONARQUBE_TOKEN_NAME="${SONARQUBE_TOKEN_NAME:-e2e-token}"
SONARQUBE_SECRET_NAME="${SONARQUBE_SECRET_NAME:-sonarqube-credentials}"
SONARQUBE_SECRET_KEY="${SONARQUBE_SECRET_KEY:-token}"
SONARQUBE_PROVIDERCONFIG_NAME="${SONARQUBE_PROVIDERCONFIG_NAME:-e2e}"
SONARQUBE_READY_ATTEMPTS="${SONARQUBE_READY_ATTEMPTS:-3}"
SONARQUBE_READY_TIMEOUT_SECS="${SONARQUBE_READY_TIMEOUT_SECS:-300}"

SONARQUBE_BASE_URL_IN_CLUSTER="http://${SONARQUBE_SERVICE}.${SONARQUBE_NAMESPACE}.svc.cluster.local:9000/api"
SONARQUBE_LOCAL_URL="http://localhost:${SONARQUBE_LOCAL_PORT}"

manifest_dir="$( cd "$( dirname "${BASH_SOURCE[0]}")" && pwd )"

log() { printf '\n>>> %s\n' "$*"; }

log "Applying SonarQube manifests"
"${KUBECTL}" apply -f "${manifest_dir}/sonarqube.yaml"

log "Waiting for deployment/${SONARQUBE_DEPLOYMENT} to become Available (timeout 10m)"
"${KUBECTL}" wait --for=condition=Available \
    "deployment/${SONARQUBE_DEPLOYMENT}" \
    --namespace="${SONARQUBE_NAMESPACE}" \
    --timeout=10m

log "Establishing port-forward localhost:${SONARQUBE_LOCAL_PORT} -> ${SONARQUBE_SERVICE}:9000"
"${KUBECTL}" port-forward "service/${SONARQUBE_SERVICE}" \
    "${SONARQUBE_LOCAL_PORT}:9000" \
    --namespace="${SONARQUBE_NAMESPACE}" \
    >/dev/null 2>&1 &
port_forward_pid=$!
trap 'kill ${port_forward_pid} 2>/dev/null || true' EXIT

# Give the port-forward a moment to bind.
for _ in $(seq 1 30); do
    if curl -sf -o /dev/null "${SONARQUBE_LOCAL_URL}/api/system/status"; then
        break
    fi
    sleep 1
done

log "Waiting for SonarQube to report status=UP"
ready=false
for attempt in $(seq 1 "${SONARQUBE_READY_ATTEMPTS}"); do
    deadline=$((SECONDS + SONARQUBE_READY_TIMEOUT_SECS))
    while [ "${SECONDS}" -lt "${deadline}" ]; do
        status_body="$(curl -sf "${SONARQUBE_LOCAL_URL}/api/system/status" 2>/dev/null || true)"
        if echo "${status_body}" | grep -q '"status":"UP"'; then
            ready=true
            break 2
        fi
        sleep 5
    done
    log "attempt ${attempt} timed out after ${SONARQUBE_READY_TIMEOUT_SECS}s, retrying"
done
if [ "${ready}" != "true" ]; then
    echo "ERROR: SonarQube did not become ready" >&2
    exit 1
fi

log "Setting admin password (idempotent)"
http_status="$(curl -s -o /dev/null -w '%{http_code}' \
    -u "admin:admin" \
    -X POST "${SONARQUBE_LOCAL_URL}/api/users/change_password" \
    --data-urlencode "login=admin" \
    --data-urlencode "password=${SONARQUBE_ADMIN_PASSWORD}" \
    --data-urlencode "previousPassword=admin")"
case "${http_status}" in
    2*) log "admin password updated" ;;
    401) log "admin password already updated (default creds rejected)" ;;
    *) echo "ERROR: unexpected HTTP ${http_status} from change_password" >&2; exit 1 ;;
esac

auth=(-u "admin:${SONARQUBE_ADMIN_PASSWORD}")

log "Accepting plugin risk consent"
curl -s -o /dev/null "${auth[@]}" -X POST \
    "${SONARQUBE_LOCAL_URL}/api/settings/set" \
    --data-urlencode "key=sonar.plugins.risk.consent" \
    --data-urlencode "value=ACCEPTED"

log "Revoking any pre-existing token named '${SONARQUBE_TOKEN_NAME}'"
curl -s -o /dev/null "${auth[@]}" -X POST \
    "${SONARQUBE_LOCAL_URL}/api/user_tokens/revoke" \
    --data-urlencode "name=${SONARQUBE_TOKEN_NAME}" || true

log "Generating token '${SONARQUBE_TOKEN_NAME}'"
token_response="$(curl -sf "${auth[@]}" -X POST \
    "${SONARQUBE_LOCAL_URL}/api/user_tokens/generate" \
    --data-urlencode "name=${SONARQUBE_TOKEN_NAME}")"
sonar_token="$(printf '%s' "${token_response}" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')"
if [ -z "${sonar_token}" ]; then
    echo "ERROR: failed to extract token from response: ${token_response}" >&2
    exit 1
fi

log "Writing Secret ${SONARQUBE_NAMESPACE}/${SONARQUBE_SECRET_NAME}"
"${KUBECTL}" create secret generic "${SONARQUBE_SECRET_NAME}" \
    --namespace="${SONARQUBE_NAMESPACE}" \
    --from-literal="${SONARQUBE_SECRET_KEY}=${sonar_token}" \
    --dry-run=client -o yaml | "${KUBECTL}" apply -f -

log "Applying ClusterProviderConfig '${SONARQUBE_PROVIDERCONFIG_NAME}'"
"${KUBECTL}" apply -f - <<EOF
apiVersion: sonarqube.crossplane.io/v1alpha1
kind: ClusterProviderConfig
metadata:
  name: ${SONARQUBE_PROVIDERCONFIG_NAME}
spec:
  baseUrl: ${SONARQUBE_BASE_URL_IN_CLUSTER}
  token:
    source: Secret
    secretRef:
      namespace: ${SONARQUBE_NAMESPACE}
      name: ${SONARQUBE_SECRET_NAME}
      key: ${SONARQUBE_SECRET_KEY}
EOF

log "SonarQube ready at ${SONARQUBE_BASE_URL_IN_CLUSTER} (token in ${SONARQUBE_NAMESPACE}/${SONARQUBE_SECRET_NAME})"
