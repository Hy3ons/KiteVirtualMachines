#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_SHARED_INFRA_NAMESPACE="${TEST_SHARED_INFRA_NAMESPACE:-kite-protect-external}"
TEST_EXTERNAL_VM_NAME="${TEST_EXTERNAL_VM_NAME:-external-vm}"
TEST_EXTERNAL_PVC_NAME="${TEST_EXTERNAL_PVC_NAME:-external-longhorn-pvc}"
TEST_KITE_NAMESPACE_EXTERNAL_PVC_NAME="${TEST_KITE_NAMESPACE_EXTERNAL_PVC_NAME:-external-kite-namespace-longhorn-pvc}"
TEST_EXTERNAL_PVC_SIZE="${TEST_EXTERNAL_PVC_SIZE:-1Gi}"
TEST_SHARED_INFRA_TIMEOUT="${TEST_SHARED_INFRA_TIMEOUT:-180s}"
TEST_SHARED_INFRA_CLEANUP_TARGET="${TEST_SHARED_INFRA_CLEANUP_TARGET:-uninstall}"
TEST_CLEANUP_EXTERNAL="${TEST_CLEANUP_EXTERNAL:-true}"
EXTERNAL_PV_NAME=""
KITE_NAMESPACE_EXTERNAL_PV_NAME=""

log() {
  printf '[kite-shared-infra-test] %s\n' "$*"
}

warn() {
  printf '[kite-shared-infra-test] WARNING: %s\n' "$*" >&2
}

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

require_cluster_resource() {
  local resource="$1"
  if ! kubectl get "${resource}" >/dev/null 2>&1; then
    warn "required cluster resource is missing: ${resource}"
    exit 1
  fi
}

create_external_resources() {
  log "creating external namespace ${TEST_SHARED_INFRA_NAMESPACE}"
  kubectl create namespace "${TEST_SHARED_INFRA_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

  log "creating external Longhorn PVC ${TEST_EXTERNAL_PVC_NAME}"
  kubectl -n "${TEST_SHARED_INFRA_NAMESPACE}" apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ${TEST_EXTERNAL_PVC_NAME}
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: longhorn
  resources:
    requests:
      storage: ${TEST_EXTERNAL_PVC_SIZE}
EOF
  kubectl -n "${TEST_SHARED_INFRA_NAMESPACE}" wait --for=jsonpath='{.status.phase}'=Bound "pvc/${TEST_EXTERNAL_PVC_NAME}" --timeout="${TEST_SHARED_INFRA_TIMEOUT}"
  EXTERNAL_PV_NAME="$(kubectl -n "${TEST_SHARED_INFRA_NAMESPACE}" get "pvc/${TEST_EXTERNAL_PVC_NAME}" -o jsonpath='{.spec.volumeName}')"
  if [[ -z "${EXTERNAL_PV_NAME}" ]]; then
    warn "external PVC did not bind to a PV"
    exit 1
  fi

  log "creating external Longhorn PVC ${TEST_KITE_NAMESPACE_EXTERNAL_PVC_NAME} in namespace kite"
  kubectl create namespace kite --dry-run=client -o yaml | kubectl apply -f -
  kubectl -n kite apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ${TEST_KITE_NAMESPACE_EXTERNAL_PVC_NAME}
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: longhorn
  resources:
    requests:
      storage: ${TEST_EXTERNAL_PVC_SIZE}
EOF
  kubectl -n kite wait --for=jsonpath='{.status.phase}'=Bound "pvc/${TEST_KITE_NAMESPACE_EXTERNAL_PVC_NAME}" --timeout="${TEST_SHARED_INFRA_TIMEOUT}"
  KITE_NAMESPACE_EXTERNAL_PV_NAME="$(kubectl -n kite get "pvc/${TEST_KITE_NAMESPACE_EXTERNAL_PVC_NAME}" -o jsonpath='{.spec.volumeName}')"
  if [[ -z "${KITE_NAMESPACE_EXTERNAL_PV_NAME}" ]]; then
    warn "external kite namespace PVC did not bind to a PV"
    exit 1
  fi

  log "creating external KubeVirt VM ${TEST_EXTERNAL_VM_NAME}"
  kubectl -n "${TEST_SHARED_INFRA_NAMESPACE}" apply -f - <<EOF
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: ${TEST_EXTERNAL_VM_NAME}
spec:
  running: false
  template:
    metadata:
      labels:
        app: ${TEST_EXTERNAL_VM_NAME}
    spec:
      domain:
        devices:
          disks:
            - name: containerdisk
              disk:
                bus: virtio
        resources:
          requests:
            memory: 64Mi
      volumes:
        - name: containerdisk
          containerDisk:
            image: quay.io/kubevirt/cirros-container-disk-demo:latest
EOF
}

