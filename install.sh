#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: install.sh
# Description: curl 또는 checkout에서 pull 기반 Kite 설치를 시작한다. GHCR 이미지를 사용하는 배포 스크립트로 위임한다.
#
# Usage:
#   ./install.sh
#
# Environment Variables:
#   KITE_INSTALL_REPOSITORY: default Hy3ons/KiteVirtualMachines
#   KITE_INSTALL_REF: default main
#   KITE_INSTALL_ARCHIVE_URL: default (empty)
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   Kubernetes 리소스 적용, dependency 설치, host sshd handoff를 수행할 수 있다.
# ==============================================================================

KITE_INSTALL_REPOSITORY="${KITE_INSTALL_REPOSITORY:-Hy3ons/KiteVirtualMachines}"
KITE_INSTALL_REF="${KITE_INSTALL_REF:-main}"
KITE_INSTALL_ARCHIVE_URL="${KITE_INSTALL_ARCHIVE_URL:-}"
KITE_INSTALL_TMPDIR=""

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
    printf "\033[0;32m[kite-install] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-install] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-install] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-install] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}

# @description 필수 CLI가 없으면 중간 단계에서 애매하게 실패하지 않도록 초기에 중단한다.
# @param $1 검사할 명령어 이름이다.
# @exitcode 1 명령어가 없을 경우 종료한다.
require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

# @description 현재 스크립트가 checkout 안에서 실행 중인지 판단하기 위해 자신의 위치를 구한다.
# @stdout 절대 경로 형태의 스크립트 디렉터리를 출력한다.
script_dir() {
  cd "$(dirname "${BASH_SOURCE[0]}")" && pwd
}

# @description 이미 checkout 안이면 임시 다운로드 없이 repo 안의 실제 설치 스크립트로 교체 실행한다.
# @param $1 repository root 디렉터리다.
# @param $@ 하위 install-all.sh로 그대로 전달할 인자다.
install_from_checkout() {
  local root_dir="$1"
  shift

  exec "${root_dir}/build/deploy/scripts/install-all.sh" "$@"
}

# @description 기본 GitHub archive URL을 만들거나, 명시된 archive URL을 그대로 반환한다.
# @stdout 다운로드할 tar.gz archive URL을 출력한다.
archive_url() {
  if [[ -n "${KITE_INSTALL_ARCHIVE_URL}" ]]; then
    printf '%s\n' "${KITE_INSTALL_ARCHIVE_URL}"
    return
  fi

  printf 'https://github.com/%s/archive/%s.tar.gz\n' "${KITE_INSTALL_REPOSITORY}" "${KITE_INSTALL_REF}"
}

# @description curl 실행 경로에서 archive를 임시 디렉터리에 풀고 pull 기반 설치를 이어간다.
# @param $@ build/deploy/scripts/install-all.sh로 그대로 전달할 인자다.
# @sideeffect 임시 디렉터리를 만들고 GitHub archive를 다운로드한다.
install_from_remote_archive() {
  local url

  require_command curl
  require_command tar
  require_command mktemp

  trap cleanup EXIT
  KITE_INSTALL_TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/kite-install.XXXXXX")"

  url="$(archive_url)"
  log "downloading Kite installer from ${url}"
  curl -fsSL "${url}" | tar -xz --strip-components=1 -C "${KITE_INSTALL_TMPDIR}"

  log "running pull-based installer without git clone from ${KITE_INSTALL_REPOSITORY}@${KITE_INSTALL_REF}"
  "${KITE_INSTALL_TMPDIR}/build/deploy/scripts/install-all.sh" "$@"
}

# @description curl 설치 중 만든 임시 디렉터리를 성공/실패와 관계없이 정리한다.
cleanup() {
  if [[ -n "${KITE_INSTALL_TMPDIR:-}" ]]; then
    rm -rf "${KITE_INSTALL_TMPDIR}"
  fi
}

# @description checkout 실행과 curl 실행을 구분하는 최상위 dispatcher다.
# @param $@ 하위 설치 스크립트로 그대로 전달할 인자다.
main() {
  local root_dir

  root_dir="$(script_dir)"
  if [[ -x "${root_dir}/build/deploy/scripts/install-all.sh" ]]; then
    install_from_checkout "${root_dir}" "$@"
    return
  fi

  install_from_remote_archive "$@"
}

main "$@"
