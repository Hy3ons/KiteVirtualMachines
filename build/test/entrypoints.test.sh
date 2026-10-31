#!/usr/bin/env bash
set -euo pipefail

# Exercise public and developer bootstrap paths using disposable executor fixtures.
# No fixture runs Kubernetes commands, builds images, or removes installed data.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
fixture="$(mktemp -d "${TMPDIR:-/tmp}/kite-entrypoints.XXXXXX")"
trap 'rm -rf "${fixture}"' EXIT
checkout="${fixture}/checkout with spaces"
mkdir -p "${checkout}/build/deploy/scripts" "${checkout}/build/dev" "${fixture}/outside" "${fixture}/tmp" "${fixture}/bin"
export ENTRYPOINT_RESULT="${fixture}/result"
export TMPDIR="${fixture}/tmp"
export KITE_ASSUME_DEFAULTS=true
export NO_COLOR=1

# write_executor writes a harmless child that records arguments and routing env.
# $1 selects the executor path and $2 identifies the expected delegation target.
write_executor() {
  local target="$1" name="$2"
  printf '#!/usr/bin/env bash\nset -euo pipefail\nprintf "%%s\\n" %q > "${ENTRYPOINT_RESULT}"\n' "${name}" > "${target}"
  cat >> "${target}" <<'CHILD'
printf 'arg=<%s>\n' "$@" >> "${ENTRYPOINT_RESULT}"
printf 'stage=%s|%s|%s|%s\n' "${KITE_GHCR_INSTALL_REF:-}" "${KITE_INSTALL_IMAGE_TAG:-}" "${KITE_INSTALL_IMAGE_PULL_POLICY:-}" "${KITE_INSTALL_FORCE_ROLLOUT:-}" >> "${ENTRYPOINT_RESULT}"
printf 'source=%s\nrevision=%s\n' "${KITE_UPDATE_SOURCE_DIR:-}" "${KITE_UPDATE_SOURCE_REVISION:-}" >> "${ENTRYPOINT_RESULT}"
CHILD
  chmod +x "${target}"
}

# expect_result fails if the delegated target or an argument was lost in transit.
expect_result() {
  local name="$1"
  [[ "$(head -n 1 "${ENTRYPOINT_RESULT}")" == "${name}" ]]
  grep -Fx 'arg=<two words>' "${ENTRYPOINT_RESULT}" >/dev/null
}

# expect_quiet_bootstrap rejects shell warnings hidden by a successful child.
expect_quiet_bootstrap() {
  if [[ -s "${fixture}/stream-stderr" ]]; then
    cat "${fixture}/stream-stderr" >&2
    exit 1
  fi
}

# expect_clean rejects leaked downloads and disposable source clones.
expect_clean() {
  [[ -z "$(find "${TMPDIR}" -mindepth 1 -print -quit)" ]]
}

cp "${ROOT_DIR}/install.sh" "${ROOT_DIR}/update.sh" "${ROOT_DIR}/uninstall.sh" "${checkout}/"
for name in build-install build-clear build-update stage-install; do
  cp "${ROOT_DIR}/build/dev/${name}.sh" "${checkout}/build/dev/"
done
write_executor "${checkout}/build/deploy/scripts/install-all.sh" install
write_executor "${checkout}/build/deploy/scripts/update-ghcr.sh" update
write_executor "${checkout}/build/deploy/scripts/clean.sh" uninstall
write_executor "${checkout}/build/dev/all-in-one.sh" build-install
write_executor "${checkout}/build/dev/clear.sh" build-clear
write_executor "${checkout}/build/dev/update-local.sh" fallback-source

expected_root_scripts=$'install.sh\nuninstall.sh\nupdate.sh'
[[ "$(find "${ROOT_DIR}" -maxdepth 1 -type f -name '*.sh' -exec basename {} \; | LC_ALL=C sort)" == "${expected_root_scripts}" ]]
printf 'PASS root contains only the three public shell entrypoints\n'
cd "${fixture}/outside"
for name in install update uninstall; do
  bash "${checkout}/${name}.sh" 'two words'
  expect_result "${name}"
  printf 'PASS local %s from unrelated cwd with spaces\n' "${name}"
done
for name in build-install build-clear; do
  bash "${checkout}/build/dev/${name}.sh" 'two words'
  expect_result "${name}"
  printf 'PASS moved developer %s from unrelated cwd\n' "${name}"
done
bash "${checkout}/build/dev/stage-install.sh" 'two words'
expect_result install
grep -Fx 'stage=stage|stage|Always|true' "${ENTRYPOINT_RESULT}" >/dev/null
printf 'PASS moved stage installer delegates to root with stage defaults\n'

