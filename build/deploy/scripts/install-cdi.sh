#!/usr/bin/env bash
set -euo pipefail

CDI_VERSION="${CDI_VERSION:-}"

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

  if [[ -z "${CDI_VERSION}" ]]; then
    require_command curl
    CDI_VERSION="$(curl -fsSL https://api.github.com/repos/kubevirt/containerized-data-importer/releases/latest | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  fi

  log "installing CDI ${CDI_VERSION}"
  kubectl apply -f "https://github.com/kubevirt/containerized-data-importer/releases/download/${CDI_VERSION}/cdi-operator.yaml"
  kubectl apply -f "https://github.com/kubevirt/containerized-data-importer/releases/download/${CDI_VERSION}/cdi-cr.yaml"
}

main "$@"
