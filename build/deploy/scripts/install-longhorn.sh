#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/install-longhorn.sh
# Description: Longhorn 기본 manifest를 설치한다.
#
# Usage:
#   build/deploy/scripts/install-longhorn.sh
#
# Environment Variables:
#   LONGHORN_VERSION: default (empty)
#   LONGHORN_MANIFEST_URL: default (empty)
#   KITE_INSTALL_LONGHORN_HOST_PACKAGES: default true
#   KITE_LONGHORN_NAMESPACE_DELETE_TIMEOUT_SECONDS: default 300
#   KITE_LONGHORN_OWNER_LABEL_KEY: default hy3ons.github.io/kite-installed-longhorn
#   KITE_LONGHORN_OWNER_LABEL_VALUE: default true
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   Kubernetes 리소스 적용, 컨테이너 이미지 빌드/주입, rollout 대기를 수행할 수 있다.
# ==============================================================================

LONGHORN_VERSION="${LONGHORN_VERSION:-}"
LONGHORN_MANIFEST_URL="${LONGHORN_MANIFEST_URL:-}"
KITE_INSTALL_LONGHORN_HOST_PACKAGES="${KITE_INSTALL_LONGHORN_HOST_PACKAGES:-true}"
KITE_LONGHORN_NAMESPACE_DELETE_TIMEOUT_SECONDS="${KITE_LONGHORN_NAMESPACE_DELETE_TIMEOUT_SECONDS:-300}"
KITE_LONGHORN_OWNER_LABEL_KEY="${KITE_LONGHORN_OWNER_LABEL_KEY:-hy3ons.github.io/kite-installed-longhorn}"
KITE_LONGHORN_OWNER_LABEL_VALUE="${KITE_LONGHORN_OWNER_LABEL_VALUE:-true}"

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


# kubectl/curl 누락을 초기에 확인해 Longhorn 설치가 중간에 멈추지 않게 한다.
require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

sudo_command() {
  if [[ "$(id -u)" -eq 0 ]]; then
    "$@"
    return
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    warn "sudo is required to install Longhorn host packages; install open-iscsi and nfs-common manually or run as root"
    exit 1
  fi

  sudo "$@"
}

install_longhorn_host_packages() {
  if [[ "${KITE_INSTALL_LONGHORN_HOST_PACKAGES}" != "true" ]]; then
    return 0
  fi
  if command -v iscsiadm >/dev/null 2>&1 && command -v mount.nfs >/dev/null 2>&1; then
    return 0
  fi
  if ! command -v apt-get >/dev/null 2>&1; then
    warn "Longhorn host packages are missing, but this installer only auto-installs them on apt-based hosts"
    warn "install open-iscsi and nfs-common manually, or set KITE_INSTALL_LONGHORN_HOST_PACKAGES=false if the host is already prepared"
    exit 1
  fi

  log "installing Longhorn host packages with apt-get"
  sudo_command apt-get update
  sudo_command env DEBIAN_FRONTEND=noninteractive apt-get install -y open-iscsi nfs-common
  if command -v systemctl >/dev/null 2>&1; then
    sudo_command systemctl enable --now iscsid >/dev/null 2>&1 || true
  fi
}

longhorn_namespace_deletion_timestamp() {
  kubectl get namespace longhorn-system -o jsonpath='{.metadata.deletionTimestamp}' 2>/dev/null || true
}

longhorn_namespace_exists() {
  kubectl get namespace longhorn-system >/dev/null 2>&1
}

wait_for_previous_longhorn_namespace_deletion() {
  local deletion_timestamp
  local deadline

  deletion_timestamp="$(longhorn_namespace_deletion_timestamp)"
  [[ -z "${deletion_timestamp}" ]] && return 0

  log "waiting for previous longhorn-system namespace deletion to finish"
  deadline=$((SECONDS + KITE_LONGHORN_NAMESPACE_DELETE_TIMEOUT_SECONDS))
  while kubectl get namespace longhorn-system >/dev/null 2>&1; do
    if [[ "${SECONDS}" -ge "${deadline}" ]]; then
      warn "longhorn-system namespace is still terminating after ${KITE_LONGHORN_NAMESPACE_DELETE_TIMEOUT_SECONDS}s"
      exit 1
    fi
    sleep 2
  done
}

mark_longhorn_installed_by_kite() {
  log "marking longhorn-system as installed by Kite"
  kubectl label namespace longhorn-system "${KITE_LONGHORN_OWNER_LABEL_KEY}=${KITE_LONGHORN_OWNER_LABEL_VALUE}" --overwrite
}

# Longhorn 기본 manifest를 설치한다. 이미 클러스터에 Longhorn이 있으면 apply로 수렴시킨다.
main() {
  local kite_created_longhorn="false"

  require_command kubectl

  if [[ -z "${LONGHORN_MANIFEST_URL}" ]]; then
    if [[ -z "${LONGHORN_VERSION}" ]]; then
      require_command curl
      # 명시 버전이 없으면 upstream latest release tag를 가져와 기본 manifest URL에 사용한다.
      LONGHORN_VERSION="$(curl -fsSL https://api.github.com/repos/longhorn/longhorn/releases/latest | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
    fi
    LONGHORN_MANIFEST_URL="https://raw.githubusercontent.com/longhorn/longhorn/${LONGHORN_VERSION}/deploy/longhorn.yaml"
  fi

  log "installing Longhorn from ${LONGHORN_MANIFEST_URL}"
  install_longhorn_host_packages
  wait_for_previous_longhorn_namespace_deletion
  if ! longhorn_namespace_exists; then
    kite_created_longhorn="true"
  fi
  # Longhorn은 namespace, CRD, controller를 한 manifest에서 설치한다.
  kubectl apply -f "${LONGHORN_MANIFEST_URL}"
  if [[ "${kite_created_longhorn}" == "true" ]]; then
    mark_longhorn_installed_by_kite
  fi
}

main "$@"
