#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/install-longhorn.sh
# Description: Longhorn 기본 manifest를 설치한다.
#
# Usage:
#   build/deploy/scripts/install-longhorn.sh
#
# Environment Variables:
#   LONGHORN_VERSION: default (empty)
#   LONGHORN_MANIFEST_URL: default (empty)
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   Kubernetes 리소스 적용, 컨테이너 이미지 빌드/주입, rollout 대기를 수행할 수 있다.
# ==============================================================================

LONGHORN_VERSION="${LONGHORN_VERSION:-}"
LONGHORN_MANIFEST_URL="${LONGHORN_MANIFEST_URL:-}"

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


# kubectl/curl 누락을 초기에 확인해 Longhorn 설치가 중간에 멈추지 않게 한다.
require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

# Longhorn 기본 manifest를 설치한다. 이미 클러스터에 Longhorn이 있으면 apply로 수렴시킨다.
main() {
  require_command kubectl

  if [[ -z "${LONGHORN_MANIFEST_URL}" ]]; then
    if [[ -z "${LONGHORN_VERSION}" ]]; then
      require_command curl
      # 명시 버전이 없으면 upstream latest release tag를 가져와 기본 manifest URL에 사용한다.
      LONGHORN_VERSION="$(curl -fsSL https://api.github.com/repos/longhorn/longhorn/releases/latest | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
    fi
    LONGHORN_MANIFEST_URL="https://raw.githubusercontent.com/longhorn/longhorn/${LONGHORN_VERSION}/deploy/longhorn.yaml"
  fi

  log "installing Longhorn from ${LONGHORN_MANIFEST_URL}"
  # Longhorn은 namespace, CRD, controller를 한 manifest에서 설치한다.
  kubectl apply -f "${LONGHORN_MANIFEST_URL}"
}

main "$@"
