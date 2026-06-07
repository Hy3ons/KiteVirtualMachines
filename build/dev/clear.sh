#!/usr/bin/env bash
set -euo pipefail

# clear.sh removes Kite development resources from the selected local cluster.
# Supported targets:
#   KITE_CLUSTER=minikube build/dev/clear.sh
#   KITE_CLUSTER=k3s build/dev/clear.sh
#   KITE_CLUSTER=k3d build/dev/clear.sh
#   KITE_CLUSTER=kind build/dev/clear.sh
#   KITE_CLUSTER=k8s build/dev/clear.sh
#   KITE_CLUSTER=current build/dev/clear.sh
#
# minikube mode can optionally delete the Minikube profile.
# k3s/current mode deletes only Kite Kubernetes resources and local Kite images, not the cluster.
# Longhorn cleanup is opt-in because it deletes VM disk data.

KITE_CLUSTER="${KITE_CLUSTER:-auto}"
KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MINIKUBE_PROFILE="${MINIKUBE_PROFILE:-minikube}"
MINIKUBE_PURGE="${MINIKUBE_PURGE:-false}"
K3S_CTR_CMD="${K3S_CTR_CMD:-sudo k3s ctr -n k8s.io}"
CLEAR_IMAGES="${CLEAR_IMAGES:-true}"
CLEAR_LONGHORN="${CLEAR_LONGHORN:-false}"
CLEAR_LONGHORN_FORCE="${CLEAR_LONGHORN_FORCE:-false}"
CLEAR_LONGHORN_DATA="${CLEAR_LONGHORN_DATA:-false}"
CLEAR_LONGHORN_DATA_CONFIRM="${CLEAR_LONGHORN_DATA_CONFIRM:-false}"
KITE_LONGHORN_DISK_NAME="${KITE_LONGHORN_DISK_NAME:-kite-longhorn}"
KITE_LONGHORN_DISK_TAG="${KITE_LONGHORN_DISK_TAG:-kite}"

log() {
  echo "[kite] $*"
}

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "[kite] missing required command: ${name}" >&2
    exit 1
  fi
}

detect_cluster() {
  local context

  if [[ "${KITE_CLUSTER}" != "auto" ]]; then
    echo "${KITE_CLUSTER}"
    return
  fi

  context="$(kubectl config current-context 2>/dev/null || true)"
  case "${context}" in
    minikube|*minikube*)
      echo "minikube"
      ;;
    *k3d*)
      echo "k3d"
      ;;
    *k3s*)
      echo "k3s"
      ;;
    kind-*|*kind*)
      echo "kind"
      ;;
    *)
      if command -v k3s >/dev/null 2>&1; then
        echo "k3s"
      else
        echo "current"
      fi
      ;;
  esac
}

delete_kite_resources() {
  require_command kubectl

  log "deleting Kite storage manifests"
  kubectl delete -f "${ROOT_DIR}/build/kite-storage/golden-images" --ignore-not-found=true || true
  kubectl delete -f "${ROOT_DIR}/build/kite-storage/longhorn" --ignore-not-found=true || true
  remove_kite_longhorn_disks

  log "stopping Kite workloads"
  kubectl delete -f "${ROOT_DIR}/build/kite/host-agent.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/controller.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/api.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/frontend.yaml" --ignore-not-found=true --wait=false || true

  log "deleting Kite custom resources"
  delete_kite_custom_resources

  log "deleting Kite shared manifests"
  kubectl delete -f "${ROOT_DIR}/build/kite/config.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/rbac.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/serviceaccount.yaml" --ignore-not-found=true --wait=false || true

  log "deleting Kite namespace-scoped resources from namespace/${KITE_NAMESPACE}"
  kubectl delete namespace "${KITE_NAMESPACE}" --ignore-not-found=true --wait=false || true

  log "deleting remaining Kite cluster-scoped resources"
  kubectl delete -f "${ROOT_DIR}/build/kite/crds.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete clusterrole kite-control-plane-role --ignore-not-found=true --wait=false || true
  kubectl delete clusterrolebinding kite-control-plane-binding --ignore-not-found=true --wait=false || true
}

