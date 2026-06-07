#!/usr/bin/env bash
set -euo pipefail

KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
DELETE_GOLDEN_IMAGE="${DELETE_GOLDEN_IMAGE:-false}"
DELETE_LONGHORN="${DELETE_LONGHORN:-false}"
DELETE_LONGHORN_FORCE="${DELETE_LONGHORN_FORCE:-false}"
DELETE_LONGHORN_DATA="${DELETE_LONGHORN_DATA:-false}"
DELETE_LONGHORN_DATA_CONFIRM="${DELETE_LONGHORN_DATA_CONFIRM:-false}"
KITE_LONGHORN_DISK_NAME="${KITE_LONGHORN_DISK_NAME:-kite-longhorn}"

log() {
  echo "[kite-deploy] $*"
}

remove_kite_longhorn_disks() {
  if ! kubectl get crd nodes.longhorn.io >/dev/null 2>&1; then
    return
  fi

  log "removing Kite Longhorn disk entries from Longhorn node resources"
  kubectl -n longhorn-system get nodes.longhorn.io -o name 2>/dev/null \
    | while read -r node; do
        [[ -z "${node}" ]] && continue
        kubectl -n longhorn-system patch "${node}" --type=json -p "[{\"op\":\"remove\",\"path\":\"/spec/disks/${KITE_LONGHORN_DISK_NAME}\"}]" 2>/dev/null || true
      done
}

longhorn_pv_count() {
  kubectl get pv -o jsonpath='{range .items[?(@.spec.csi.driver=="driver.longhorn.io")]}{.metadata.name}{"\n"}{end}' 2>/dev/null \
    | sed '/^[[:space:]]*$/d' \
    | wc -l \
    | tr -d ' '
}

delete_kite_longhorn_host_data() {
  if [[ "${DELETE_LONGHORN_DATA}" != "true" ]]; then
    log "skipping Longhorn host data cleanup because DELETE_LONGHORN_DATA=${DELETE_LONGHORN_DATA}"
    return
  fi
  if [[ "${DELETE_LONGHORN_DATA_CONFIRM}" != "true" ]]; then
    echo "[kite-deploy] refusing Longhorn host data deletion without DELETE_LONGHORN_DATA_CONFIRM=true" >&2
    exit 1
  fi
  if [[ "${DELETE_LONGHORN_FORCE}" != "true" ]]; then
    local pv_count
    pv_count="$(longhorn_pv_count)"
    if [[ "${pv_count}" != "0" ]]; then
      log "skipping Kite Longhorn host data cleanup because ${pv_count} Longhorn PV(s) still exist"
      log "delete remaining Longhorn PVC/PV resources first, or set DELETE_LONGHORN_FORCE=true"
      return
    fi
  fi

  log "deleting Kite Longhorn host data on every node"
  kubectl apply -f "${ROOT_DIR}/build/kite-storage/longhorn-cleanup"
  kubectl -n longhorn-system rollout status daemonset/kite-longhorn-disk-cleanup --timeout=180s || true
  kubectl delete -f "${ROOT_DIR}/build/kite-storage/longhorn-cleanup" --ignore-not-found=true || true
}

delete_longhorn_resources() {
  if [[ "${DELETE_LONGHORN}" != "true" ]]; then
    return
  fi

  remove_kite_longhorn_disks

  if [[ "${DELETE_LONGHORN_FORCE}" != "true" ]]; then
    local pv_count
    pv_count="$(longhorn_pv_count)"
    if [[ "${pv_count}" != "0" ]]; then
      log "skipping Longhorn uninstall because ${pv_count} Longhorn PV(s) still exist"
      log "delete remaining Longhorn PVC/PV resources first, or set DELETE_LONGHORN_FORCE=true"
      return
    fi
  fi

  log "deleting Longhorn resources"
  kubectl delete storageclass longhorn --ignore-not-found=true || true
  kubectl delete namespace longhorn-system --ignore-not-found=true --wait=false || true

  kubectl api-resources --api-group=longhorn.io --verbs=list -o name 2>/dev/null \
    | while read -r resource; do
        [[ -z "${resource}" ]] && continue
        kubectl get "${resource}" -A -o name 2>/dev/null \
          | xargs -r -n 1 kubectl patch --type=merge -p '{"metadata":{"finalizers":[]}}' || true
      done
}

main() {
  if [[ "${DELETE_GOLDEN_IMAGE}" == "true" ]]; then
    log "deleting golden image DataVolumes and PVCs"
    kubectl delete -f "${ROOT_DIR}/build/kite-storage/golden-images" --ignore-not-found=true || true
    kubectl -n "${KITE_NAMESPACE}" delete datavolume ubuntu-22.04 --ignore-not-found=true
    kubectl -n "${KITE_NAMESPACE}" delete pvc ubuntu-22.04 --ignore-not-found=true
  fi

  log "deleting Kite manifests"
  kubectl delete -k "${ROOT_DIR}/build/kite" --ignore-not-found=true || true
  kubectl delete -f "${ROOT_DIR}/build/kite-storage/longhorn" --ignore-not-found=true || true
  remove_kite_longhorn_disks
  delete_kite_longhorn_host_data

  log "deleting remaining Kite namespace and cluster-scoped resources"
  kubectl delete namespace "${KITE_NAMESPACE}" --ignore-not-found=true
  kubectl delete crd kiteusers.hy3ons.github.io kitevirtualmachines.hy3ons.github.io --ignore-not-found=true
  kubectl delete clusterrole kite-control-plane-role --ignore-not-found=true
  kubectl delete clusterrolebinding kite-control-plane-binding --ignore-not-found=true

  delete_longhorn_resources
}

main "$@"
