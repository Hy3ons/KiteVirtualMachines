#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/verify.sh
# Description: Kite 설치 후 storage, dependency, workload, golden image 상태를 확인한다.
#
# Usage:
#   build/deploy/scripts/verify.sh
#
# Environment Variables:
#   KITE_NAMESPACE: default kite
#   KITE_LONGHORN_DISK_TAG: default kite
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   주로 상태 조회와 대기를 수행하며, test는 임시 port-forward process를 생성한다.
# ==============================================================================

KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
KITE_LONGHORN_DISK_TAG="${KITE_LONGHORN_DISK_TAG:-kite}"

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


# 각 node에 Ready/Schedulable 상태이고 kite tag가 붙은 Longhorn disk가 있는지 확인한다.
require_kite_longhorn_disk() {
  local node="$1"
  local disks
  local found
  local name
  local allow_scheduling
  local tags
  local ready
  local schedulable

  # Longhorn disk의 spec과 status를 한 번에 읽어 VM disk scheduling 가능 여부를 판단한다.
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

# Kubernetes node 전체를 순회하며 Kite VM용 Longhorn disk 조건을 검증한다.
verify_kite_longhorn_disks() {
  kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' \
    | while read -r node; do
        [[ -z "${node}" ]] && continue
        require_kite_longhorn_disk "${node}"
      done
}

# 설치 후 사람이 보기 쉬운 smoke verification을 순서대로 실행한다.
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
