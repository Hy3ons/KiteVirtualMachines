#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/wait-kubevirt.sh
# Description: KubeVirt control plane이 사용 가능할 때까지 대기한다.
#
# Usage:
#   build/deploy/scripts/wait-kubevirt.sh
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


# KubeVirt 컨트롤 플레인이 Available 상태가 될 때까지 대기한다.
main() {
  log "waiting for KubeVirt control plane"
  # KubeVirt CR이 Available이어야 VM/VMI reconcile이 정상적으로 동작한다.
  kubectl wait -n kubevirt kubevirt/kubevirt --for=condition=Available --timeout="${TIMEOUT_SECONDS}s"
  kubectl wait --for=condition=Ready pod -n kubevirt --all --timeout="${TIMEOUT_SECONDS}s"
}

main "$@"
