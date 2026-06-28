#!/usr/bin/env bash
set -euo pipefail

# dev.sh builds Kite images from the local source tree and deploys them to a Kubernetes cluster.
# Supported targets:
#   KITE_CLUSTER=minikube build/dev/dev.sh
#   KITE_CLUSTER=k3s build/dev/dev.sh
#   KITE_CLUSTER=k3d build/dev/dev.sh
#   KITE_CLUSTER=kind build/dev/dev.sh
#   KITE_CLUSTER=k8s build/dev/dev.sh
#   KITE_CLUSTER=current build/dev/dev.sh
#
# minikube mode can start the profile and load images into the Minikube runtime.
# k3s mode builds images with local Docker and imports them into k3s containerd.
# current mode builds local Docker images and applies Kubernetes manifests. Set
# PUSH_IMAGES=true with IMAGE_REGISTRY only when a remote cluster must pull from a registry.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
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

# Minikube knobs. They are used only when KITE_CLUSTER=minikube or auto detects minikube.
MINIKUBE_PROFILE="${MINIKUBE_PROFILE:-minikube}"
MINIKUBE_DRIVER="${MINIKUBE_DRIVER:-}"
MINIKUBE_CPUS="${MINIKUBE_CPUS:-4}"
MINIKUBE_MEMORY="${MINIKUBE_MEMORY:-8192}"
MINIKUBE_START="${MINIKUBE_START:-true}"
MINIKUBE_BUILD_STRATEGY="${MINIKUBE_BUILD_STRATEGY:-${BUILD_STRATEGY:-auto}}"

# k3s image import command. Kubernetes reads images from the k8s.io containerd namespace.
# Override when sudo is not needed:
#   K3S_CTR_CMD="k3s ctr -n k8s.io" KITE_CLUSTER=k3s build/dev/dev.sh
K3S_CTR_CMD="${K3S_CTR_CMD:-sudo k3s ctr -n k8s.io}"
K3S_IMPORT_IMAGES="${K3S_IMPORT_IMAGES:-true}"
K3D_CLUSTER_NAME="${K3D_CLUSTER_NAME:-}"
K3D_LOAD_IMAGES="${K3D_LOAD_IMAGES:-true}"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-}"
KIND_LOAD_IMAGES="${KIND_LOAD_IMAGES:-true}"

cleanup() {
  rm -rf "${KUSTOMIZE_OVERLAY_DIR}"
  rm -f "${RENDERED_MANIFEST}"
}
trap cleanup EXIT

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

image_name() {
  local component="$1"
  echo "${IMAGE_REGISTRY}/${component}:${IMAGE_TAG}"
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

host_os() {
  uname -s 2>/dev/null || echo unknown
}

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

build_local_image() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  shift 3
  local build_args=("$@")

  require_command docker
  log "building ${image}"
  DOCKER_BUILDKIT=1 docker build --progress=plain "${build_args[@]}" -t "${image}" -f "${dockerfile}" "${context}"
}

push_local_image() {
  local image="$1"

  if [[ "${PUSH_IMAGES}" != "true" ]]; then
    return
  fi

  require_command docker
  log "pushing ${image}"
  docker push "${image}"
}

build_minikube_image_with_minikube() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  shift 3
  local build_args=("$@")

  minikube -p "${MINIKUBE_PROFILE}" image build "${build_args[@]}" -t "${image}" -f "${dockerfile}" "${context}"
}

build_minikube_image_with_docker_env() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  shift 3
  local build_args=("$@")
  local status

  require_command docker
  eval "$(minikube -p "${MINIKUBE_PROFILE}" docker-env)"
  if DOCKER_BUILDKIT=1 docker build --progress=plain "${build_args[@]}" -t "${image}" -f "${dockerfile}" "${context}"; then
    status=0
  else
    status=$?
  fi
  eval "$(minikube -p "${MINIKUBE_PROFILE}" docker-env --unset)"
  return "${status}"
}

