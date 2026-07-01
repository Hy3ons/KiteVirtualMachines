#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/manage-host-sshd.sh
# Description: Kite gateway가 22번 포트를 쓰도록 host sshd 포트를 옮기거나 복원한다.
#
# Usage:
#   build/deploy/scripts/manage-host-sshd.sh <ensure|restore|restore-after-port-free>
#
# Environment Variables:
#   KITE_MANAGE_HOST_SSHD: default ask
#   KITE_RESTORE_HOST_SSHD: default ask
#   KITE_HOST_SSHD_PORT: default 2222
#   KITE_GATEWAY_EXTERNAL_PORT: default 22
#   KITE_HOST_SSHD_CONFIG: default /etc/ssh/sshd_config
#   KITE_HOST_SSHD_STATE_DIR: default /etc/kite/host-sshd
#   KITE_HOST_SSHD_BACKUP: default ${KITE_HOST_SSHD_STATE_DIR}/sshd_config.before-kite
#   KITE_HOST_SSHD_STATE: default ${KITE_HOST_SSHD_STATE_DIR}/state.env
#   KITE_HOST_SSHD_RESTORE_WAIT_TIMEOUT_SECONDS: default 90
#   KITE_HOST_SSHD_RESTORE_WAIT_RETRY_SECONDS: default 1
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   호스트 /etc/ssh/sshd_config와 systemd ssh 서비스를 변경할 수 있다.
# ==============================================================================

ACTION="${1:-ensure}"
KITE_MANAGE_HOST_SSHD="${KITE_MANAGE_HOST_SSHD:-ask}"
KITE_RESTORE_HOST_SSHD="${KITE_RESTORE_HOST_SSHD:-ask}"
KITE_HOST_SSHD_PORT="${KITE_HOST_SSHD_PORT:-2222}"
KITE_GATEWAY_EXTERNAL_PORT="${KITE_GATEWAY_EXTERNAL_PORT:-22}"
KITE_HOST_SSHD_CONFIG="${KITE_HOST_SSHD_CONFIG:-/etc/ssh/sshd_config}"
KITE_HOST_SSHD_STATE_DIR="${KITE_HOST_SSHD_STATE_DIR:-/etc/kite/host-sshd}"
KITE_HOST_SSHD_BACKUP="${KITE_HOST_SSHD_BACKUP:-${KITE_HOST_SSHD_STATE_DIR}/sshd_config.before-kite}"
KITE_HOST_SSHD_STATE="${KITE_HOST_SSHD_STATE:-${KITE_HOST_SSHD_STATE_DIR}/state.env}"
KITE_HOST_SSHD_RESTORE_WAIT_TIMEOUT_SECONDS="${KITE_HOST_SSHD_RESTORE_WAIT_TIMEOUT_SECONDS:-90}"
KITE_HOST_SSHD_RESTORE_WAIT_RETRY_SECONDS="${KITE_HOST_SSHD_RESTORE_WAIT_RETRY_SECONDS:-1}"

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
    printf "\033[0;32m[kite-host-sshd] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-host-sshd] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-host-sshd] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-host-sshd] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}


# host sshd handoff는 systemd/Linux 전제를 사용하므로 Linux에서만 실행한다.
is_linux() {
  [[ "$(uname -s 2>/dev/null || true)" == "Linux" ]]
}

# root가 아니면 sudo로 권한 명령을 실행한다.
sudo_cmd() {
  if [[ "${EUID:-$(id -u)}" == "0" ]]; then
    "$@"
  else
    sudo "$@"
  fi
}

# sudo를 쓸 수 있는지 확인해 host sshd 설정 변경 가능 여부를 판단한다.
has_sudo() {
  [[ "${EUID:-$(id -u)}" == "0" ]] || command -v sudo >/dev/null 2>&1
}

# 배포판별 OpenSSH systemd unit 이름이 ssh.service/sshd.service 중 무엇인지 찾는다.
ssh_service_name() {
  if ! command -v systemctl >/dev/null 2>&1; then
    return 1
  fi
  if systemctl list-unit-files --no-legend ssh.service 2>/dev/null | awk '{ print $1 }' | grep -qx ssh.service; then
    echo "ssh.service"
    return 0
  fi
  if systemctl list-unit-files --no-legend sshd.service 2>/dev/null | awk '{ print $1 }' | grep -qx sshd.service; then
    echo "sshd.service"
    return 0
  fi
  return 1
}

