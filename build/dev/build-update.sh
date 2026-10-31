#!/usr/bin/env bash
set -euo pipefail

# Fetch the requested release into a new temporary checkout, then build and update
# installed Kite containers. VM/storage/settings are never reinstalled here.
# KITE_UPDATE_REPOSITORY defaults to the official repository; KITE_UPDATE_REF
# selects a branch, tag or commit (default main). Requires git, Docker and K3s.
# Use KITE_UPDATE_REF=stage ./build/dev/build-update.sh for stage source builds.
KITE_UPDATE_REPOSITORY="${KITE_UPDATE_REPOSITORY:-https://github.com/Hy3ons/KiteVirtualMachines.git}"
KITE_UPDATE_REF="${KITE_UPDATE_REF:-main}"
UPDATE_BOOTSTRAP_DIR=""
if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
  UPDATE_BOOTSTRAP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi
UPDATE_SOURCE_TEMP=""

# cleanup_source removes only this invocation's disposable clone after execution.
cleanup_source() {
  if [[ -n "${UPDATE_SOURCE_TEMP}" ]]; then
    rm -rf "${UPDATE_SOURCE_TEMP}"
  fi
}

# main fetches a single immutable commit and invokes the source build updater.
# Arguments are forwarded to the updater; an installed checkout provides a
# bootstrap implementation when an older selected source lacks update-local.sh.
main() {
  local revision
  local executor

  command -v git >/dev/null || { echo '[kite-source-update] git is required' >&2; return 1; }
  UPDATE_SOURCE_TEMP="$(mktemp -d "${TMPDIR:-/tmp}/kite-source-update.XXXXXX")"
  trap cleanup_source EXIT
  git init --quiet "${UPDATE_SOURCE_TEMP}/source"
  git -C "${UPDATE_SOURCE_TEMP}/source" remote add origin "${KITE_UPDATE_REPOSITORY}"
  git -C "${UPDATE_SOURCE_TEMP}/source" fetch --quiet --depth=1 origin "${KITE_UPDATE_REF}"
  git -C "${UPDATE_SOURCE_TEMP}/source" checkout --quiet --detach FETCH_HEAD
  revision="$(git -C "${UPDATE_SOURCE_TEMP}/source" rev-parse HEAD)"
  printf '[kite-source-update] source ref=%s commit=%s\n' "${KITE_UPDATE_REF}" "${revision}"
  export KITE_UPDATE_SOURCE_DIR="${UPDATE_SOURCE_TEMP}/source"
  export KITE_UPDATE_SOURCE_REVISION="${revision}"
  executor="${KITE_UPDATE_SOURCE_DIR}/build/dev/update-local.sh"
  if [[ ! -x "${executor}" && -n "${UPDATE_BOOTSTRAP_DIR}" ]]; then
    executor="${UPDATE_BOOTSTRAP_DIR}/update-local.sh"
  fi
  if [[ ! -x "${executor}" ]]; then
    echo '[kite-source-update] selected source has no local updater; use a release containing update-local.sh or a current updater checkout' >&2
    return 1
  fi
  "${executor}" "$@"
}

main "$@"
