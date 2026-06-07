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
KITE_LONGHORN_DISK_TAG="${KITE_LONGHORN_DISK_TAG:-kite}"
DELETE_HOST_DNS="${DELETE_HOST_DNS:-true}"
DELETE_HOST_ACCOUNTS="${DELETE_HOST_ACCOUNTS:-true}"

log() {
  echo "[kite-deploy] $*"
}

reset_kite_host_dns() {
  if [[ "${DELETE_HOST_DNS}" != "true" ]]; then
    log "skipping host DNS cleanup because DELETE_HOST_DNS=${DELETE_HOST_DNS}"
    return
  fi
  if ! kubectl -n "${KITE_NAMESPACE}" get pods -l app=kite-host-agent >/dev/null 2>&1; then
    return
  fi

  log "removing Kite cluster.local host DNS routing"
  kubectl -n "${KITE_NAMESPACE}" get pods -l app=kite-host-agent -o name 2>/dev/null \
    | while read -r pod; do
        [[ -z "${pod}" ]] && continue
        kubectl -n "${KITE_NAMESPACE}" exec "${pod}" -- nsenter -t 1 -m -u -i -n -p -- sh -c '
          if command -v resolvectl >/dev/null 2>&1; then
            iface="$(ip route show default 2>/dev/null | awk "{print \$5; exit}")"
            if [ -n "$iface" ]; then
              resolvectl revert "$iface" >/dev/null 2>&1 || true
            fi
          fi
        ' >/dev/null 2>&1 || true
      done
}

clear_kite_host_accounts() {
  if [[ "${DELETE_HOST_ACCOUNTS}" != "true" ]]; then
    log "skipping host account cleanup because DELETE_HOST_ACCOUNTS=${DELETE_HOST_ACCOUNTS}"
    return
  fi
  if ! kubectl -n "${KITE_NAMESPACE}" get pods -l app=kite-host-agent >/dev/null 2>&1; then
    return
  fi

  log "removing Kite-managed host accounts and metadata"
  kubectl -n "${KITE_NAMESPACE}" get pods -l app=kite-host-agent -o name 2>/dev/null \
    | while read -r pod; do
        [[ -z "${pod}" ]] && continue
        kubectl -n "${KITE_NAMESPACE}" exec "${pod}" -- nsenter -t 1 -m -u -i -n -p -- sh -c '
          account_root="/var/lib/kite/accounts"
          [ -d "$account_root" ] || exit 0

          for metadata in "$account_root"/*.json; do
            [ -e "$metadata" ] || continue
            username="${metadata##*/}"
            username="${username%.json}"
            if ! printf "%s\n" "$username" | grep -Eq "^[a-z_][a-z0-9_-]{0,31}$"; then
              continue
            fi

            pkill -KILL -u "$username" >/dev/null 2>&1 || true
            userdel -r "$username" >/dev/null 2>&1 || true
            rm -f "$metadata"
          done

          rmdir "$account_root" /var/lib/kite 2>/dev/null || true
        ' >/dev/null 2>&1 || true
      done
}

remove_kite_longhorn_disks() {
  if ! kubectl get crd nodes.longhorn.io >/dev/null 2>&1; then
    return
  fi

  log "removing Kite Longhorn disk entries and tags from Longhorn node resources"
  kubectl -n longhorn-system get nodes.longhorn.io -o name 2>/dev/null \
    | while read -r node; do
        [[ -z "${node}" ]] && continue
        kubectl -n longhorn-system patch "${node}" --type=json -p "[{\"op\":\"remove\",\"path\":\"/spec/disks/${KITE_LONGHORN_DISK_NAME}\"}]" 2>/dev/null || true
        remove_kite_longhorn_disk_tags "${node}"
      done
}

remove_kite_longhorn_disk_tags() {
  local node="$1"
  local disks
  local disk
  local tags
  local next_tags

  disks="$(kubectl -n longhorn-system get "${node}" -o 'go-template={{ range $name, $disk := .spec.disks }}{{ $name }}|{{ range $disk.tags }}{{ . }} {{ end }}{{ "\n" }}{{ end }}' 2>/dev/null || true)"
  while IFS="|" read -r disk tags; do
    [[ -z "${disk}" ]] && continue
    if [[ " ${tags} " != *" ${KITE_LONGHORN_DISK_TAG} "* ]]; then
      continue
    fi

    next_tags="$(printf '%s\n' "${tags}" | xargs -n 1 | awk -v tag="${KITE_LONGHORN_DISK_TAG}" '$0 != tag && NF && !seen[$0]++ { printf "%s\"%s\"", sep, $0; sep=", " }')"
    kubectl -n longhorn-system patch "${node}" --type=json -p "[{\"op\":\"replace\",\"path\":\"/spec/disks/${disk}/tags\",\"value\":[${next_tags}]}]" 2>/dev/null || true
  done <<< "${disks}"
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
  reset_kite_host_dns
  clear_kite_host_accounts
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
