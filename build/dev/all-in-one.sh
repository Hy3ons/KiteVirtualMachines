#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/dev/all-in-one.sh
# Description: 로컬 소스 빌드 기반 개발 설치 전체 흐름을 실행한다.
#
# Usage:
#   build/dev/all-in-one.sh
#
# Environment Variables:
#   KITE_CLUSTER: default auto
#   INSTALL_LONGHORN: default true
#   INSTALL_KUBEVIRT: default true
#   INSTALL_CDI: default true
#   CONFIGURE_LONGHORN: default true
#   APPLY_STORAGECLASS: default true
#   APPLY_GOLDEN_IMAGE: default true
#   DEPLOY_KITE: default true
#   RUN_VERIFY: default true
#   MANAGE_HOST_SSHD: default true
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   Kubernetes 리소스 적용, 컨테이너 이미지 빌드/주입, rollout 대기를 수행할 수 있다.
# ==============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
INSTALL_LONGHORN_WAS_SET="${INSTALL_LONGHORN+x}"
INSTALL_KUBEVIRT_WAS_SET="${INSTALL_KUBEVIRT+x}"
INSTALL_CDI_WAS_SET="${INSTALL_CDI+x}"
CONFIGURE_LONGHORN_WAS_SET="${CONFIGURE_LONGHORN+x}"
APPLY_STORAGECLASS_WAS_SET="${APPLY_STORAGECLASS+x}"
APPLY_GOLDEN_IMAGE_WAS_SET="${APPLY_GOLDEN_IMAGE+x}"
DEPLOY_KITE_WAS_SET="${DEPLOY_KITE+x}"
RUN_VERIFY_WAS_SET="${RUN_VERIFY+x}"
MANAGE_HOST_SSHD_WAS_SET="${MANAGE_HOST_SSHD+x}"
KITE_CLUSTER="${KITE_CLUSTER:-auto}"
INSTALL_LONGHORN="${INSTALL_LONGHORN:-true}"
INSTALL_KUBEVIRT="${INSTALL_KUBEVIRT:-true}"
INSTALL_CDI="${INSTALL_CDI:-true}"
CONFIGURE_LONGHORN="${CONFIGURE_LONGHORN:-true}"
APPLY_STORAGECLASS="${APPLY_STORAGECLASS:-true}"
APPLY_GOLDEN_IMAGE="${APPLY_GOLDEN_IMAGE:-true}"
DEPLOY_KITE="${DEPLOY_KITE:-true}"
RUN_VERIFY="${RUN_VERIFY:-true}"
MANAGE_HOST_SSHD="${MANAGE_HOST_SSHD:-true}"

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
    printf "\033[0;32m[kite-all-in-one] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-all-in-one] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-all-in-one] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-all-in-one] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}


# 필수 CLI가 없으면 일부 리소스만 적용된 상태로 멈추지 않도록 초기에 중단한다.
require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

# enabled 값이 true인 단계만 실행한다. 환경변수로 설치 단계를 세밀하게 건너뛸 때 사용한다.
run_step() {
  local enabled="$1"
  local description="$2"
  shift 2

  if [[ "${enabled}" != "true" ]]; then
    log "skipping ${description}"
    return
  fi

  log "${description}"
  "$@"
}

# storage/golden image와 app manifest가 공통으로 쓰는 kite namespace를 먼저 만든다.
ensure_kite_namespace() {
  log "ensuring kite namespace exists for storage and app manifests"
  kubectl apply -f "${ROOT_DIR}/build/kite/namespace.yaml"
}

