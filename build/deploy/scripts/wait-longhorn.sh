#!/usr/bin/env bash
set -euo pipefail

TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-900}"

log() {
  echo "[kite-deploy] $*"
}

main() {
  log "waiting for Longhorn namespace"
  kubectl wait --for=condition=Ready pod -n longhorn-system --all --timeout="${TIMEOUT_SECONDS}s"
}

main "$@"
