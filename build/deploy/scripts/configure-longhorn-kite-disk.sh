#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
KITE_LONGHORN_DISK_NAME="${KITE_LONGHORN_DISK_NAME:-kite-longhorn}"
KITE_LONGHORN_DISK_PATH="${KITE_LONGHORN_DISK_PATH:-/mnt/kite-longhorn}"
KITE_LONGHORN_DISK_TAG="${KITE_LONGHORN_DISK_TAG:-kite}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-180}"
PATCH_RETRY_INTERVAL_SECONDS="${PATCH_RETRY_INTERVAL_SECONDS:-5}"

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

patch_longhorn_node_disk() {
  local node="$1"
  local deadline
  local output

  deadline=$((SECONDS + TIMEOUT_SECONDS))

  log "configuring Longhorn disk ${KITE_LONGHORN_DISK_NAME} on node/${node}"
  while true; do
    if output="$(kubectl -n longhorn-system patch "nodes.longhorn.io/${node}" --type=merge -p "{
      \"spec\": {
        \"disks\": {
          \"${KITE_LONGHORN_DISK_NAME}\": {
            \"path\": \"${KITE_LONGHORN_DISK_PATH}\",
            \"allowScheduling\": true,
            \"tags\": [\"${KITE_LONGHORN_DISK_TAG}\"]
          }
        }
      }
    }" 2>&1)"; then
      echo "${output}"
      return
    fi

    if [[ "${output}" != *"being syncing"* && "${output}" != *"please retry later"* ]]; then
      echo "${output}" >&2
      return 1
    fi

    if (( SECONDS >= deadline )); then
      echo "${output}" >&2
      echo "[kite-deploy] timed out waiting for Longhorn node/${node} disk sync" >&2
      return 1
    fi

    log "Longhorn node/${node} is still syncing disks; retrying in ${PATCH_RETRY_INTERVAL_SECONDS}s"
    sleep "${PATCH_RETRY_INTERVAL_SECONDS}"
  done
}

main() {
  require_command kubectl

  log "creating host disk directories for Kite Longhorn storage"
  kubectl apply -f "${ROOT_DIR}/build/kite-storage/longhorn/disk-directory-daemonset.yaml"
  kubectl -n longhorn-system rollout status daemonset/kite-longhorn-disk-directory --timeout="${TIMEOUT_SECONDS}s"
  kubectl delete -f "${ROOT_DIR}/build/kite-storage/longhorn/disk-directory-daemonset.yaml" --ignore-not-found=true

  kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' \
    | while read -r node; do
        [[ -z "${node}" ]] && continue
        patch_longhorn_node_disk "${node}"
      done
}

main "$@"