configure_interactive_install_options() {
  kite_prompt_interactive || return 0

  log "interactive install options"
  kite_prompt_configure_bool MANAGE_HOST_SSHD "${MANAGE_HOST_SSHD_WAS_SET}" "Kite gateway가 22번을 쓸 수 있게 host sshd handoff를 확인할까요?"
  kite_prompt_configure_bool INSTALL_LONGHORN "${INSTALL_LONGHORN_WAS_SET}" "Longhorn을 설치할까요?"
  kite_prompt_configure_bool CONFIGURE_LONGHORN "${CONFIGURE_LONGHORN_WAS_SET}" "Longhorn disk/tag 설정을 적용할까요?"
  kite_prompt_configure_bool APPLY_STORAGECLASS "${APPLY_STORAGECLASS_WAS_SET}" "Kite VM용 Longhorn StorageClass를 적용할까요?"
  kite_prompt_configure_bool INSTALL_KUBEVIRT "${INSTALL_KUBEVIRT_WAS_SET}" "KubeVirt를 설치할까요?"
  kite_prompt_configure_bool INSTALL_CDI "${INSTALL_CDI_WAS_SET}" "CDI를 설치할까요?"
  kite_prompt_configure_bool APPLY_GOLDEN_IMAGE "${APPLY_GOLDEN_IMAGE_WAS_SET}" "Ubuntu golden image DataVolume을 적용할까요?"
  kite_prompt_configure_bool DEPLOY_KITE "${DEPLOY_KITE_WAS_SET}" "Kite API/controller/gateway/frontend를 빌드하고 배포할까요?"
  kite_prompt_configure_bool RUN_VERIFY "${RUN_VERIFY_WAS_SET}" "설치 후 verify 스크립트를 실행할까요?"

  log "install choices: MANAGE_HOST_SSHD=${MANAGE_HOST_SSHD}, INSTALL_LONGHORN=${INSTALL_LONGHORN}, CONFIGURE_LONGHORN=${CONFIGURE_LONGHORN}, APPLY_STORAGECLASS=${APPLY_STORAGECLASS}, INSTALL_KUBEVIRT=${INSTALL_KUBEVIRT}, INSTALL_CDI=${INSTALL_CDI}, APPLY_GOLDEN_IMAGE=${APPLY_GOLDEN_IMAGE}, DEPLOY_KITE=${DEPLOY_KITE}, RUN_VERIFY=${RUN_VERIFY}"
}

# 개발용 전체 설치 순서다. 각 단계는 run_step으로 감싸져 있어 환경변수로 on/off할 수 있다.
main() {
  require_command kubectl

  log "checking Kubernetes connectivity"
  kubectl get nodes >/dev/null
  configure_interactive_install_options
  # gateway가 22번 포트를 쓰는 배포에서는 host sshd handoff가 먼저 필요할 수 있다.
  run_step "${MANAGE_HOST_SSHD}" "checking host sshd handoff for Kite gateway" "${ROOT_DIR}/build/deploy/scripts/manage-host-sshd.sh" ensure

  run_step "${INSTALL_LONGHORN}" "installing Longhorn" "${ROOT_DIR}/build/deploy/scripts/install-longhorn.sh"
  run_step "${CONFIGURE_LONGHORN}" "waiting for Longhorn" "${ROOT_DIR}/build/deploy/scripts/wait-longhorn.sh"
  run_step "${CONFIGURE_LONGHORN}" "configuring Kite Longhorn disk" "${ROOT_DIR}/build/deploy/scripts/configure-longhorn-kite-disk.sh"
  run_step "${APPLY_STORAGECLASS}" "applying Kite VM StorageClass" kubectl apply -f "${ROOT_DIR}/build/kite-storage/longhorn/storageclass.yaml"

  run_step "${INSTALL_KUBEVIRT}" "installing KubeVirt" "${ROOT_DIR}/build/deploy/scripts/install-kubevirt.sh"
  run_step "true" "waiting for KubeVirt" "${ROOT_DIR}/build/deploy/scripts/wait-kubevirt.sh"

  run_step "${INSTALL_CDI}" "installing CDI" "${ROOT_DIR}/build/deploy/scripts/install-cdi.sh"
  run_step "true" "waiting for CDI" "${ROOT_DIR}/build/deploy/scripts/wait-cdi.sh"

  ensure_kite_namespace
  # golden image는 VM DataVolume clone 원본으로 쓰이므로 app 배포 전에 준비한다.
  run_step "${APPLY_GOLDEN_IMAGE}" "applying Ubuntu golden image" kubectl apply -f "${ROOT_DIR}/build/kite-storage/golden-images"
  run_step "${APPLY_GOLDEN_IMAGE}" "waiting for Ubuntu golden image" "${ROOT_DIR}/build/deploy/scripts/wait-golden-image.sh" ubuntu-22.04

  run_step "${DEPLOY_KITE}" "building local Kite images and deploying Kite" "${ROOT_DIR}/build/dev/dev.sh"
  run_step "${RUN_VERIFY}" "verifying Kite installation" "${ROOT_DIR}/build/deploy/scripts/verify.sh"

  log "all-in-one install complete"
}

main "$@"
