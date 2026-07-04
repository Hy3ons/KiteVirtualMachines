#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/update-ghcr.sh
# Description: 기존 Kite 설치를 보존하면서 GHCR 이미지와 런타임 manifest를 재적용한다.
#
# Usage:
#   build/deploy/scripts/update-ghcr.sh
#
# Environment Variables:
#   KITE_NAMESPACE: default kite
#   KITE_UPDATE_REGISTRY: default ghcr.io/hy3ons
#   KITE_UPDATE_TAG: default latest
#   KITE_UPDATE_COMPONENTS: default api,controller,gateway,frontend
#   KITE_UPDATE_APPLY_CRDS: default true
#   KITE_UPDATE_APPLY_RBAC: default true
#   KITE_UPDATE_WAIT: default true
#   KITE_UPDATE_HEALTH_CHECK: default true
#   KITE_UPDATE_RUN_VERIFY: default false
#   KITE_UPDATE_ROLLBACK_ON_FAIL: default true
#   KITE_UPDATE_DRY_RUN: default false
#   KITE_UPDATE_API_LOCAL_PORT: default 18080
#   KITE_UPDATE_FRONTEND_LOCAL_PORT: default 18081
#   KITE_ROLLOUT_TIMEOUT: default 15m
#   KITE_ASSUME_DEFAULTS: default false
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default empty
#
# Side Effects:
#   Kubernetes manifests are applied, Kite Deployment images are updated, and
#   selected Deployments may roll back to their previous image on rollout failure.
# ==============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
KITE_UPDATE_TAG_WAS_SET="${KITE_UPDATE_TAG+x}"
KITE_UPDATE_COMPONENTS_WAS_SET="${KITE_UPDATE_COMPONENTS+x}"
KITE_UPDATE_APPLY_CRDS_WAS_SET="${KITE_UPDATE_APPLY_CRDS+x}"
KITE_UPDATE_APPLY_RBAC_WAS_SET="${KITE_UPDATE_APPLY_RBAC+x}"
KITE_UPDATE_WAIT_WAS_SET="${KITE_UPDATE_WAIT+x}"
KITE_UPDATE_HEALTH_CHECK_WAS_SET="${KITE_UPDATE_HEALTH_CHECK+x}"
KITE_UPDATE_RUN_VERIFY_WAS_SET="${KITE_UPDATE_RUN_VERIFY+x}"
KITE_UPDATE_ROLLBACK_ON_FAIL_WAS_SET="${KITE_UPDATE_ROLLBACK_ON_FAIL+x}"
KITE_UPDATE_DRY_RUN_WAS_SET="${KITE_UPDATE_DRY_RUN+x}"
KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
KITE_UPDATE_REGISTRY="${KITE_UPDATE_REGISTRY:-ghcr.io/hy3ons}"
KITE_UPDATE_TAG="${KITE_UPDATE_TAG:-latest}"
KITE_UPDATE_COMPONENTS="${KITE_UPDATE_COMPONENTS:-api,controller,gateway,frontend}"
KITE_UPDATE_APPLY_CRDS="${KITE_UPDATE_APPLY_CRDS:-true}"
KITE_UPDATE_APPLY_RBAC="${KITE_UPDATE_APPLY_RBAC:-true}"
KITE_UPDATE_WAIT="${KITE_UPDATE_WAIT:-true}"
KITE_UPDATE_HEALTH_CHECK="${KITE_UPDATE_HEALTH_CHECK:-true}"
KITE_UPDATE_RUN_VERIFY="${KITE_UPDATE_RUN_VERIFY:-false}"
KITE_UPDATE_ROLLBACK_ON_FAIL="${KITE_UPDATE_ROLLBACK_ON_FAIL:-true}"
KITE_UPDATE_DRY_RUN="${KITE_UPDATE_DRY_RUN:-false}"
KITE_UPDATE_API_LOCAL_PORT="${KITE_UPDATE_API_LOCAL_PORT:-18080}"
KITE_UPDATE_FRONTEND_LOCAL_PORT="${KITE_UPDATE_FRONTEND_LOCAL_PORT:-18081}"
KITE_ROLLOUT_TIMEOUT="${KITE_ROLLOUT_TIMEOUT:-15m}"
KITE_MANIFEST_DIR="${ROOT_DIR}/build/kite"
KITE_UPDATE_PORT_FORWARD_PIDS=()
KITE_UPDATE_OLD_IMAGES=()

