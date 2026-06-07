#!/usr/bin/env bash
set -euo pipefail

KUBEVIRT_VERSION="${KUBEVIRT_VERSION:-}"

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

  if [[ -z "${KUBEVIRT_VERSION}" ]]; then
    require_command curl
    KUBEVIRT_VERSION="$(curl -fsSL https://api.github.com/repos/kubevirt/kubevirt/releases/latest | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  fi

  log "installing KubeVirt ${KUBEVIRT_VERSION}"
  kubectl apply -f "https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-operator.yaml"
  kubectl apply -f "https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-cr.yaml"
}

main "$@"
