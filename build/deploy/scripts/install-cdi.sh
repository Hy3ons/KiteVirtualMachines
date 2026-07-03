#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/install-cdi.sh
# Description: CDI Operator와 CR을 설치한다.
#
# Usage:
#   build/deploy/scripts/install-cdi.sh
#
# Environment Variables:
#   CDI_VERSION: default (empty)
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   Kubernetes 리소스 적용, 컨테이너 이미지 빌드/주입, rollout 대기를 수행할 수 있다.
# ==============================================================================

CDI_VERSION="${CDI_VERSION:-}"

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


# kubectl/curl 누락을 초기에 확인해 CRD 설치가 절반만 진행되는 상황을 피한다.
require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

# CDI Operator와 CR을 설치한다. CDI는 golden image를 DataVolume으로 가져올 때 필요하다.
main() {
  require_command kubectl

  if [[ -z "${CDI_VERSION}" ]]; then
    require_command curl
    # 명시 버전이 없으면 GitHub latest release에서 tag_name만 뽑아 같은 버전의 manifest를 쓴다.
    CDI_VERSION="$(curl -fsSL https://api.github.com/repos/kubevirt/containerized-data-importer/releases/latest | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  fi

  log "installing CDI ${CDI_VERSION}"
  # operator는 컨트롤러를 설치하고, cr.yaml은 실제 CDI 인스턴스를 생성한다.
  kubectl apply -f "https://github.com/kubevirt/containerized-data-importer/releases/download/${CDI_VERSION}/cdi-operator.yaml"
  kubectl apply -f "https://github.com/kubevirt/containerized-data-importer/releases/download/${CDI_VERSION}/cdi-cr.yaml"
}

main "$@"