# systemd socket activation이 켜진 ssh.socket/sshd.socket을 찾는다.
active_ssh_socket_unit() {
  local socket

  if ! command -v systemctl >/dev/null 2>&1; then
    return 1
  fi

  for socket in ssh.socket sshd.socket; do
    if systemctl list-unit-files --no-legend "${socket}" 2>/dev/null | awk '{ print $1 }' | grep -qx "${socket}" \
      && systemctl is-active --quiet "${socket}" 2>/dev/null; then
      echo "${socket}"
      return 0
    fi
  done

  return 1
}

# sshd 설정 문법 검사용 바이너리 경로를 찾는다.
sshd_binary() {
  if command -v sshd >/dev/null 2>&1; then
    command -v sshd
    return 0
  fi
  if [[ -x /usr/sbin/sshd ]]; then
    echo /usr/sbin/sshd
    return 0
  fi
  return 1
}

# 특정 TCP 포트에 LISTEN socket이 있는지 ss/netstat로 확인한다.
listening_port_is_open() {
  local port="$1"

  if command -v ss >/dev/null 2>&1; then
    ss -ltnH 2>/dev/null | awk -v port="${port}" '
      {
        endpoint = $4
        gsub(/\[|\]/, "", endpoint)
        if (endpoint ~ "(^|[:.])" port "$") {
          found = 1
        }
      }
      END {
        exit found ? 0 : 1
      }
    '
    return
  fi

  if command -v netstat >/dev/null 2>&1; then
    netstat -ltn 2>/dev/null | awk -v port="${port}" '
      {
        endpoint = $4
        gsub(/\[|\]/, "", endpoint)
        if (endpoint ~ "(^|[:.])" port "$") {
          found = 1
        }
      }
      END {
        exit found ? 0 : 1
      }
    '
    return
  fi

  log "ss/netstat was not found; skipping listen-port verification for ${port}"
  return 0
}

# 가능하면 해당 포트를 잡은 프로세스가 sshd인지 확인하고, 권한상 안 보이면 포트 LISTEN만 확인한다.
sshd_port_is_open() {
  local port="$1"

  if command -v ss >/dev/null 2>&1; then
    if sudo_cmd ss -ltnpH 2>/dev/null | awk -v port="${port}" '
      {
        endpoint = $4
        gsub(/\[|\]/, "", endpoint)
        if (endpoint ~ "(^|[:.])" port "$" && $0 ~ /sshd/) {
          found = 1
        }
      }
      END {
        exit found ? 0 : 1
      }
    '; then
      return 0
    fi
  fi

  listening_port_is_open "${port}"
}

# true/false/ask 모드를 공통으로 처리해 위험 작업 전 사용자 확인을 받는다.
confirm() {
  local mode="$1"
  local prompt="$2"

  case "${mode}" in
    true|yes|1)
      return 0
      ;;
    false|no|0)
      return 1
      ;;
    ask)
      if [[ ! -t 0 ]]; then
        log "non-interactive shell; skipping because confirmation is required"
        return 1
      fi
      while true; do
        printf '%s\n' "${prompt}" >&2
        printf '  1) 예\n' >&2
        printf '  2) 아니오\n' >&2
        read -r -p "선택 [1/2, 기본: 2] " answer
        answer="${answer:-2}"
        case "${answer}" in
          1|y|Y|yes|YES)
            return 0
            ;;
          2|n|N|no|NO)
            return 1
            ;;
          *)
            warn "1 또는 2를 입력하세요"
            ;;
        esac
      done
      ;;
    *)
      log "unknown confirmation mode ${mode}; expected ask, true, or false"
      return 1
      ;;
  esac
}

