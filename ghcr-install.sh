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
#   KITE_ASSUME_DEFAULTS: default false; true면 모든 interactive 질문을 건너뛰고 env/default 값으로 실행한다.
#   MANAGE_HOST_SSHD: default true; gateway가 22번을 쓰도록 host sshd handoff를 수행할지 초반에 묻는다.
#   KITE_HOST_SSHD_PORT: default 2222; host sshd handoff 대상 포트다. 원격 curl 실행에서도 초반에 한 번 묻는다.
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
#   TTY에서 실행하고 env가 없는 항목은 설치 초반에 모두 질문한다. curl pipe 실행은
#   stdin이 pipe이므로 /dev/tty가 있으면 host sshd 이동 포트만 bootstrap에서 한 번 묻는다.
#   하위 스크립트 실행 중에는 같은 항목을 다시 묻지 않도록 env를 export한다.
#
# Noninteractive Behavior:
#   env가 있으면 그 값을 그대로 쓰고 질문하지 않는다. env가 없으면 위 기본값으로
#   진행한다. 일반 k8s/k3s 의존성 준비 실패는 하위 스크립트가 명확히 중단한다.
#
# Dangerous Options:
#   MANAGE_HOST_SSHD=true가 기본값이며 host sshd 포트를 바꿀 수 있다. KITE_HOST_SSHD_PORT는
#   적용 전 점유 확인을 거친다.
#
# Side Effects:
#   Kubernetes 리소스 적용, dependency 설치, host sshd handoff를 수행할 수 있다.
# ==============================================================================

KITE_GHCR_INSTALL_REPOSITORY="${KITE_GHCR_INSTALL_REPOSITORY:-Hy3ons/KiteVirtualMachines}"
KITE_GHCR_INSTALL_REF="${KITE_GHCR_INSTALL_REF:-main}"
KITE_GHCR_INSTALL_ARCHIVE_URL="${KITE_GHCR_INSTALL_ARCHIVE_URL:-}"
KITE_GHCR_INSTALL_TMPDIR=""
MANAGE_HOST_SSHD="${MANAGE_HOST_SSHD:-true}"
KITE_HOST_SSHD_PORT_WAS_SET="${KITE_HOST_SSHD_PORT+x}"
KITE_HOST_SSHD_PORT="${KITE_HOST_SSHD_PORT:-2222}"

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

valid_tcp_port() {
  local port="$1"

  [[ "${port}" =~ ^[0-9]+$ ]] || return 1
  (( port >= 1 && port <= 65535 ))
}

prompt_remote_host_sshd_port() {
  local answer

  [[ "${MANAGE_HOST_SSHD}" == "true" ]] || return 0
  [[ "${KITE_ASSUME_DEFAULTS:-false}" != "true" ]] || return 0
  [[ -z "${KITE_HOST_SSHD_PORT_WAS_SET}" ]] || return 0
  [[ ! -t 0 && -r /dev/tty && -w /dev/tty ]] || return 0

  while true; do
    {
      printf '%s\n' "Kite gateway가 외부 SSH 22번을 사용하기 전에 host sshd를 어떤 포트로 옮길까요?"
      printf '  기본값은 %s입니다. 이 포트는 나중에 host SSH 직접 접속과 gateway fallback에 사용됩니다.\n' "${KITE_HOST_SSHD_PORT}"
      printf '  포트 점유 여부는 실제 적용 직전에 다시 확인합니다.\n'
      printf '포트 입력 [기본: %s] ' "${KITE_HOST_SSHD_PORT}"
    } > /dev/tty
    IFS= read -r answer < /dev/tty
    answer="${answer:-${KITE_HOST_SSHD_PORT}}"
    if valid_tcp_port "${answer}"; then
      KITE_HOST_SSHD_PORT="${answer}"
      export KITE_HOST_SSHD_PORT
      return 0
    fi
    printf '[kite-ghcr-install] WARNING: port must be a number between 1 and 65535\n' > /dev/tty
  done
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
  prompt_remote_host_sshd_port

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
