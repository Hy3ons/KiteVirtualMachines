#!/usr/bin/env bash
set -euo pipefail

# all-in-one.sh is the local-build all-in-one installer.
# It installs or waits for the virtualization/storage stack, imports the golden VM image,
# builds Kite containers from this checkout, and applies the Kite Deployments/DaemonSet.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
KITE_CLUSTER="${KITE_CLUSTER:-auto}"
INSTALL_LONGHORN="${INSTALL_LONGHORN:-true}"
INSTALL_KUBEVIRT="${INSTALL_KUBEVIRT:-true}"
INSTALL_CDI="${INSTALL_CDI:-true}"
CONFIGURE_LONGHORN="${CONFIGURE_LONGHORN:-true}"
APPLY_STORAGECLASS="${APPLY_STORAGECLASS:-true}"
APPLY_GOLDEN_IMAGE="${APPLY_GOLDEN_IMAGE:-true}"
DEPLOY_KITE="${DEPLOY_KITE:-true}"
RUN_VERIFY="${RUN_VERIFY:-true}"

log() {
  echo "[kite-all-in-one] $*"
}

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "[kite-all-in-one] missing required command: ${name}" >&2
    exit 1
  fi
}

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

ensure_kite_namespace() {
  log "ensuring kite namespace exists for storage and app manifests"
  kubectl apply -f "${ROOT_DIR}/build/kite/namespace.yaml"
}

main() {
  require_command kubectl

  log "checking Kubernetes connectivity"
  kubectl get nodes >/dev/null

  run_step "${INSTALL_LONGHORN}" "installing Longhorn" "${ROOT_DIR}/build/deploy/scripts/install-longhorn.sh"
  run_step "${CONFIGURE_LONGHORN}" "waiting for Longhorn" "${ROOT_DIR}/build/deploy/scripts/wait-longhorn.sh"
  run_step "${CONFIGURE_LONGHORN}" "configuring Kite Longhorn disk" "${ROOT_DIR}/build/deploy/scripts/configure-longhorn-kite-disk.sh"
  run_step "${APPLY_STORAGECLASS}" "applying Kite VM StorageClass" kubectl apply -f "${ROOT_DIR}/build/kite-storage/longhorn/storageclass.yaml"

  run_step "${INSTALL_KUBEVIRT}" "installing KubeVirt" "${ROOT_DIR}/build/deploy/scripts/install-kubevirt.sh"
  run_step "true" "waiting for KubeVirt" "${ROOT_DIR}/build/deploy/scripts/wait-kubevirt.sh"

  run_step "${INSTALL_CDI}" "installing CDI" "${ROOT_DIR}/build/deploy/scripts/install-cdi.sh"
  run_step "true" "waiting for CDI" "${ROOT_DIR}/build/deploy/scripts/wait-cdi.sh"

  ensure_kite_namespace
  run_step "${APPLY_GOLDEN_IMAGE}" "applying Ubuntu golden image" kubectl apply -f "${ROOT_DIR}/build/kite-storage/golden-images"
  run_step "${APPLY_GOLDEN_IMAGE}" "waiting for Ubuntu golden image" "${ROOT_DIR}/build/deploy/scripts/wait-golden-image.sh" ubuntu-22.04

  run_step "${DEPLOY_KITE}" "building local Kite images and deploying Kite" "${ROOT_DIR}/build/dev/dev.sh"
  run_step "${RUN_VERIFY}" "verifying Kite installation" "${ROOT_DIR}/build/deploy/scripts/verify.sh"

  log "all-in-one install complete"
}

main "$@"
