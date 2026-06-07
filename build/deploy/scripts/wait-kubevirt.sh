#!/usr/bin/env bash
set -euo pipefail

TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-900}"

log() {
  echo "[kite-deploy] $*"
}

main() {
  log "waiting for KubeVirt control plane"
  kubectl wait -n kubevirt kubevirt/kubevirt --for=condition=Available --timeout="${TIMEOUT_SECONDS}s"
  kubectl wait --for=condition=Ready pod -n kubevirt --all --timeout="${TIMEOUT_SECONDS}s"
}

main "$@"
