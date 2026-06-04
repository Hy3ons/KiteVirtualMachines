#!/usr/bin/env bash
set -euo pipefail

# dev.sh builds the current Kite source into Minikube-visible images and deploys it.
# It is intended to work on normal developer laptops as long as minikube, kubectl, and Docker are installed.
# QEMU-based Minikube drivers often cannot read host paths such as /Users from inside the VM, so this
# script avoids `minikube image build` for those drivers and sends the Docker build context through
# Minikube's Docker daemon instead.

# Resolve the repository root from this script location so it can be run from any cwd.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Local development knobs. Override these from the shell when needed:
#   MINIKUBE_PROFILE=kite-dev MINIKUBE_DRIVER=docker MINIKUBE_CPUS=6 MINIKUBE_MEMORY=12288 ./dev.sh
#   BUILD_STRATEGY=docker-env ./dev.sh
MINIKUBE_PROFILE="${MINIKUBE_PROFILE:-minikube}"
MINIKUBE_DRIVER="${MINIKUBE_DRIVER:-}"
MINIKUBE_CPUS="${MINIKUBE_CPUS:-4}"
MINIKUBE_MEMORY="${MINIKUBE_MEMORY:-8192}"
BUILD_STRATEGY="${BUILD_STRATEGY:-auto}"

# Kite's own runtime resources are deployed in the kite namespace.
# install.yaml is intentionally written for this namespace, so keep this value as kite unless the manifest is changed too.
KITE_NAMESPACE="kite"

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

minikube_driver() {
  local driver

  # First try the configured driver. This is available when the profile was started with --driver.
  driver="$(minikube -p "${MINIKUBE_PROFILE}" config get driver 2>/dev/null || true)"
  if [[ -n "${driver}" ]]; then
    echo "${driver}"
    return
  fi

  # Fall back to profile metadata. Minikube versions format this JSON differently, so keep the parser loose.
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

  # Allow a developer to pin a driver, while keeping Minikube's default driver selection otherwise.
  if [[ -n "${MINIKUBE_DRIVER}" ]]; then
    args+=(--driver="${MINIKUBE_DRIVER}")
  fi

  log "starting minikube profile=${MINIKUBE_PROFILE}"
  minikube "${args[@]}"

  # Make plain kubectl commands talk to the selected profile.
  minikube -p "${MINIKUBE_PROFILE}" update-context
}

build_image_with_minikube() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"

  minikube -p "${MINIKUBE_PROFILE}" image build -t "${image}" -f "${dockerfile}" "${context}"
}

build_image_with_docker_env() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  local status

  require_command docker

  # docker-env points the local docker CLI at Minikube's Docker daemon. The build context is streamed
  # from the host, so macOS paths such as /Users do not need to exist inside the Minikube VM.
  eval "$(minikube -p "${MINIKUBE_PROFILE}" docker-env)"
  if DOCKER_BUILDKIT=1 docker build --progress=plain -t "${image}" -f "${dockerfile}" "${context}"; then
    status=0
  else
    status=$?
  fi
  eval "$(minikube -p "${MINIKUBE_PROFILE}" docker-env --unset)"
  return "${status}"
}

build_image_with_local_load() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"

  require_command docker

  # Last-resort path for drivers/runtimes where docker-env is not available. Build on the host Docker
  # daemon, then ask Minikube to load the image into the cluster runtime.
  DOCKER_BUILDKIT=1 docker build --progress=plain -t "${image}" -f "${dockerfile}" "${context}"
  minikube -p "${MINIKUBE_PROFILE}" image load "${image}"
}

