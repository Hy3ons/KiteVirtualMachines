#!/usr/bin/env bash
set -euo pipefail

# install.sh is the pull-based installer.
# It installs the virtualization/storage stack and applies Kite manifests that pull
# prebuilt images from GHCR, without building local Docker images.
# No git binary or git clone is required. When run through curl, this script
# downloads a GitHub release branch/tag archive with curl+tar into a temporary
# directory and runs the real installer from there.

KITE_INSTALL_REPOSITORY="${KITE_INSTALL_REPOSITORY:-Hy3ons/KiteVirtualMachines}"
KITE_INSTALL_REF="${KITE_INSTALL_REF:-main}"
KITE_INSTALL_ARCHIVE_URL="${KITE_INSTALL_ARCHIVE_URL:-}"
KITE_INSTALL_TMPDIR=""

log() {
  echo "[kite-install] $*"
}

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "[kite-install] missing required command: ${name}" >&2
    exit 1
  fi
}

script_dir() {
  cd "$(dirname "${BASH_SOURCE[0]}")" && pwd
}

install_from_checkout() {
  local root_dir="$1"
  shift

  exec "${root_dir}/build/deploy/scripts/install-all.sh" "$@"
}

archive_url() {
  if [[ -n "${KITE_INSTALL_ARCHIVE_URL}" ]]; then
    printf '%s\n' "${KITE_INSTALL_ARCHIVE_URL}"
    return
  fi

  printf 'https://github.com/%s/archive/%s.tar.gz\n' "${KITE_INSTALL_REPOSITORY}" "${KITE_INSTALL_REF}"
}

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

cleanup() {
  if [[ -n "${KITE_INSTALL_TMPDIR}" ]]; then
    rm -rf "${KITE_INSTALL_TMPDIR}"
  fi
}

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
