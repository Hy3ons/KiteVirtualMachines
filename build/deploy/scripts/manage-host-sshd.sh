#!/usr/bin/env bash
set -euo pipefail

# manage-host-sshd.sh optionally moves the host OpenSSH daemon away from port 22.
# Kite uses this only when the operator explicitly allows the Kubernetes gateway
# Service to own external SSH port 22.

ACTION="${1:-ensure}"
KITE_MANAGE_HOST_SSHD="${KITE_MANAGE_HOST_SSHD:-ask}"
KITE_RESTORE_HOST_SSHD="${KITE_RESTORE_HOST_SSHD:-ask}"
KITE_HOST_SSHD_PORT="${KITE_HOST_SSHD_PORT:-2222}"
KITE_GATEWAY_EXTERNAL_PORT="${KITE_GATEWAY_EXTERNAL_PORT:-22}"
KITE_HOST_SSHD_CONFIG="${KITE_HOST_SSHD_CONFIG:-/etc/ssh/sshd_config}"
KITE_HOST_SSHD_STATE_DIR="${KITE_HOST_SSHD_STATE_DIR:-/etc/kite/host-sshd}"
KITE_HOST_SSHD_BACKUP="${KITE_HOST_SSHD_BACKUP:-${KITE_HOST_SSHD_STATE_DIR}/sshd_config.before-kite}"
KITE_HOST_SSHD_STATE="${KITE_HOST_SSHD_STATE:-${KITE_HOST_SSHD_STATE_DIR}/state.env}"

log() {
  echo "[kite-host-sshd] $*"
}

is_linux() {
  [[ "$(uname -s 2>/dev/null || true)" == "Linux" ]]
}

sudo_cmd() {
  if [[ "${EUID:-$(id -u)}" == "0" ]]; then
    "$@"
  else
    sudo "$@"
  fi
}

has_sudo() {
  [[ "${EUID:-$(id -u)}" == "0" ]] || command -v sudo >/dev/null 2>&1
}

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
      read -r -p "${prompt} [y/N] " answer
      [[ "${answer}" == "y" || "${answer}" == "Y" || "${answer}" == "yes" || "${answer}" == "YES" ]]
      ;;
    *)
      log "unknown confirmation mode ${mode}; expected ask, true, or false"
      return 1
      ;;
  esac
}

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

ensure_host_sshd() {
  local service
  local sshd
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

  sudo_cmd systemctl restart "${service}"
  log "host sshd now listens on ${KITE_HOST_SSHD_PORT}; Kite gateway can bind external port 22"
}

restore_host_sshd() {
  local state
  local service
  local config
  local backup
  local sshd

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

  sudo_cmd cp -p "${backup}" "${config}"
  if [[ -n "${service}" ]]; then
    sudo_cmd systemctl restart "${service}"
  fi
  sudo_cmd rm -rf "${KITE_HOST_SSHD_STATE_DIR}"
  log "host sshd config restored from Kite backup"
}

case "${ACTION}" in
  ensure)
    ensure_host_sshd
    ;;
  restore)
    restore_host_sshd
    ;;
  *)
    echo "[kite-host-sshd] unknown action ${ACTION}; use ensure or restore" >&2
    exit 1
    ;;
esac
