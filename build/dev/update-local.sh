#!/usr/bin/env bash
set -Eeuo pipefail

# Build fresh release images natively on a single-node K3s host, import all of
# them into its containerd, then reuse the state-preserving runtime updater.
# build/dev/build-update.sh supplies KITE_UPDATE_SOURCE_DIR and KITE_UPDATE_SOURCE_REVISION.
# Requires Docker and K3s access; it never installs or resets infrastructure.
LOCAL_UPDATE_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KITE_UPDATE_SOURCE_DIR="${KITE_UPDATE_SOURCE_DIR:?Run build/dev/build-update.sh to fetch the release source}"
KITE_UPDATE_SOURCE_REVISION="${KITE_UPDATE_SOURCE_REVISION:?Missing source commit}"
KITE_UPDATE_REGISTRY="localhost/kite-local"
KITE_UPDATE_TAG="source-${KITE_UPDATE_SOURCE_REVISION:0:12}-$(date +%s)-$$"
KITE_UPDATE_PULL_POLICY=Never

# shellcheck source=build/deploy/scripts/update-ghcr.sh
source "${LOCAL_UPDATE_SCRIPT_DIR}/../deploy/scripts/update-ghcr.sh"
KITE_MANIFEST_DIR="${KITE_UPDATE_SOURCE_DIR}/build/kite"

# prepare_update_images builds every selected image before importing any image.
# It verifies local single-node K3s identity to prevent loading the wrong runtime.
# Build/import failure exits before the shared updater can mutate deployments.
prepare_update_images() {
  local nodes
  local local_nodes
  local node_uid
  local architecture
  local extra
  local component
  local context
  local dockerfile
  local images=()

  if [[ "${KITE_UPDATE_DRY_RUN}" == true ]]; then
    log "dry-run: build selected source images, then import all into local K3s containerd"
    return 0
  fi
  require_command docker
  require_command k3s
  nodes="$(kubectl get nodes -o "jsonpath={range .items[*]}{.metadata.uid}{' '}{.status.nodeInfo.architecture}{'\n'}{end}")"
  local_nodes="$(k3s kubectl get nodes -o "jsonpath={range .items[*]}{.metadata.uid}{' '}{.status.nodeInfo.architecture}{'\n'}{end}")"
  if [[ "${nodes}" != "${local_nodes}" || "${nodes}" == *$'\n'* ]]; then
    warn "local source updates require this host's single-node K3s cluster"
    return 1
  fi
  read -r node_uid architecture extra <<< "${nodes}"
  case "${architecture}" in
    arm64|amd64) ;;
    *) warn "unsupported build architecture=${architecture}"; return 1 ;;
  esac
  log "building source=${KITE_UPDATE_SOURCE_REVISION} platform=linux/${architecture}"
  while read -r component; do
    if [[ "${component}" == frontend ]]; then
      context="${KITE_UPDATE_SOURCE_DIR}/kite-frontend"
      dockerfile="${context}/Dockerfile"
    else
      context="${KITE_UPDATE_SOURCE_DIR}/kite"
      dockerfile="${context}/Dockerfile.${component}"
    fi
    log "building $(component_image "${component}")"
    run_cmd env DOCKER_BUILDKIT=1 docker build --pull --progress=plain \
      --platform "linux/${architecture}" \
      --label "org.opencontainers.image.revision=${KITE_UPDATE_SOURCE_REVISION}" \
      --build-arg NEXT_PUBLIC_KITE_MODE=production \
      --build-arg NEXT_PUBLIC_API_BASE_URL=/api/v1 \
      --build-arg NEXT_PUBLIC_USE_MOCK=false \
      -t "$(component_image "${component}")" -f "${dockerfile}" "${context}"
    images+=("$(component_image "${component}")")
  done <<< "${KITE_UPDATE_COMPONENT_LIST}"

  log "all builds complete; importing images into local K3s containerd"
  docker save "${images[@]}" | k3s ctr -n k8s.io images import -
}

main "$@"
