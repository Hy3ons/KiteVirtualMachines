#!/usr/bin/env bash
set -euo pipefail

KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
KITE_LONGHORN_DISK_TAG="${KITE_LONGHORN_DISK_TAG:-kite}"

log() {
  echo "[kite-deploy] $*"
}

require_kite_longhorn_disk() {
  local node="$1"
  local disks
  local found
  local name
  local allow_scheduling
  local tags
  local ready
  local schedulable

  disks="$(kubectl -n longhorn-system get "nodes.longhorn.io/${node}" -o 'go-template={{ range $name, $disk := .spec.disks }}{{ $status := index $.status.diskStatus $name }}{{ $name }}|{{ $disk.allowScheduling }}|{{ range $disk.tags }}{{ . }} {{ end }}|{{ if $status }}{{ range $status.conditions }}{{ if eq .type "Ready" }}{{ .status }}{{ end }}{{ end }}{{ end }}|{{ if $status }}{{ range $status.conditions }}{{ if eq .type "Schedulable" }}{{ .status }}{{ end }}{{ end }}{{ end }}{{ "\n" }}{{ end }}')"

  found="false"
  while IFS="|" read -r name allow_scheduling tags ready schedulable; do
    [[ -z "${name}" ]] && continue
    if [[ "${allow_scheduling}" == "true" && " ${tags} " == *" ${KITE_LONGHORN_DISK_TAG} "* && "${ready}" == "True" && "${schedulable}" == "True" ]]; then
      found="true"
      break
    fi
  done <<< "${disks}"

  if [[ "${found}" != "true" ]]; then
    echo "[kite-deploy] node/${node} has no Ready/Schedulable Longhorn disk tagged ${KITE_LONGHORN_DISK_TAG}" >&2
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
  kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-gateway --timeout=180s
  kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-frontend --timeout=180s
  kubectl -n "${KITE_NAMESPACE}" get service kite-gateway

  log "checking golden image"
  kubectl -n "${KITE_NAMESPACE}" get datavolume ubuntu-22.04
  kubectl -n "${KITE_NAMESPACE}" get pvc ubuntu-22.04
}

main "$@"