run_kite_cleanup() {
  log "running Kite cleanup target ${TEST_SHARED_INFRA_CLEANUP_TARGET}"
  case "${TEST_SHARED_INFRA_CLEANUP_TARGET}" in
    uninstall)
      env \
        KITE_ASSUME_DEFAULTS=true \
        KITE_UNINSTALL_PRESET=full \
        DELETE_LONGHORN=true \
        DELETE_LONGHORN_FORCE=true \
        DELETE_LONGHORN_DATA=true \
        DELETE_LONGHORN_DATA_CONFIRM=true \
        "${ROOT_DIR}/uninstall.sh"
      ;;
    build-clear)
      env \
        KITE_ASSUME_DEFAULTS=true \
        CLEAR_LONGHORN=true \
        CLEAR_LONGHORN_FORCE=true \
        CLEAR_LONGHORN_DATA=true \
        CLEAR_LONGHORN_DATA_CONFIRM=true \
        "${ROOT_DIR}/build/dev/clear.sh"
      ;;
    *)
      warn "TEST_SHARED_INFRA_CLEANUP_TARGET must be uninstall or build-clear"
      exit 1
      ;;
  esac
}

verify_external_resources_survived() {
  log "checking external resources survived cleanup"
  kubectl get namespace "${TEST_SHARED_INFRA_NAMESPACE}" >/dev/null
  kubectl -n "${TEST_SHARED_INFRA_NAMESPACE}" get "virtualmachines.kubevirt.io/${TEST_EXTERNAL_VM_NAME}" >/dev/null
  kubectl -n "${TEST_SHARED_INFRA_NAMESPACE}" get "pvc/${TEST_EXTERNAL_PVC_NAME}" >/dev/null
  kubectl get "pv/${EXTERNAL_PV_NAME}" >/dev/null
  kubectl -n kite get "pvc/${TEST_KITE_NAMESPACE_EXTERNAL_PVC_NAME}" >/dev/null
  kubectl get "pv/${KITE_NAMESPACE_EXTERNAL_PV_NAME}" >/dev/null
  kubectl get namespace longhorn-system >/dev/null
  kubectl get namespace kubevirt >/dev/null
  kubectl get namespace cdi >/dev/null
  kubectl get crd virtualmachines.kubevirt.io >/dev/null
  kubectl get storageclass longhorn >/dev/null
}

cleanup_external_resources() {
  if [[ "${TEST_CLEANUP_EXTERNAL}" != "true" ]]; then
    log "leaving external namespace ${TEST_SHARED_INFRA_NAMESPACE} for inspection"
    return
  fi
  log "deleting external namespace ${TEST_SHARED_INFRA_NAMESPACE}"
  kubectl delete namespace "${TEST_SHARED_INFRA_NAMESPACE}" --ignore-not-found=true --wait=false
  log "deleting external kite namespace PVC ${TEST_KITE_NAMESPACE_EXTERNAL_PVC_NAME}"
  kubectl -n kite delete pvc "${TEST_KITE_NAMESPACE_EXTERNAL_PVC_NAME}" --ignore-not-found=true --wait=false
}

main() {
  require_command kubectl
  require_cluster_resource namespace/longhorn-system
  require_cluster_resource namespace/kubevirt
  require_cluster_resource namespace/cdi
  require_cluster_resource crd/virtualmachines.kubevirt.io
  require_cluster_resource storageclass/longhorn
  create_external_resources
  run_kite_cleanup
  verify_external_resources_survived
  cleanup_external_resources
  log "shared infrastructure protection test passed"
}

main "$@"