# A real tar archive and file:// curl exercise download, extraction and cleanup.
tar -czf "${fixture}/release.tar.gz" -C "${fixture}" 'checkout with spaces'
export KITE_GHCR_INSTALL_ARCHIVE_URL="file://${fixture}/release.tar.gz"
export KITE_GHCR_UPDATE_ARCHIVE_URL="${KITE_GHCR_INSTALL_ARCHIVE_URL}"
export KITE_UNINSTALL_ARCHIVE_URL="${KITE_GHCR_INSTALL_ARCHIVE_URL}"
for name in install update uninstall; do
  bash -s -- 'two words' < "${ROOT_DIR}/${name}.sh" 2> "${fixture}/stream-stderr"
  expect_quiet_bootstrap
  expect_result "${name}"
  expect_clean
  printf 'PASS streamed %s downloads archive and cleans up\n' "${name}"
done

# Only the stage raw bootstrap request is intercepted; archive curl stays real.
export ENTRYPOINT_REAL_CURL="$(command -v curl)"
export ENTRYPOINT_INSTALL_SCRIPT="${ROOT_DIR}/install.sh"
export ENTRYPOINT_RAW_REQUEST="${fixture}/raw-request"
cat > "${fixture}/bin/curl" <<'CURL'
#!/usr/bin/env bash
set -euo pipefail
for arg in "$@"; do
  if [[ "${arg}" == https://raw.githubusercontent.com/* ]]; then
    printf '%s\n' "${arg}" > "${ENTRYPOINT_RAW_REQUEST}"
    cat "${ENTRYPOINT_INSTALL_SCRIPT}"
    exit
  fi
done
exec "${ENTRYPOINT_REAL_CURL}" "$@"
CURL
chmod +x "${fixture}/bin/curl"
PATH="${fixture}/bin:${PATH}" bash -s -- 'two words' < "${ROOT_DIR}/build/dev/stage-install.sh" 2> "${fixture}/stream-stderr"
expect_quiet_bootstrap
expect_result install
grep -Fx 'https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/stage/install.sh' "${ENTRYPOINT_RAW_REQUEST}" >/dev/null
grep -Fx 'stage=stage|stage|Always|true' "${ENTRYPOINT_RESULT}" >/dev/null
expect_clean
printf 'PASS streamed stage installer uses new root raw URL\n'

# Fake only Git transport; execute the real source bootstrap and child selection.
export ENTRYPOINT_FETCHED_EXECUTOR="${fixture}/source-executor"
write_executor "${ENTRYPOINT_FETCHED_EXECUTOR}" fetched-source
cat > "${fixture}/bin/git" <<'GIT'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" == init ]]; then mkdir -p "$3"; exit; fi
[[ "$1" == -C ]]
source_dir="$2"
shift 2
case "$1" in
  remote|fetch) ;;
  checkout)
    if [[ "${ENTRYPOINT_OLD_SOURCE:-false}" != true ]]; then
      mkdir -p "${source_dir}/build/dev"
      cp "${ENTRYPOINT_FETCHED_EXECUTOR}" "${source_dir}/build/dev/update-local.sh"
    fi
    ;;
  rev-parse) printf '0123456789abcdef\n' ;;
  *) exit 2 ;;
esac
GIT
chmod +x "${fixture}/bin/git"
PATH="${fixture}/bin:${PATH}" bash "${checkout}/build/dev/build-update.sh" 'two words'
expect_result fetched-source
grep -Fx 'revision=0123456789abcdef' "${ENTRYPOINT_RESULT}" >/dev/null
expect_clean
printf 'PASS source update prefers fetched build/dev executor\n'
PATH="${fixture}/bin:${PATH}" ENTRYPOINT_OLD_SOURCE=true bash "${checkout}/build/dev/build-update.sh" 'two words'
expect_result fallback-source
expect_clean
printf 'PASS older source uses moved local fallback executor\n'
PATH="${fixture}/bin:${PATH}" bash -s -- 'two words' < "${ROOT_DIR}/build/dev/build-update.sh" 2> "${fixture}/stream-stderr"
expect_quiet_bootstrap
expect_result fetched-source
expect_clean
printf 'PASS streamed source updater finds fetched executor\n'
rm -f "${ENTRYPOINT_RESULT}"
if PATH="${fixture}/bin:${PATH}" ENTRYPOINT_OLD_SOURCE=true bash -s -- 'two words' < "${ROOT_DIR}/build/dev/build-update.sh"; then
  echo 'FAIL missing source executor returned success' >&2
  exit 1
fi
[[ ! -e "${ENTRYPOINT_RESULT}" ]]
expect_clean
printf 'PASS missing executor fails and cleans disposable clone\n'

rm -f "${ENTRYPOINT_RESULT}"
if KITE_GHCR_UPDATE_ARCHIVE_URL="file://${fixture}/missing.tar.gz" bash -s < "${ROOT_DIR}/update.sh"; then
  echo 'FAIL missing archive returned success' >&2
  exit 1
fi
[[ ! -e "${ENTRYPOINT_RESULT}" ]]
expect_clean
printf 'PASS failed download never delegates and cleans temporary files\n'
