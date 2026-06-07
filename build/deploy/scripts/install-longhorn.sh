#!/usr/bin/env bash
set -euo pipefail

LONGHORN_VERSION="${LONGHORN_VERSION:-}"
LONGHORN_MANIFEST_URL="${LONGHORN_MANIFEST_URL:-}"

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

  if [[ -z "${LONGHORN_MANIFEST_URL}" ]]; then
    if [[ -z "${LONGHORN_VERSION}" ]]; then
      require_command curl
      LONGHORN_VERSION="$(curl -fsSL https://api.github.com/repos/longhorn/longhorn/releases/latest | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
    fi
    LONGHORN_MANIFEST_URL="https://raw.githubusercontent.com/longhorn/longhorn/${LONGHORN_VERSION}/deploy/longhorn.yaml"
  fi

  log "installing Longhorn from ${LONGHORN_MANIFEST_URL}"
  kubectl apply -f "${LONGHORN_MANIFEST_URL}"
}

main "$@"
