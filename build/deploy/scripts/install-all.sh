#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/install-all.sh
# Description: pull 기반 설치 전체 흐름을 실행한다. host sshd handoff, dependency 설치, Kite manifest 적용을 포함한다.
#
# Usage:
#   build/deploy/scripts/install-all.sh
#
# Environment Variables:
#   KITE_NAMESPACE: default kite
#   INSTALL_LONGHORN: default true
#   KITE_INSTALL_LONGHORN_HOST_PACKAGES: default true
#   CONFIGURE_LONGHORN: default true
#   APPLY_STORAGECLASS: default true
#   INSTALL_KUBEVIRT: default true
#   INSTALL_CDI: default true
#   APPLY_GOLDEN_IMAGE: default true
#   MANAGE_HOST_SSHD: default true
#   KITE_HOST_SSHD_PORT: default 2222
#   KITE_LONGHORN_USE_DEDICATED_DISK: default false
#   KITE_GATEWAY_HOST_KEY_REFRESH: default false
#   KITE_ROLLOUT_TIMEOUT: default 15m
#   RUN_VERIFY: default true
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   Kubernetes 리소스 적용, 컨테이너 이미지 빌드/주입, rollout 대기를 수행할 수 있다.
# ==============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
INSTALL_LONGHORN_WAS_SET="${INSTALL_LONGHORN+x}"
CONFIGURE_LONGHORN_WAS_SET="${CONFIGURE_LONGHORN+x}"
APPLY_STORAGECLASS_WAS_SET="${APPLY_STORAGECLASS+x}"
INSTALL_KUBEVIRT_WAS_SET="${INSTALL_KUBEVIRT+x}"
INSTALL_CDI_WAS_SET="${INSTALL_CDI+x}"
APPLY_GOLDEN_IMAGE_WAS_SET="${APPLY_GOLDEN_IMAGE+x}"
MANAGE_HOST_SSHD_WAS_SET="${MANAGE_HOST_SSHD+x}"
KITE_HOST_SSHD_PORT_WAS_SET="${KITE_HOST_SSHD_PORT+x}"
KITE_LONGHORN_USE_DEDICATED_DISK_WAS_SET="${KITE_LONGHORN_USE_DEDICATED_DISK+x}"
KITE_GATEWAY_HOST_KEY_REFRESH_WAS_SET="${KITE_GATEWAY_HOST_KEY_REFRESH+x}"
RUN_VERIFY_WAS_SET="${RUN_VERIFY+x}"
KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
INSTALL_LONGHORN="${INSTALL_LONGHORN:-true}"
CONFIGURE_LONGHORN="${CONFIGURE_LONGHORN:-true}"
APPLY_STORAGECLASS="${APPLY_STORAGECLASS:-true}"
INSTALL_KUBEVIRT="${INSTALL_KUBEVIRT:-true}"
INSTALL_CDI="${INSTALL_CDI:-true}"
APPLY_GOLDEN_IMAGE="${APPLY_GOLDEN_IMAGE:-true}"
MANAGE_HOST_SSHD="${MANAGE_HOST_SSHD:-true}"
KITE_HOST_SSHD_PORT="${KITE_HOST_SSHD_PORT:-2222}"
KITE_HOST_SSHD_STATE="${KITE_HOST_SSHD_STATE:-/etc/kite/host-sshd/state.env}"
KITE_LONGHORN_USE_DEDICATED_DISK="${KITE_LONGHORN_USE_DEDICATED_DISK:-false}"
KITE_GATEWAY_HOST_KEY_REFRESH="${KITE_GATEWAY_HOST_KEY_REFRESH:-false}"
KITE_ROLLOUT_TIMEOUT="${KITE_ROLLOUT_TIMEOUT:-15m}"
RUN_VERIFY="${RUN_VERIFY:-true}"

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
    printf "\033[0;32m[kite-deploy] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-deploy] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-deploy] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-deploy] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}


