#!/usr/bin/env bash
set -Eeuo pipefail

# ==============================================================================
# Script: build/deploy/scripts/update-ghcr.sh
# Description: 기존 VM과 운영 설정을 보존하며 Kite Deployment의 이미지만 갱신한다.
#
# Usage:
#   build/deploy/scripts/update-ghcr.sh
#
# Environment Variables:
#   KITE_NAMESPACE: default kite
#   KITE_UPDATE_REGISTRY: default ghcr.io/hy3ons
#   KITE_UPDATE_TAG: default latest
#   KITE_UPDATE_PULL_POLICY: default Always; source updater sets Never after import.
#   KITE_UPDATE_COMPONENTS: default api,controller,gateway,frontend
#   KITE_UPDATE_APPLY_CRDS: default false; 명시적인 스키마 마이그레이션에만 사용한다.
#   KITE_UPDATE_APPLY_RBAC: default false; 명시적인 권한 마이그레이션에만 사용한다.
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
#   Existing Deployment images and refresh metadata are patched. Explicit CRD/RBAC
#   opt-ins apply only those manifests. VM, storage, Service and settings are preserved.
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
KITE_UPDATE_PULL_POLICY="${KITE_UPDATE_PULL_POLICY:-Always}"
KITE_UPDATE_COMPONENTS="${KITE_UPDATE_COMPONENTS:-api,controller,gateway,frontend}"
KITE_UPDATE_APPLY_CRDS="${KITE_UPDATE_APPLY_CRDS:-false}"
KITE_UPDATE_APPLY_RBAC="${KITE_UPDATE_APPLY_RBAC:-false}"
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
KITE_UPDATE_COMPONENT_LIST=""

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

# save_old_image records rollback image and pull policy for one installed component.
# A ready Pod's immutable digest is preferred so a moving tag can be rolled back.
# main calls this for every selected component before making any changes.
save_old_image() {
  local component="$1"
  local deployment
  local snapshot
  local image
  local policy
  local pod_images
  local ready
  local image_id

  deployment="$(component_deployment "${component}")"
  snapshot="$(kubectl -n "${KITE_NAMESPACE}" get deployment "${deployment}" \
    -o "jsonpath={.spec.template.spec.containers[?(@.name=='${deployment}')].image}{'|'}{.spec.template.spec.containers[?(@.name=='${deployment}')].imagePullPolicy}")"
  IFS='|' read -r image policy <<< "${snapshot}"
  if [[ -z "${image}" || -z "${policy}" ]]; then
    warn "missing container image or pull policy for ${deployment}"
    return 1
  fi
  pod_images="$(kubectl -n "${KITE_NAMESPACE}" get pods -l "app=${deployment}" \
    --field-selector=status.phase=Running \
    -o "jsonpath={range .items[*]}{range .status.containerStatuses[?(@.name=='${deployment}')]}{.ready}{' '}{.imageID}{'\n'}{end}{end}")"
  while read -r ready image_id; do
    image_id="${image_id#docker-pullable://}"
    if [[ "${ready}" == true && "${image_id}" == *@sha256:* ]]; then
      image="${image_id}"
      break
    fi
  done <<< "${pod_images}"

  KITE_UPDATE_OLD_IMAGES+=("${component}|${image}|${policy}")
}

# old_image_for_component returns the saved component, image and policy record.
# rollback_component uses this snapshot without reading changed live templates.
old_image_for_component() {
  local component="$1"
  local entry

  for entry in "${KITE_UPDATE_OLD_IMAGES[@]}"; do
    if [[ "${entry%%|*}" == "${component}" ]]; then
      echo "${entry}"
      return 0
    fi
  done
}

