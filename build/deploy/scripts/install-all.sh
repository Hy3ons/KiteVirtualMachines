#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/install-all.sh
# Description: pull 기반 설치 전체 흐름을 실행한다. dependency 설치와 Kite manifest 적용을 포함한다.
#
# Usage:
#   build/deploy/scripts/install-all.sh
#
# Environment Variables:
#   KITE_NAMESPACE: default kite; Kite runtime을 설치할 namespace다. interactive에서 초반에 묻는다.
#   KITE_INSTALL_REGISTRY: default ghcr.io/hy3ons
#   KITE_INSTALL_IMAGE_TAG: default production
#   KITE_INSTALL_IMAGE_PULL_POLICY: default IfNotPresent
#   KITE_INSTALL_FORCE_ROLLOUT: default false
#   INSTALL_LONGHORN: default true
#   KITE_INSTALL_LONGHORN_HOST_PACKAGES: default true
#   CONFIGURE_LONGHORN: default true
#   APPLY_STORAGECLASS: default true
#   INSTALL_KUBEVIRT: default true
#   INSTALL_CDI: default true
#   APPLY_GOLDEN_IMAGE: default true
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
KITE_NAMESPACE_WAS_SET="${KITE_NAMESPACE+x}"
INSTALL_LONGHORN_WAS_SET="${INSTALL_LONGHORN+x}"
CONFIGURE_LONGHORN_WAS_SET="${CONFIGURE_LONGHORN+x}"
APPLY_STORAGECLASS_WAS_SET="${APPLY_STORAGECLASS+x}"
INSTALL_KUBEVIRT_WAS_SET="${INSTALL_KUBEVIRT+x}"
INSTALL_CDI_WAS_SET="${INSTALL_CDI+x}"
APPLY_GOLDEN_IMAGE_WAS_SET="${APPLY_GOLDEN_IMAGE+x}"
KITE_LONGHORN_USE_DEDICATED_DISK_WAS_SET="${KITE_LONGHORN_USE_DEDICATED_DISK+x}"
KITE_GATEWAY_HOST_KEY_REFRESH_WAS_SET="${KITE_GATEWAY_HOST_KEY_REFRESH+x}"
RUN_VERIFY_WAS_SET="${RUN_VERIFY+x}"
KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
KITE_INSTALL_REGISTRY="${KITE_INSTALL_REGISTRY:-ghcr.io/hy3ons}"
KITE_INSTALL_IMAGE_TAG="${KITE_INSTALL_IMAGE_TAG:-production}"
KITE_INSTALL_IMAGE_PULL_POLICY="${KITE_INSTALL_IMAGE_PULL_POLICY:-IfNotPresent}"
KITE_INSTALL_FORCE_ROLLOUT="${KITE_INSTALL_FORCE_ROLLOUT:-false}"
INSTALL_LONGHORN="${INSTALL_LONGHORN:-true}"
CONFIGURE_LONGHORN="${CONFIGURE_LONGHORN:-true}"
APPLY_STORAGECLASS="${APPLY_STORAGECLASS:-true}"
INSTALL_KUBEVIRT="${INSTALL_KUBEVIRT:-true}"
INSTALL_CDI="${INSTALL_CDI:-true}"
APPLY_GOLDEN_IMAGE="${APPLY_GOLDEN_IMAGE:-true}"
KITE_LONGHORN_USE_DEDICATED_DISK="${KITE_LONGHORN_USE_DEDICATED_DISK:-false}"
KITE_GATEWAY_HOST_KEY_REFRESH="${KITE_GATEWAY_HOST_KEY_REFRESH:-false}"
KITE_ROLLOUT_TIMEOUT="${KITE_ROLLOUT_TIMEOUT:-15m}"
RUN_VERIFY="${RUN_VERIFY:-true}"
KITE_INSTALL_KUSTOMIZE_DIR=""
KITE_INSTALL_EXISTING_RUNTIME="false"

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

validate_namespace() {
  if [[ -z "${KITE_NAMESPACE}" || ! "${KITE_NAMESPACE}" =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ || "${#KITE_NAMESPACE}" -gt 63 ]]; then
    echo "[kite-deploy] KITE_NAMESPACE must be a Kubernetes namespace name: lowercase letters, numbers, hyphen, max 63 chars" >&2
    exit 1
  fi
}

validate_install_options() {
  validate_namespace
  kite_validate_bool KITE_INSTALL_FORCE_ROLLOUT
  kite_validate_bool INSTALL_LONGHORN
  kite_validate_bool CONFIGURE_LONGHORN
  kite_validate_bool APPLY_STORAGECLASS
  kite_validate_bool INSTALL_KUBEVIRT
  kite_validate_bool INSTALL_CDI
  kite_validate_bool APPLY_GOLDEN_IMAGE
  kite_validate_bool KITE_LONGHORN_USE_DEDICATED_DISK
  kite_validate_bool KITE_GATEWAY_HOST_KEY_REFRESH
  kite_validate_bool RUN_VERIFY
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
  kite_prompt_value KITE_NAMESPACE "${KITE_NAMESPACE_WAS_SET}" "Kite를 어떤 Kubernetes namespace에 설치할까요?" "기본값은 kite입니다. 기존 설치를 업데이트하려면 이전 설치와 같은 namespace를 입력해야 합니다."
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

  log "install choices: namespace=${KITE_NAMESPACE}($(kite_option_source "${KITE_NAMESPACE_WAS_SET}")), INSTALL_LONGHORN=${INSTALL_LONGHORN}, CONFIGURE_LONGHORN=${CONFIGURE_LONGHORN}, KITE_LONGHORN_USE_DEDICATED_DISK=${KITE_LONGHORN_USE_DEDICATED_DISK}, APPLY_STORAGECLASS=${APPLY_STORAGECLASS}, INSTALL_KUBEVIRT=${INSTALL_KUBEVIRT}, INSTALL_CDI=${INSTALL_CDI}, APPLY_GOLDEN_IMAGE=${APPLY_GOLDEN_IMAGE}, KITE_GATEWAY_HOST_KEY_REFRESH=${KITE_GATEWAY_HOST_KEY_REFRESH}, RUN_VERIFY=${RUN_VERIFY}"
}

