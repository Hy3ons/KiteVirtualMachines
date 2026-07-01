#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/dev/dev.sh
# Description: 로컬 소스 트리에서 모든 Kite 이미지를 빌드하고 클러스터에 배포한다.
#
# Usage:
#   build/dev/dev.sh
#
# Environment Variables:
#   KITE_NAMESPACE: default kite
#   KITE_CLUSTER: default auto
#   IMAGE_REGISTRY: default kite-dev
#   IMAGE_TAG: default dev-$(date +%Y%m%d%H%M%S)
#   PUSH_IMAGES: default false
#   FRONTEND_VITE_BUILD_MODE: default production
#   FRONTEND_VITE_API_BASE_URL: default /api/v1
#   FRONTEND_VITE_USE_MOCK: default false
#   MINIKUBE_PROFILE: default minikube
#   MINIKUBE_DRIVER: default (empty)
#   MINIKUBE_CPUS: default 4
#   MINIKUBE_MEMORY: default 8192
#   MINIKUBE_START: default true
#   MINIKUBE_BUILD_STRATEGY: default BUILD_STRATEGY 값 또는 auto
#   K3S_CTR_CMD: default sudo k3s ctr -n k8s.io
#   K3S_IMPORT_IMAGES: default true
#   K3D_CLUSTER_NAME: default (empty)
#   K3D_LOAD_IMAGES: default true
#   KIND_CLUSTER_NAME: default (empty)
#   KIND_LOAD_IMAGES: default true
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   Kubernetes 리소스 적용, 컨테이너 이미지 빌드/주입, rollout 대기를 수행할 수 있다.
# ==============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PUSH_IMAGES_WAS_SET="${PUSH_IMAGES+x}"
FRONTEND_VITE_USE_MOCK_WAS_SET="${FRONTEND_VITE_USE_MOCK+x}"
MINIKUBE_START_WAS_SET="${MINIKUBE_START+x}"
K3S_IMPORT_IMAGES_WAS_SET="${K3S_IMPORT_IMAGES+x}"
K3D_LOAD_IMAGES_WAS_SET="${K3D_LOAD_IMAGES+x}"
KIND_LOAD_IMAGES_WAS_SET="${KIND_LOAD_IMAGES+x}"
KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
KITE_CLUSTER="${KITE_CLUSTER:-auto}"
IMAGE_REGISTRY="${IMAGE_REGISTRY:-kite-dev}"
IMAGE_TAG="${IMAGE_TAG:-dev-$(date +%Y%m%d%H%M%S)}"
KITE_MANIFEST_DIR="${ROOT_DIR}/build/kite"
TMP_ROOT="${TMPDIR:-/tmp}"
TMP_ROOT="${TMP_ROOT%/}"
KUSTOMIZE_OVERLAY_DIR="$(mktemp -d "${ROOT_DIR}/build/dev/.kustomize.XXXXXX")"
RENDERED_MANIFEST="$(mktemp "${TMP_ROOT}/kite-install.XXXXXX")"
PUSH_IMAGES="${PUSH_IMAGES:-false}"
FRONTEND_VITE_BUILD_MODE="${FRONTEND_VITE_BUILD_MODE:-production}"
FRONTEND_VITE_API_BASE_URL="${FRONTEND_VITE_API_BASE_URL:-/api/v1}"
FRONTEND_VITE_USE_MOCK="${FRONTEND_VITE_USE_MOCK:-false}"

# Minikube 옵션은 KITE_CLUSTER=minikube이거나 auto가 minikube context를 감지할 때만 사용된다.
MINIKUBE_PROFILE="${MINIKUBE_PROFILE:-minikube}"
MINIKUBE_DRIVER="${MINIKUBE_DRIVER:-}"
MINIKUBE_CPUS="${MINIKUBE_CPUS:-4}"
MINIKUBE_MEMORY="${MINIKUBE_MEMORY:-8192}"
MINIKUBE_START="${MINIKUBE_START:-true}"
MINIKUBE_BUILD_STRATEGY="${MINIKUBE_BUILD_STRATEGY:-${BUILD_STRATEGY:-auto}}"

