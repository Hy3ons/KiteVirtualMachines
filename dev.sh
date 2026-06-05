#!/usr/bin/env bash
set -euo pipefail

# dev.sh builds Kite images from the local source tree and deploys them to a Kubernetes cluster.
# Supported targets:
#   KITE_CLUSTER=minikube ./dev.sh
#   KITE_CLUSTER=k3s ./dev.sh
#   KITE_CLUSTER=current ./dev.sh
#
# minikube mode can start the profile and load images into the Minikube runtime.
# k3s mode builds images with local Docker and imports them into k3s containerd.
# current mode only builds local Docker images and applies Kubernetes manifests; use it when the
# current cluster can already pull the configured image names from a registry.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
KITE_CLUSTER="${KITE_CLUSTER:-auto}"
IMAGE_REGISTRY="${IMAGE_REGISTRY:-anacnu.com}"
IMAGE_TAG="${IMAGE_TAG:-dev-$(date +%Y%m%d%H%M%S)}"
MANIFEST_TEMPLATE="${ROOT_DIR}/kite-yaml/install.yaml"
RENDERED_MANIFEST="$(mktemp "${TMPDIR:-/tmp}/kite-install.XXXXXX.yaml")"

# Minikube knobs. They are used only when KITE_CLUSTER=minikube or auto detects minikube.
MINIKUBE_PROFILE="${MINIKUBE_PROFILE:-minikube}"
MINIKUBE_DRIVER="${MINIKUBE_DRIVER:-}"
MINIKUBE_CPUS="${MINIKUBE_CPUS:-4}"
MINIKUBE_MEMORY="${MINIKUBE_MEMORY:-8192}"
MINIKUBE_START="${MINIKUBE_START:-true}"
MINIKUBE_BUILD_STRATEGY="${MINIKUBE_BUILD_STRATEGY:-auto}"

# k3s image import command. Kubernetes reads images from the k8s.io containerd namespace.
# Override when sudo is not needed:
#   K3S_CTR_CMD="k3s ctr -n k8s.io" KITE_CLUSTER=k3s ./dev.sh
K3S_CTR_CMD="${K3S_CTR_CMD:-sudo k3s ctr -n k8s.io}"
K3S_IMPORT_IMAGES="${K3S_IMPORT_IMAGES:-true}"

cleanup() {
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

  require_command docker
  log "building ${image}"
  DOCKER_BUILDKIT=1 docker build --progress=plain -t "${image}" -f "${dockerfile}" "${context}"
}

build_minikube_image_with_minikube() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"

  minikube -p "${MINIKUBE_PROFILE}" image build -t "${image}" -f "${dockerfile}" "${context}"
}

build_minikube_image_with_docker_env() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  local status

  require_command docker
  eval "$(minikube -p "${MINIKUBE_PROFILE}" docker-env)"
  if DOCKER_BUILDKIT=1 docker build --progress=plain -t "${image}" -f "${dockerfile}" "${context}"; then
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

  build_local_image "${image}" "${dockerfile}" "${context}"
  minikube -p "${MINIKUBE_PROFILE}" image load "${image}"
}

build_minikube_image() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  local driver
  local os_name

  driver="$(minikube_driver || true)"
  os_name="$(host_os)"
  log "build strategy=${MINIKUBE_BUILD_STRATEGY}, minikube driver=${driver:-unknown}, host=${os_name}"

  case "${MINIKUBE_BUILD_STRATEGY}" in
    minikube)
      build_minikube_image_with_minikube "${image}" "${dockerfile}" "${context}"
      return
      ;;
    docker-env)
      build_minikube_image_with_docker_env "${image}" "${dockerfile}" "${context}"
      return
      ;;
    local-load)
      build_minikube_image_with_local_load "${image}" "${dockerfile}" "${context}"
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
    if build_minikube_image_with_docker_env "${image}" "${dockerfile}" "${context}"; then
      return
    fi

    log "docker-env build failed for ${image}; retrying local build plus minikube image load"
    build_minikube_image_with_local_load "${image}" "${dockerfile}" "${context}"
    return
  fi

  if build_minikube_image_with_minikube "${image}" "${dockerfile}" "${context}"; then
    return
  fi

  log "minikube image build failed for ${image}; retrying docker-env"
  if build_minikube_image_with_docker_env "${image}" "${dockerfile}" "${context}"; then
    return
  fi

  log "docker-env build failed for ${image}; retrying local build plus minikube image load"
  build_minikube_image_with_local_load "${image}" "${dockerfile}" "${context}"
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

