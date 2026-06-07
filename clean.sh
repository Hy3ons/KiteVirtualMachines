#!/usr/bin/env bash
set -euo pipefail

# clean.sh removes Kite resources without requiring git or a repository clone.
# When run from a checkout it uses local files. When run through curl, it downloads
# the selected GitHub branch/tag archive with curl+tar and runs build/dev/clear.sh.

KITE_CLEAN_REPOSITORY="${KITE_CLEAN_REPOSITORY:-Hy3ons/KiteVirtualMachines}"
KITE_CLEAN_REF="${KITE_CLEAN_REF:-main}"
KITE_CLEAN_ARCHIVE_URL="${KITE_CLEAN_ARCHIVE_URL:-}"
KITE_CLEAN_TMPDIR=""

log() {
  echo "[kite-clean] $*"
}

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "[kite-clean] missing required command: ${name}" >&2
    exit 1
  fi
}

script_dir() {
  cd "$(dirname "${BASH_SOURCE[0]}")" && pwd
}

archive_url() {
  if [[ -n "${KITE_CLEAN_ARCHIVE_URL}" ]]; then
    printf '%s\n' "${KITE_CLEAN_ARCHIVE_URL}"
    return
  fi

  printf 'https://github.com/%s/archive/%s.tar.gz\n' "${KITE_CLEAN_REPOSITORY}" "${KITE_CLEAN_REF}"
}

cleanup() {
  if [[ -n "${KITE_CLEAN_TMPDIR}" ]]; then
    rm -rf "${KITE_CLEAN_TMPDIR}"
  fi
}

clean_from_checkout() {
  local root_dir="$1"
  shift

  exec "${root_dir}/build/dev/clear.sh" "$@"
}

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
