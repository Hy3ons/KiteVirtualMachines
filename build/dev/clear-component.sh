#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=build/dev/clear.sh
source "${SCRIPT_DIR}/clear.sh"

usage() {
  cat >&2 <<'EOF'
usage: build/dev/clear-component.sh <api|controller|gateway|frontend>

examples:
  KITE_CLUSTER=k3s build/dev/clear-component.sh frontend
  CLEAR_IMAGES=false KITE_CLUSTER=k3s build/dev/clear-frontend.sh
EOF
}

normalize_component() {
  local component="$1"

  case "${component}" in
    api|kite-api)
      echo "api"
      ;;
    controller|kite-controller)
      echo "controller"
      ;;
    gateway|kite-gateway)
      echo "gateway"
      ;;
    frontend|kite-frontend)
      echo "frontend"
      ;;
    *)
      echo "[kite] unknown component=${component}; use api, controller, gateway, or frontend" >&2
      exit 1
      ;;
  esac
}

component_name() {
  echo "kite-$1"
}

component_manifest() {
  echo "${ROOT_DIR}/build/kite/$1.yaml"
}

delete_component_resources() {
  local component="$1"
  local name

  require_command kubectl

  name="$(component_name "${component}")"
  log "deleting ${name} Kubernetes resources"
  kubectl delete -f "$(component_manifest "${component}")" --ignore-not-found=true --wait=false || true
  kubectl -n "${KITE_NAMESPACE}" delete pod -l "app=${name}" --ignore-not-found=true --wait=false || true
}

delete_local_component_images() {
  local component="$1"
  local name

  if [[ "${CLEAR_IMAGES}" != "true" ]]; then
    log "skipping local Docker image cleanup because CLEAR_IMAGES=${CLEAR_IMAGES}"
    return
  fi
  if ! command -v docker >/dev/null 2>&1; then
    log "docker is not installed; skipping local Docker image cleanup"
    return
  fi

  name="$(component_name "${component}")"
  log "removing local Docker images for ${name}"
  docker image ls --format '{{.Repository}}:{{.Tag}}' \
    | grep -E "(^|/)${name}:" \
    | xargs -r docker rmi -f || true
}

delete_k3s_component_images() {
  local component="$1"
  local name

  if [[ "${CLEAR_IMAGES}" != "true" ]]; then
    log "skipping k3s image cleanup because CLEAR_IMAGES=${CLEAR_IMAGES}"
    return
  fi

  name="$(component_name "${component}")"
  log "removing k3s containerd images for ${name}"
  ${K3S_CTR_CMD} images ls -q \
    | grep -E "(^|/)${name}:" \
    | xargs -r -n 1 ${K3S_CTR_CMD} images rm || true
}

main() {
  local component_arg="${1:-${KITE_COMPONENT:-}}"
  local component
  local cluster

  if [[ -z "${component_arg}" ]]; then
    usage
    exit 1
  fi

  component="$(normalize_component "${component_arg}")"
  cluster="$(detect_cluster)"
  log "target cluster=${cluster}"
  log "target component=${component}"

  case "${cluster}" in
    minikube|k3d|kind|current|k8s|kubernetes)
      delete_component_resources "${component}"
      delete_local_component_images "${component}"
      ;;
    k3s)
      delete_component_resources "${component}"
      delete_k3s_component_images "${component}"
      delete_local_component_images "${component}"
      ;;
    *)
      echo "[kite] unknown KITE_CLUSTER=${cluster}; use auto, minikube, k3s, k3d, kind, k8s, kubernetes, or current" >&2
      exit 1
      ;;
  esac

  log "${component} clear complete"
}

main "$@"