build_image() {
  local image="$1"
  local dockerfile="$2"
  local context="$3"
  local driver
  local os_name

  log "building ${image}"
  driver="$(minikube_driver || true)"
  os_name="$(host_os)"
  log "build strategy=${BUILD_STRATEGY}, minikube driver=${driver:-unknown}, host=${os_name}"

  case "${BUILD_STRATEGY}" in
    minikube)
      build_image_with_minikube "${image}" "${dockerfile}" "${context}"
      return
      ;;
    docker-env)
      build_image_with_docker_env "${image}" "${dockerfile}" "${context}"
      return
      ;;
    local-load)
      build_image_with_local_load "${image}" "${dockerfile}" "${context}"
      return
      ;;
    auto)
      ;;
    *)
      echo "[kite] unknown BUILD_STRATEGY=${BUILD_STRATEGY}; use auto, minikube, docker-env, or local-load" >&2
      exit 1
      ;;
  esac

  # QEMU and qemu2 commonly fail with:
  #   lstat /Users: no such file or directory
  # because `minikube image build` may resolve the host project path inside the VM.
  # On macOS, prefer docker-env even if driver detection fails, because host paths under /Users are common.
  if [[ "${driver}" == "qemu" || "${driver}" == "qemu2" || "${os_name}" == "Darwin" ]]; then
    log "using docker-env build to avoid Minikube VM host path issues"
    if build_image_with_docker_env "${image}" "${dockerfile}" "${context}"; then
      return
    fi

    log "docker-env build failed for ${image}; retrying with local docker build plus minikube image load"
    build_image_with_local_load "${image}" "${dockerfile}" "${context}"
    return
  fi

  # For docker/podman drivers, `minikube image build` is usually the fastest path.
  if build_image_with_minikube "${image}" "${dockerfile}" "${context}"; then
    return
  fi

  log "minikube image build failed for ${image}; retrying with docker-env fallback"
  if build_image_with_docker_env "${image}" "${dockerfile}" "${context}"; then
    return
  fi

  log "docker-env build failed for ${image}; retrying with local docker build plus minikube image load"
  build_image_with_local_load "${image}" "${dockerfile}" "${context}"
}

ensure_namespace() {
  log "creating namespace"

  # The namespace also exists in install.yaml. Applying it here first keeps repeated local runs idempotent.
  kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: ${KITE_NAMESPACE}
EOF
}

apply_manifest() {
  log "applying install.yaml"
  kubectl apply -f "${ROOT_DIR}/kite-yaml/install.yaml"
}

show_debug() {
  log "deployment did not become ready; printing debug information"
  kubectl -n "${KITE_NAMESPACE}" get pods -o wide || true
  kubectl -n "${KITE_NAMESPACE}" get events --sort-by=.lastTimestamp | tail -40 || true
}

wait_for_rollout() {
  local deployment="$1"
  log "waiting for deployment/${deployment}"

  if ! kubectl -n "${KITE_NAMESPACE}" rollout status "deployment/${deployment}" --timeout=180s; then
    show_debug
    exit 1
  fi
}

main() {
  require_command minikube
  require_command kubectl

  start_minikube

  build_image "anacnu.com/kite-api:latest" "${ROOT_DIR}/kite/Dockerfile.api" "${ROOT_DIR}/kite"
  build_image "anacnu.com/kite-controller:latest" "${ROOT_DIR}/kite/Dockerfile.controller" "${ROOT_DIR}/kite"
  build_image "anacnu.com/kite-account:latest" "${ROOT_DIR}/kite/Dockerfile.account" "${ROOT_DIR}/kite"
  build_image "anacnu.com/kite-frontend:latest" "${ROOT_DIR}/kite-frontend/Dockerfile" "${ROOT_DIR}/kite-frontend"

  ensure_namespace
  apply_manifest

  log "waiting for deployments"
  wait_for_rollout kite-api
  wait_for_rollout kite-controller
  kubectl -n "${KITE_NAMESPACE}" rollout status daemonset/kite-account --timeout=180s || {
    show_debug
    exit 1
  }
  wait_for_rollout kite-frontend

  log "deployment complete"
  log "try API health:"
  echo "  kubectl -n ${KITE_NAMESPACE} port-forward svc/kite-api 8080:8080"
  echo "  curl http://127.0.0.1:8080/health"
}

main "$@"
