#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/test/test.sh
# Description: 배포된 Kite API에 대해 port-forward 기반 smoke test를 실행한다.
#
# Usage:
#   build/test/test.sh
#
# Environment Variables:
#   KITE_NAMESPACE: default kite
#   API_LOCAL_PORT: default 18080
#   TEST_USERNAME: default test-$(date +%s)
#   TEST_EMAIL: default test@gmail.com
#   TEST_PASSWORD: default password
#
# Side Effects:
#   주로 상태 조회와 대기를 수행하며, test는 임시 port-forward process를 생성한다.
# ==============================================================================

KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
API_LOCAL_PORT="${API_LOCAL_PORT:-18080}"
TEST_USERNAME="${TEST_USERNAME:-test-$(date +%s)}"
TEST_EMAIL="${TEST_EMAIL:-test@gmail.com}"
TEST_PASSWORD="${TEST_PASSWORD:-password}"

# smoke test에 필요한 외부 CLI가 없으면 API 요청 전에 바로 실패시킨다.
require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

# port-forward는 background process라 테스트 종료 시 항상 정리한다.
cleanup() {
  if [[ -n "${PORT_FORWARD_PID:-}" ]]; then
    kill "${PORT_FORWARD_PID}" >/dev/null 2>&1 || true
  fi
}

# port-forward 직후 API가 뜰 때까지 health endpoint를 짧게 재시도한다.
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
# API/controller가 기대하는 Kite CRD가 설치되어 있는지 확인한다.
kubectl get crd kiteusers.hy3ons.github.io >/dev/null
kubectl get crd kitevirtualmachines.hy3ons.github.io >/dev/null

echo "[kite] checking deployments"
kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-api --timeout=120s
kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-controller --timeout=120s
kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-frontend --timeout=120s

echo "[kite] port-forwarding API service to localhost:${API_LOCAL_PORT}"
# 로컬 smoke test가 cluster 내부 Service에 접근할 수 있도록 임시 port-forward를 띄운다.
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
# signup/login은 API 서버가 Kubernetes CRD write 경로까지 정상 동작하는지 확인한다.
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
kubectl get kiteusers.hy3ons.github.io

echo "[kite] smoke test complete"
