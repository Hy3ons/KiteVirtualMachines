#!/usr/bin/env bash
set -euo pipefail

KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
API_LOCAL_PORT="${API_LOCAL_PORT:-18080}"
TEST_USERNAME="${TEST_USERNAME:-test-$(date +%s)}"
TEST_EMAIL="${TEST_EMAIL:-test@gmail.com}"
TEST_PASSWORD="${TEST_PASSWORD:-password}"

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "[kite] missing required command: ${name}" >&2
    exit 1
  fi
}

cleanup() {
  if [[ -n "${PORT_FORWARD_PID:-}" ]]; then
    kill "${PORT_FORWARD_PID}" >/dev/null 2>&1 || true
  fi
}

wait_for_http() {
  local url="$1"
  local tries="${2:-30}"
  for _ in $(seq 1 "${tries}"); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

require_command kubectl
require_command curl

trap cleanup EXIT

echo "[kite] checking namespace"
kubectl get namespace "${KITE_NAMESPACE}" >/dev/null

echo "[kite] checking CRDs"
kubectl get crd kiteusers.anacnu.com >/dev/null
kubectl get crd kitevirtualmachines.anacnu.com >/dev/null

echo "[kite] checking deployments"
kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-api --timeout=120s
kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-controller --timeout=120s
kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-frontend --timeout=120s

echo "[kite] port-forwarding API service to localhost:${API_LOCAL_PORT}"
kubectl -n "${KITE_NAMESPACE}" port-forward svc/kite-api "${API_LOCAL_PORT}:8080" >/tmp/kite-api-port-forward.log 2>&1 &
PORT_FORWARD_PID="$!"

if ! wait_for_http "http://127.0.0.1:${API_LOCAL_PORT}/health" 30; then
  echo "[kite] API health check failed" >&2
  cat /tmp/kite-api-port-forward.log >&2 || true
  exit 1
fi

echo "[kite] health"
curl -fsS "http://127.0.0.1:${API_LOCAL_PORT}/health"
echo

echo "[kite] signup smoke test"
signup_response="$(curl -fsS -X POST "http://127.0.0.1:${API_LOCAL_PORT}/api/signup" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${TEST_USERNAME}\",\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\"}")"
echo "${signup_response}"
echo

echo "[kite] login smoke test"
login_response="$(curl -fsS -X POST "http://127.0.0.1:${API_LOCAL_PORT}/api/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${TEST_USERNAME}\",\"password\":\"${TEST_PASSWORD}\"}")"
echo "${login_response}"
echo

echo "[kite] checking created KiteUser"
kubectl get kiteusers.anacnu.com

echo "[kite] smoke test complete"
