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
#   INSTALL_LONGHORN: default false
#   CONFIGURE_LONGHORN: default true
#   APPLY_STORAGECLASS: default true
#   INSTALL_KUBEVIRT: default true
#   INSTALL_CDI: default true
#   APPLY_GOLDEN_IMAGE: default true
#   MANAGE_HOST_SSHD: default true
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
RUN_VERIFY_WAS_SET="${RUN_VERIFY+x}"
KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
INSTALL_LONGHORN="${INSTALL_LONGHORN:-false}"
CONFIGURE_LONGHORN="${CONFIGURE_LONGHORN:-true}"
APPLY_STORAGECLASS="${APPLY_STORAGECLASS:-true}"
INSTALL_KUBEVIRT="${INSTALL_KUBEVIRT:-true}"
INSTALL_CDI="${INSTALL_CDI:-true}"
APPLY_GOLDEN_IMAGE="${APPLY_GOLDEN_IMAGE:-true}"
MANAGE_HOST_SSHD="${MANAGE_HOST_SSHD:-true}"
RUN_VERIFY="${RUN_VERIFY:-true}"

# shellcheck source=build/scripts/prompt.sh
source "${ROOT_DIR}/build/scripts/prompt.sh"

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

configure_interactive_install_options() {
  kite_prompt_interactive || return 0

  log "interactive install options"
  kite_prompt_configure_bool MANAGE_HOST_SSHD "${MANAGE_HOST_SSHD_WAS_SET}" "Kite gateway가 22번을 쓸 수 있게 host sshd handoff를 확인할까요?"
  kite_prompt_configure_bool INSTALL_LONGHORN "${INSTALL_LONGHORN_WAS_SET}" "Longhorn 기본 manifest를 설치할까요?"
  kite_prompt_configure_bool CONFIGURE_LONGHORN "${CONFIGURE_LONGHORN_WAS_SET}" "Longhorn에 Kite 전용 disk/tag 설정을 적용할까요?"
  kite_prompt_configure_bool APPLY_STORAGECLASS "${APPLY_STORAGECLASS_WAS_SET}" "Kite 전용 Longhorn StorageClass를 적용할까요?"
  kite_prompt_configure_bool INSTALL_KUBEVIRT "${INSTALL_KUBEVIRT_WAS_SET}" "KubeVirt를 설치할까요?"
  kite_prompt_configure_bool INSTALL_CDI "${INSTALL_CDI_WAS_SET}" "CDI를 설치할까요?"
  kite_prompt_configure_bool APPLY_GOLDEN_IMAGE "${APPLY_GOLDEN_IMAGE_WAS_SET}" "Ubuntu golden image DataVolume을 적용할까요?"
  kite_prompt_configure_bool RUN_VERIFY "${RUN_VERIFY_WAS_SET}" "설치 후 verify 스크립트를 실행할까요?"

  log "install choices: MANAGE_HOST_SSHD=${MANAGE_HOST_SSHD}, INSTALL_LONGHORN=${INSTALL_LONGHORN}, CONFIGURE_LONGHORN=${CONFIGURE_LONGHORN}, APPLY_STORAGECLASS=${APPLY_STORAGECLASS}, INSTALL_KUBEVIRT=${INSTALL_KUBEVIRT}, INSTALL_CDI=${INSTALL_CDI}, APPLY_GOLDEN_IMAGE=${APPLY_GOLDEN_IMAGE}, RUN_VERIFY=${RUN_VERIFY}"
}

# pull 기반 설치의 전체 순서다. host sshd handoff, Longhorn/KubeVirt/CDI 준비,
# Kite 매니페스트 적용, golden image 적용, 기본 검증까지 한 번에 진행한다.
main() {
  require_command kubectl

  kubectl get nodes >/dev/null
  configure_interactive_install_options
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
    log "skipping Longhorn install; set INSTALL_LONGHORN=true to apply the default manifest"
  fi
  if [[ "${CONFIGURE_LONGHORN}" == "true" ]]; then
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
  # build/kite kustomization에는 API/controller/gateway/frontend 런타임 리소스가 모여 있다.
  kubectl apply -k "${ROOT_DIR}/build/kite"

  log "waiting for Kite workloads"
  kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-api --timeout=180s
  kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-controller --timeout=180s
  kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-gateway --timeout=180s
  kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-frontend --timeout=180s

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