build_minikube_image_with_local_load() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  shift 3
  local build_args=("$@")

  build_local_image "${image}" "${dockerfile}" "${context}" "${build_args[@]}"
  minikube -p "${MINIKUBE_PROFILE}" image load "${image}"
}

build_minikube_image() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  shift 3
  local build_args=("$@")
  local driver
  local os_name

  driver="$(minikube_driver || true)"
  os_name="$(host_os)"
  log "build strategy=${MINIKUBE_BUILD_STRATEGY}, minikube driver=${driver:-unknown}, host=${os_name}"

  case "${MINIKUBE_BUILD_STRATEGY}" in
    minikube)
      build_minikube_image_with_minikube "${image}" "${dockerfile}" "${context}" "${build_args[@]}"
      return
      ;;
    docker-env)
      build_minikube_image_with_docker_env "${image}" "${dockerfile}" "${context}" "${build_args[@]}"
      return
      ;;
    local-load)
      build_minikube_image_with_local_load "${image}" "${dockerfile}" "${context}" "${build_args[@]}"
      return
      ;;
    auto)
      ;;
    *)
      echo "[kite] unknown MINIKUBE_BUILD_STRATEGY=${MINIKUBE_BUILD_STRATEGY}; use auto, minikube, docker-env, or local-load" >&2
      exit 1
      ;;
  esac

  # qemu/qemu2 and macOS commonly fail when minikube image build tries to resolve /Users inside the VM.
  if [[ "${driver}" == "qemu" || "${driver}" == "qemu2" || "${os_name}" == "Darwin" ]]; then
    log "using docker-env build to avoid Minikube VM host path issues"
    if build_minikube_image_with_docker_env "${image}" "${dockerfile}" "${context}" "${build_args[@]}"; then
      return
    fi

    log "docker-env build failed for ${image}; retrying local build plus minikube image load"
    build_minikube_image_with_local_load "${image}" "${dockerfile}" "${context}" "${build_args[@]}"
    return
  fi

  if build_minikube_image_with_minikube "${image}" "${dockerfile}" "${context}" "${build_args[@]}"; then
    return
  fi

  log "minikube image build failed for ${image}; retrying docker-env"
  if build_minikube_image_with_docker_env "${image}" "${dockerfile}" "${context}" "${build_args[@]}"; then
    return
  fi

  log "docker-env build failed for ${image}; retrying local build plus minikube image load"
  build_minikube_image_with_local_load "${image}" "${dockerfile}" "${context}" "${build_args[@]}"
}

load_image_into_k3s() {
  local image="$1"

  if [[ "${K3S_IMPORT_IMAGES}" != "true" ]]; then
    log "skipping k3s image import for ${image}"
    return
  fi

  require_command docker
  log "importing ${image} into k3s containerd"
  docker save "${image}" | ${K3S_CTR_CMD} images import -
}

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

render_manifest() {
  local pull_policy="Never"

  if [[ "${PUSH_IMAGES}" == "true" ]]; then
    pull_policy="IfNotPresent"
  fi

  log "rendering manifest with image tag ${IMAGE_TAG}"

  require_command kubectl
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

apply_manifest() {
  "${ROOT_DIR}/build/deploy/scripts/ensure-gateway-host-key-secret.sh"

  log "applying ${RENDERED_MANIFEST}"
  kubectl apply -f "${RENDERED_MANIFEST}"
}

show_debug() {
  log "rollout did not become ready; printing debug information"
  kubectl -n "${KITE_NAMESPACE}" get pods -o wide || true
  kubectl -n "${KITE_NAMESPACE}" get events --sort-by=.lastTimestamp | tail -60 || true
}

wait_for_deployment() {
  local deployment="$1"

  log "waiting for deployment/${deployment}"
  if ! kubectl -n "${KITE_NAMESPACE}" rollout status "deployment/${deployment}" --timeout=180s; then
    show_debug
    exit 1
  fi
}

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

main() {
  local cluster

  require_command kubectl
  cluster="$(detect_cluster)"
  log "target cluster=${cluster}"

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
