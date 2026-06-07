#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
KITE_LONGHORN_DISK_NAME="${KITE_LONGHORN_DISK_NAME:-kite-longhorn}"
KITE_LONGHORN_DISK_PATH="${KITE_LONGHORN_DISK_PATH:-/mnt/kite-longhorn}"
KITE_LONGHORN_DISK_TAG="${KITE_LONGHORN_DISK_TAG:-kite}"
KITE_LONGHORN_USE_DEDICATED_DISK="${KITE_LONGHORN_USE_DEDICATED_DISK:-false}"
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

patch_longhorn_node_dedicated_disk() {
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

tag_existing_longhorn_disks() {
  local node="$1"
  local disks
  local disk
  local allow_scheduling
  local ready
  local schedulable
  local tags
  local next_tags

  disks="$(kubectl -n longhorn-system get "nodes.longhorn.io/${node}" -o 'go-template={{ range $name, $disk := .spec.disks }}{{ $status := index $.status.diskStatus $name }}{{ $name }}|{{ $disk.allowScheduling }}|{{ if $status }}{{ range $status.conditions }}{{ if eq .type "Ready" }}{{ .status }}{{ end }}{{ end }}{{ end }}|{{ if $status }}{{ range $status.conditions }}{{ if eq .type "Schedulable" }}{{ .status }}{{ end }}{{ end }}{{ end }}|{{ range $disk.tags }}{{ . }} {{ end }}{{ "\n" }}{{ end }}')"
  if [[ -z "${disks}" ]]; then
    echo "[kite-deploy] no Longhorn disks found on node/${node}" >&2
    return 1
  fi

  log "tagging existing Ready Longhorn disks on node/${node} with ${KITE_LONGHORN_DISK_TAG}"
  while IFS="|" read -r disk allow_scheduling ready schedulable tags; do
    [[ -z "${disk}" ]] && continue
    if [[ "${allow_scheduling}" != "true" || "${ready}" != "True" || "${schedulable}" != "True" ]]; then
      log "skipping Longhorn disk ${disk} on node/${node} because ready=${ready} schedulable=${schedulable} allowScheduling=${allow_scheduling}"
      continue
    fi
    if [[ " ${tags} " == *" ${KITE_LONGHORN_DISK_TAG} "* ]]; then
      continue
    fi

    next_tags="$(printf '%s\n%s\n' "${tags}" "${KITE_LONGHORN_DISK_TAG}" | xargs -n 1 | awk 'NF && !seen[$0]++ { printf "%s\"%s\"", sep, $0; sep=", " }')"
    kubectl -n longhorn-system patch "nodes.longhorn.io/${node}" --type=json -p "[{\"op\":\"replace\",\"path\":\"/spec/disks/${disk}/tags\",\"value\":[${next_tags}]}]"
  done <<< "${disks}"
}

main() {
  require_command kubectl

  if [[ "${KITE_LONGHORN_USE_DEDICATED_DISK}" == "true" ]]; then
    log "creating host disk directories for Kite Longhorn storage"
    kubectl apply -f "${ROOT_DIR}/build/kite-storage/longhorn/disk-directory-daemonset.yaml"
    kubectl -n longhorn-system rollout status daemonset/kite-longhorn-disk-directory --timeout="${TIMEOUT_SECONDS}s"
    kubectl delete -f "${ROOT_DIR}/build/kite-storage/longhorn/disk-directory-daemonset.yaml" --ignore-not-found=true
  else
    log "using existing Longhorn disks; set KITE_LONGHORN_USE_DEDICATED_DISK=true to add ${KITE_LONGHORN_DISK_PATH}"
  fi

  kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' \
    | while read -r node; do
        [[ -z "${node}" ]] && continue
        if [[ "${KITE_LONGHORN_USE_DEDICATED_DISK}" == "true" ]]; then
          patch_longhorn_node_dedicated_disk "${node}"
        else
          tag_existing_longhorn_disks "${node}"
        fi
      done
}

main "$@"