export_install_options() {
  export KITE_NAMESPACE
  export KITE_LONGHORN_USE_DEDICATED_DISK
  export KITE_GATEWAY_HOST_KEY_REFRESH
}

cleanup_install_overlay() {
  if [[ -n "${KITE_INSTALL_KUSTOMIZE_DIR:-}" ]]; then
    rm -rf "${KITE_INSTALL_KUSTOMIZE_DIR}"
  fi
}

render_install_manifest() {
  require_command mktemp

  KITE_INSTALL_KUSTOMIZE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/kite-ghcr-install-kustomize.XXXXXX")"
  cp -R "${ROOT_DIR}/build/kite" "${KITE_INSTALL_KUSTOMIZE_DIR}/kite"
  cat > "${KITE_INSTALL_KUSTOMIZE_DIR}/kustomization.yaml" <<EOF
resources:
  - kite

images:
  - name: ghcr.io/hy3ons/kite-api
    newName: ${KITE_INSTALL_REGISTRY}/kite-api
    newTag: ${KITE_INSTALL_IMAGE_TAG}
  - name: ghcr.io/hy3ons/kite-controller
    newName: ${KITE_INSTALL_REGISTRY}/kite-controller
    newTag: ${KITE_INSTALL_IMAGE_TAG}
  - name: ghcr.io/hy3ons/kite-gateway
    newName: ${KITE_INSTALL_REGISTRY}/kite-gateway
    newTag: ${KITE_INSTALL_IMAGE_TAG}
  - name: ghcr.io/hy3ons/kite-frontend
    newName: ${KITE_INSTALL_REGISTRY}/kite-frontend
    newTag: ${KITE_INSTALL_IMAGE_TAG}

patches:
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: kite-api
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: kite-api
      spec:
        template:
          spec:
            containers:
              - name: kite-api
                imagePullPolicy: ${KITE_INSTALL_IMAGE_PULL_POLICY}
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: kite-controller
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: kite-controller
      spec:
        template:
          spec:
            containers:
              - name: kite-controller
                imagePullPolicy: ${KITE_INSTALL_IMAGE_PULL_POLICY}
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: kite-gateway
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: kite-gateway
      spec:
        template:
          spec:
            containers:
              - name: kite-gateway
                imagePullPolicy: ${KITE_INSTALL_IMAGE_PULL_POLICY}
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: kite-frontend
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: kite-frontend
      spec:
        template:
          spec:
            containers:
              - name: kite-frontend
                imagePullPolicy: ${KITE_INSTALL_IMAGE_PULL_POLICY}
EOF
}

record_existing_runtime() {
  local deployment

  KITE_INSTALL_EXISTING_RUNTIME="false"
  for deployment in kite-api kite-controller kite-gateway kite-frontend; do
    if kubectl -n "${KITE_NAMESPACE}" get deployment "${deployment}" >/dev/null 2>&1; then
      KITE_INSTALL_EXISTING_RUNTIME="true"
      return 0
    fi
  done
}

restart_existing_runtime_if_requested() {
  if [[ "${KITE_INSTALL_FORCE_ROLLOUT}" != "true" ]]; then
    return 0
  fi
  if [[ "${KITE_INSTALL_EXISTING_RUNTIME}" != "true" ]]; then
    return 0
  fi

  log "restarting existing Kite workloads so mutable image tags are pulled again"
  kubectl -n "${KITE_NAMESPACE}" rollout restart \
    deployment/kite-api \
    deployment/kite-controller \
    deployment/kite-gateway \
    deployment/kite-frontend
}

apply_kite_manifests() {
  log "applying Kite manifests from ${KITE_INSTALL_REGISTRY}/<component>:${KITE_INSTALL_IMAGE_TAG}"
  "${ROOT_DIR}/build/deploy/scripts/ensure-gateway-host-key-secret.sh"
  render_install_manifest
  record_existing_runtime
  kubectl apply -k "${KITE_INSTALL_KUSTOMIZE_DIR}"
  restart_existing_runtime_if_requested
}

# pull 기반 설치의 전체 순서다. Longhorn/KubeVirt/CDI 준비,
# Kite 매니페스트 적용, golden image 적용, 기본 검증까지 한 번에 진행한다.
main() {
  require_command kubectl

  trap cleanup_install_overlay EXIT

  kubectl get nodes >/dev/null
  configure_interactive_install_options
  validate_install_options
  export_install_options

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
  apply_kite_manifests

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