# shellcheck source=build/lib/prompt.sh
source "${ROOT_DIR}/build/lib/prompt.sh"

log_color_enabled() {
  [[ "${KITE_LOG_COLOR:-auto}" != "false" && -z "${NO_COLOR:-}" && -t 1 ]]
}

log_timestamp() {
  date +"%Y-%m-%dT%H:%M:%S%z"
}

log() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[0;32m[kite-ghcr-update] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-ghcr-update] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-ghcr-update] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-ghcr-update] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

run_cmd() {
  if [[ "${KITE_UPDATE_DRY_RUN}" == "true" ]]; then
    printf '[kite-ghcr-update] dry-run:'
    printf ' %q' "$@"
    printf '\n'
    return 0
  fi

  "$@"
}

component_deployment() {
  case "$1" in
    api) echo "kite-api" ;;
    controller) echo "kite-controller" ;;
    gateway) echo "kite-gateway" ;;
    frontend) echo "kite-frontend" ;;
    *)
      echo "[kite-ghcr-update] unknown component=$1" >&2
      exit 1
      ;;
  esac
}

component_image() {
  echo "${KITE_UPDATE_REGISTRY}/$(component_deployment "$1"):${KITE_UPDATE_TAG}"
}

component_manifest() {
  echo "${KITE_MANIFEST_DIR}/$1.yaml"
}

normalize_components() {
  local raw="$1"
  local component
  local normalized=()

  raw="${raw// /}"
  IFS=',' read -r -a normalized <<< "${raw}"
  if [[ "${#normalized[@]}" -eq 0 ]]; then
    echo "[kite-ghcr-update] KITE_UPDATE_COMPONENTS cannot be empty" >&2
    exit 1
  fi

  for component in "${normalized[@]}"; do
    component_deployment "${component}" >/dev/null
  done

  printf '%s\n' "${normalized[@]}"
}

gateway_fallback_address() {
  kubectl -n "${KITE_NAMESPACE}" get deployment kite-gateway \
    -o jsonpath='{range .spec.template.spec.containers[?(@.name=="kite-gateway")].env[?(@.name=="KITE_GATEWAY_HOST_SSHD_ADDRESS")]}{.value}{end}' \
    2>/dev/null || true
}

save_old_image() {
  local component="$1"
  local deployment
  local container
  local image

  deployment="$(component_deployment "${component}")"
  container="${deployment}"
  image="$(kubectl -n "${KITE_NAMESPACE}" get deployment "${deployment}" \
    -o "jsonpath={.spec.template.spec.containers[?(@.name=='${container}')].image}" 2>/dev/null || true)"

  KITE_UPDATE_OLD_IMAGES+=("${component}|${image}")
}

old_image_for_component() {
  local component="$1"
  local entry

  for entry in "${KITE_UPDATE_OLD_IMAGES[@]}"; do
    if [[ "${entry%%|*}" == "${component}" ]]; then
      echo "${entry#*|}"
      return 0
    fi
  done
}

configure_interactive_update_options() {
  kite_prompt_interactive || return 0

  log "interactive update options"
  kite_prompt_value KITE_UPDATE_TAG "${KITE_UPDATE_TAG_WAS_SET}" "어떤 GHCR image tag로 업데이트할까요?" "기본 latest. main publish 직후에는 latest/main/production/sha-<commit> tag를 사용할 수 있습니다."
  kite_prompt_value KITE_UPDATE_COMPONENTS "${KITE_UPDATE_COMPONENTS_WAS_SET}" "업데이트할 컴포넌트를 콤마로 입력하세요." "가능한 값: api,controller,gateway,frontend"
  kite_prompt_configure_bool KITE_UPDATE_APPLY_CRDS "${KITE_UPDATE_APPLY_CRDS_WAS_SET}" "CRD manifest도 kubectl apply로 갱신할까요?"
  kite_prompt_configure_bool KITE_UPDATE_APPLY_RBAC "${KITE_UPDATE_APPLY_RBAC_WAS_SET}" "ServiceAccount/RBAC manifest도 갱신할까요?"
  kite_prompt_configure_bool KITE_UPDATE_WAIT "${KITE_UPDATE_WAIT_WAS_SET}" "Deployment rollout 완료까지 기다릴까요?"
  kite_prompt_configure_bool KITE_UPDATE_HEALTH_CHECK "${KITE_UPDATE_HEALTH_CHECK_WAS_SET}" "업데이트 후 API/frontend smoke check를 실행할까요?"
  kite_prompt_configure_bool KITE_UPDATE_RUN_VERIFY "${KITE_UPDATE_RUN_VERIFY_WAS_SET}" "기존 deploy verify 스크립트까지 실행할까요?"
  kite_prompt_configure_bool KITE_UPDATE_ROLLBACK_ON_FAIL "${KITE_UPDATE_ROLLBACK_ON_FAIL_WAS_SET}" "rollout 실패 시 이전 image로 되돌릴까요?"
  kite_prompt_configure_bool KITE_UPDATE_DRY_RUN "${KITE_UPDATE_DRY_RUN_WAS_SET}" "실제 변경 없이 계획만 출력할까요?"
}

