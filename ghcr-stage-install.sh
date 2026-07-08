#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: ghcr-stage-install.sh
# Purpose:
#   Maintainer QA용 stage GHCR 설치 진입점이다. 일반 사용자용 ghcr-install.sh와
#   같은 설치 흐름을 사용하되, GitHub archive ref와 image tag를 stage로 고정한다.
#
# Usage:
#   ./ghcr-stage-install.sh
#   curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/stage/ghcr-stage-install.sh | bash
#
# Environment Variables:
#   KITE_GHCR_INSTALL_REPOSITORY: default Hy3ons/KiteVirtualMachines
#   KITE_GHCR_INSTALL_REF: default stage
#   KITE_INSTALL_REGISTRY: default ghcr.io/hy3ons
#   KITE_INSTALL_IMAGE_TAG: default stage
#   KITE_INSTALL_IMAGE_PULL_POLICY: default Always
#   KITE_INSTALL_FORCE_ROLLOUT: default true
#   KITE_ASSUME_DEFAULTS: default false; true면 ghcr-install.sh 하위 질문도 기본값/env로 진행한다.
#
# Side Effects:
#   ghcr-install.sh와 동일하게 Kubernetes dependency와 Kite runtime 리소스를 적용할 수 있다.
# ==============================================================================

KITE_GHCR_INSTALL_REPOSITORY="${KITE_GHCR_INSTALL_REPOSITORY:-Hy3ons/KiteVirtualMachines}"
KITE_GHCR_INSTALL_REF="${KITE_GHCR_INSTALL_REF:-stage}"
KITE_INSTALL_REGISTRY="${KITE_INSTALL_REGISTRY:-ghcr.io/hy3ons}"
KITE_INSTALL_IMAGE_TAG="${KITE_INSTALL_IMAGE_TAG:-stage}"
KITE_INSTALL_IMAGE_PULL_POLICY="${KITE_INSTALL_IMAGE_PULL_POLICY:-Always}"
KITE_INSTALL_FORCE_ROLLOUT="${KITE_INSTALL_FORCE_ROLLOUT:-true}"

export KITE_GHCR_INSTALL_REPOSITORY
export KITE_GHCR_INSTALL_REF
export KITE_INSTALL_REGISTRY
export KITE_INSTALL_IMAGE_TAG
export KITE_INSTALL_IMAGE_PULL_POLICY
export KITE_INSTALL_FORCE_ROLLOUT

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
    printf "\033[0;32m[kite-ghcr-stage-install] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-ghcr-stage-install] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-ghcr-stage-install] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-ghcr-stage-install] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

script_dir() {
  cd "$(dirname "${BASH_SOURCE[0]}")" && pwd
}

run_from_checkout_if_possible() {
  local root_dir

  root_dir="$(script_dir)"
  if [[ -x "${root_dir}/ghcr-install.sh" ]]; then
    log "running stage install through local ghcr-install.sh"
    exec "${root_dir}/ghcr-install.sh" "$@"
  fi
}

run_from_remote_raw() {
  local url

  require_command curl
  url="https://raw.githubusercontent.com/${KITE_GHCR_INSTALL_REPOSITORY}/${KITE_GHCR_INSTALL_REF}/ghcr-install.sh"
  log "downloading stage ghcr-install.sh from ${url}"
  curl -fsSL "${url}" | bash -s -- "$@"
}

main() {
  log "stage install defaults: ref=${KITE_GHCR_INSTALL_REF}, image=${KITE_INSTALL_REGISTRY}/<component>:${KITE_INSTALL_IMAGE_TAG}"
  run_from_checkout_if_possible "$@"
  run_from_remote_raw "$@"
}

main "$@"
