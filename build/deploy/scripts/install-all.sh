#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
INSTALL_LONGHORN="${INSTALL_LONGHORN:-false}"
INSTALL_KUBEVIRT="${INSTALL_KUBEVIRT:-true}"
INSTALL_CDI="${INSTALL_CDI:-true}"
APPLY_GOLDEN_IMAGE="${APPLY_GOLDEN_IMAGE:-true}"

log() {
  echo "[kite-deploy] $*"
}

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "[kite-deploy] missing required command: ${name}" >&2
    exit 1
  fi
}

main() {
  require_command kubectl

  kubectl get nodes >/dev/null

  if [[ "${INSTALL_LONGHORN}" == "true" ]]; then
    "${ROOT_DIR}/build/deploy/scripts/install-longhorn.sh"
  else
    log "skipping Longhorn install; set INSTALL_LONGHORN=true to apply the default manifest"
  fi
  "${ROOT_DIR}/build/deploy/scripts/wait-longhorn.sh"
  "${ROOT_DIR}/build/deploy/scripts/configure-longhorn-kite-disk.sh"
  kubectl apply -f "${ROOT_DIR}/build/kite-storage/longhorn"

  if [[ "${INSTALL_KUBEVIRT}" == "true" ]]; then
    "${ROOT_DIR}/build/deploy/scripts/install-kubevirt.sh"
  fi
  "${ROOT_DIR}/build/deploy/scripts/wait-kubevirt.sh"

  if [[ "${INSTALL_CDI}" == "true" ]]; then
    "${ROOT_DIR}/build/deploy/scripts/install-cdi.sh"
  fi
  "${ROOT_DIR}/build/deploy/scripts/wait-cdi.sh"

  log "applying Kite manifests"
  kubectl apply -k "${ROOT_DIR}/build/kite"

  log "waiting for Kite workloads"
  kubectl -n kite rollout status deployment/kite-api --timeout=180s
  kubectl -n kite rollout status deployment/kite-controller --timeout=180s
  kubectl -n kite rollout status deployment/kite-frontend --timeout=180s
  kubectl -n kite rollout status daemonset/kite-host-agent --timeout=180s

  if [[ "${APPLY_GOLDEN_IMAGE}" == "true" ]]; then
    log "applying golden image"
    kubectl apply -f "${ROOT_DIR}/build/kite-storage/golden-images"
    "${ROOT_DIR}/build/deploy/scripts/wait-golden-image.sh" ubuntu-22.04
  fi

  "${ROOT_DIR}/build/deploy/scripts/verify.sh"
}

main "$@"
