#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/wait-golden-image.sh
# Description: golden image DataVolume import가 완료될 때까지 대기한다.
#
# Usage:
#   build/deploy/scripts/wait-golden-image.sh [datavolume-name]
#
# Environment Variables:
#   TIMEOUT_SECONDS: default 1800
#   SLEEP_SECONDS: default 10
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   주로 상태 조회와 대기를 수행하며, test는 임시 port-forward process를 생성한다.
# ==============================================================================

IMAGE_NAME="${1:-ubuntu-22.04}"
NAMESPACE="${KITE_NAMESPACE:-kite}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-1800}"
SLEEP_SECONDS="${SLEEP_SECONDS:-10}"

log_color_enabled() {
  [[ "${KITE_LOG_COLOR:-auto}" != "false" && -z "${NO_COLOR:-}" && -t 1 ]]
}

log_timestamp() {
  date +"%Y-%m-%dT%H:%M:%S%z"
}

log() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[0;32m[kite-deploy] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-deploy] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-deploy] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-deploy] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}


# golden image DataVolume import가 끝나고 Succeeded 상태가 될 때까지 폴링한다.
main() {
  local elapsed=0
  local phase
  local progress

  log "waiting for DataVolume ${NAMESPACE}/${IMAGE_NAME} to reach Succeeded"
  while (( elapsed <= TIMEOUT_SECONDS )); do
    # DataVolume phase/progress는 CDI가 비동기로 갱신하므로 짧게 반복 조회한다.
    phase="$(kubectl -n "${NAMESPACE}" get datavolume "${IMAGE_NAME}" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
    progress="$(kubectl -n "${NAMESPACE}" get datavolume "${IMAGE_NAME}" -o jsonpath='{.status.progress}' 2>/dev/null || true)"
    if [[ "${phase}" == "Succeeded" ]]; then
      log "DataVolume ${NAMESPACE}/${IMAGE_NAME} is Succeeded progress=${progress:-100%}"
      return
    fi
    if [[ "${phase}" == "Failed" ]]; then
      kubectl -n "${NAMESPACE}" describe datavolume "${IMAGE_NAME}" || true
      echo "[kite-deploy] DataVolume ${NAMESPACE}/${IMAGE_NAME} failed" >&2
      exit 1
    fi

    log "DataVolume phase=${phase:-unknown} progress=${progress:-unknown}; waiting"
    sleep "${SLEEP_SECONDS}"
    elapsed=$((elapsed + SLEEP_SECONDS))
  done

  kubectl -n "${NAMESPACE}" describe datavolume "${IMAGE_NAME}" || true
  echo "[kite-deploy] timed out waiting for DataVolume ${NAMESPACE}/${IMAGE_NAME}" >&2
  exit 1
}

main "$@"