render_manifest() {
  local pull_policy="IfNotPresent"

  if [[ "${1}" == "minikube" || "${1}" == "k3s" ]]; then
    pull_policy="Never"
  fi

  log "rendering manifest with image tag ${IMAGE_TAG}"

  sed \
    -e "s#anacnu.com/kite-api:latest#$(image_name kite-api)#g" \
    -e "s#anacnu.com/kite-controller:latest#$(image_name kite-controller)#g" \
    -e "s#anacnu.com/kite-host-agent:latest#$(image_name kite-host-agent)#g" \
    -e "s#anacnu.com/kite-frontend:latest#$(image_name kite-frontend)#g" \
    -e "s#imagePullPolicy: IfNotPresent#imagePullPolicy: ${pull_policy}#g" \
    "${MANIFEST_TEMPLATE}" > "${RENDERED_MANIFEST}"
}

apply_manifest() {
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

wait_for_daemonset() {
  local daemonset="$1"

  log "waiting for daemonset/${daemonset}"
  if ! kubectl -n "${KITE_NAMESPACE}" rollout status "daemonset/${daemonset}" --timeout=180s; then
    show_debug
    exit 1
  fi
}

build_images_for_cluster() {
  local cluster="$1"
  local api_image
  local controller_image
  local host_agent_image
  local frontend_image

  api_image="$(image_name kite-api)"
  controller_image="$(image_name kite-controller)"
  host_agent_image="$(image_name kite-host-agent)"
  frontend_image="$(image_name kite-frontend)"

  case "${cluster}" in
    minikube)
      build_minikube_image "${api_image}" "${ROOT_DIR}/kite/Dockerfile.api" "${ROOT_DIR}/kite"
      build_minikube_image "${controller_image}" "${ROOT_DIR}/kite/Dockerfile.controller" "${ROOT_DIR}/kite"
      build_minikube_image "${host_agent_image}" "${ROOT_DIR}/kite/Dockerfile.host-agent" "${ROOT_DIR}/kite"
      build_minikube_image "${frontend_image}" "${ROOT_DIR}/kite-frontend/Dockerfile" "${ROOT_DIR}/kite-frontend"
      ;;
    k3s)
      build_local_image "${api_image}" "${ROOT_DIR}/kite/Dockerfile.api" "${ROOT_DIR}/kite"
      build_local_image "${controller_image}" "${ROOT_DIR}/kite/Dockerfile.controller" "${ROOT_DIR}/kite"
      build_local_image "${host_agent_image}" "${ROOT_DIR}/kite/Dockerfile.host-agent" "${ROOT_DIR}/kite"
      build_local_image "${frontend_image}" "${ROOT_DIR}/kite-frontend/Dockerfile" "${ROOT_DIR}/kite-frontend"
      load_image_into_k3s "${api_image}"
      load_image_into_k3s "${controller_image}"
      load_image_into_k3s "${host_agent_image}"
      load_image_into_k3s "${frontend_image}"
      ;;
    current)
      build_local_image "${api_image}" "${ROOT_DIR}/kite/Dockerfile.api" "${ROOT_DIR}/kite"
      build_local_image "${controller_image}" "${ROOT_DIR}/kite/Dockerfile.controller" "${ROOT_DIR}/kite"
      build_local_image "${host_agent_image}" "${ROOT_DIR}/kite/Dockerfile.host-agent" "${ROOT_DIR}/kite"
      build_local_image "${frontend_image}" "${ROOT_DIR}/kite-frontend/Dockerfile" "${ROOT_DIR}/kite-frontend"
      ;;
    *)
      echo "[kite] unknown KITE_CLUSTER=${cluster}; use auto, minikube, k3s, or current" >&2
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
  wait_for_daemonset kite-host-agent
  wait_for_deployment kite-frontend

  print_summary "${cluster}"
}

main "$@"