# 기존 sshd_config에서 전역 Port만 주석 처리하고, Match 블록 앞에 Kite 관리 Port를 삽입한다.
render_sshd_config() {
  local source="$1"
  local port="$2"

  awk -v port="${port}" '
    BEGIN {
      printed = 0
      in_match = 0
    }
    function print_block() {
      if (!printed) {
        print "# BEGIN KITE MANAGED SSHD"
        print "Port " port
        print "# END KITE MANAGED SSHD"
        printed = 1
      }
    }
    /^[[:space:]]*Match[[:space:]]/ {
      print_block()
      in_match = 1
    }
    !in_match && /^[[:space:]]*Port[[:space:]]+/ {
      print "# " $0 " # disabled by Kite"
      next
    }
    {
      print
    }
    END {
      print_block()
    }
  ' "${source}"
}

# host sshd가 gateway 외부 포트와 충돌하는 설정인지 판단한다.
config_needs_handoff() {
  local config="$1"
  local external_port="$2"
  local ports

  ports="$(awk '
    /^[[:space:]]*Match[[:space:]]/ { exit }
    /^[[:space:]]*#/ { next }
    /^[[:space:]]*Port[[:space:]]+/ { print $2 }
  ' "${config}")"

  if [[ -z "${ports}" ]]; then
    return 0
  fi
  if printf '%s\n' "${ports}" | grep -qx "${external_port}"; then
    return 0
  fi
  return 1
}

# Kite gateway가 22번을 사용할 수 있게 host sshd를 KITE_HOST_SSHD_PORT로 옮긴다.
ensure_host_sshd() {
  local service
  local sshd
  local socket
  local tmp

  if ! is_linux; then
    log "host sshd handoff is supported only on Linux; skipping"
    return 0
  fi
  if [[ "${KITE_GATEWAY_EXTERNAL_PORT}" != "22" ]]; then
    log "gateway external port is ${KITE_GATEWAY_EXTERNAL_PORT}; host sshd handoff is not needed"
    return 0
  fi
  if [[ ! -f "${KITE_HOST_SSHD_CONFIG}" ]]; then
    log "${KITE_HOST_SSHD_CONFIG} does not exist; no host sshd config to change"
    return 0
  fi
  if ! service="$(ssh_service_name)"; then
    log "systemd ssh.service/sshd.service was not found; skipping host sshd handoff"
    return 0
  fi
  # systemd socket activation이 켜진 환경에서는 sshd_config의 Port만 바꿔도 실제
  # 리스너는 ssh.socket/sshd.socket이 계속 22번을 잡고 있을 수 있다. 이 상태에서
  # gateway까지 22번을 잡으려 하면 설치 실패나 원격 접속 단절로 이어질 수 있으므로
  # 자동 handoff를 건너뛰고 운영자가 socket unit까지 직접 조정하게 한다.
  if socket="$(active_ssh_socket_unit)"; then
    if [[ "${KITE_MANAGE_HOST_SSHD}" == "true" ]]; then
      echo "[kite-host-sshd] ${socket} is active; move or disable the socket unit before forcing host sshd handoff" >&2
      exit 1
    fi
    log "${socket} is active; skipping host sshd handoff because socket activation must be moved manually"
    return 0
  fi
  if ! sshd="$(sshd_binary)"; then
    log "sshd binary was not found; skipping host sshd handoff"
    return 0
  fi
  if [[ -f "${KITE_HOST_SSHD_STATE}" ]]; then
    log "host sshd handoff state already exists at ${KITE_HOST_SSHD_STATE}; leaving current config unchanged"
    return 0
  fi
  if ! config_needs_handoff "${KITE_HOST_SSHD_CONFIG}" "${KITE_GATEWAY_EXTERNAL_PORT}"; then
    log "host sshd config already avoids port ${KITE_GATEWAY_EXTERNAL_PORT}; no handoff needed"
    return 0
  fi
  if ! has_sudo; then
    if [[ "${KITE_MANAGE_HOST_SSHD}" == "true" ]]; then
      echo "[kite-host-sshd] sudo is required to manage host sshd" >&2
      exit 1
    fi
    log "sudo is not available; skipping host sshd handoff"
    return 0
  fi
  if ! confirm "${KITE_MANAGE_HOST_SSHD}" "Move host sshd from port 22 to ${KITE_HOST_SSHD_PORT} so Kite gateway can own port 22? This may affect your current SSH session."; then
    log "host sshd handoff skipped"
    return 0
  fi

  tmp="$(mktemp "${TMPDIR:-/tmp}/kite-sshd-config.XXXXXX")"
  render_sshd_config "${KITE_HOST_SSHD_CONFIG}" "${KITE_HOST_SSHD_PORT}" > "${tmp}"
  sudo_cmd "${sshd}" -t -f "${tmp}"

  sudo_cmd install -d -m 0755 "${KITE_HOST_SSHD_STATE_DIR}"
  sudo_cmd cp -p "${KITE_HOST_SSHD_CONFIG}" "${KITE_HOST_SSHD_BACKUP}"
  sudo_cmd install -m 0644 "${tmp}" "${KITE_HOST_SSHD_CONFIG}"
  rm -f "${tmp}"

  {
    echo "SERVICE=${service}"
    echo "CONFIG=${KITE_HOST_SSHD_CONFIG}"
    echo "BACKUP=${KITE_HOST_SSHD_BACKUP}"
    echo "PORT=${KITE_HOST_SSHD_PORT}"
  } | sudo_cmd tee "${KITE_HOST_SSHD_STATE}" >/dev/null

  # sshd -t는 설정 파일 문법만 검증한다. systemd 재시작은 성공했지만 방화벽,
  # socket activation, 배포판별 unit 차이 때문에 새 포트가 실제로 열리지 않는
  # 경우가 있어, 2222가 열리지 않으면 즉시 원래 백업으로 되돌린다.
  if ! sudo_cmd systemctl restart "${service}"; then
    log "failed to restart ${service}; rolling back host sshd config"
    sudo_cmd cp -p "${KITE_HOST_SSHD_BACKUP}" "${KITE_HOST_SSHD_CONFIG}" || true
    sudo_cmd systemctl restart "${service}" || true
    sudo_cmd rm -rf "${KITE_HOST_SSHD_STATE_DIR}" || true
    exit 1
  fi
  if ! sshd_port_is_open "${KITE_HOST_SSHD_PORT}"; then
    log "host sshd did not open port ${KITE_HOST_SSHD_PORT}; rolling back host sshd config"
    sudo_cmd cp -p "${KITE_HOST_SSHD_BACKUP}" "${KITE_HOST_SSHD_CONFIG}" || true
    sudo_cmd systemctl restart "${service}" || true
    sudo_cmd rm -rf "${KITE_HOST_SSHD_STATE_DIR}" || true
    exit 1
  fi
  log "host sshd now listens on ${KITE_HOST_SSHD_PORT}; Kite gateway can bind external port 22"
}

