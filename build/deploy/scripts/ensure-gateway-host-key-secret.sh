#!/usr/bin/env bash
set -euo pipefail

# ensure-gateway-host-key-secret.sh creates the SSH host key Secret used by kite-gateway.
# It prefers the current Linux host's OpenSSH server key so clients that already trusted
# the host on port 22 can keep the same fingerprint after Kite takes over that port.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
KITE_GATEWAY_HOST_KEY_SECRET="${KITE_GATEWAY_HOST_KEY_SECRET:-kite-gateway-host-key}"
KITE_GATEWAY_HOST_KEY_SOURCE="${KITE_GATEWAY_HOST_KEY_SOURCE:-auto}"
KITE_GATEWAY_HOST_KEY_REFRESH="${KITE_GATEWAY_HOST_KEY_REFRESH:-false}"
KITE_GATEWAY_HOST_KEY_FILE_NAME="${KITE_GATEWAY_HOST_KEY_FILE_NAME:-ssh_host_rsa_key}"
GATEWAY_HOST_KEY_TMPDIR=""

log() {
  echo "[kite-deploy] $*"
}

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "[kite-deploy] missing required command: ${name}" >&2
    exit 1
  fi
}

is_linux() {
  [[ "$(uname -s 2>/dev/null || true)" == "Linux" ]]
}

host_key_candidates() {
  printf '%s\n' \
    /etc/ssh/ssh_host_ed25519_key \
    /etc/ssh/ssh_host_ecdsa_key \
    /etc/ssh/ssh_host_rsa_key
}

host_key_exists() {
  local path="$1"

  [[ -f "${path}" ]] && return 0
  if command -v sudo >/dev/null 2>&1; then
    sudo test -f "${path}" 2>/dev/null
    return $?
  fi
  return 1
}

copy_key_file() {
  local source="$1"
  local target="$2"

  if [[ -r "${source}" ]]; then
    cp "${source}" "${target}"
  elif command -v sudo >/dev/null 2>&1; then
    sudo cat "${source}" > "${target}"
  else
    return 1
  fi
  chmod 0600 "${target}"
}

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

write_generated_key() {
  local target="$1"

  require_command ssh-keygen
  ssh-keygen -q -t rsa -b 4096 -N "" -f "${target}"
}

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

cleanup() {
  if [[ -n "${GATEWAY_HOST_KEY_TMPDIR}" ]]; then
    rm -rf "${GATEWAY_HOST_KEY_TMPDIR}"
  fi
}

main() {
  local key_path

  require_command kubectl
  trap cleanup EXIT
  kubectl apply -f "${ROOT_DIR}/build/kite/namespace.yaml"

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

  kubectl -n "${KITE_NAMESPACE}" create secret generic "${KITE_GATEWAY_HOST_KEY_SECRET}" \
    --from-file="${KITE_GATEWAY_HOST_KEY_FILE_NAME}=${key_path}" \
    --dry-run=client -o yaml \
    | kubectl apply -f -
}

main "$@"