delete_kite_custom_resources() {
  if kubectl get crd kitevirtualmachines.hy3ons.github.io >/dev/null 2>&1; then
    kubectl get kitevirtualmachines.hy3ons.github.io -A -o jsonpath='{range .items[*]}{.metadata.namespace}{" "}{.metadata.name}{"\n"}{end}' 2>/dev/null \
      | while read -r namespace name; do
          [[ -z "${namespace}" || -z "${name}" ]] && continue
          kubectl -n "${namespace}" patch kitevirtualmachines.hy3ons.github.io "${name}" --type=merge -p '{"metadata":{"finalizers":[]}}' || true
        done
    kubectl delete kitevirtualmachines.hy3ons.github.io -A --all --ignore-not-found=true --wait=false || true
  fi

  if kubectl get crd kiteusers.hy3ons.github.io >/dev/null 2>&1; then
    kubectl get kiteusers.hy3ons.github.io -o name 2>/dev/null \
      | xargs -r -n 1 kubectl patch --type=merge -p '{"metadata":{"finalizers":[]}}' || true
    kubectl delete kiteusers.hy3ons.github.io --all --ignore-not-found=true --wait=false || true
  fi
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
  if [[ "${CLEAR_LONGHORN_DATA}" != "true" ]]; then
    log "skipping Longhorn host data cleanup because CLEAR_LONGHORN_DATA=${CLEAR_LONGHORN_DATA}"
    return
  fi
  if [[ "${CLEAR_LONGHORN_DATA_CONFIRM}" != "true" ]]; then
    echo "[kite] refusing Longhorn host data deletion without CLEAR_LONGHORN_DATA_CONFIRM=true" >&2
    exit 1
  fi
  if [[ "${CLEAR_LONGHORN_FORCE}" != "true" ]]; then
    local pv_count
    pv_count="$(longhorn_pv_count)"
    if [[ "${pv_count}" != "0" ]]; then
      log "skipping Kite Longhorn host data cleanup because ${pv_count} Longhorn PV(s) still exist"
      log "delete remaining Longhorn PVC/PV resources first, or set CLEAR_LONGHORN_FORCE=true"
      return
    fi
  fi

  log "deleting Kite Longhorn host data on every node"
  kubectl apply -f "${ROOT_DIR}/build/kite-storage/longhorn-cleanup"
  kubectl -n longhorn-system rollout status daemonset/kite-longhorn-disk-cleanup --timeout=180s || true
  kubectl delete -f "${ROOT_DIR}/build/kite-storage/longhorn-cleanup" --ignore-not-found=true || true
}

delete_longhorn_resources() {
  if [[ "${CLEAR_LONGHORN}" != "true" ]]; then
    log "skipping Longhorn cleanup because CLEAR_LONGHORN=${CLEAR_LONGHORN}"
    return
  fi

  require_command kubectl
  remove_kite_longhorn_disks

  if [[ "${CLEAR_LONGHORN_FORCE}" != "true" ]]; then
    local pv_count
    pv_count="$(longhorn_pv_count)"
    if [[ "${pv_count}" != "0" ]]; then
      log "skipping Longhorn uninstall because ${pv_count} Longhorn PV(s) still exist"
      log "delete remaining Longhorn PVC/PV resources first, or set CLEAR_LONGHORN_FORCE=true"
      return
    fi
  fi

  log "deleting Longhorn workloads and custom resources"
  kubectl delete storageclass longhorn --ignore-not-found=true || true
  kubectl delete namespace longhorn-system --ignore-not-found=true --wait=false || true

  log "removing Longhorn finalizers from terminating resources when present"
  kubectl api-resources --api-group=longhorn.io --verbs=list -o name 2>/dev/null \
    | while read -r resource; do
        [[ -z "${resource}" ]] && continue
        kubectl get "${resource}" -A -o name 2>/dev/null \
          | xargs -r -n 1 kubectl patch --type=merge -p '{"metadata":{"finalizers":[]}}' || true
      done
}

delete_local_docker_images() {
  if [[ "${CLEAR_IMAGES}" != "true" ]]; then
    log "skipping local Docker image cleanup because CLEAR_IMAGES=${CLEAR_IMAGES}"
    return
  fi
  if ! command -v docker >/dev/null 2>&1; then
    log "docker is not installed; skipping local Docker image cleanup"
    return
  fi

  log "removing local Docker Kite images"
  docker image ls --format '{{.Repository}}:{{.Tag}}' \
    | grep -E '(^|/)kite-(api|controller|host-agent|frontend):' \
    | xargs -r docker rmi -f
}

delete_k3s_images() {
  if [[ "${CLEAR_IMAGES}" != "true" ]]; then
    log "skipping k3s image cleanup because CLEAR_IMAGES=${CLEAR_IMAGES}"
    return
  fi

  log "removing k3s containerd Kite images"
  ${K3S_CTR_CMD} images ls -q \
    | grep -E '(^|/)kite-(api|controller|host-agent|frontend):' \
    | xargs -r -n 1 ${K3S_CTR_CMD} images rm || true
}

delete_minikube_profile() {
  require_command minikube

  if [[ "${MINIKUBE_PURGE}" == "true" ]]; then
    log "deleting every minikube cluster and purging local minikube state"
    minikube delete --all --purge
    return
  fi

  log "deleting minikube profile=${MINIKUBE_PROFILE}"
  minikube -p "${MINIKUBE_PROFILE}" delete
}

main() {
  local cluster

  cluster="$(detect_cluster)"
  log "target cluster=${cluster}"

  case "${cluster}" in
    minikube)
      delete_minikube_profile
      delete_local_docker_images
      ;;
    k3s)
      delete_kite_resources
      delete_kite_longhorn_host_data
      delete_longhorn_resources
      delete_k3s_images
      delete_local_docker_images
      ;;
    k3d|kind|current|k8s|kubernetes)
      delete_kite_resources
      delete_kite_longhorn_host_data
      delete_longhorn_resources
      delete_local_docker_images
      ;;
    *)
      echo "[kite] unknown KITE_CLUSTER=${cluster}; use auto, minikube, k3s, k3d, kind, k8s, kubernetes, or current" >&2
      exit 1
      ;;
  esac

  log "clear complete"
}

main "$@"
