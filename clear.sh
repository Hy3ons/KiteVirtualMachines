#!/usr/bin/env bash
set -euo pipefail

# clear.sh removes Kite development resources from the selected local cluster.
# Supported targets:
#   KITE_CLUSTER=minikube ./clear.sh
#   KITE_CLUSTER=k3s ./clear.sh
#   KITE_CLUSTER=current ./clear.sh
#
# minikube mode can optionally delete the Minikube profile.
# k3s/current mode deletes only Kite Kubernetes resources and local Kite images, not the cluster.

KITE_CLUSTER="${KITE_CLUSTER:-auto}"
KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
MINIKUBE_PROFILE="${MINIKUBE_PROFILE:-minikube}"
MINIKUBE_PURGE="${MINIKUBE_PURGE:-false}"
K3S_CTR_CMD="${K3S_CTR_CMD:-sudo k3s ctr -n k8s.io}"
CLEAR_IMAGES="${CLEAR_IMAGES:-true}"

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
    *k3s*|*k3d*)
      echo "k3s"
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

  log "deleting Kite namespace-scoped resources from namespace/${KITE_NAMESPACE}"
  kubectl delete namespace "${KITE_NAMESPACE}" --ignore-not-found=true

  log "deleting Kite cluster-scoped resources"
  kubectl delete crd kiteusers.anacnu.com kitevirtualmachines.anacnu.com --ignore-not-found=true
  kubectl delete clusterrole kite-control-plane-role --ignore-not-found=true
  kubectl delete clusterrolebinding kite-control-plane-binding --ignore-not-found=true
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
      delete_k3s_images
      delete_local_docker_images
      ;;
    current)
      delete_kite_resources
      delete_local_docker_images
      ;;
    *)
      echo "[kite] unknown KITE_CLUSTER=${cluster}; use auto, minikube, k3s, or current" >&2
      exit 1
      ;;
  esac

  log "clear complete"
}

main "$@"
