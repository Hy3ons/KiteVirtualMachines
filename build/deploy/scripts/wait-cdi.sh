#!/usr/bin/env bash
set -euo pipefail

TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-900}"

log() {
  echo "[kite-deploy] $*"
}

main() {
  log "waiting for CDI control plane"
  kubectl wait -n cdi cdi/cdi --for=condition=Available --timeout="${TIMEOUT_SECONDS}s"
  kubectl wait --for=condition=Ready pod -n cdi --all --timeout="${TIMEOUT_SECONDS}s"
}

main "$@"