# k3s는 Kubernetes가 k8s.io containerd namespace의 이미지를 읽으므로 그 namespace로 import한다.
# sudo가 필요 없는 환경에서는 다음처럼 덮어쓴다:
#   K3S_CTR_CMD="k3s ctr -n k8s.io" KITE_CLUSTER=k3s build/dev/dev.sh
K3S_CTR_CMD="${K3S_CTR_CMD:-sudo k3s ctr -n k8s.io}"
K3S_IMPORT_IMAGES="${K3S_IMPORT_IMAGES:-true}"
K3D_CLUSTER_NAME="${K3D_CLUSTER_NAME:-}"
K3D_LOAD_IMAGES="${K3D_LOAD_IMAGES:-true}"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-}"
KIND_LOAD_IMAGES="${KIND_LOAD_IMAGES:-true}"

# shellcheck source=build/scripts/prompt.sh
source "${ROOT_DIR}/build/scripts/prompt.sh"

cleanup() {
  rm -rf "${KUSTOMIZE_OVERLAY_DIR}"
  rm -f "${RENDERED_MANIFEST}"
}
trap cleanup EXIT

# 공통 로그 prefix를 붙인다.
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
    printf "\033[0;32m[kite] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}


# 외부 CLI 누락을 명확한 메시지로 중단한다.
require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

# registry/repository/tag 조합을 한 곳에서 만든다.
image_name() {
  local component="$1"
  echo "${IMAGE_REGISTRY}/${component}:${IMAGE_TAG}"
}

# kube context와 로컬 명령 존재 여부를 보고 대상 클러스터 종류를 추정한다.
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

configure_interactive_dev_options() {
  local cluster="$1"
  local component="${2:-all}"

  kite_prompt_interactive || return 0

  log "interactive dev deploy options"
  if [[ "${component}" == "all" || "${component}" == "frontend" ]]; then
    kite_prompt_configure_bool FRONTEND_VITE_USE_MOCK "${FRONTEND_VITE_USE_MOCK_WAS_SET}" "frontend 이미지를 mock API 모드로 빌드할까요?"
  fi

  case "${cluster}" in
    minikube)
      kite_prompt_configure_bool MINIKUBE_START "${MINIKUBE_START_WAS_SET}" "배포 전에 minikube profile을 시작/갱신할까요?"
      ;;
    k3s)
      kite_prompt_configure_bool K3S_IMPORT_IMAGES "${K3S_IMPORT_IMAGES_WAS_SET}" "빌드한 이미지를 k3s containerd로 import할까요?"
      ;;
    k3d)
      kite_prompt_configure_bool K3D_LOAD_IMAGES "${K3D_LOAD_IMAGES_WAS_SET}" "빌드한 이미지를 k3d cluster로 load할까요?"
      ;;
    kind)
      kite_prompt_configure_bool KIND_LOAD_IMAGES "${KIND_LOAD_IMAGES_WAS_SET}" "빌드한 이미지를 kind cluster로 load할까요?"
      ;;
    current|k8s|kubernetes)
      kite_prompt_configure_bool PUSH_IMAGES "${PUSH_IMAGES_WAS_SET}" "현재 클러스터가 이미지를 pull할 수 있도록 registry에 push할까요?"
      ;;
  esac

  log "dev deploy choices: FRONTEND_VITE_USE_MOCK=${FRONTEND_VITE_USE_MOCK}, PUSH_IMAGES=${PUSH_IMAGES}, MINIKUBE_START=${MINIKUBE_START}, K3S_IMPORT_IMAGES=${K3S_IMPORT_IMAGES}, K3D_LOAD_IMAGES=${K3D_LOAD_IMAGES}, KIND_LOAD_IMAGES=${KIND_LOAD_IMAGES}"
}

# minikube image build 전략을 고를 때 현재 profile의 driver를 확인한다.
minikube_driver() {
  local driver

  driver="$(minikube -p "${MINIKUBE_PROFILE}" config get driver 2>/dev/null || true)"
  if [[ -n "${driver}" ]]; then
    echo "${driver}"
    return
  fi

  minikube -p "${MINIKUBE_PROFILE}" profile list -o json 2>/dev/null \
    | tr '\n' ' ' \
    | sed -n 's/.*"Driver"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'
}

# macOS/Darwin 여부는 minikube build fallback 선택에 영향을 준다.
host_os() {
  uname -s 2>/dev/null || echo unknown
}

