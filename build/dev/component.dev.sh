#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/dev/component.dev.sh
# Description: 선택한 단일 Kite 컴포넌트를 로컬 빌드 후 현재 클러스터에 재배포한다.
#
# Usage:
#   build/dev/component.dev.sh <api|controller|gateway|frontend>
#
# Environment Variables:
#   없음: 이 wrapper는 인자와 하위 스크립트의 환경변수를 그대로 전달한다.
#
# Side Effects:
#   Kubernetes 리소스 적용, 컨테이너 이미지 빌드/주입, rollout 대기를 수행할 수 있다.
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KITE_DEV_DRY_RUN_WAS_SET="${KITE_DEV_DRY_RUN+x}"
# shellcheck source=build/dev/dev.sh
source "${SCRIPT_DIR}/dev.sh"

# 단일 컴포넌트 재배포 CLI의 사용법을 출력한다.
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

# 사용자가 api/kite-api처럼 섞어 입력해도 내부 표준 이름으로 맞춘다.
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

# Docker image 이름에 들어갈 컴포넌트 이름을 만든다.
component_image_component() {
  echo "kite-$1"
}

# Kubernetes Deployment 이름은 현재 매니페스트에서 kite-<component> 형식을 사용한다.
component_deployment() {
  echo "kite-$1"
}

# 컴포넌트별 원본 manifest 경로를 반환한다.
component_manifest() {
  echo "${KITE_MANIFEST_DIR}/$1.yaml"
}

# 컴포넌트별 Dockerfile 경로를 반환한다.
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

# Docker build context는 Go 컴포넌트와 frontend가 서로 다르다.
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

# 선택한 클러스터 종류에 맞춰 이미지 빌드와 이미지 주입/import를 수행한다.
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
    # frontend는 Vite build arg가 이미지 안의 API 경로와 mock 여부를 결정한다.
    build_args=(
      --build-arg "VITE_BUILD_MODE=${FRONTEND_VITE_BUILD_MODE}"
      --build-arg "VITE_API_BASE_URL=${FRONTEND_VITE_API_BASE_URL}"
      --build-arg "VITE_USE_MOCK=${FRONTEND_VITE_USE_MOCK}"
    )
  fi

  case "${cluster}" in
    minikube)
      build_minikube_image "${image}" "${dockerfile}" "${context}" "${build_args[@]+"${build_args[@]}"}"
      ;;
    k3s)
      build_local_image "${image}" "${dockerfile}" "${context}" "${build_args[@]+"${build_args[@]}"}"
      load_image_into_k3s "${image}"
      ;;
    k3d)
      build_local_image "${image}" "${dockerfile}" "${context}" "${build_args[@]+"${build_args[@]}"}"
      load_image_into_k3d "${image}"
      ;;
    kind)
      build_local_image "${image}" "${dockerfile}" "${context}" "${build_args[@]+"${build_args[@]}"}"
      load_image_into_kind "${image}"
      ;;
    current|k8s|kubernetes)
      build_local_image "${image}" "${dockerfile}" "${context}" "${build_args[@]+"${build_args[@]}"}"
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

# registry에 push하는 모드면 kubelet이 pull할 수 있게 IfNotPresent를 쓰고, 로컬 주입이면 Never를 쓴다.
pull_policy_for_cluster() {
  if [[ "${PUSH_IMAGES}" == "true" ]]; then
    echo "IfNotPresent"
    return
  fi

  echo "Never"
}

# plan/done 출력의 label 폭을 맞춰 사람이 스캔하기 쉽게 만든다.
print_plan_row() {
  local label="$1"
  local value="$2"

  printf '  %-16s %s\n' "${label}:" "${value}"
}

# dry-run 또는 재배포 전 확인용으로 어떤 이미지/manifest가 쓰일지 출력한다.
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

# minikube는 profile start 단계가 하나 더 있어 진행 단계 수가 다르다.
step_total_for_cluster() {
  if [[ "$1" == "minikube" ]]; then
    echo 4
    return
  fi

  echo 3
}

# 단계 번호를 출력하고 dry-run이면 실제 명령 대신 실행 예정 명령만 보여준다.
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

# 컴포넌트 재배포가 끝난 뒤 namespace와 image tag를 요약한다.
print_done() {
  local component="$1"

  echo
  log "${component} redeploy complete"
  print_plan_row "namespace" "${KITE_NAMESPACE}"
  print_plan_row "image tag" "${IMAGE_TAG}"
}

# namespace, 공통 RBAC/config, 컴포넌트 manifest를 적용하고 Deployment image를 교체한다.
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

  # API/controller/gateway는 Kubernetes API 접근 권한과 공통 config가 필요하다.
  if [[ "${component}" == "api" || "${component}" == "controller" || "${component}" == "gateway" ]]; then
    kubectl apply -f "${KITE_MANIFEST_DIR}/serviceaccount.yaml"
    kubectl apply -f "${KITE_MANIFEST_DIR}/config.yaml"
    kubectl apply -f "${KITE_MANIFEST_DIR}/rbac.yaml"
  fi

  # gateway는 SSH 서버로 동작하므로 host key Secret이 먼저 있어야 한다.
  if [[ "${component}" == "gateway" ]]; then
    "${ROOT_DIR}/build/deploy/scripts/ensure-gateway-host-key-secret.sh"
  fi

  kubectl apply -f "$(component_manifest "${component}")"

  log "setting deployment/${deployment} image=${image}"
  # set image로 새 이미지를 반영하고, pullPolicy를 cluster별 이미지 주입 방식에 맞춘다.
  kubectl -n "${KITE_NAMESPACE}" set image "deployment/${deployment}" "${container}=${image}"
  kubectl -n "${KITE_NAMESPACE}" patch "deployment/${deployment}" --type=strategic \
    -p "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"${container}\",\"imagePullPolicy\":\"${pull_policy}\"}]}}}}"
}

# 단일 컴포넌트 재배포의 전체 흐름이다. 입력 검증, 클러스터 감지, 빌드, 적용, rollout 대기를 수행한다.
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
  if kite_prompt_interactive; then
    kite_prompt_configure_bool KITE_DEV_DRY_RUN "${KITE_DEV_DRY_RUN_WAS_SET}" "실제 배포 없이 계획만 출력하는 dry-run으로 실행할까요?"
  fi
  configure_interactive_dev_options "${cluster}" "${component}"
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