# Kite 제거 후 백업해 둔 sshd_config를 복원해 host sshd가 다시 22번을 사용하게 한다.
restore_host_sshd() {
  local state
  local service
  local config
  local backup
  local sshd
  local rollback

  if ! is_linux; then
    return 0
  fi
  if [[ ! -f "${KITE_HOST_SSHD_STATE}" ]]; then
    log "no Kite host sshd state found; nothing to restore"
    return 0
  fi
  if ! has_sudo; then
    if [[ "${KITE_RESTORE_HOST_SSHD}" == "true" ]]; then
      echo "[kite-host-sshd] sudo is required to restore host sshd" >&2
      exit 1
    fi
    log "sudo is not available; skipping host sshd restore"
    return 0
  fi
  if ! confirm "${KITE_RESTORE_HOST_SSHD}" "Restore host sshd config from Kite backup and release port 22 back to the host?"; then
    log "host sshd restore skipped"
    return 0
  fi

  state="$(sudo_cmd cat "${KITE_HOST_SSHD_STATE}")"
  service="$(printf '%s\n' "${state}" | awk -F= '$1 == "SERVICE" { print $2; exit }')"
  config="$(printf '%s\n' "${state}" | awk -F= '$1 == "CONFIG" { print $2; exit }')"
  backup="$(printf '%s\n' "${state}" | awk -F= '$1 == "BACKUP" { print $2; exit }')"

  service="${service:-$(ssh_service_name || true)}"
  config="${config:-${KITE_HOST_SSHD_CONFIG}}"
  backup="${backup:-${KITE_HOST_SSHD_BACKUP}}"

  if [[ -z "${service}" ]]; then
    log "ssh systemd service was not found; restoring config without restart"
  fi
  if [[ ! -f "${backup}" ]]; then
    echo "[kite-host-sshd] backup ${backup} does not exist; cannot restore safely" >&2
    exit 1
  fi
  if sshd="$(sshd_binary)"; then
    sudo_cmd "${sshd}" -t -f "${backup}"
  fi

  rollback="$(mktemp "${TMPDIR:-/tmp}/kite-sshd-restore-rollback.XXXXXX")"
  sudo_cmd cp -p "${config}" "${rollback}"
  sudo_cmd cp -p "${backup}" "${config}"
  if [[ -n "${service}" ]]; then
    # restore는 원격 접속 경로를 2222에서 22로 되돌리는 작업이다. 백업 적용 후
    # 22번이 실제로 열렸는지 확인하기 전에는 Kite 상태 파일을 지우지 않는다.
    # 실패하면 직전 Kite-managed 설정으로 되돌려 최소한 기존 2222 경로를 살린다.
    if ! sudo_cmd systemctl restart "${service}"; then
      log "failed to restart ${service}; rolling back to Kite-managed sshd config"
      sudo_cmd cp -p "${rollback}" "${config}" || true
      sudo_cmd systemctl restart "${service}" || true
      sudo_cmd rm -f "${rollback}" || true
      exit 1
    fi
    if ! sshd_port_is_open "${KITE_GATEWAY_EXTERNAL_PORT}"; then
      log "host sshd did not open port ${KITE_GATEWAY_EXTERNAL_PORT}; rolling back to Kite-managed sshd config"
      sudo_cmd cp -p "${rollback}" "${config}" || true
      sudo_cmd systemctl restart "${service}" || true
      sudo_cmd rm -f "${rollback}" || true
      exit 1
    fi
  fi
  sudo_cmd rm -f "${rollback}" || true
  sudo_cmd rm -rf "${KITE_HOST_SSHD_STATE_DIR}"
  log "host sshd config restored from Kite backup"
}

