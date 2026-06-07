#!/usr/bin/env bash
set -euo pipefail

KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"

log() {
  echo "[kite-deploy] $*"
}

main() {
  log "checking storage"
  kubectl get storageclass kite-vm-storage
  kubectl -n longhorn-system get daemonset kite-longhorn-disk-directory

  log "checking dependencies"
  kubectl get pods -n longhorn-system
  kubectl get pods -n kubevirt
  kubectl get pods -n cdi

  log "checking Kite APIs"
  kubectl get crd kiteusers.anacnu.com
  kubectl get crd kitevirtualmachines.anacnu.com

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