# 설치에 필요한 외부 CLI를 먼저 확인해, 클러스터를 일부만 변경한 뒤 실패하는 일을 줄인다.
require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

# Kite-managed host sshd state is written by manage-host-sshd.sh and tells the gateway where fallback SSH lives.
managed_host_sshd_port() {
  local state
  local port

  if [[ -f "${KITE_HOST_SSHD_STATE}" ]]; then
    state="$(cat "${KITE_HOST_SSHD_STATE}")"
  elif command -v sudo >/dev/null 2>&1 && sudo test -f "${KITE_HOST_SSHD_STATE}" 2>/dev/null; then
    state="$(sudo cat "${KITE_HOST_SSHD_STATE}")"
  else
    port="$("${ROOT_DIR}/build/deploy/scripts/manage-host-sshd.sh" print-port 2>/dev/null || true)"
    echo "${port:-${KITE_HOST_SSHD_PORT}}"
    return 0
  fi

  port="$(printf '%s\n' "${state}" | awk -F= '$1 == "PORT" { print $2; exit }')"
  echo "${port:-${KITE_HOST_SSHD_PORT}}"
}

# The gateway pod needs the same host sshd port that the host handoff selected.
patch_gateway_host_sshd_address() {
  local port

  if [[ "${MANAGE_HOST_SSHD}" != "true" ]]; then
    return 0
  fi

  port="$(managed_host_sshd_port)"
  log "configuring kite-gateway host fallback address with host sshd port ${port}"
  kubectl -n "${KITE_NAMESPACE}" set env deployment/kite-gateway "KITE_GATEWAY_HOST_SSHD_ADDRESS=\$(KITE_NODE_IP):${port}"
}

configure_gateway_service_exposure() {
  local node_port

  if [[ "${MANAGE_HOST_SSHD}" == "true" ]]; then
    log "exposing kite-gateway Service on external SSH port 22"
    kubectl -n "${KITE_NAMESPACE}" patch service kite-gateway --type=merge -p '{
      "spec": {
        "type": "LoadBalancer",
        "ports": [
          {
            "name": "ssh",
            "port": 22,
            "targetPort": "ssh",
            "protocol": "TCP"
          }
        ]
      }
    }'
    return
  fi

  log "keeping kite-gateway Service internal because MANAGE_HOST_SSHD=false"
  node_port="$(kubectl -n "${KITE_NAMESPACE}" get service kite-gateway -o jsonpath='{.spec.ports[0].nodePort}' 2>/dev/null || true)"
  if [[ -n "${node_port}" ]]; then
    kubectl -n "${KITE_NAMESPACE}" patch service kite-gateway --type=json -p='[
      {"op":"replace","path":"/spec/type","value":"ClusterIP"},
      {"op":"remove","path":"/spec/ports/0/nodePort"}
    ]'
  else
    kubectl -n "${KITE_NAMESPACE}" patch service kite-gateway --type=merge -p '{"spec":{"type":"ClusterIP"}}'
  fi
}

ensure_longhorn_available_for_configuration() {
  if kubectl get namespace longhorn-system >/dev/null 2>&1; then
    return 0
  fi

  warn "CONFIGURE_LONGHORN=true requires Longhorn, but namespace/longhorn-system does not exist"
  warn "run with INSTALL_LONGHORN=true, or set CONFIGURE_LONGHORN=false if Longhorn is managed elsewhere"
  exit 1
}

