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
#   KITE_LONGHORN_USE_DEDICATED_DISK: default false
#   KITE_GATEWAY_HOST_KEY_REFRESH: default false
#   FRONTEND_VITE_USE_MOCK: default false
#   K3S_IMPORT_IMAGES: default true
#   K3D_LOAD_IMAGES: default true
#   KIND_LOAD_IMAGES: default true
#   MINIKUBE_START: default true
#   PUSH_IMAGES: default false
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
KITE_LONGHORN_USE_DEDICATED_DISK_WAS_SET="${KITE_LONGHORN_USE_DEDICATED_DISK+x}"
KITE_GATEWAY_HOST_KEY_REFRESH_WAS_SET="${KITE_GATEWAY_HOST_KEY_REFRESH+x}"
FRONTEND_VITE_USE_MOCK_WAS_SET="${FRONTEND_VITE_USE_MOCK+x}"
K3S_IMPORT_IMAGES_WAS_SET="${K3S_IMPORT_IMAGES+x}"
K3D_LOAD_IMAGES_WAS_SET="${K3D_LOAD_IMAGES+x}"
KIND_LOAD_IMAGES_WAS_SET="${KIND_LOAD_IMAGES+x}"
MINIKUBE_START_WAS_SET="${MINIKUBE_START+x}"
PUSH_IMAGES_WAS_SET="${PUSH_IMAGES+x}"
KITE_CLUSTER="${KITE_CLUSTER:-auto}"
INSTALL_LONGHORN="${INSTALL_LONGHORN:-true}"
INSTALL_KUBEVIRT="${INSTALL_KUBEVIRT:-true}"
INSTALL_CDI="${INSTALL_CDI:-true}"
CONFIGURE_LONGHORN="${CONFIGURE_LONGHORN:-true}"
APPLY_STORAGECLASS="${APPLY_STORAGECLASS:-true}"
APPLY_GOLDEN_IMAGE="${APPLY_GOLDEN_IMAGE:-true}"
DEPLOY_KITE="${DEPLOY_KITE:-true}"
RUN_VERIFY="${RUN_VERIFY:-true}"
KITE_LONGHORN_USE_DEDICATED_DISK="${KITE_LONGHORN_USE_DEDICATED_DISK:-false}"
KITE_GATEWAY_HOST_KEY_REFRESH="${KITE_GATEWAY_HOST_KEY_REFRESH:-false}"
FRONTEND_VITE_USE_MOCK="${FRONTEND_VITE_USE_MOCK:-false}"
K3S_IMPORT_IMAGES="${K3S_IMPORT_IMAGES:-true}"
K3D_LOAD_IMAGES="${K3D_LOAD_IMAGES:-true}"
KIND_LOAD_IMAGES="${KIND_LOAD_IMAGES:-true}"
MINIKUBE_START="${MINIKUBE_START:-true}"
PUSH_IMAGES="${PUSH_IMAGES:-false}"

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
  kite_prompt_configure_bool INSTALL_LONGHORN "${INSTALL_LONGHORN_WAS_SET}" "Longhorn을 설치할까요?"
  kite_prompt_configure_bool CONFIGURE_LONGHORN "${CONFIGURE_LONGHORN_WAS_SET}" "Longhorn disk/tag 설정을 적용할까요?"
  if [[ "${CONFIGURE_LONGHORN}" == "true" ]]; then
    kite_prompt_configure_bool KITE_LONGHORN_USE_DEDICATED_DISK "${KITE_LONGHORN_USE_DEDICATED_DISK_WAS_SET}" "Longhorn에 Kite 전용 host path disk entry를 만들까요? 아니오면 기존 Ready disk에 kite tag만 붙입니다."
  fi
  kite_prompt_configure_bool APPLY_STORAGECLASS "${APPLY_STORAGECLASS_WAS_SET}" "Kite VM용 Longhorn StorageClass를 적용할까요?"
  kite_prompt_configure_bool INSTALL_KUBEVIRT "${INSTALL_KUBEVIRT_WAS_SET}" "KubeVirt를 설치할까요?"
  kite_prompt_configure_bool INSTALL_CDI "${INSTALL_CDI_WAS_SET}" "CDI를 설치할까요?"
  kite_prompt_configure_bool APPLY_GOLDEN_IMAGE "${APPLY_GOLDEN_IMAGE_WAS_SET}" "Ubuntu golden image DataVolume을 적용할까요?"
  kite_prompt_configure_bool DEPLOY_KITE "${DEPLOY_KITE_WAS_SET}" "Kite API/controller/gateway/frontend를 빌드하고 배포할까요?"
  if [[ "${DEPLOY_KITE}" == "true" ]]; then
    kite_prompt_configure_bool FRONTEND_VITE_USE_MOCK "${FRONTEND_VITE_USE_MOCK_WAS_SET}" "frontend 이미지를 mock API 모드로 빌드할까요?"
    case "${KITE_CLUSTER}" in
      minikube)
        kite_prompt_configure_bool MINIKUBE_START "${MINIKUBE_START_WAS_SET}" "배포 전에 minikube profile을 시작/갱신할까요?"
        ;;
      k3s)
        kite_prompt_configure_bool K3S_IMPORT_IMAGES "${K3S_IMPORT_IMAGES_WAS_SET}" "빌드한 이미지를 k3s containerd로 import할까요?"
        ;;
      k3d)
        kite_prompt_configure_bool K3D_LOAD_IMAGES "${K3D_LOAD_IMAGES_WAS_SET}" "빌드한 이미지를 k3d cluster로 load할까요?"
        ;;
      kind)
        kite_prompt_configure_bool KIND_LOAD_IMAGES "${KIND_LOAD_IMAGES_WAS_SET}" "빌드한 이미지를 kind cluster로 load할까요?"
        ;;
      current|k8s|kubernetes)
        kite_prompt_configure_bool PUSH_IMAGES "${PUSH_IMAGES_WAS_SET}" "현재 클러스터가 이미지를 pull할 수 있도록 registry에 push할까요?"
        ;;
    esac
    kite_prompt_configure_bool KITE_GATEWAY_HOST_KEY_REFRESH "${KITE_GATEWAY_HOST_KEY_REFRESH_WAS_SET}" "기존 kite-gateway host key Secret이 있으면 새 key로 갱신할까요?"
  fi
  kite_prompt_configure_bool RUN_VERIFY "${RUN_VERIFY_WAS_SET}" "설치 후 verify 스크립트를 실행할까요?"

  log "install choices: INSTALL_LONGHORN=${INSTALL_LONGHORN}, CONFIGURE_LONGHORN=${CONFIGURE_LONGHORN}, KITE_LONGHORN_USE_DEDICATED_DISK=${KITE_LONGHORN_USE_DEDICATED_DISK}, APPLY_STORAGECLASS=${APPLY_STORAGECLASS}, INSTALL_KUBEVIRT=${INSTALL_KUBEVIRT}, INSTALL_CDI=${INSTALL_CDI}, APPLY_GOLDEN_IMAGE=${APPLY_GOLDEN_IMAGE}, DEPLOY_KITE=${DEPLOY_KITE}, FRONTEND_VITE_USE_MOCK=${FRONTEND_VITE_USE_MOCK}, K3S_IMPORT_IMAGES=${K3S_IMPORT_IMAGES}, K3D_LOAD_IMAGES=${K3D_LOAD_IMAGES}, KIND_LOAD_IMAGES=${KIND_LOAD_IMAGES}, MINIKUBE_START=${MINIKUBE_START}, PUSH_IMAGES=${PUSH_IMAGES}, KITE_GATEWAY_HOST_KEY_REFRESH=${KITE_GATEWAY_HOST_KEY_REFRESH}, RUN_VERIFY=${RUN_VERIFY}"
}

export_install_options() {
  export KITE_LONGHORN_USE_DEDICATED_DISK
  export KITE_GATEWAY_HOST_KEY_REFRESH
  export FRONTEND_VITE_USE_MOCK
  export K3S_IMPORT_IMAGES
  export K3D_LOAD_IMAGES
  export KIND_LOAD_IMAGES
  export MINIKUBE_START
  export PUSH_IMAGES
}

# 개발용 전체 설치 순서다. 각 단계는 run_step으로 감싸져 있어 환경변수로 on/off할 수 있다.
main() {
  require_command kubectl

  log "checking Kubernetes connectivity"
  kubectl get nodes >/dev/null
  configure_interactive_install_options
  export_install_options

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