print_plan() {
  local component

  echo
  echo "[kite-ghcr-update] update plan"
  printf '  %-18s %s\n' "namespace:" "${KITE_NAMESPACE}"
  printf '  %-18s %s\n' "registry:" "${KITE_UPDATE_REGISTRY}"
  printf '  %-18s %s\n' "tag:" "${KITE_UPDATE_TAG}"
  printf '  %-18s %s\n' "components:" "${KITE_UPDATE_COMPONENTS}"
  printf '  %-18s %s\n' "apply CRDs:" "${KITE_UPDATE_APPLY_CRDS}"
  printf '  %-18s %s\n' "apply RBAC:" "${KITE_UPDATE_APPLY_RBAC}"
  printf '  %-18s %s\n' "health check:" "${KITE_UPDATE_HEALTH_CHECK}"
  printf '  %-18s %s\n' "verify:" "${KITE_UPDATE_RUN_VERIFY}"
  printf '  %-18s %s\n' "rollback:" "${KITE_UPDATE_ROLLBACK_ON_FAIL}"
  printf '  %-18s %s\n' "dry run:" "${KITE_UPDATE_DRY_RUN}"
  while read -r component; do
    [[ -z "${component}" ]] && continue
    printf '  %-18s %s\n' "${component} image:" "$(component_image "${component}")"
  done < <(normalize_components "${KITE_UPDATE_COMPONENTS}")
  echo
}

apply_common_manifests() {
  if [[ "${KITE_UPDATE_APPLY_CRDS}" == "true" ]]; then
    run_cmd kubectl apply -f "${KITE_MANIFEST_DIR}/crds.yaml"
  fi

  run_cmd kubectl apply -f "${KITE_MANIFEST_DIR}/namespace.yaml"

  if [[ "${KITE_UPDATE_APPLY_RBAC}" == "true" ]]; then
    run_cmd kubectl apply -f "${KITE_MANIFEST_DIR}/serviceaccount.yaml"
    run_cmd kubectl apply -f "${KITE_MANIFEST_DIR}/rbac.yaml"
  fi

  if [[ "${KITE_UPDATE_DRY_RUN}" != "true" ]] && ! kubectl -n "${KITE_NAMESPACE}" get configmap kite-runtime-config >/dev/null 2>&1; then
    run_cmd kubectl apply -f "${KITE_MANIFEST_DIR}/config.yaml"
  elif [[ "${KITE_UPDATE_DRY_RUN}" == "true" ]]; then
    run_cmd kubectl apply -f "${KITE_MANIFEST_DIR}/config.yaml"
  else
    log "preserving existing kite-runtime-config"
  fi
}

apply_component() {
  local component="$1"
  local deployment
  local image

  deployment="$(component_deployment "${component}")"
  image="$(component_image "${component}")"

  save_old_image "${component}"
  run_cmd kubectl apply -f "$(component_manifest "${component}")"
  run_cmd kubectl -n "${KITE_NAMESPACE}" set image "deployment/${deployment}" "${deployment}=${image}"
  run_cmd kubectl -n "${KITE_NAMESPACE}" patch "deployment/${deployment}" --type=strategic \
    -p "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"${deployment}\",\"imagePullPolicy\":\"IfNotPresent\"}]}}}}"
}

component_selected() {
  local needle="$1"
  local component

  while read -r component; do
    [[ "${component}" == "${needle}" ]] && return 0
  done < <(normalize_components "${KITE_UPDATE_COMPONENTS}")

  return 1
}

apply_gateway_preserved_settings() {
  local fallback_address="$1"

  if ! component_selected gateway; then
    return 0
  fi

  run_cmd "${ROOT_DIR}/build/deploy/scripts/ensure-gateway-host-key-secret.sh"
  if [[ -n "${fallback_address}" ]]; then
    run_cmd kubectl -n "${KITE_NAMESPACE}" set env deployment/kite-gateway "KITE_GATEWAY_HOST_SSHD_ADDRESS=${fallback_address}"
  fi
}

