#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/dev/clear-component.sh
# Description: 선택한 단일 Kite 컴포넌트의 Kubernetes 리소스와 이미지 캐시를 정리한다.
#
# Usage:
#   build/dev/clear-component.sh <api|controller|gateway|frontend>
#
# Environment Variables:
#   없음: 이 wrapper는 인자와 하위 스크립트의 환경변수를 그대로 전달한다.
#
# Side Effects:
#   Kubernetes 리소스, 이미지 캐시, 선택적 Longhorn 상태를 변경하거나 삭제할 수 있다.
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=build/dev/clear.sh
source "${SCRIPT_DIR}/clear.sh"

# 단일 컴포넌트 삭제 CLI의 사용법을 출력한다.
usage() {
  cat >&2 <<'EOF'
usage: build/dev/clear-component.sh <api|controller|gateway|frontend>

examples:
  KITE_CLUSTER=k3s build/dev/clear-component.sh frontend
  CLEAR_IMAGES=false KITE_CLUSTER=k3s build/dev/clear-frontend.sh
EOF
}

# api/kite-api처럼 섞인 입력을 내부 표준 컴포넌트 이름으로 맞춘다.
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

# Kubernetes 리소스와 이미지 이름에서 쓰는 kite-<component> 이름을 만든다.
component_name() {
  echo "kite-$1"
}

# 컴포넌트별 manifest 파일 경로를 반환한다.
component_manifest() {
  echo "${ROOT_DIR}/build/kite/$1.yaml"
}

# Deployment/Service 등 해당 컴포넌트의 Kubernetes 리소스와 남은 Pod를 삭제한다.
delete_component_resources() {
  local component="$1"
  local name

  require_command kubectl

  name="$(component_name "${component}")"
  log "deleting ${name} Kubernetes resources"
  # manifest 삭제 후 label로 남은 Pod도 지워 rollout/termination 잔여 상태를 줄인다.
  kubectl delete -f "$(component_manifest "${component}")" --ignore-not-found=true --wait=false || true
  kubectl -n "${KITE_NAMESPACE}" delete pod -l "app=${name}" --ignore-not-found=true --wait=false || true
}

# 로컬 Docker daemon에 남은 해당 컴포넌트 이미지를 삭제한다.
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
  # tag가 다양한 개발 이미지를 repository 이름 기준으로 찾아 한 번에 제거한다.
  docker image ls --format '{{.Repository}}:{{.Tag}}' \
    | grep -E "(^|/)${name}:" \
    | xargs -r docker rmi -f || true
}

# k3s containerd에 import된 해당 컴포넌트 이미지를 삭제한다.
delete_k3s_component_images() {
  local component="$1"
  local name

  if [[ "${CLEAR_IMAGES}" != "true" ]]; then
    log "skipping k3s image cleanup because CLEAR_IMAGES=${CLEAR_IMAGES}"
    return
  fi

  name="$(component_name "${component}")"
  log "removing k3s containerd images for ${name}"
  # k3s는 Docker가 아니라 containerd k8s.io namespace에 이미지를 보관한다.
  ${K3S_CTR_CMD} images ls -q \
    | grep -E "(^|/)${name}:" \
    | xargs -r -n 1 ${K3S_CTR_CMD} images rm || true
}

# 대상 컴포넌트와 클러스터 종류에 맞춰 Kubernetes 리소스와 이미지 캐시를 정리한다.
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
  if interactive_clear_enabled && [[ -z "${CLEAR_IMAGES_WAS_SET}" ]]; then
    if ask_numbered_bool "로컬/k3s에 남은 ${component} 개발 이미지도 삭제할까요?" "${CLEAR_IMAGES}"; then
      CLEAR_IMAGES=true
    else
      CLEAR_IMAGES=false
    fi
    log "component cleanup choices: CLEAR_IMAGES=${CLEAR_IMAGES}"
  fi

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
