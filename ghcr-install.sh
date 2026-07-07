#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: ghcr-install.sh
# Purpose:
#   일반 사용자/운영자가 GHCR에 published 된 Kite 이미지를 pull해서 클러스터에
#   설치하는 공개 진입점이다.
#
# Usage:
#   ./ghcr-install.sh
#   curl -fsSL https://raw.githubusercontent.com/Hy3ons/KiteVirtualMachines/main/ghcr-install.sh | bash
#
# Required Commands:
#   checkout 실행: kubectl
#   curl 실행: curl, tar, mktemp, kubectl
#
# Environment Variables:
#   KITE_GHCR_INSTALL_REPOSITORY: default Hy3ons/KiteVirtualMachines; curl 실행 시 받을 GitHub repository다. prompt 없음.
#   KITE_GHCR_INSTALL_REF: default main; curl 실행 시 받을 branch/tag다. prompt 없음.
#   KITE_GHCR_INSTALL_ARCHIVE_URL: default empty; 직접 archive URL을 지정할 때 쓴다. prompt 없음.
#   KITE_INSTALL_REGISTRY: default ghcr.io/hy3ons; component image registry다. prompt 없음.
#   KITE_INSTALL_IMAGE_TAG: default production; 적용할 GHCR image tag다. prompt 없음.
#   KITE_INSTALL_IMAGE_PULL_POLICY: default IfNotPresent; Kite runtime Deployment imagePullPolicy다. prompt 없음.
#   KITE_ASSUME_DEFAULTS: default false; true면 모든 interactive 질문을 건너뛰고 env/default 값으로 실행한다.
#   INSTALL_LONGHORN: default true; Longhorn 기본 manifest를 설치할지 초반에 묻는다.
#   KITE_INSTALL_LONGHORN_HOST_PACKAGES: default true; apt 기반 host에서 Longhorn 필수 패키지를 자동 설치할지 정한다.
#   CONFIGURE_LONGHORN: default true; Longhorn disk/tag 설정을 적용할지 초반에 묻는다.
#   KITE_LONGHORN_USE_DEDICATED_DISK: default false; 전용 Longhorn host path disk를 만들지 초반에 묻는다.
#   APPLY_STORAGECLASS: default true; Kite VM StorageClass를 적용할지 초반에 묻는다.
#   INSTALL_KUBEVIRT: default true; KubeVirt 설치 여부를 초반에 묻는다.
#   INSTALL_CDI: default true; CDI 설치 여부를 초반에 묻는다.
#   APPLY_GOLDEN_IMAGE: default true; Ubuntu golden image DataVolume 적용 여부를 초반에 묻는다.
#   KITE_GATEWAY_HOST_KEY_REFRESH: default false; 기존 gateway host key Secret 갱신 여부를 초반에 묻는다.
#   RUN_VERIFY: default true; 설치 후 verify 실행 여부를 초반에 묻는다.
#   KITE_LOG_COLOR: default auto; 컬러 로그 여부다.
#   NO_COLOR: default empty; 설정하면 컬러 로그를 끈다.
#
# Interactive Behavior:
#   TTY에서 실행하고 env가 없는 항목은 설치 초반에 모두 질문한다.
#   하위 스크립트 실행 중에는 같은 항목을 다시 묻지 않도록 env를 export한다.
#
# Noninteractive Behavior:
#   env가 있으면 그 값을 그대로 쓰고 질문하지 않는다. env가 없으면 위 기본값으로
#   진행한다. 일반 k8s/k3s 의존성 준비 실패는 하위 스크립트가 명확히 중단한다.
#
# Dangerous Options:
#   Longhorn/KubeVirt/CDI 설치와 Kubernetes 리소스 적용이 클러스터 상태를 변경한다.
#
# Side Effects:
#   Kubernetes 리소스 적용과 dependency 설치를 수행할 수 있다.
# ==============================================================================

KITE_GHCR_INSTALL_REPOSITORY="${KITE_GHCR_INSTALL_REPOSITORY:-Hy3ons/KiteVirtualMachines}"
KITE_GHCR_INSTALL_REF="${KITE_GHCR_INSTALL_REF:-main}"
KITE_GHCR_INSTALL_ARCHIVE_URL="${KITE_GHCR_INSTALL_ARCHIVE_URL:-}"
KITE_GHCR_INSTALL_TMPDIR=""

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
    printf "\033[0;32m[kite-ghcr-install] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-ghcr-install] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-ghcr-install] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-ghcr-install] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
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
  if [[ -n "${KITE_GHCR_INSTALL_ARCHIVE_URL}" ]]; then
    printf '%s\n' "${KITE_GHCR_INSTALL_ARCHIVE_URL}"
    return
  fi

  printf 'https://github.com/%s/archive/%s.tar.gz\n' "${KITE_GHCR_INSTALL_REPOSITORY}" "${KITE_GHCR_INSTALL_REF}"
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
  KITE_GHCR_INSTALL_TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/kite-ghcr-install.XXXXXX")"

  url="$(archive_url)"
  log "downloading Kite installer from ${url}"
  curl -fsSL "${url}" | tar -xz --strip-components=1 -C "${KITE_GHCR_INSTALL_TMPDIR}"

  log "running pull-based installer without git clone from ${KITE_GHCR_INSTALL_REPOSITORY}@${KITE_GHCR_INSTALL_REF}"
  "${KITE_GHCR_INSTALL_TMPDIR}/build/deploy/scripts/install-all.sh" "$@"
}

# @description curl 설치 중 만든 임시 디렉터리를 성공/실패와 관계없이 정리한다.
cleanup() {
  if [[ -n "${KITE_GHCR_INSTALL_TMPDIR:-}" ]]; then
    rm -rf "${KITE_GHCR_INSTALL_TMPDIR}"
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
