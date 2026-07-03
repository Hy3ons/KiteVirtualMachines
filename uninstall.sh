#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: uninstall.sh
# Purpose:
#   일반 사용자/운영자가 Kite 설치를 제거하는 공개 진입점이다. 기본값은 안전
#   삭제이며, 초반 질문 또는 env/preset으로 golden image, Longhorn host data,
#   Longhorn 설치 제거까지 확장할 수 있다.
#
# Usage:
#   ./uninstall.sh
#   curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/uninstall.sh | bash
#
# Required Commands:
#   checkout 실행: kubectl
#   curl 실행: curl, tar, mktemp, kubectl
#
# Environment Variables:
#   KITE_UNINSTALL_REPOSITORY: default Hy3ons/KiteVirtualMachines; curl 실행 시 받을 GitHub repository다. prompt 없음.
#   KITE_UNINSTALL_REF: default main; curl 실행 시 받을 branch/tag다. prompt 없음.
#   KITE_UNINSTALL_ARCHIVE_URL: default empty; 직접 archive URL을 지정할 때 쓴다. prompt 없음.
#   KITE_UNINSTALL_PRESET: default safe; safe 또는 full. full은 위험 삭제 선택값을 켠다.
#   KITE_ASSUME_DEFAULTS: default false; true면 모든 interactive 질문을 건너뛰고 env/default 값으로 실행한다.
#   DELETE_GOLDEN_IMAGE: default false; Ubuntu golden image DataVolume/PVC 삭제 여부를 초반에 묻는다.
#   DELETE_LONGHORN_DATA: default false; Kite Longhorn host data 삭제 여부를 초반에 묻는다.
#   DELETE_LONGHORN_DATA_CONFIRM: default false; host data 삭제 명시 확인값이다.
#   DELETE_LONGHORN: default false; Longhorn 설치 자체 제거 여부를 초반에 묻는다.
#   DELETE_LONGHORN_FORCE: default false; Longhorn PV가 남아도 강제 삭제할지 초반에 묻는다.
#   RESTORE_HOST_SSHD: default true; host sshd를 22번으로 복원할지 초반에 묻는다.
#   KITE_RESTORE_HOST_SSHD: default ask; restore worker 확인값이며 루트에서 초반 선택으로 true/false를 확정한다.
#   KITE_LOG_COLOR: default auto; 컬러 로그 여부다.
#   NO_COLOR: default empty; 설정하면 컬러 로그를 끈다.
#
# Interactive Behavior:
#   TTY에서 실행하고 env가 없는 항목은 삭제 초반에 모두 질문한다. 삭제 계획 요약을
#   보여준 뒤 하위 스크립트 실행 중에는 같은 항목을 다시 묻지 않는다.
#
# Noninteractive Behavior:
#   env가 있으면 그 값을 그대로 쓰고 질문하지 않는다. env가 없으면 safe 기본값으로
#   진행한다. KITE_UNINSTALL_PRESET=full은 위험 삭제 선택값을 켜지만
#   DELETE_LONGHORN_FORCE=false는 유지한다.
#
# Dangerous Options:
#   DELETE_LONGHORN_DATA, DELETE_LONGHORN, DELETE_LONGHORN_FORCE는 VM disk
#   infrastructure를 삭제할 수 있다. force 기본값은 항상 false다.
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

script_dir() {
  cd "$(dirname "${BASH_SOURCE[0]}")" && pwd
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

uninstall_from_checkout() {
  local root_dir="$1"
  shift

  exec "${root_dir}/build/deploy/scripts/clean.sh" "$@"
}

uninstall_from_remote_archive() {
  local url

  require_command curl
  require_command tar
  require_command mktemp

  trap cleanup EXIT
  KITE_UNINSTALL_TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/kite-uninstall.XXXXXX")"

  url="$(archive_url)"
  log "downloading Kite uninstaller from ${url}"
  curl -fsSL "${url}" | tar -xz --strip-components=1 -C "${KITE_UNINSTALL_TMPDIR}"

  log "running Kite uninstaller without git clone from ${KITE_UNINSTALL_REPOSITORY}@${KITE_UNINSTALL_REF}"
  "${KITE_UNINSTALL_TMPDIR}/build/deploy/scripts/clean.sh" "$@"
}

main() {
  local root_dir

  root_dir="$(script_dir)"
  if [[ -x "${root_dir}/build/deploy/scripts/clean.sh" ]]; then
    uninstall_from_checkout "${root_dir}" "$@"
    return
  fi

  uninstall_from_remote_archive "$@"
}

main "$@"