configure_interactive_install_options() {
  kite_prompt_interactive || return 0

  log "interactive install options"
  kite_prompt_configure_bool MANAGE_HOST_SSHD "${MANAGE_HOST_SSHD_WAS_SET}" "Kite gateway가 22번을 쓸 수 있게 host sshd handoff를 확인할까요?"
  if [[ "${MANAGE_HOST_SSHD}" == "true" ]]; then
    kite_prompt_value KITE_HOST_SSHD_PORT "${KITE_HOST_SSHD_PORT_WAS_SET}" "KITE_HOST_SSHD_PORT 값을 정합니다." "host sshd가 22번에서 이동할 포트입니다. 실제 적용 전 점유 확인을 거칩니다."
  fi
  kite_prompt_configure_bool INSTALL_LONGHORN "${INSTALL_LONGHORN_WAS_SET}" "Longhorn 기본 manifest를 설치할까요?"
  kite_prompt_configure_bool CONFIGURE_LONGHORN "${CONFIGURE_LONGHORN_WAS_SET}" "Longhorn에 Kite 전용 disk/tag 설정을 적용할까요?"
  if [[ "${CONFIGURE_LONGHORN}" == "true" ]]; then
    kite_prompt_configure_bool KITE_LONGHORN_USE_DEDICATED_DISK "${KITE_LONGHORN_USE_DEDICATED_DISK_WAS_SET}" "Longhorn에 Kite 전용 host path disk entry를 만들까요? 아니오면 기존 Ready disk에 kite tag만 붙입니다."
  fi
  kite_prompt_configure_bool APPLY_STORAGECLASS "${APPLY_STORAGECLASS_WAS_SET}" "Kite 전용 Longhorn StorageClass를 적용할까요?"
  kite_prompt_configure_bool INSTALL_KUBEVIRT "${INSTALL_KUBEVIRT_WAS_SET}" "KubeVirt를 설치할까요?"
  kite_prompt_configure_bool INSTALL_CDI "${INSTALL_CDI_WAS_SET}" "CDI를 설치할까요?"
  kite_prompt_configure_bool APPLY_GOLDEN_IMAGE "${APPLY_GOLDEN_IMAGE_WAS_SET}" "Ubuntu golden image DataVolume을 적용할까요?"
  kite_prompt_configure_bool KITE_GATEWAY_HOST_KEY_REFRESH "${KITE_GATEWAY_HOST_KEY_REFRESH_WAS_SET}" "기존 kite-gateway host key Secret이 있으면 새 key로 갱신할까요?"
  kite_prompt_configure_bool RUN_VERIFY "${RUN_VERIFY_WAS_SET}" "설치 후 verify 스크립트를 실행할까요?"

  log "install choices: MANAGE_HOST_SSHD=${MANAGE_HOST_SSHD}, KITE_HOST_SSHD_PORT=${KITE_HOST_SSHD_PORT}, INSTALL_LONGHORN=${INSTALL_LONGHORN}, CONFIGURE_LONGHORN=${CONFIGURE_LONGHORN}, KITE_LONGHORN_USE_DEDICATED_DISK=${KITE_LONGHORN_USE_DEDICATED_DISK}, APPLY_STORAGECLASS=${APPLY_STORAGECLASS}, INSTALL_KUBEVIRT=${INSTALL_KUBEVIRT}, INSTALL_CDI=${INSTALL_CDI}, APPLY_GOLDEN_IMAGE=${APPLY_GOLDEN_IMAGE}, KITE_GATEWAY_HOST_KEY_REFRESH=${KITE_GATEWAY_HOST_KEY_REFRESH}, RUN_VERIFY=${RUN_VERIFY}"
}

export_install_options() {
  export KITE_HOST_SSHD_PORT
  export KITE_LONGHORN_USE_DEDICATED_DISK
  export KITE_GATEWAY_HOST_KEY_REFRESH
  if [[ "${MANAGE_HOST_SSHD}" == "true" ]]; then
    if [[ -z "${KITE_MANAGE_HOST_SSHD:-}" ]]; then
      export KITE_MANAGE_HOST_SSHD=true
    fi
  else
    export KITE_MANAGE_HOST_SSHD=false
  fi
}

