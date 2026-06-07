#!/usr/bin/env bash
set -euo pipefail

KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
KITE_LONGHORN_DISK_NAME="${KITE_LONGHORN_DISK_NAME:-kite-longhorn}"
KITE_LONGHORN_DISK_PATH="${KITE_LONGHORN_DISK_PATH:-/mnt/kite-longhorn}"
KITE_LONGHORN_DISK_TAG="${KITE_LONGHORN_DISK_TAG:-kite}"

log() {
  echo "[kite-deploy] $*"
}

require_kite_longhorn_disk() {
  local node="$1"
  local disk
  local path
  local allow_scheduling
  local tags

  disk="$(kubectl -n longhorn-system get "nodes.longhorn.io/${node}" -o "go-template={{ with index .spec.disks \"${KITE_LONGHORN_DISK_NAME}\" }}{{ .path }}|{{ .allowScheduling }}|{{ range .tags }}{{ . }} {{ end }}{{ end }}")"
  if [[ -z "${disk}" ]]; then
    echo "[kite-deploy] missing Longhorn disk ${KITE_LONGHORN_DISK_NAME} on node/${node}" >&2
    exit 1
  fi

  IFS="|" read -r path allow_scheduling tags <<< "${disk}"
  if [[ "${path}" != "${KITE_LONGHORN_DISK_PATH}" ]]; then
    echo "[kite-deploy] Longhorn disk ${KITE_LONGHORN_DISK_NAME} on node/${node} uses path ${path}, expected ${KITE_LONGHORN_DISK_PATH}" >&2
    exit 1
  fi
  if [[ "${allow_scheduling}" != "true" ]]; then
    echo "[kite-deploy] Longhorn disk ${KITE_LONGHORN_DISK_NAME} on node/${node} does not allow scheduling" >&2
    exit 1
  fi
  if [[ " ${tags} " != *" ${KITE_LONGHORN_DISK_TAG} "* ]]; then
    echo "[kite-deploy] Longhorn disk ${KITE_LONGHORN_DISK_NAME} on node/${node} is missing tag ${KITE_LONGHORN_DISK_TAG}" >&2
    exit 1
  fi
}

verify_kite_longhorn_disks() {
  kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' \
    | while read -r node; do
        [[ -z "${node}" ]] && continue
        require_kite_longhorn_disk "${node}"
      done
}

main() {
  log "checking storage"
  kubectl get storageclass kite-vm-storage
  verify_kite_longhorn_disks

  log "checking dependencies"
  kubectl get pods -n longhorn-system
  kubectl get pods -n kubevirt
  kubectl get pods -n cdi

  log "checking Kite APIs"
  kubectl get crd kiteusers.hy3ons.github.io
  kubectl get crd kitevirtualmachines.hy3ons.github.io

  log "checking Kite workloads"
  kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-api --timeout=180s
  kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-controller --timeout=180s
  kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-frontend --timeout=180s
  kubectl -n "${KITE_NAMESPACE}" rollout status daemonset/kite-host-agent --timeout=180s

  log "checking golden image"
  kubectl -n "${KITE_NAMESPACE}" get datavolume ubuntu-22.04
  kubectl -n "${KITE_NAMESPACE}" get pvc ubuntu-22.04
}

main "$@"