# restore_host_sshd_after_port_release waits for the gateway listener to release port 22 before restoring sshd.
# This is used by clear/uninstall before deleting kite-gateway because the active SSH session may die immediately
# after the Kubernetes Service releases port 22.
restore_host_sshd_after_port_release() {
  local deadline

  if ! is_linux; then
    return 0
  fi
  if [[ ! -f "${KITE_HOST_SSHD_STATE}" ]]; then
    log "no Kite host sshd state found; nothing to restore"
    return 0
  fi
  if ! confirm "${KITE_RESTORE_HOST_SSHD}" "Restore host sshd config from Kite backup after Kite gateway releases port ${KITE_GATEWAY_EXTERNAL_PORT}?"; then
    log "host sshd restore skipped"
    return 0
  fi
  if ! command -v ss >/dev/null 2>&1 && ! command -v netstat >/dev/null 2>&1; then
    log "ss/netstat was not found; attempting host sshd restore without port-release wait"
    KITE_RESTORE_HOST_SSHD=true restore_host_sshd
    return
  fi

  deadline=$((SECONDS + KITE_HOST_SSHD_RESTORE_WAIT_TIMEOUT_SECONDS))
  while listening_port_is_open "${KITE_GATEWAY_EXTERNAL_PORT}"; do
    if [[ "${SECONDS}" -ge "${deadline}" ]]; then
      echo "[kite-host-sshd] port ${KITE_GATEWAY_EXTERNAL_PORT} did not become free within ${KITE_HOST_SSHD_RESTORE_WAIT_TIMEOUT_SECONDS}s; cannot restore host sshd safely" >&2
      exit 1
    fi
    log "waiting for port ${KITE_GATEWAY_EXTERNAL_PORT} to be released before restoring host sshd"
    sleep "${KITE_HOST_SSHD_RESTORE_WAIT_RETRY_SECONDS}"
  done

  KITE_RESTORE_HOST_SSHD=true restore_host_sshd
}

case "${ACTION}" in
  ensure)
    ensure_host_sshd
    ;;
  restore)
    restore_host_sshd
    ;;
  restore-after-port-free)
    restore_host_sshd_after_port_release
    ;;
  *)
    echo "[kite-host-sshd] unknown action ${ACTION}; use ensure, restore, or restore-after-port-free" >&2
    exit 1
    ;;
esac