configure_interactive_update_options() {
  kite_prompt_interactive || return 0

  log "interactive update options"
  kite_prompt_value KITE_UPDATE_TAG "${KITE_UPDATE_TAG_WAS_SET}" "어떤 GHCR image tag로 업데이트할까요?" "기본 latest. main publish 직후에는 latest/main/production/sha-<commit> tag를 사용할 수 있습니다."
  kite_prompt_value KITE_UPDATE_COMPONENTS "${KITE_UPDATE_COMPONENTS_WAS_SET}" "업데이트할 컴포넌트를 콤마로 입력하세요." "가능한 값: api,controller,gateway,frontend"
  kite_prompt_configure_bool KITE_UPDATE_APPLY_CRDS "${KITE_UPDATE_APPLY_CRDS_WAS_SET}" "이번 버전에 필요한 CRD 마이그레이션도 적용할까요? 기본은 보존입니다."
  kite_prompt_configure_bool KITE_UPDATE_APPLY_RBAC "${KITE_UPDATE_APPLY_RBAC_WAS_SET}" "이번 버전에 필요한 ServiceAccount/RBAC 마이그레이션도 적용할까요? 기본은 보존입니다."
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

# apply_common_manifests applies only explicitly requested CRD/RBAC migrations.
# Normal image updates never create or reapply settings, namespaces or Services.
apply_common_manifests() {
  if [[ "${KITE_UPDATE_APPLY_CRDS}" == "true" ]]; then
    run_cmd kubectl apply -f "${KITE_MANIFEST_DIR}/crds.yaml"
  fi

  if [[ "${KITE_UPDATE_APPLY_RBAC}" == "true" ]]; then
    run_cmd kubectl apply -f "${KITE_MANIFEST_DIR}/serviceaccount.yaml"
    run_cmd kubectl apply -f "${KITE_MANIFEST_DIR}/rbac.yaml"
  fi

}

# apply_component replaces one existing container image and requests fresh Pods.
# Strategic merge by container name preserves env, resources, volumes and replicas.
# component selects the installed Deployment; no installer manifest is reapplied.
apply_component() {
  local component="$1"
  local deployment
  local image

  deployment="$(component_deployment "${component}")"
  image="$(component_image "${component}")"

  run_cmd kubectl -n "${KITE_NAMESPACE}" patch "deployment/${deployment}" --type=strategic \
    -p "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"kite.hy3ons.github.io/update-id\":\"$(date -u +%Y%m%dT%H%M%SZ)-$$\"}},\"spec\":{\"containers\":[{\"name\":\"${deployment}\",\"image\":\"${image}\",\"imagePullPolicy\":\"${KITE_UPDATE_PULL_POLICY}\"}]}}}}"
}

# prepare_update_images is the pre-mutation preparation hook for source builds.
# The GHCR path resolves images through kubelet; update-local overrides this hook
# to finish every build and import before main patches any Deployment.
prepare_update_images() {
  :
}

wait_rollout() {
  local component="$1"
  local deployment

  [[ "${KITE_UPDATE_DRY_RUN}" == "true" ]] && return 0
  [[ "${KITE_UPDATE_WAIT}" == "true" ]] || return 0

  deployment="$(component_deployment "${component}")"
  kubectl -n "${KITE_NAMESPACE}" rollout status "deployment/${deployment}" --timeout="${KITE_ROLLOUT_TIMEOUT}"
}

# rollback_component restores the saved image/digest and pull policy only.
# It leaves VM resources, settings and other container configuration untouched.
rollback_component() {
  local component="$1"
  local deployment
  local old_image
  local old_policy
  local saved_component

  IFS='|' read -r saved_component old_image old_policy <<< "$(old_image_for_component "${component}")"
  [[ -n "${old_image}" ]] || return 0

  deployment="$(component_deployment "${component}")"
  warn "rolling back ${deployment} to ${old_image}"
  kubectl -n "${KITE_NAMESPACE}" patch "deployment/${deployment}" --type=strategic \
    -p "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"${deployment}\",\"image\":\"${old_image}\",\"imagePullPolicy\":\"${old_policy}\"}]}}}}" || return 1
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

# fail_update preserves a failing command's exit status and restores saved images.
# It is the ERR trap for mutation and verification commands in the update flow.
fail_update() {
  local failure_code="$1"

  trap - ERR
  warn "update failed; restoring previous images"
  rollback_all
  exit "${failure_code}"
}

cleanup_port_forwards() {
  local pid

  for pid in ${KITE_UPDATE_PORT_FORWARD_PIDS[@]+"${KITE_UPDATE_PORT_FORWARD_PIDS[@]}"}; do
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
  log "checking kite-api health"
  start_port_forward kite-api "${KITE_UPDATE_API_LOCAL_PORT}" 8080 || return 1
  curl -fsS "http://127.0.0.1:${KITE_UPDATE_API_LOCAL_PORT}/api/v1/health" >/dev/null || return 1

  log "checking kite-frontend index"
  start_port_forward kite-frontend "${KITE_UPDATE_FRONTEND_LOCAL_PORT}" 80 || return 1
  curl -fsS "http://127.0.0.1:${KITE_UPDATE_FRONTEND_LOCAL_PORT}/" >/dev/null || return 1
}

run_verify() {
  [[ "${KITE_UPDATE_DRY_RUN}" == "true" ]] && return 0
  [[ "${KITE_UPDATE_RUN_VERIFY}" == "true" ]] || return 0

  "${ROOT_DIR}/build/deploy/scripts/verify.sh"
}

main() {
  local component

  require_command kubectl
  configure_interactive_update_options
  # Validate once in the parent shell, before process substitutions or mutations.
  KITE_UPDATE_COMPONENT_LIST="$(normalize_components "${KITE_UPDATE_COMPONENTS}")"
  case "${KITE_UPDATE_PULL_POLICY}" in
    Always|IfNotPresent|Never) ;;
    *) warn "invalid image pull policy"; return 1 ;;
  esac
  print_plan
  trap cleanup_port_forwards EXIT

  if [[ "${KITE_UPDATE_DRY_RUN}" != "true" ]]; then
    kubectl get nodes >/dev/null
    while read -r component; do
      save_old_image "${component}"
    done <<< "${KITE_UPDATE_COMPONENT_LIST}"
  fi

  prepare_update_images
  apply_common_manifests
  trap 'fail_update "$?"' ERR

  while read -r component; do
    [[ -z "${component}" ]] && continue
    apply_component "${component}"
  done <<< "${KITE_UPDATE_COMPONENT_LIST}"

  while read -r component; do
    [[ -z "${component}" ]] && continue
    wait_rollout "${component}"
  done <<< "${KITE_UPDATE_COMPONENT_LIST}"

  health_check

  run_verify
  log "Kite update complete"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
