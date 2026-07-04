#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/ensure-gateway-host-key-secret.sh
# Description: kite-gateway가 사용할 SSH host key Secret을 생성하거나 갱신한다.
#
# Usage:
#   build/deploy/scripts/ensure-gateway-host-key-secret.sh
#
# Environment Variables:
#   KITE_NAMESPACE: default kite
#   KITE_GATEWAY_HOST_KEY_SECRET: default kite-gateway-host-key
#   KITE_GATEWAY_HOST_KEY_SOURCE: default auto
#   KITE_GATEWAY_HOST_KEY_REFRESH: default false
#   KITE_GATEWAY_HOST_KEY_FILE_NAME: default ssh_host_rsa_key
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   대상 클러스터 또는 로컬 개발 환경 상태를 변경할 수 있다.
# ==============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
KITE_GATEWAY_HOST_KEY_REFRESH_WAS_SET="${KITE_GATEWAY_HOST_KEY_REFRESH+x}"
KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
KITE_GATEWAY_HOST_KEY_SECRET="${KITE_GATEWAY_HOST_KEY_SECRET:-kite-gateway-host-key}"
KITE_GATEWAY_HOST_KEY_SOURCE="${KITE_GATEWAY_HOST_KEY_SOURCE:-auto}"
KITE_GATEWAY_HOST_KEY_REFRESH="${KITE_GATEWAY_HOST_KEY_REFRESH:-false}"
KITE_GATEWAY_HOST_KEY_FILE_NAME="${KITE_GATEWAY_HOST_KEY_FILE_NAME:-ssh_host_rsa_key}"
GATEWAY_HOST_KEY_TMPDIR=""

# shellcheck source=build/lib/prompt.sh
source "${ROOT_DIR}/build/lib/prompt.sh"

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
    printf "\033[0;32m[kite-deploy] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-deploy] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-deploy] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-deploy] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}


sudo_cmd() {
  if kite_prompt_interactive; then
    sudo "$@"
  elif [[ -n "${SUDO_ASKPASS:-}" ]]; then
    sudo -A "$@"
  else
    sudo -n "$@"
  fi
}

# kubectl/ssh-keygen 같은 필수 명령을 사용하는 지점 전에 명확히 실패시킨다.
require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

# host key 재사용은 Linux의 /etc/ssh 기본 경로를 전제로 한다.
is_linux() {
  [[ "$(uname -s 2>/dev/null || true)" == "Linux" ]]
}

# OpenSSH가 일반적으로 만드는 host key 후보를 선호도 순서대로 반환한다.
host_key_candidates() {
  printf '%s\n' \
    /etc/ssh/ssh_host_ed25519_key \
    /etc/ssh/ssh_host_ecdsa_key \
    /etc/ssh/ssh_host_rsa_key
}

# root 소유 host key는 일반 사용자에게 읽기 권한이 없을 수 있어 sudo test까지 시도한다.
host_key_exists() {
  local path="$1"

  [[ -f "${path}" ]] && return 0
  if command -v sudo >/dev/null 2>&1; then
    sudo_cmd test -f "${path}" 2>/dev/null
    return $?
  fi
  return 1
}

# host key를 임시 파일로 복사한다. Secret 생성용 파일은 Kubernetes에 올린 뒤 삭제된다.
copy_key_file() {
  local source="$1"
  local target="$2"

  if [[ -r "${source}" ]]; then
    cp "${source}" "${target}"
  elif command -v sudo >/dev/null 2>&1; then
    sudo_cmd cat "${source}" > "${target}"
  else
    return 1
  fi
  chmod 0600 "${target}"
}

# 사용할 수 있는 첫 번째 Linux host key를 찾는다.
select_host_key() {
  local candidate

  is_linux || return 1
  while IFS= read -r candidate; do
    if host_key_exists "${candidate}"; then
      printf '%s\n' "${candidate}"
      return 0
    fi
  done < <(host_key_candidates)
  return 1
}