# 필요하면 Minikube profile을 시작하고 kubectl context를 해당 profile로 맞춘다.
start_minikube() {
  local args=(
    start
    -p "${MINIKUBE_PROFILE}"
    --cpus="${MINIKUBE_CPUS}"
    --memory="${MINIKUBE_MEMORY}"
  )

  if [[ "${MINIKUBE_START}" != "true" ]]; then
    log "skipping minikube start because MINIKUBE_START=${MINIKUBE_START}"
    minikube -p "${MINIKUBE_PROFILE}" update-context
    return
  fi

  if [[ -n "${MINIKUBE_DRIVER}" ]]; then
    args+=(--driver="${MINIKUBE_DRIVER}")
  fi

  log "starting minikube profile=${MINIKUBE_PROFILE}"
  minikube "${args[@]}"
  minikube -p "${MINIKUBE_PROFILE}" update-context
}

# 로컬 Docker daemon으로 이미지를 빌드한다.
build_local_image() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  shift 3

  require_command docker
  log "building ${image}"
  DOCKER_BUILDKIT=1 docker build --progress=plain "$@" -t "${image}" -f "${dockerfile}" "${context}"
}

# 원격 클러스터가 registry에서 이미지를 가져가야 할 때만 push한다.
push_local_image() {
  local image="$1"

  if [[ "${PUSH_IMAGES}" != "true" ]]; then
    return
  fi

  require_command docker
  log "pushing ${image}"
  docker push "${image}"
}

# Minikube 내부 builder를 사용해 profile runtime에 직접 이미지를 만든다.
build_minikube_image_with_minikube() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  shift 3

  minikube -p "${MINIKUBE_PROFILE}" image build "$@" -t "${image}" -f "${dockerfile}" "${context}"
}

# Minikube docker-env로 로컬 docker CLI를 profile 내부 Docker daemon에 연결해 빌드한다.
build_minikube_image_with_docker_env() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  shift 3
  local status

  require_command docker
  # docker-env는 현재 shell의 Docker 환경변수를 바꾸므로 성공/실패 후 반드시 unset한다.
  eval "$(minikube -p "${MINIKUBE_PROFILE}" docker-env)"
  if DOCKER_BUILDKIT=1 docker build --progress=plain "$@" -t "${image}" -f "${dockerfile}" "${context}"; then
    status=0
  else
    status=$?
  fi
  eval "$(minikube -p "${MINIKUBE_PROFILE}" docker-env --unset)"
  return "${status}"
}

# 로컬 Docker에서 빌드한 뒤 minikube image load로 profile runtime에 주입한다.
build_minikube_image_with_local_load() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  shift 3

  build_local_image "${image}" "${dockerfile}" "${context}" "$@"
  minikube -p "${MINIKUBE_PROFILE}" image load "${image}"
}

# Minikube driver/OS별로 가장 성공 가능성이 높은 빌드 전략을 고르고 실패 시 fallback한다.
build_minikube_image() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  shift 3
  local driver
  local os_name

  driver="$(minikube_driver || true)"
  os_name="$(host_os)"
  log "build strategy=${MINIKUBE_BUILD_STRATEGY}, minikube driver=${driver:-unknown}, host=${os_name}"

  case "${MINIKUBE_BUILD_STRATEGY}" in
    minikube)
      build_minikube_image_with_minikube "${image}" "${dockerfile}" "${context}" "$@"
      return
      ;;
    docker-env)
      build_minikube_image_with_docker_env "${image}" "${dockerfile}" "${context}" "$@"
      return
      ;;
    local-load)
      build_minikube_image_with_local_load "${image}" "${dockerfile}" "${context}" "$@"
      return
      ;;
    auto)
      ;;
    *)
      echo "[kite] unknown MINIKUBE_BUILD_STRATEGY=${MINIKUBE_BUILD_STRATEGY}; use auto, minikube, docker-env, or local-load" >&2
      exit 1
      ;;
  esac

  # qemu/qemu2와 macOS는 minikube image build가 VM 안에서 /Users 경로를 해석하다 실패하기 쉽다.
  if [[ "${driver}" == "qemu" || "${driver}" == "qemu2" || "${os_name}" == "Darwin" ]]; then
    log "using docker-env build to avoid Minikube VM host path issues"
    if build_minikube_image_with_docker_env "${image}" "${dockerfile}" "${context}" "$@"; then
      return
    fi

    log "docker-env build failed for ${image}; retrying local build plus minikube image load"
    build_minikube_image_with_local_load "${image}" "${dockerfile}" "${context}" "$@"
    return
  fi

  if build_minikube_image_with_minikube "${image}" "${dockerfile}" "${context}" "$@"; then
    return
  fi

  log "minikube image build failed for ${image}; retrying docker-env"
  if build_minikube_image_with_docker_env "${image}" "${dockerfile}" "${context}" "$@"; then
    return
  fi

  log "docker-env build failed for ${image}; retrying local build plus minikube image load"
  build_minikube_image_with_local_load "${image}" "${dockerfile}" "${context}" "$@"
}

