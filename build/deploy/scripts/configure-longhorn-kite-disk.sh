#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/configure-longhorn-kite-disk.sh
# Description: Longhorn node disk에 Kite 태그를 붙이거나 전용 disk entry를 구성한다.
#
# Usage:
#   build/deploy/scripts/configure-longhorn-kite-disk.sh
#
# Environment Variables:
#   KITE_LONGHORN_DISK_NAME: default kite-longhorn
#   KITE_LONGHORN_DISK_PATH: default /mnt/kite-longhorn
#   KITE_LONGHORN_DISK_TAG: default kite
#   KITE_LONGHORN_USE_DEDICATED_DISK: default false
#   TIMEOUT_SECONDS: default 180
#   PATCH_RETRY_INTERVAL_SECONDS: default 5
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   대상 클러스터 또는 로컬 개발 환경 상태를 변경할 수 있다.
# ==============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
KITE_LONGHORN_USE_DEDICATED_DISK_WAS_SET="${KITE_LONGHORN_USE_DEDICATED_DISK+x}"
KITE_LONGHORN_DISK_NAME="${KITE_LONGHORN_DISK_NAME:-kite-longhorn}"
KITE_LONGHORN_DISK_PATH="${KITE_LONGHORN_DISK_PATH:-/mnt/kite-longhorn}"
KITE_LONGHORN_DISK_TAG="${KITE_LONGHORN_DISK_TAG:-kite}"
KITE_LONGHORN_USE_DEDICATED_DISK="${KITE_LONGHORN_USE_DEDICATED_DISK:-false}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-180}"
PATCH_RETRY_INTERVAL_SECONDS="${PATCH_RETRY_INTERVAL_SECONDS:-5}"

# shellcheck source=build/lib/prompt.sh
source "${ROOT_DIR}/build/lib/prompt.sh"

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
    printf "\033[0;32m[kite-deploy] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-deploy] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-deploy] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-deploy] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}


# kubectl이 없으면 Longhorn node CR patch를 할 수 없으므로 시작 전에 확인한다.
require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

# 전용 host path를 쓰는 모드에서 Longhorn node CR에 Kite 전용 disk 항목을 추가한다.
# Longhorn node가 disk sync 중이면 patch를 거절하므로, 지정 시간 동안 재시도한다.
patch_longhorn_node_dedicated_disk() {
  local node="$1"
  local deadline
  local output

  deadline=$((SECONDS + TIMEOUT_SECONDS))

  log "configuring Longhorn disk ${KITE_LONGHORN_DISK_NAME} on node/${node}"
  while true; do
    # allowScheduling=true와 kite tag를 같이 넣어 Kite VM DataVolume이 이 디스크를 고르게 한다.
    if output="$(kubectl -n longhorn-system patch "nodes.longhorn.io/${node}" --type=merge -p "{
      \"spec\": {
        \"disks\": {
          \"${KITE_LONGHORN_DISK_NAME}\": {
            \"path\": \"${KITE_LONGHORN_DISK_PATH}\",
            \"allowScheduling\": true,
            \"tags\": [\"${KITE_LONGHORN_DISK_TAG}\"]
          }
        }
      }
    }" 2>&1)"; then
      echo "${output}"
      return
    fi

    if [[ "${output}" != *"being syncing"* && "${output}" != *"please retry later"* ]]; then
      echo "${output}" >&2
      return 1
    fi

    if (( SECONDS >= deadline )); then
      echo "${output}" >&2
      echo "[kite-deploy] timed out waiting for Longhorn node/${node} disk sync" >&2
      return 1
    fi

    log "Longhorn node/${node} is still syncing disks; retrying in ${PATCH_RETRY_INTERVAL_SECONDS}s"
    sleep "${PATCH_RETRY_INTERVAL_SECONDS}"
  done
}