# host key를 재사용할 수 없거나 generate 모드일 때 gateway 전용 RSA key를 만든다.
write_generated_key() {
  local target="$1"

  require_command ssh-keygen
  ssh-keygen -q -t rsa -b 4096 -N "" -f "${target}"
}

# 환경변수 설정(auto/host/generate/절대경로)에 따라 gateway host key 파일을 준비한다.
write_gateway_key() {
  local target="$1"
  local host_key

  case "${KITE_GATEWAY_HOST_KEY_SOURCE}" in
    auto)
      if host_key="$(select_host_key)"; then
        log "using existing host SSH key ${host_key} for kite-gateway host key"
        copy_key_file "${host_key}" "${target}"
      else
        log "host SSH key was not found; generating kite-gateway host key"
        write_generated_key "${target}"
      fi
      ;;
    host)
      if ! host_key="$(select_host_key)"; then
        echo "[kite-deploy] host SSH key was not found" >&2
        exit 1
      fi
      log "using existing host SSH key ${host_key} for kite-gateway host key"
      copy_key_file "${host_key}" "${target}"
      ;;
    generate)
      log "generating kite-gateway host key"
      write_generated_key "${target}"
      ;;
    /*)
      log "using configured SSH host key ${KITE_GATEWAY_HOST_KEY_SOURCE}"
      copy_key_file "${KITE_GATEWAY_HOST_KEY_SOURCE}" "${target}"
      ;;
    *)
      echo "[kite-deploy] KITE_GATEWAY_HOST_KEY_SOURCE must be auto, host, generate, or an absolute file path" >&2
      exit 1
      ;;
  esac
}

# 임시 key 파일이 디스크에 남지 않도록 종료 시 정리한다.
cleanup() {
  if [[ -n "${GATEWAY_HOST_KEY_TMPDIR:-}" ]]; then
    rm -rf "${GATEWAY_HOST_KEY_TMPDIR}"
  fi
}

# Secret이 없거나 refresh가 요청된 경우 gateway host key Secret을 생성/갱신한다.
main() {
  local key_path

  require_command kubectl
  trap cleanup EXIT
  # Secret을 만들 namespace가 먼저 있어야 하므로 namespace manifest를 idempotent하게 적용한다.
  kubectl apply -f "${ROOT_DIR}/build/kite/namespace.yaml"
  kite_prompt_configure_bool KITE_GATEWAY_HOST_KEY_REFRESH "${KITE_GATEWAY_HOST_KEY_REFRESH_WAS_SET}" "기존 kite-gateway host key Secret이 있으면 새 key로 갱신할까요?"

  if kubectl -n "${KITE_NAMESPACE}" get secret "${KITE_GATEWAY_HOST_KEY_SECRET}" >/dev/null 2>&1; then
    if [[ "${KITE_GATEWAY_HOST_KEY_REFRESH}" != "true" ]]; then
      log "gateway host key Secret ${KITE_NAMESPACE}/${KITE_GATEWAY_HOST_KEY_SECRET} already exists; keeping it"
      return
    fi
    log "refreshing gateway host key Secret ${KITE_NAMESPACE}/${KITE_GATEWAY_HOST_KEY_SECRET}"
  fi

  GATEWAY_HOST_KEY_TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/kite-gateway-host-key.XXXXXX")"
  key_path="${GATEWAY_HOST_KEY_TMPDIR}/${KITE_GATEWAY_HOST_KEY_FILE_NAME}"
  write_gateway_key "${key_path}"

  # create --dry-run 출력 YAML을 apply에 넘기면 생성과 갱신을 같은 경로로 처리할 수 있다.
  kubectl -n "${KITE_NAMESPACE}" create secret generic "${KITE_GATEWAY_HOST_KEY_SECRET}" \
    --from-file="${KITE_GATEWAY_HOST_KEY_FILE_NAME}=${key_path}" \
    --dry-run=client -o yaml \
    | kubectl apply -f -
}

main "$@"