# k3s containerd의 k8s.io namespace로 Docker image tar를 import한다.
load_image_into_k3s() {
  local image="$1"

  if [[ "${K3S_IMPORT_IMAGES}" != "true" ]]; then
    log "skipping k3s image import for ${image}"
    return
  fi

  require_command docker
  log "importing ${image} into k3s containerd"
  # docker save 출력 tar stream을 바로 ctr import에 넘겨 임시 tar 파일을 만들지 않는다.
  docker save "${image}" | ${K3S_CTR_CMD} images import -
}

# k3d cluster runtime에 이미지를 주입한다.
load_image_into_k3d() {
  local image="$1"
  local args=()

  if [[ "${K3D_LOAD_IMAGES}" != "true" ]]; then
    log "skipping k3d image import for ${image}"
    return
  fi

  require_command k3d
  if [[ -n "${K3D_CLUSTER_NAME}" ]]; then
    args+=("--cluster" "${K3D_CLUSTER_NAME}")
  fi

  log "importing ${image} into k3d"
  k3d image import "${image}" "${args[@]}"
}

# kind cluster runtime에 이미지를 주입한다.
load_image_into_kind() {
  local image="$1"
  local args=()

  if [[ "${KIND_LOAD_IMAGES}" != "true" ]]; then
    log "skipping kind image load for ${image}"
    return
  fi

  require_command kind
  if [[ -n "${KIND_CLUSTER_NAME}" ]]; then
    args+=(--name "${KIND_CLUSTER_NAME}")
  fi

  log "loading ${image} into kind"
  kind load docker-image "${image}" "${args[@]}"
}

# 로컬 이미지 tag와 pullPolicy를 반영한 임시 kustomize overlay를 렌더링한다.
render_manifest() {
  local pull_policy="Never"

  if [[ "${PUSH_IMAGES}" == "true" ]]; then
    pull_policy="IfNotPresent"
  fi

  log "rendering manifest with image tag ${IMAGE_TAG}"

  require_command kubectl
  # 원본 build/kite를 건드리지 않고 임시 overlay에서 image name/tag와 pullPolicy만 바꾼다.
  cat > "${KUSTOMIZE_OVERLAY_DIR}/kustomization.yaml" <<EOF
resources:
  - ../../kite

images:
  - name: ghcr.io/hy3ons/kite-api
    newName: ${IMAGE_REGISTRY}/kite-api
    newTag: ${IMAGE_TAG}
  - name: ghcr.io/hy3ons/kite-controller
    newName: ${IMAGE_REGISTRY}/kite-controller
    newTag: ${IMAGE_TAG}
  - name: ghcr.io/hy3ons/kite-gateway
    newName: ${IMAGE_REGISTRY}/kite-gateway
    newTag: ${IMAGE_TAG}
  - name: ghcr.io/hy3ons/kite-frontend
    newName: ${IMAGE_REGISTRY}/kite-frontend
    newTag: ${IMAGE_TAG}

patches:
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: kite-api
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: kite-api
      spec:
        template:
          spec:
            containers:
              - name: kite-api
                imagePullPolicy: ${pull_policy}
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: kite-controller
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: kite-controller
      spec:
        template:
          spec:
            containers:
              - name: kite-controller
                imagePullPolicy: ${pull_policy}
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: kite-gateway
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: kite-gateway
      spec:
        template:
          spec:
            containers:
              - name: kite-gateway
                imagePullPolicy: ${pull_policy}
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: kite-frontend
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: kite-frontend
      spec:
        template:
          spec:
            containers:
              - name: kite-frontend
                imagePullPolicy: ${pull_policy}
EOF
  kubectl kustomize "${KUSTOMIZE_OVERLAY_DIR}" > "${RENDERED_MANIFEST}"
}

