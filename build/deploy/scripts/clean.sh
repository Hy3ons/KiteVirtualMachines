#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/clean.sh
# Description: checkout 또는 curl 경로에서 Kite uninstall을 시작한다.
#
# Usage:
#   build/deploy/scripts/clean.sh
#
# Environment Variables:
#   KITE_UNINSTALL_REPOSITORY: default Hy3ons/KiteVirtualMachines
#   KITE_UNINSTALL_REF: default main
#   KITE_UNINSTALL_ARCHIVE_URL: default (empty)
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   Kubernetes 리소스, 선택적 Longhorn/host sshd 상태를 변경하거나 삭제할 수 있다.
# ==============================================================================

KITE_UNINSTALL_REPOSITORY="${KITE_UNINSTALL_REPOSITORY:-Hy3ons/KiteVirtualMachines}"
KITE_UNINSTALL_REF="${KITE_UNINSTALL_REF:-main}"
KITE_UNINSTALL_ARCHIVE_URL="${KITE_UNINSTALL_ARCHIVE_URL:-}"
KITE_UNINSTALL_TMPDIR=""

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
    printf "\033[0;32m[kite-uninstall] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-uninstall] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-uninstall] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-uninstall] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

script_root() {
  cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd
}

archive_url() {
  if [[ -n "${KITE_UNINSTALL_ARCHIVE_URL}" ]]; then
    printf '%s\n' "${KITE_UNINSTALL_ARCHIVE_URL}"
    return
  fi

  printf 'https://github.com/%s/archive/%s.tar.gz\n' "${KITE_UNINSTALL_REPOSITORY}" "${KITE_UNINSTALL_REF}"
}

cleanup() {
  if [[ -n "${KITE_UNINSTALL_TMPDIR:-}" ]]; then
    rm -rf "${KITE_UNINSTALL_TMPDIR}"
  fi
}

clean_from_checkout() {
  local root_dir="$1"
  shift

  exec "${root_dir}/build/deploy/scripts/uninstall-kite.sh" "$@"
}

clean_from_remote_archive() {
  local url

  require_command curl
  require_command tar
  require_command mktemp

  trap cleanup EXIT
  KITE_UNINSTALL_TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/kite-uninstall.XXXXXX")"

  url="$(archive_url)"
  log "downloading Kite deploy cleaner from ${url}"
  curl -fsSL "${url}" | tar -xz --strip-components=1 -C "${KITE_UNINSTALL_TMPDIR}"

  log "running Kite uninstaller without git clone from ${KITE_UNINSTALL_REPOSITORY}@${KITE_UNINSTALL_REF}"
  "${KITE_UNINSTALL_TMPDIR}/build/deploy/scripts/uninstall-kite.sh" "$@"
}

main() {
  local root_dir

  root_dir="$(script_root)"
  if [[ -x "${root_dir}/build/deploy/scripts/uninstall-kite.sh" ]]; then
    clean_from_checkout "${root_dir}" "$@"
    return
  fi

  clean_from_remote_archive "$@"
}

main "$@"