# pull 기반 설치의 전체 순서다. host sshd handoff, Longhorn/KubeVirt/CDI 준비,
# Kite 매니페스트 적용, golden image 적용, 기본 검증까지 한 번에 진행한다.
main() {
  require_command kubectl

  kubectl get nodes >/dev/null
  configure_interactive_install_options
  export_install_options
  # gateway가 외부 22번을 쓰려면 host sshd를 다른 포트로 옮겨야 할 수 있다.
  # 원격 서버에서는 접속 경로가 바뀌므로 manage-host-sshd.sh가 별도로 확인/백업한다.
  if [[ "${MANAGE_HOST_SSHD}" == "true" ]]; then
    "${ROOT_DIR}/build/deploy/scripts/manage-host-sshd.sh" ensure
  else
    log "skipping host sshd handoff because MANAGE_HOST_SSHD=${MANAGE_HOST_SSHD}"
  fi

  if [[ "${INSTALL_LONGHORN}" == "true" ]]; then
    "${ROOT_DIR}/build/deploy/scripts/install-longhorn.sh"
  else
    log "skipping Longhorn install because INSTALL_LONGHORN=${INSTALL_LONGHORN}"
  fi
  if [[ "${CONFIGURE_LONGHORN}" == "true" ]]; then
    ensure_longhorn_available_for_configuration
    "${ROOT_DIR}/build/deploy/scripts/wait-longhorn.sh"
    "${ROOT_DIR}/build/deploy/scripts/configure-longhorn-kite-disk.sh"
  else
    log "skipping Longhorn disk/tag configuration because CONFIGURE_LONGHORN=${CONFIGURE_LONGHORN}"
  fi
  if [[ "${APPLY_STORAGECLASS}" == "true" ]]; then
    # VM DataVolume이 Longhorn을 쓰도록 Kite 전용 StorageClass를 적용한다.
    kubectl apply -f "${ROOT_DIR}/build/kite-storage/longhorn/storageclass.yaml"
  else
    log "skipping Kite StorageClass apply because APPLY_STORAGECLASS=${APPLY_STORAGECLASS}"
  fi

  if [[ "${INSTALL_KUBEVIRT}" == "true" ]]; then
    "${ROOT_DIR}/build/deploy/scripts/install-kubevirt.sh"
  fi
  "${ROOT_DIR}/build/deploy/scripts/wait-kubevirt.sh"

  if [[ "${INSTALL_CDI}" == "true" ]]; then
    "${ROOT_DIR}/build/deploy/scripts/install-cdi.sh"
  fi
  "${ROOT_DIR}/build/deploy/scripts/wait-cdi.sh"

  log "applying Kite manifests"
  "${ROOT_DIR}/build/deploy/scripts/ensure-gateway-host-key-secret.sh"
  if kubectl -n "${KITE_NAMESPACE}" get service kite-gateway >/dev/null 2>&1; then
    configure_gateway_service_exposure
  fi
  # build/kite kustomization에는 API/controller/gateway/frontend 런타임 리소스가 모여 있다.
  kubectl apply -k "${ROOT_DIR}/build/kite"
  configure_gateway_service_exposure
  patch_gateway_host_sshd_address

  log "waiting for Kite workloads"
  kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-api --timeout="${KITE_ROLLOUT_TIMEOUT}"
  kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-controller --timeout="${KITE_ROLLOUT_TIMEOUT}"
  kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-gateway --timeout="${KITE_ROLLOUT_TIMEOUT}"
  kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-frontend --timeout="${KITE_ROLLOUT_TIMEOUT}"

  if [[ "${APPLY_GOLDEN_IMAGE}" == "true" ]]; then
    log "applying golden image"
    kubectl apply -f "${ROOT_DIR}/build/kite-storage/golden-images"
    "${ROOT_DIR}/build/deploy/scripts/wait-golden-image.sh" ubuntu-22.04
  fi

  if [[ "${RUN_VERIFY}" == "true" ]]; then
    "${ROOT_DIR}/build/deploy/scripts/verify.sh"
  else
    log "skipping verify because RUN_VERIFY=${RUN_VERIFY}"
  fi
}

main "$@"