# gateway host key Secret을 보장한 뒤 렌더링된 manifest를 클러스터에 적용한다.
apply_manifest() {
  "${ROOT_DIR}/build/deploy/scripts/ensure-gateway-host-key-secret.sh"

  log "applying ${RENDERED_MANIFEST}"
  kubectl apply -f "${RENDERED_MANIFEST}"
}

# rollout 실패 시 바로 원인 파악할 수 있게 Pod 상태와 최근 이벤트를 출력한다.
show_debug() {
  log "rollout did not become ready; printing debug information"
  kubectl -n "${KITE_NAMESPACE}" get pods -o wide || true
  kubectl -n "${KITE_NAMESPACE}" get events --sort-by=.lastTimestamp | tail -60 || true
}

# Deployment rollout을 기다리고 실패하면 debug 정보를 출력한 뒤 중단한다.
wait_for_deployment() {
  local deployment="$1"

  log "waiting for deployment/${deployment}"
  if ! kubectl -n "${KITE_NAMESPACE}" rollout status "deployment/${deployment}" --timeout=180s; then
    show_debug
    exit 1
  fi
}

# 클러스터 종류별로 네 컴포넌트 이미지를 빌드하고 필요한 runtime에 주입한다.
build_images_for_cluster() {
  local cluster="$1"
  local api_image
  local controller_image
  local gateway_image
  local frontend_image
  local -a frontend_build_args

  api_image="$(image_name kite-api)"
  controller_image="$(image_name kite-controller)"
  gateway_image="$(image_name kite-gateway)"
  frontend_image="$(image_name kite-frontend)"
  # frontend 이미지는 빌드 시점에 Vite 환경값이 정적으로 들어간다.
  frontend_build_args=(
    --build-arg "VITE_BUILD_MODE=${FRONTEND_VITE_BUILD_MODE}"
    --build-arg "VITE_API_BASE_URL=${FRONTEND_VITE_API_BASE_URL}"
    --build-arg "VITE_USE_MOCK=${FRONTEND_VITE_USE_MOCK}"
  )

  case "${cluster}" in
    minikube)
      build_minikube_image "${api_image}" "${ROOT_DIR}/kite/Dockerfile.api" "${ROOT_DIR}/kite"
      build_minikube_image "${controller_image}" "${ROOT_DIR}/kite/Dockerfile.controller" "${ROOT_DIR}/kite"
      build_minikube_image "${gateway_image}" "${ROOT_DIR}/kite/Dockerfile.gateway" "${ROOT_DIR}/kite"
      build_minikube_image "${frontend_image}" "${ROOT_DIR}/kite-frontend/Dockerfile" "${ROOT_DIR}/kite-frontend" "${frontend_build_args[@]}"
      ;;
    k3s)
      build_local_image "${api_image}" "${ROOT_DIR}/kite/Dockerfile.api" "${ROOT_DIR}/kite"
      build_local_image "${controller_image}" "${ROOT_DIR}/kite/Dockerfile.controller" "${ROOT_DIR}/kite"
      build_local_image "${gateway_image}" "${ROOT_DIR}/kite/Dockerfile.gateway" "${ROOT_DIR}/kite"
      build_local_image "${frontend_image}" "${ROOT_DIR}/kite-frontend/Dockerfile" "${ROOT_DIR}/kite-frontend" "${frontend_build_args[@]}"
      load_image_into_k3s "${api_image}"
      load_image_into_k3s "${controller_image}"
      load_image_into_k3s "${gateway_image}"
      load_image_into_k3s "${frontend_image}"
      ;;
    k3d)
      build_local_image "${api_image}" "${ROOT_DIR}/kite/Dockerfile.api" "${ROOT_DIR}/kite"
      build_local_image "${controller_image}" "${ROOT_DIR}/kite/Dockerfile.controller" "${ROOT_DIR}/kite"
      build_local_image "${gateway_image}" "${ROOT_DIR}/kite/Dockerfile.gateway" "${ROOT_DIR}/kite"
      build_local_image "${frontend_image}" "${ROOT_DIR}/kite-frontend/Dockerfile" "${ROOT_DIR}/kite-frontend" "${frontend_build_args[@]}"
      load_image_into_k3d "${api_image}"
      load_image_into_k3d "${controller_image}"
      load_image_into_k3d "${gateway_image}"
      load_image_into_k3d "${frontend_image}"
      ;;
    kind)
      build_local_image "${api_image}" "${ROOT_DIR}/kite/Dockerfile.api" "${ROOT_DIR}/kite"
      build_local_image "${controller_image}" "${ROOT_DIR}/kite/Dockerfile.controller" "${ROOT_DIR}/kite"
      build_local_image "${gateway_image}" "${ROOT_DIR}/kite/Dockerfile.gateway" "${ROOT_DIR}/kite"
      build_local_image "${frontend_image}" "${ROOT_DIR}/kite-frontend/Dockerfile" "${ROOT_DIR}/kite-frontend" "${frontend_build_args[@]}"
      load_image_into_kind "${api_image}"
      load_image_into_kind "${controller_image}"
      load_image_into_kind "${gateway_image}"
      load_image_into_kind "${frontend_image}"
      ;;
    current|k8s|kubernetes)
      build_local_image "${api_image}" "${ROOT_DIR}/kite/Dockerfile.api" "${ROOT_DIR}/kite"
      build_local_image "${controller_image}" "${ROOT_DIR}/kite/Dockerfile.controller" "${ROOT_DIR}/kite"
      build_local_image "${gateway_image}" "${ROOT_DIR}/kite/Dockerfile.gateway" "${ROOT_DIR}/kite"
      build_local_image "${frontend_image}" "${ROOT_DIR}/kite-frontend/Dockerfile" "${ROOT_DIR}/kite-frontend" "${frontend_build_args[@]}"
      push_local_image "${api_image}"
      push_local_image "${controller_image}"
      push_local_image "${gateway_image}"
      push_local_image "${frontend_image}"
      if [[ "${PUSH_IMAGES}" != "true" ]]; then
        log "using locally built images only; this works for local Docker-backed clusters or preloaded image names"
        log "set PUSH_IMAGES=true when the current Kubernetes cluster must pull from ${IMAGE_REGISTRY}"
      fi
      ;;
    *)
      echo "[kite] unknown KITE_CLUSTER=${cluster}; use auto, minikube, k3s, k3d, kind, k8s, kubernetes, or current" >&2
      exit 1
      ;;
  esac
}

