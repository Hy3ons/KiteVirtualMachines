#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=build/dev/dev.sh
source "${SCRIPT_DIR}/dev.sh"

usage() {
  cat >&2 <<'EOF'
usage: build/dev/component.dev.sh <api|controller|gateway|frontend>

examples:
  KITE_CLUSTER=k3s build/dev/component.dev.sh frontend
  IMAGE_TAG=qa-frontend KITE_CLUSTER=k3s build/dev/frontend.dev.sh
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

component_image_component() {
  echo "kite-$1"
}

component_deployment() {
  echo "kite-$1"
}

component_manifest() {
  echo "${KITE_MANIFEST_DIR}/$1.yaml"
}

component_dockerfile() {
  case "$1" in
    api)
      echo "${ROOT_DIR}/kite/Dockerfile.api"
      ;;
    controller)
      echo "${ROOT_DIR}/kite/Dockerfile.controller"
      ;;
    gateway)
      echo "${ROOT_DIR}/kite/Dockerfile.gateway"
      ;;
    frontend)
      echo "${ROOT_DIR}/kite-frontend/Dockerfile"
      ;;
  esac
}

component_context() {
  case "$1" in
    api|controller|gateway)
      echo "${ROOT_DIR}/kite"
      ;;
    frontend)
      echo "${ROOT_DIR}/kite-frontend"
      ;;
  esac
}

build_component_for_cluster() {
  local cluster="$1"
  local component="$2"
  local image
  local dockerfile
  local context
  local -a build_args=()

  image="$(image_name "$(component_image_component "${component}")")"
  dockerfile="$(component_dockerfile "${component}")"
  context="$(component_context "${component}")"

  if [[ "${component}" == "frontend" ]]; then
    build_args=(
      --build-arg "VITE_BUILD_MODE=${FRONTEND_VITE_BUILD_MODE}"
      --build-arg "VITE_API_BASE_URL=${FRONTEND_VITE_API_BASE_URL}"
      --build-arg "VITE_USE_MOCK=${FRONTEND_VITE_USE_MOCK}"
    )
  fi

  case "${cluster}" in
    minikube)
      build_minikube_image "${image}" "${dockerfile}" "${context}" "${build_args[@]}"
      ;;
    k3s)
      build_local_image "${image}" "${dockerfile}" "${context}" "${build_args[@]}"
      load_image_into_k3s "${image}"
      ;;
    k3d)
      build_local_image "${image}" "${dockerfile}" "${context}" "${build_args[@]}"
      load_image_into_k3d "${image}"
      ;;
    kind)
      build_local_image "${image}" "${dockerfile}" "${context}" "${build_args[@]}"
      load_image_into_kind "${image}"
      ;;
    current|k8s|kubernetes)
      build_local_image "${image}" "${dockerfile}" "${context}" "${build_args[@]}"
      push_local_image "${image}"
      if [[ "${PUSH_IMAGES}" != "true" ]]; then
        log "using locally built ${image}; set PUSH_IMAGES=true when the cluster must pull from a registry"
      fi
      ;;
    *)
      echo "[kite] unknown KITE_CLUSTER=${cluster}; use auto, minikube, k3s, k3d, kind, k8s, kubernetes, or current" >&2
      exit 1
      ;;
  esac
}

pull_policy_for_cluster() {
  case "$1" in
    minikube|k3s|k3d|kind)
      echo "Never"
      ;;
    *)
      echo "IfNotPresent"
      ;;
  esac
}

apply_component_manifest() {
  local cluster="$1"
  local component="$2"
  local deployment
  local container
  local image
  local pull_policy

  require_command kubectl

  deployment="$(component_deployment "${component}")"
  container="$(component_image_component "${component}")"
  image="$(image_name "${container}")"
  pull_policy="$(pull_policy_for_cluster "${cluster}")"

  log "applying namespace and ${component} manifest"
  kubectl apply -f "${KITE_MANIFEST_DIR}/namespace.yaml"

  if [[ "${component}" == "api" || "${component}" == "controller" || "${component}" == "gateway" ]]; then
    kubectl apply -f "${KITE_MANIFEST_DIR}/serviceaccount.yaml"
    kubectl apply -f "${KITE_MANIFEST_DIR}/config.yaml"
    kubectl apply -f "${KITE_MANIFEST_DIR}/rbac.yaml"
  fi

  if [[ "${component}" == "gateway" ]]; then
    "${ROOT_DIR}/build/deploy/scripts/ensure-gateway-host-key-secret.sh"
  fi

  kubectl apply -f "$(component_manifest "${component}")"

  log "setting deployment/${deployment} image=${image}"
  kubectl -n "${KITE_NAMESPACE}" set image "deployment/${deployment}" "${container}=${image}"
  kubectl -n "${KITE_NAMESPACE}" patch "deployment/${deployment}" --type=strategic \
    -p "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"${container}\",\"imagePullPolicy\":\"${pull_policy}\"}]}}}}"
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

  if [[ "${cluster}" == "minikube" ]]; then
    require_command minikube
    start_minikube
  fi

  build_component_for_cluster "${cluster}" "${component}"
  apply_component_manifest "${cluster}" "${component}"
  wait_for_deployment "$(component_deployment "${component}")"

  log "${component} redeploy complete"
  echo "  namespace: ${KITE_NAMESPACE}"
  echo "  image tag: ${IMAGE_TAG}"
}

main "$@"
