#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/wait-longhorn.sh
# Description: Longhorn Pod가 Ready 상태가 될 때까지 대기한다.
#
# Usage:
#   build/deploy/scripts/wait-longhorn.sh
#
# Environment Variables:
#   TIMEOUT_SECONDS: default 900
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   주로 상태 조회와 대기를 수행하며, test는 임시 port-forward process를 생성한다.
# ==============================================================================

TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-900}"

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


wait_for_longhorn_pods_to_exist() {
  local deadline
  local pod_count

  deadline=$((SECONDS + TIMEOUT_SECONDS))
  while true; do
    pod_count="$(kubectl -n longhorn-system get pod --no-headers 2>/dev/null | sed '/^[[:space:]]*$/d' | wc -l | tr -d ' ')"
    if [[ "${pod_count}" != "0" ]]; then
      return
    fi
    if (( SECONDS >= deadline )); then
      warn "no Longhorn pods appeared in namespace/longhorn-system after ${TIMEOUT_SECONDS}s"
      return 1
    fi
    log "waiting for Longhorn pods to appear"
    sleep 2
  done
}

# Longhorn Pod가 모두 Ready가 될 때까지 기다린다. 이후 StorageClass/디스크 태그 설정이 이어진다.
main() {
  log "waiting for Longhorn namespace"
  wait_for_longhorn_pods_to_exist
  # --all 대기는 manager/UI/engine 관련 Pod가 남아 있는 초기화 시간을 흡수한다.
  kubectl wait --for=condition=Ready pod -n longhorn-system --all --timeout="${TIMEOUT_SECONDS}s"
}

main "$@"