wait_rollout() {
  local component="$1"
  local deployment

  [[ "${KITE_UPDATE_DRY_RUN}" == "true" ]] && return 0
  [[ "${KITE_UPDATE_WAIT}" == "true" ]] || return 0

  deployment="$(component_deployment "${component}")"
  kubectl -n "${KITE_NAMESPACE}" rollout status "deployment/${deployment}" --timeout="${KITE_ROLLOUT_TIMEOUT}"
}

rollback_component() {
  local component="$1"
  local deployment
  local old_image

  old_image="$(old_image_for_component "${component}")"
  [[ -n "${old_image}" ]] || return 0

  deployment="$(component_deployment "${component}")"
  warn "rolling back ${deployment} to ${old_image}"
  kubectl -n "${KITE_NAMESPACE}" set image "deployment/${deployment}" "${deployment}=${old_image}"
  kubectl -n "${KITE_NAMESPACE}" rollout status "deployment/${deployment}" --timeout="${KITE_ROLLOUT_TIMEOUT}"
}

rollback_all() {
  local component

  [[ "${KITE_UPDATE_ROLLBACK_ON_FAIL}" == "true" ]] || return 0

  while read -r component; do
    [[ -z "${component}" ]] && continue
    rollback_component "${component}" || true
  done < <(normalize_components "${KITE_UPDATE_COMPONENTS}")
}

cleanup_port_forwards() {
  local pid

  for pid in "${KITE_UPDATE_PORT_FORWARD_PIDS[@]}"; do
    kill "${pid}" >/dev/null 2>&1 || true
  done
}

start_port_forward() {
  local service="$1"
  local local_port="$2"
  local remote_port="$3"
  local log_file
  local pid

  log_file="$(mktemp "${TMPDIR:-/tmp}/kite-ghcr-update-port-forward.XXXXXX")"
  kubectl -n "${KITE_NAMESPACE}" port-forward "svc/${service}" "${local_port}:${remote_port}" >"${log_file}" 2>&1 &
  pid="$!"
  KITE_UPDATE_PORT_FORWARD_PIDS+=("${pid}")
  sleep 2
  if ! kill -0 "${pid}" >/dev/null 2>&1; then
    cat "${log_file}" >&2 || true
    return 1
  fi
}

health_check() {
  [[ "${KITE_UPDATE_DRY_RUN}" == "true" ]] && return 0
  [[ "${KITE_UPDATE_HEALTH_CHECK}" == "true" ]] || return 0

  require_command curl
  trap cleanup_port_forwards EXIT

  log "checking kite-api health"
  start_port_forward kite-api "${KITE_UPDATE_API_LOCAL_PORT}" 8080
  curl -fsS "http://127.0.0.1:${KITE_UPDATE_API_LOCAL_PORT}/api/v1/health" >/dev/null

  log "checking kite-frontend index"
  start_port_forward kite-frontend "${KITE_UPDATE_FRONTEND_LOCAL_PORT}" 80
  curl -fsS "http://127.0.0.1:${KITE_UPDATE_FRONTEND_LOCAL_PORT}/" >/dev/null
}

run_verify() {
  [[ "${KITE_UPDATE_DRY_RUN}" == "true" ]] && return 0
  [[ "${KITE_UPDATE_RUN_VERIFY}" == "true" ]] || return 0

  "${ROOT_DIR}/build/deploy/scripts/verify.sh"
}

main() {
  local component
  local fallback_address

  require_command kubectl
  configure_interactive_update_options
  print_plan

  if [[ "${KITE_UPDATE_DRY_RUN}" != "true" ]]; then
    kubectl get nodes >/dev/null
  fi

  fallback_address="$(gateway_fallback_address)"
  apply_common_manifests

  while read -r component; do
    [[ -z "${component}" ]] && continue
    apply_component "${component}"
  done < <(normalize_components "${KITE_UPDATE_COMPONENTS}")

  apply_gateway_preserved_settings "${fallback_address}"

  if ! while read -r component; do
    [[ -z "${component}" ]] && continue
    wait_rollout "${component}"
  done < <(normalize_components "${KITE_UPDATE_COMPONENTS}"); then
    rollback_all
    exit 1
  fi

  if ! health_check; then
    rollback_all
    exit 1
  fi

  run_verify
  log "GHCR update complete"
}

main "$@"
