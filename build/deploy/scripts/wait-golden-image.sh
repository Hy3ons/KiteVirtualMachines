#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME="${1:-ubuntu-22.04}"
NAMESPACE="${KITE_NAMESPACE:-kite}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-1800}"
SLEEP_SECONDS="${SLEEP_SECONDS:-10}"

log() {
  echo "[kite-deploy] $*"
}

main() {
  local elapsed=0
  local phase

  log "waiting for DataVolume ${NAMESPACE}/${IMAGE_NAME} to reach Succeeded"
  while (( elapsed <= TIMEOUT_SECONDS )); do
    phase="$(kubectl -n "${NAMESPACE}" get datavolume "${IMAGE_NAME}" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
    if [[ "${phase}" == "Succeeded" ]]; then
      log "DataVolume ${NAMESPACE}/${IMAGE_NAME} is Succeeded"
      return
    fi
    if [[ "${phase}" == "Failed" ]]; then
      kubectl -n "${NAMESPACE}" describe datavolume "${IMAGE_NAME}" || true
      echo "[kite-deploy] DataVolume ${NAMESPACE}/${IMAGE_NAME} failed" >&2
      exit 1
    fi

    log "DataVolume phase=${phase:-unknown}; waiting"
    sleep "${SLEEP_SECONDS}"
    elapsed=$((elapsed + SLEEP_SECONDS))
  done

  kubectl -n "${NAMESPACE}" describe datavolume "${IMAGE_NAME}" || true
  echo "[kite-deploy] timed out waiting for DataVolume ${NAMESPACE}/${IMAGE_NAME}" >&2
  exit 1
}

main "$@"
