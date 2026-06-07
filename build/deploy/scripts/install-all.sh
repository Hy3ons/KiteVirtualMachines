#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
INSTALL_LONGHORN="${INSTALL_LONGHORN:-false}"
INSTALL_KUBEVIRT="${INSTALL_KUBEVIRT:-true}"
INSTALL_CDI="${INSTALL_CDI:-true}"
APPLY_GOLDEN_IMAGE="${APPLY_GOLDEN_IMAGE:-true}"
KITE_GATEWAY_HOST_KEY_SECRET="${KITE_GATEWAY_HOST_KEY_SECRET:-kite-gateway-host-key}"
MANAGE_HOST_SSHD="${MANAGE_HOST_SSHD:-true}"

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

ensure_gateway_host_key_secret() {
  local tmpdir

  kubectl apply -f "${ROOT_DIR}/build/kite/namespace.yaml"
  if kubectl -n "${KITE_NAMESPACE}" get secret "${KITE_GATEWAY_HOST_KEY_SECRET}" >/dev/null 2>&1; then
    return
  fi

  require_command ssh-keygen
  tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/kite-gateway-host-key.XXXXXX")"
  ssh-keygen -q -t rsa -b 4096 -N "" -f "${tmpdir}/ssh_host_rsa_key"
  kubectl -n "${KITE_NAMESPACE}" create secret generic "${KITE_GATEWAY_HOST_KEY_SECRET}" \
    --from-file=ssh_host_rsa_key="${tmpdir}/ssh_host_rsa_key"
  rm -rf "${tmpdir}"
}

main() {
  require_command kubectl

  kubectl get nodes >/dev/null
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
  "${ROOT_DIR}/build/deploy/scripts/wait-longhorn.sh"
  "${ROOT_DIR}/build/deploy/scripts/configure-longhorn-kite-disk.sh"
  kubectl apply -f "${ROOT_DIR}/build/kite-storage/longhorn/storageclass.yaml"

  if [[ "${INSTALL_KUBEVIRT}" == "true" ]]; then
    "${ROOT_DIR}/build/deploy/scripts/install-kubevirt.sh"
  fi
  "${ROOT_DIR}/build/deploy/scripts/wait-kubevirt.sh"

  if [[ "${INSTALL_CDI}" == "true" ]]; then
    "${ROOT_DIR}/build/deploy/scripts/install-cdi.sh"
  fi
  "${ROOT_DIR}/build/deploy/scripts/wait-cdi.sh"

  log "applying Kite manifests"
  ensure_gateway_host_key_secret
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

  "${ROOT_DIR}/build/deploy/scripts/verify.sh"
}

main "$@"
