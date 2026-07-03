#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/install-kubevirt.sh
# Description: KubeVirt Operator와 CR을 설치한다.
#
# Usage:
#   build/deploy/scripts/install-kubevirt.sh
#
# Environment Variables:
#   KUBEVIRT_VERSION: default (empty)
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   Kubernetes 리소스 적용, 컨테이너 이미지 빌드/주입, rollout 대기를 수행할 수 있다.
# ==============================================================================

KUBEVIRT_VERSION="${KUBEVIRT_VERSION:-}"

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


# kubectl/curl 누락을 초기에 확인해 KubeVirt 설치가 중간에 멈추지 않게 한다.
require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

# KubeVirt Operator와 CR을 설치한다. VM/VMI 리소스는 이 단계가 끝난 뒤 사용할 수 있다.
main() {
  require_command kubectl

  if [[ -z "${KUBEVIRT_VERSION}" ]]; then
    require_command curl
    # 명시 버전이 없으면 upstream latest release tag를 가져와 operator/cr manifest URL에 사용한다.
    KUBEVIRT_VERSION="$(curl -fsSL https://api.github.com/repos/kubevirt/kubevirt/releases/latest | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  fi

  log "installing KubeVirt ${KUBEVIRT_VERSION}"
  # operator 설치 후 kubevirt-cr.yaml을 적용해야 virt-api/virt-controller 등이 생성된다.
  kubectl apply -f "https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-operator.yaml"
  kubectl apply -f "https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-cr.yaml"
}

main "$@"