# 전용 디스크를 만들지 않는 기본 모드에서 기존 Ready/Schedulable 디스크에 kite tag를 붙인다.
# 기존 디스크를 재사용해야 단일 디스크 서버에서도 VM clone/import가 진행된다.
tag_existing_longhorn_disks() {
  local node="$1"
  local disks
  local disk
  local allow_scheduling
  local ready
  local schedulable
  local tags
  local next_tags

  # go-template로 spec/status를 한 번에 뽑아 patch 가능한 디스크만 골라낸다.
  disks="$(kubectl -n longhorn-system get "nodes.longhorn.io/${node}" -o 'go-template={{ range $name, $disk := .spec.disks }}{{ $status := index $.status.diskStatus $name }}{{ $name }}|{{ $disk.allowScheduling }}|{{ if $status }}{{ range $status.conditions }}{{ if eq .type "Ready" }}{{ .status }}{{ end }}{{ end }}{{ end }}|{{ if $status }}{{ range $status.conditions }}{{ if eq .type "Schedulable" }}{{ .status }}{{ end }}{{ end }}{{ end }}|{{ range $disk.tags }}{{ . }} {{ end }}{{ "\n" }}{{ end }}')"
  if [[ -z "${disks}" ]]; then
    echo "[kite-deploy] no Longhorn disks found on node/${node}" >&2
    return 1
  fi

  log "tagging existing Ready Longhorn disks on node/${node} with ${KITE_LONGHORN_DISK_TAG}"
  while IFS="|" read -r disk allow_scheduling ready schedulable tags; do
    [[ -z "${disk}" ]] && continue
    if [[ "${allow_scheduling}" != "true" || "${ready}" != "True" || "${schedulable}" != "True" ]]; then
      log "skipping Longhorn disk ${disk} on node/${node} because ready=${ready} schedulable=${schedulable} allowScheduling=${allow_scheduling}"
      continue
    fi
    if [[ " ${tags} " == *" ${KITE_LONGHORN_DISK_TAG} "* ]]; then
      continue
    fi

    # 기존 tag를 유지하면서 중복 없이 kite tag를 추가해 다른 Longhorn 정책과 충돌을 줄인다.
    next_tags="$(printf '%s\n%s\n' "${tags}" "${KITE_LONGHORN_DISK_TAG}" | xargs -n 1 | awk 'NF && !seen[$0]++ { printf "%s\"%s\"", sep, $0; sep=", " }')"
    kubectl -n longhorn-system patch "nodes.longhorn.io/${node}" --type=json -p "[{\"op\":\"replace\",\"path\":\"/spec/disks/${disk}/tags\",\"value\":[${next_tags}]}]"
  done <<< "${disks}"
}

# 모든 Kubernetes node에 대해 전용 디스크 생성 또는 기존 디스크 tag 부여를 수행한다.
main() {
  require_command kubectl
  kite_prompt_configure_bool KITE_LONGHORN_USE_DEDICATED_DISK "${KITE_LONGHORN_USE_DEDICATED_DISK_WAS_SET}" "Longhorn에 Kite 전용 host path disk entry를 만들까요? 아니오면 기존 Ready disk에 kite tag만 붙입니다."

  if [[ "${KITE_LONGHORN_USE_DEDICATED_DISK}" == "true" ]]; then
    log "creating host disk directories for Kite Longhorn storage"
    # DaemonSet이 각 node에 host path 디렉터리를 만들고 권한을 맞춘 뒤 바로 제거된다.
    kubectl apply -f "${ROOT_DIR}/build/kite-storage/longhorn/disk-directory-daemonset.yaml"
    kubectl -n longhorn-system rollout status daemonset/kite-longhorn-disk-directory --timeout="${TIMEOUT_SECONDS}s"
    kubectl delete -f "${ROOT_DIR}/build/kite-storage/longhorn/disk-directory-daemonset.yaml" --ignore-not-found=true
  else
    log "using existing Longhorn disks; set KITE_LONGHORN_USE_DEDICATED_DISK=true to add ${KITE_LONGHORN_DISK_PATH}"
  fi

  # Longhorn node CR 이름은 Kubernetes node 이름과 같으므로 node 목록을 기준으로 순회한다.
  kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' \
    | while read -r node; do
        [[ -z "${node}" ]] && continue
        if [[ "${KITE_LONGHORN_USE_DEDICATED_DISK}" == "true" ]]; then
          patch_longhorn_node_dedicated_disk "${node}"
        else
          tag_existing_longhorn_disks "${node}"
        fi
      done
}

main "$@"
