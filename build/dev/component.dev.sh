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

options:
  -h, --help    show this help

environment:
  KITE_CLUSTER                auto|minikube|k3s|k3d|kind|k8s|current
  KITE_NAMESPACE              target namespace, default kite
  IMAGE_REGISTRY              image prefix, default kite-dev
  IMAGE_TAG                   image tag, default dev-<timestamp>
  KITE_DEV_SHOW_PLAN          print plan table, default true
  KITE_DEV_DRY_RUN            print steps without running commands, default false
  FRONTEND_VITE_BUILD_MODE    frontend-only build mode, default production
  FRONTEND_VITE_API_BASE_URL  frontend-only API base URL, default /api/v1
  FRONTEND_VITE_USE_MOCK      frontend-only mock mode, default false
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
  if [[ "${PUSH_IMAGES}" == "true" ]]; then
    echo "IfNotPresent"
    return
  fi

  echo "Never"
}

print_plan_row() {
  local label="$1"
  local value="$2"

  printf '  %-16s %s\n' "${label}:" "${value}"
}

print_component_plan() {
  local cluster="$1"
  local component="$2"
  local image

  if [[ "${KITE_DEV_SHOW_PLAN:-true}" != "true" ]]; then
    return
  fi

  image="$(image_name "$(component_image_component "${component}")")"

  echo
  echo "[kite] component deploy plan"
  print_plan_row "component" "${component}"
  print_plan_row "cluster" "${cluster}"
  print_plan_row "namespace" "${KITE_NAMESPACE}"
  print_plan_row "image" "${image}"
  print_plan_row "manifest" "$(component_manifest "${component}")"
  print_plan_row "dockerfile" "$(component_dockerfile "${component}")"
  print_plan_row "context" "$(component_context "${component}")"
  print_plan_row "dry run" "${KITE_DEV_DRY_RUN:-false}"

  if [[ "${component}" == "frontend" ]]; then
    print_plan_row "vite mode" "${FRONTEND_VITE_BUILD_MODE}"
    print_plan_row "vite api" "${FRONTEND_VITE_API_BASE_URL}"
    print_plan_row "vite mock" "${FRONTEND_VITE_USE_MOCK}"
  fi
  echo
}

step_total_for_cluster() {
  if [[ "$1" == "minikube" ]]; then
    echo 4
    return
  fi

  echo 3
}

run_step() {
  local title="$1"
  shift

  STEP_INDEX=$((STEP_INDEX + 1))
  printf '[kite] [%d/%d] %s\n' "${STEP_INDEX}" "${STEP_TOTAL}" "${title}"
  if [[ "${KITE_DEV_DRY_RUN:-false}" == "true" ]]; then
    printf '[kite]       dry-run: %s\n' "$*"
    return
  fi
  "$@"
}

print_done() {
  local component="$1"

  echo
  log "${component} redeploy complete"
  print_plan_row "namespace" "${KITE_NAMESPACE}"
  print_plan_row "image tag" "${IMAGE_TAG}"
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

  if [[ "${component_arg}" == "-h" || "${component_arg}" == "--help" || "${2:-}" == "-h" || "${2:-}" == "--help" ]]; then
    usage
    exit 0
  fi

  if [[ -z "${component_arg}" ]]; then
    usage
    exit 1
  fi

  if [[ "$#" -gt 1 ]]; then
    echo "[kite] unexpected argument: $2" >&2
    usage
    exit 1
  fi

  if [[ "${KITE_DEV_DRY_RUN:-false}" != "true" ]]; then
    require_command kubectl
  fi

  component="$(normalize_component "${component_arg}")"
  cluster="$(detect_cluster)"
  STEP_INDEX=0
  STEP_TOTAL="$(step_total_for_cluster "${cluster}")"

  print_component_plan "${cluster}" "${component}"

  if [[ "${cluster}" == "minikube" ]]; then
    if [[ "${KITE_DEV_DRY_RUN:-false}" != "true" ]]; then
      require_command minikube
    fi
    run_step "start minikube profile" start_minikube
  fi

  run_step "build and load $(component_image_component "${component}") image" build_component_for_cluster "${cluster}" "${component}"
  run_step "apply $(component_deployment "${component}") manifest" apply_component_manifest "${cluster}" "${component}"
  run_step "wait for $(component_deployment "${component}") rollout" wait_for_deployment "$(component_deployment "${component}")"

  print_done "${component}"
}

main "$@"
