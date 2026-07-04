#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: ghcr-update.sh
# Purpose:
#   일반 사용자/운영자가 이미 설치된 Kite를 GHCR image 기준으로 업데이트하는
#   공개 진입점이다. KiteUser, KiteVirtualMachine, VM disk, Longhorn/KubeVirt/CDI
#   공유 인프라는 삭제하지 않는다.
#
# Usage:
#   ./ghcr-update.sh
#   curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/ghcr-update.sh | bash
#
# Required Commands:
#   checkout 실행: kubectl, curl
#   curl 실행: curl, tar, mktemp, kubectl
#
# Environment Variables:
#   KITE_GHCR_UPDATE_REPOSITORY: default Hy3ons/KiteVirtualMachines; curl 실행 시 받을 GitHub repository다. prompt 없음.
#   KITE_GHCR_UPDATE_REF: default main; curl 실행 시 받을 branch/tag다. prompt 없음.
#   KITE_GHCR_UPDATE_ARCHIVE_URL: default empty; 직접 archive URL을 지정할 때 쓴다. prompt 없음.
#   KITE_UPDATE_REGISTRY: default ghcr.io/hy3ons; component image registry다.
#   KITE_UPDATE_TAG: default latest; 적용할 GHCR image tag다. interactive에서 초반에 묻는다.
#   KITE_UPDATE_COMPONENTS: default api,controller,gateway,frontend; 업데이트할 component 목록이다.
#   KITE_UPDATE_APPLY_CRDS: default true; CRD manifest 갱신 여부다.
#   KITE_UPDATE_APPLY_RBAC: default true; ServiceAccount/RBAC manifest 갱신 여부다.
#   KITE_UPDATE_WAIT: default true; rollout 대기 여부다.
#   KITE_UPDATE_HEALTH_CHECK: default true; API/frontend smoke check 여부다.
#   KITE_UPDATE_RUN_VERIFY: default false; 기존 deploy verify.sh 실행 여부다.
#   KITE_UPDATE_ROLLBACK_ON_FAIL: default true; rollout/smoke 실패 시 이전 image rollback 여부다.
#   KITE_UPDATE_DRY_RUN: default false; true면 적용 계획과 실행할 명령만 출력한다.
#   KITE_ASSUME_DEFAULTS: default false; true면 모든 interactive 질문을 건너뛰고 env/default 값으로 실행한다.
#   KITE_LOG_COLOR: default auto; 컬러 로그 여부다.
#   NO_COLOR: default empty; 설정하면 컬러 로그를 끈다.
#
# Interactive Behavior:
#   TTY에서 실행하고 env가 없는 항목은 업데이트 초반에 모두 질문한다.
#
# Noninteractive Behavior:
#   env가 있으면 그 값을 그대로 쓰고 질문하지 않는다. env가 없으면 안전 기본값으로
#   진행한다. 위험한 삭제 동작은 없다.
#
# Dangerous Options:
#   없음. 이 스크립트는 Kite runtime image와 manifest를 갱신하며 사용자 VM/PVC,
#   Longhorn/KubeVirt/CDI 설치 자체, host sshd 설정을 삭제하거나 이동하지 않는다.
#
# Side Effects:
#   Kubernetes manifest apply, Deployment image 교체, rollout 대기, 실패 시 image rollback을 수행할 수 있다.
# ==============================================================================

KITE_GHCR_UPDATE_REPOSITORY="${KITE_GHCR_UPDATE_REPOSITORY:-Hy3ons/KiteVirtualMachines}"
KITE_GHCR_UPDATE_REF="${KITE_GHCR_UPDATE_REF:-main}"
KITE_GHCR_UPDATE_ARCHIVE_URL="${KITE_GHCR_UPDATE_ARCHIVE_URL:-}"
KITE_GHCR_UPDATE_TMPDIR=""

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
    printf "\033[0;32m[kite-ghcr-update] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-ghcr-update] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-ghcr-update] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-ghcr-update] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
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

update_from_checkout() {
  local root_dir="$1"
  shift

  exec "${root_dir}/build/deploy/scripts/update-ghcr.sh" "$@"
}

archive_url() {
  if [[ -n "${KITE_GHCR_UPDATE_ARCHIVE_URL}" ]]; then
    printf '%s\n' "${KITE_GHCR_UPDATE_ARCHIVE_URL}"
    return
  fi

  printf 'https://github.com/%s/archive/%s.tar.gz\n' "${KITE_GHCR_UPDATE_REPOSITORY}" "${KITE_GHCR_UPDATE_REF}"
}

update_from_remote_archive() {
  local url

  require_command curl
  require_command tar
  require_command mktemp

  trap cleanup EXIT
  KITE_GHCR_UPDATE_TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/kite-ghcr-update.XXXXXX")"

  url="$(archive_url)"
  log "downloading Kite updater from ${url}"
  curl -fsSL "${url}" | tar -xz --strip-components=1 -C "${KITE_GHCR_UPDATE_TMPDIR}"

  log "running GHCR updater from ${KITE_GHCR_UPDATE_REPOSITORY}@${KITE_GHCR_UPDATE_REF}"
  "${KITE_GHCR_UPDATE_TMPDIR}/build/deploy/scripts/update-ghcr.sh" "$@"
}

cleanup() {
  if [[ -n "${KITE_GHCR_UPDATE_TMPDIR:-}" ]]; then
    rm -rf "${KITE_GHCR_UPDATE_TMPDIR}"
  fi
}

main() {
  local root_dir

  root_dir="$(script_dir)"
  if [[ -x "${root_dir}/build/deploy/scripts/update-ghcr.sh" ]]; then
    update_from_checkout "${root_dir}" "$@"
    return
  fi

  update_from_remote_archive "$@"
}

main "$@"
