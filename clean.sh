#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: clean.sh
# Description: curl 또는 checkout에서 Kite 정리를 시작한다. 실제 삭제 로직은 build/dev/clear.sh로 위임한다.
#
# Usage:
#   ./clean.sh
#
# Environment Variables:
#   KITE_CLEAN_REPOSITORY: default Hy3ons/KiteVirtualMachines
#   KITE_CLEAN_REF: default main
#   KITE_CLEAN_ARCHIVE_URL: default (empty)
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   Kubernetes 리소스, 이미지 캐시, 선택적 Longhorn/host sshd 상태를 변경하거나 삭제할 수 있다.
# ==============================================================================

KITE_CLEAN_REPOSITORY="${KITE_CLEAN_REPOSITORY:-Hy3ons/KiteVirtualMachines}"
KITE_CLEAN_REF="${KITE_CLEAN_REF:-main}"
KITE_CLEAN_ARCHIVE_URL="${KITE_CLEAN_ARCHIVE_URL:-}"
KITE_CLEAN_TMPDIR=""

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
    printf "\033[0;32m[kite-clean] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-clean] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-clean] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-clean] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}


# curl/tar/mktemp 같은 필수 CLI를 미리 확인해 다운로드 중간 실패를 명확히 만든다.
require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

# checkout 안에서 실행 중인지 판단하기 위해 현재 스크립트 위치를 구한다.
script_dir() {
  cd "$(dirname "${BASH_SOURCE[0]}")" && pwd
}

# 기본값은 GitHub archive URL이고, 테스트/미러 환경은 KITE_CLEAN_ARCHIVE_URL로 덮어쓴다.
archive_url() {
  if [[ -n "${KITE_CLEAN_ARCHIVE_URL}" ]]; then
    printf '%s\n' "${KITE_CLEAN_ARCHIVE_URL}"
    return
  fi

  printf 'https://github.com/%s/archive/%s.tar.gz\n' "${KITE_CLEAN_REPOSITORY}" "${KITE_CLEAN_REF}"
}

# curl 정리 경로에서 만든 임시 디렉터리를 종료 시 제거한다.
cleanup() {
  if [[ -n "${KITE_CLEAN_TMPDIR:-}" ]]; then
    rm -rf "${KITE_CLEAN_TMPDIR}"
  fi
}

# checkout 안이면 로컬 build/dev/clear.sh를 그대로 실행한다.
clean_from_checkout() {
  local root_dir="$1"
  shift

  exec "${root_dir}/build/dev/clear.sh" "$@"
}

# git clone이 없는 환경에서는 GitHub 압축본을 내려받아 그 안의 clear.sh를 실행한다.
clean_from_remote_archive() {
  local url

  require_command curl
  require_command tar
  require_command mktemp

  trap cleanup EXIT
  KITE_CLEAN_TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/kite-clean.XXXXXX")"

  url="$(archive_url)"
  log "downloading Kite cleaner from ${url}"
  curl -fsSL "${url}" | tar -xz --strip-components=1 -C "${KITE_CLEAN_TMPDIR}"

  log "running Kite cleaner without git clone from ${KITE_CLEAN_REPOSITORY}@${KITE_CLEAN_REF}"
  "${KITE_CLEAN_TMPDIR}/build/dev/clear.sh" "$@"
}

# checkout 실행과 curl 실행을 구분하는 최상위 dispatcher다.
main() {
  local root_dir

  root_dir="$(script_dir)"
  if [[ -x "${root_dir}/build/dev/clear.sh" ]]; then
    clean_from_checkout "${root_dir}" "$@"
    return
  fi

  clean_from_remote_archive "$@"
}

main "$@"