# 설치 후 사용자가 바로 확인할 수 있는 port-forward/접속 힌트를 출력한다.
print_summary() {
  log "deployment complete"
  echo "  cluster: ${1}"
  echo "  namespace: ${KITE_NAMESPACE}"
  echo "  image tag: ${IMAGE_TAG}"
  echo
  echo "  API health:"
  echo "    kubectl -n ${KITE_NAMESPACE} port-forward svc/kite-api 8080:8080"
  echo "    curl http://127.0.0.1:8080/health"
  echo
  echo "  Frontend:"
  echo "    kubectl -n ${KITE_NAMESPACE} port-forward svc/kite-frontend 8081:80"
  echo "    open http://127.0.0.1:8081"
  echo
  echo "  gateway:"
  echo "    ssh <sshId>@<node-ip>"
}

# 전체 개발 배포 흐름이다. 클러스터 감지, 이미지 빌드/주입, manifest 렌더링/적용, rollout 대기를 수행한다.
main() {
  local cluster

  require_command kubectl
  cluster="$(detect_cluster)"
  log "target cluster=${cluster}"
  configure_interactive_dev_options "${cluster}" all

  if [[ "${cluster}" == "minikube" ]]; then
    require_command minikube
    start_minikube
  fi

  build_images_for_cluster "${cluster}"
  render_manifest "${cluster}"
  apply_manifest

  wait_for_deployment kite-api
  wait_for_deployment kite-controller
  wait_for_deployment kite-gateway
  wait_for_deployment kite-frontend

  print_summary "${cluster}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
