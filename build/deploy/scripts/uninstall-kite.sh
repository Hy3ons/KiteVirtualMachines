#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/deploy/scripts/uninstall-kite.sh
# Description: pull 기반 배포 리소스, 선택적 Longhorn 데이터, host sshd 복원을 정리한다.
#
# Usage:
#   build/deploy/scripts/uninstall-kite.sh
#
# Environment Variables:
#   KITE_NAMESPACE: default kite
#   DELETE_GOLDEN_IMAGE: default false
#   DELETE_LONGHORN: default false
#   DELETE_LONGHORN_FORCE: default false
#   DELETE_LONGHORN_DATA: default false
#   DELETE_LONGHORN_DATA_CONFIRM: default false
#   KITE_UNINSTALL_PRESET: default safe
#   KITE_LONGHORN_DISK_NAME: default kite-longhorn
#   KITE_LONGHORN_DISK_TAG: default kite
#   KITE_LONGHORN_OWNER_LABEL_KEY: default hy3ons.github.io/kite-installed-longhorn
#   KITE_LONGHORN_OWNER_LABEL_VALUE: default true
#   KITE_LONGHORN_DISK_REMOVE_TIMEOUT_SECONDS: default 180
#   KITE_LONGHORN_DISK_REMOVE_RETRY_SECONDS: default 5
#   RESTORE_HOST_SSHD: default true
#   KITE_RESTORE_HOST_SSHD: default ask
#   KITE_HOST_SSHD_STATE: default /etc/kite/host-sshd/state.env
#   KITE_HOST_SSHD_RESTORE_LOG: default /tmp/kite-host-sshd-restore.log
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Side Effects:
#   Kubernetes 리소스, 이미지 캐시, 선택적 Longhorn/host sshd 상태를 변경하거나 삭제할 수 있다.
# ==============================================================================

KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
DELETE_GOLDEN_IMAGE_WAS_SET="${DELETE_GOLDEN_IMAGE+x}"
DELETE_LONGHORN_WAS_SET="${DELETE_LONGHORN+x}"
DELETE_LONGHORN_FORCE_WAS_SET="${DELETE_LONGHORN_FORCE+x}"
DELETE_LONGHORN_DATA_WAS_SET="${DELETE_LONGHORN_DATA+x}"
DELETE_LONGHORN_DATA_CONFIRM_WAS_SET="${DELETE_LONGHORN_DATA_CONFIRM+x}"
RESTORE_HOST_SSHD_WAS_SET="${RESTORE_HOST_SSHD+x}"
KITE_RESTORE_HOST_SSHD_WAS_SET="${KITE_RESTORE_HOST_SSHD+x}"
KITE_UNINSTALL_PRESET="${KITE_UNINSTALL_PRESET:-safe}"
case "${KITE_UNINSTALL_PRESET}" in
  safe)
    DELETE_GOLDEN_IMAGE="${DELETE_GOLDEN_IMAGE:-false}"
    DELETE_LONGHORN="${DELETE_LONGHORN:-false}"
    DELETE_LONGHORN_DATA="${DELETE_LONGHORN_DATA:-false}"
    DELETE_LONGHORN_DATA_CONFIRM="${DELETE_LONGHORN_DATA_CONFIRM:-false}"
    ;;
  full)
    DELETE_GOLDEN_IMAGE="${DELETE_GOLDEN_IMAGE:-true}"
    DELETE_LONGHORN="${DELETE_LONGHORN:-true}"
    DELETE_LONGHORN_DATA="${DELETE_LONGHORN_DATA:-true}"
    DELETE_LONGHORN_DATA_CONFIRM="${DELETE_LONGHORN_DATA_CONFIRM:-true}"
    ;;
  *)
    echo "[kite-deploy] KITE_UNINSTALL_PRESET must be safe or full" >&2
    exit 1
    ;;
esac
DELETE_LONGHORN_FORCE="${DELETE_LONGHORN_FORCE:-false}"
KITE_LONGHORN_DISK_NAME="${KITE_LONGHORN_DISK_NAME:-kite-longhorn}"
KITE_LONGHORN_DISK_TAG="${KITE_LONGHORN_DISK_TAG:-kite}"
KITE_LONGHORN_OWNER_LABEL_KEY="${KITE_LONGHORN_OWNER_LABEL_KEY:-hy3ons.github.io/kite-installed-longhorn}"
KITE_LONGHORN_OWNER_LABEL_VALUE="${KITE_LONGHORN_OWNER_LABEL_VALUE:-true}"
KITE_LONGHORN_DISK_REMOVE_TIMEOUT_SECONDS="${KITE_LONGHORN_DISK_REMOVE_TIMEOUT_SECONDS:-180}"
KITE_LONGHORN_DISK_REMOVE_RETRY_SECONDS="${KITE_LONGHORN_DISK_REMOVE_RETRY_SECONDS:-5}"
# RESTORE_HOST_SSHD=true이면 배포 제거 후 Kite gateway 때문에 옮겨 둔 host sshd
# 설정을 원래 백업으로 복원한다. 원격 서버에서 현재 SSH 포트를 유지해야 하거나,
# 별도 방화벽/포트포워딩 점검 전이라면 RESTORE_HOST_SSHD=false로 이 단계를 막는다.
RESTORE_HOST_SSHD="${RESTORE_HOST_SSHD:-true}"
KITE_RESTORE_HOST_SSHD="${KITE_RESTORE_HOST_SSHD:-ask}"
KITE_HOST_SSHD_STATE="${KITE_HOST_SSHD_STATE:-/etc/kite/host-sshd/state.env}"
KITE_HOST_SSHD_RESTORE_LOG="${KITE_HOST_SSHD_RESTORE_LOG:-/tmp/kite-host-sshd-restore.log}"
HOST_SSHD_RESTORE_PID=""

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

configure_interactive_uninstall_options() {
  kite_prompt_interactive || return 0

  log "interactive uninstall options (preset=${KITE_UNINSTALL_PRESET})"
  kite_prompt_configure_bool DELETE_GOLDEN_IMAGE "${DELETE_GOLDEN_IMAGE_WAS_SET}" "Ubuntu golden image DataVolume/PVC까지 삭제할까요?"
  kite_prompt_configure_bool DELETE_LONGHORN "${DELETE_LONGHORN_WAS_SET}" "Longhorn 설치 자체와 Kite Longhorn disk entry까지 제거할까요?"
  if [[ "${DELETE_LONGHORN}" == "true" ]]; then
    kite_prompt_configure_bool DELETE_LONGHORN_FORCE "${DELETE_LONGHORN_FORCE_WAS_SET}" "Longhorn PV가 남아 있어도 Longhorn 제거를 강제로 진행할까요?"
  fi
  kite_prompt_configure_bool DELETE_LONGHORN_DATA "${DELETE_LONGHORN_DATA_WAS_SET}" "노드의 Kite Longhorn host data까지 삭제할까요? VM 디스크 데이터가 사라질 수 있습니다."
  if [[ "${DELETE_LONGHORN_DATA}" == "true" ]]; then
    if [[ -z "${DELETE_LONGHORN_DATA_CONFIRM_WAS_SET}" ]]; then
      DELETE_LONGHORN_DATA_CONFIRM=true
    fi
    kite_prompt_configure_bool DELETE_LONGHORN_FORCE "${DELETE_LONGHORN_FORCE_WAS_SET}" "Longhorn PV가 남아 있어도 host data 삭제를 강제로 진행할까요?"
  fi
  kite_prompt_configure_bool RESTORE_HOST_SSHD "${RESTORE_HOST_SSHD_WAS_SET}" "Kite gateway가 22번을 쓰던 경우 host sshd를 22번으로 복원할까요?"
  if [[ "${RESTORE_HOST_SSHD}" == "true" && -z "${KITE_RESTORE_HOST_SSHD_WAS_SET}" ]]; then
    KITE_RESTORE_HOST_SSHD=true
  elif [[ "${RESTORE_HOST_SSHD}" != "true" && -z "${KITE_RESTORE_HOST_SSHD_WAS_SET}" ]]; then
    KITE_RESTORE_HOST_SSHD=false
  fi

  log "uninstall choices: namespace=${KITE_NAMESPACE}, preset=${KITE_UNINSTALL_PRESET}, DELETE_GOLDEN_IMAGE=${DELETE_GOLDEN_IMAGE}($(kite_option_source "${DELETE_GOLDEN_IMAGE_WAS_SET}")), DELETE_LONGHORN_DATA=${DELETE_LONGHORN_DATA}($(kite_option_source "${DELETE_LONGHORN_DATA_WAS_SET}")), DELETE_LONGHORN=${DELETE_LONGHORN}($(kite_option_source "${DELETE_LONGHORN_WAS_SET}")), DELETE_LONGHORN_FORCE=${DELETE_LONGHORN_FORCE}($(kite_option_source "${DELETE_LONGHORN_FORCE_WAS_SET}")), RESTORE_HOST_SSHD=${RESTORE_HOST_SSHD}($(kite_option_source "${RESTORE_HOST_SSHD_WAS_SET}"))"
}

normalize_uninstall_options() {
  if [[ "${RESTORE_HOST_SSHD}" == "true" && -z "${KITE_RESTORE_HOST_SSHD_WAS_SET}" ]]; then
    KITE_RESTORE_HOST_SSHD=true
  elif [[ "${RESTORE_HOST_SSHD}" != "true" && -z "${KITE_RESTORE_HOST_SSHD_WAS_SET}" ]]; then
    KITE_RESTORE_HOST_SSHD=false
  fi
  export DELETE_GOLDEN_IMAGE
  export DELETE_LONGHORN
  export DELETE_LONGHORN_FORCE
  export DELETE_LONGHORN_DATA
  export DELETE_LONGHORN_DATA_CONFIRM
  export RESTORE_HOST_SSHD
  export KITE_RESTORE_HOST_SSHD
}

# host_sshd_restore_state_exists checks whether Kite previously moved host sshd away from port 22.
# The state file may require sudo because it is stored under /etc/kite on remote Linux hosts.
host_sshd_restore_state_exists() {
  [[ "$(uname -s 2>/dev/null || true)" == "Linux" ]] || return 1
  if [[ -f "${KITE_HOST_SSHD_STATE}" ]]; then
    return 0
  fi
  if command -v sudo >/dev/null 2>&1 && sudo test -f "${KITE_HOST_SSHD_STATE}" 2>/dev/null; then
    return 0
  fi
  return 1
}

# confirm_host_sshd_restore_before_gateway_delete asks before scheduling a detached restore.
# The answer is exported so foreground and background restore paths use the same decision.
confirm_host_sshd_restore_before_gateway_delete() {
  local answer

  case "${KITE_RESTORE_HOST_SSHD}" in
    true|yes|1)
      export KITE_RESTORE_HOST_SSHD=true
      return 0
      ;;
    false|no|0)
      export KITE_RESTORE_HOST_SSHD=false
      return 1
      ;;
    ask)
      if [[ ! -t 0 ]]; then
        warn "non-interactive shell; not scheduling host sshd restore because KITE_RESTORE_HOST_SSHD=ask"
        export KITE_RESTORE_HOST_SSHD=false
        return 1
      fi
      if kite_prompt_bool "Kite gateway가 현재 SSH 세션을 처리 중일 수 있습니다. gateway 삭제 전에 host sshd 22번 복원 worker를 예약할까요?" "false"; then
        export KITE_RESTORE_HOST_SSHD=true
        return 0
      fi
      export KITE_RESTORE_HOST_SSHD=false
      return 1
      ;;
    *)
      warn "unknown KITE_RESTORE_HOST_SSHD=${KITE_RESTORE_HOST_SSHD}; expected ask, true, or false"
      export KITE_RESTORE_HOST_SSHD=false
      return 1
      ;;
  esac
}

prepare_host_sshd_restore_log() {
  local log_dir

  if { : >>"${KITE_HOST_SSHD_RESTORE_LOG}"; } 2>/dev/null; then
    return 0
  fi

  log_dir="${TMPDIR:-/tmp}"
  KITE_HOST_SSHD_RESTORE_LOG="$(mktemp "${log_dir%/}/kite-host-sshd-restore.XXXXXX.log")"
}

# schedule_host_sshd_restore_before_gateway_delete starts a nohup restore worker before gateway deletion.
# This keeps the port-22 recovery alive even if deleting the gateway drops the current SSH session.
schedule_host_sshd_restore_before_gateway_delete() {
  local restore_cmd

  if [[ "${RESTORE_HOST_SSHD}" != "true" ]]; then
    return 0
  fi
  if ! host_sshd_restore_state_exists; then
    return 0
  fi
  if ! confirm_host_sshd_restore_before_gateway_delete; then
    warn "host sshd restore is disabled; deleting kite-gateway may leave host SSH away from port 22"
    return 0
  fi

  if [[ "${EUID:-$(id -u)}" == "0" ]]; then
    restore_cmd=(env)
  else
    if ! sudo -v; then
      warn "sudo is required before deleting kite-gateway so host sshd restore can run after port 22 is released"
      return 1
    fi
    restore_cmd=(sudo -n env)
  fi

  prepare_host_sshd_restore_log
  log "scheduling detached host sshd restore after port 22 is released"
  nohup "${restore_cmd[@]}" \
    KITE_RESTORE_HOST_SSHD=true \
    KITE_HOST_SSHD_STATE="${KITE_HOST_SSHD_STATE}" \
    KITE_HOST_SSHD_RESTORE_WAIT_TIMEOUT_SECONDS="${KITE_HOST_SSHD_RESTORE_WAIT_TIMEOUT_SECONDS:-90}" \
    KITE_HOST_SSHD_RESTORE_WAIT_RETRY_SECONDS="${KITE_HOST_SSHD_RESTORE_WAIT_RETRY_SECONDS:-1}" \
    "${ROOT_DIR}/build/deploy/scripts/manage-host-sshd.sh" restore-after-port-free >>"${KITE_HOST_SSHD_RESTORE_LOG}" 2>&1 &
  HOST_SSHD_RESTORE_PID="$!"
  log "host sshd restore worker pid=${HOST_SSHD_RESTORE_PID}; log=${KITE_HOST_SSHD_RESTORE_LOG}"
}

# finish_host_sshd_restore waits for the scheduled restore worker or runs a foreground restore fallback.
# When the SSH session survived gateway deletion this gives uninstall a deterministic success/failure result.
finish_host_sshd_restore() {
  if [[ "${RESTORE_HOST_SSHD}" != "true" ]]; then
    log "skipping host sshd restore because RESTORE_HOST_SSHD=${RESTORE_HOST_SSHD}"
    return 0
  fi
  if [[ -n "${HOST_SSHD_RESTORE_PID}" ]]; then
    if wait "${HOST_SSHD_RESTORE_PID}"; then
      log "host sshd restore worker completed"
    else
      warn "host sshd restore worker failed; check ${KITE_HOST_SSHD_RESTORE_LOG}"
      return 1
    fi
    return 0
  fi

  "${ROOT_DIR}/build/deploy/scripts/manage-host-sshd.sh" restore-after-port-free
}


# Longhorn CRD가 있는 클러스터에서 Kite가 만든/태그한 disk entry를 node별로 제거한다.
remove_kite_longhorn_disks() {
  if ! kubectl get crd nodes.longhorn.io >/dev/null 2>&1; then
    return
  fi

  log "removing Kite Longhorn disk entries from Longhorn node resources"
  kubectl -n longhorn-system get nodes.longhorn.io -o name 2>/dev/null \
    | while read -r node; do
        [[ -z "${node}" ]] && continue
        remove_kite_longhorn_disks_from_node "${node}"
      done
}

# JSON Patch path에서 /와 ~는 escape해야 disk 이름에 특수문자가 있어도 patch가 안전하다.
json_pointer_escape() {
  local value="$1"

  value="${value//\~/~0}"
  value="${value//\//~1}"
  printf '%s' "${value}"
}

# node.longhorn.io의 특정 disk entry를 제거한다. Longhorn이 replica/backing image를
# 아직 잡고 있으면 scheduling을 잠시 끄고 controller 정리를 기다린 뒤 재시도한다.
remove_longhorn_disk_from_node() {
  local node="$1"
  local disk="$2"
  local reason="$3"
  local escaped_disk
  local deadline
  local start_time
  local original_allow_scheduling
  local output

  escaped_disk="$(json_pointer_escape "${disk}")"
  start_time="${SECONDS}"
  original_allow_scheduling="$(kubectl -n longhorn-system get "${node}" -o "go-template={{ index .spec.disks \"${disk}\" \"allowScheduling\" }}" 2>/dev/null || true)"

  # Longhorn은 replica가 남아 있으면 disk 삭제를 막는다. 새 replica 배치를 막기 위해
  # allowScheduling부터 꺼 두고, 실패하면 아래에서 원래 값으로 되돌린다.
  kubectl -n longhorn-system patch "${node}" --type=json -p "[{\"op\":\"replace\",\"path\":\"/spec/disks/${escaped_disk}/allowScheduling\",\"value\":false}]" >/dev/null 2>&1 || true

  # PVC/PV 삭제가 Longhorn controller에 반영되는 데 시간이 걸리므로 제한 시간 안에서 재시도한다.
  deadline=$((SECONDS + KITE_LONGHORN_DISK_REMOVE_TIMEOUT_SECONDS))
  while true; do
    if output="$(kubectl -n longhorn-system patch "${node}" --type=json -p "[{\"op\":\"remove\",\"path\":\"/spec/disks/${escaped_disk}\"}]" 2>&1)"; then
      log "removed Longhorn disk ${disk} from ${node} (${reason})"
      return
    fi

    if [[ "${output}" != *"remove all replicas and backing images first"* || "${SECONDS}" -ge "${deadline}" ]]; then
      break
    fi

    log "waiting to remove Longhorn disk ${disk} from ${node}; Longhorn still sees replicas or backing images (elapsed=$((SECONDS - start_time))s timeout=${KITE_LONGHORN_DISK_REMOVE_TIMEOUT_SECONDS}s)"
    sleep "${KITE_LONGHORN_DISK_REMOVE_RETRY_SECONDS}"
  done

  # 끝까지 실패하면 cleanup 때문에 클러스터가 scheduling 불가 상태로 남지 않도록 원래 값을 복원한다.
  if [[ "${original_allow_scheduling}" == "true" || "${original_allow_scheduling}" == "false" ]]; then
    kubectl -n longhorn-system patch "${node}" --type=json -p "[{\"op\":\"replace\",\"path\":\"/spec/disks/${escaped_disk}/allowScheduling\",\"value\":${original_allow_scheduling}}]" >/dev/null 2>&1 || true
  fi
  log "could not remove Longhorn disk ${disk} from ${node} after $((SECONDS - start_time))s (${reason}): ${output}"
}

remove_longhorn_disk_tag_from_node() {
  local node="$1"
  local disk="$2"
  local tags="$3"
  local escaped_disk
  local next_tags

  escaped_disk="$(json_pointer_escape "${disk}")"
  next_tags="$(printf '%s\n' "${tags}" | xargs -n 1 | awk -v remove="${KITE_LONGHORN_DISK_TAG}" 'NF && $0 != remove && !seen[$0]++ { printf "%s\"%s\"", sep, $0; sep=", " }')"
  log "removing Longhorn disk tag ${KITE_LONGHORN_DISK_TAG} from ${node}/${disk}"
  kubectl -n longhorn-system patch "${node}" --type=json -p "[{\"op\":\"replace\",\"path\":\"/spec/disks/${escaped_disk}/tags\",\"value\":[${next_tags}]}]" >/dev/null 2>&1 || true
}

# 설치 방식에 따라 Kite 전용 disk 이름이 있거나 기존 disk에 kite tag만 있을 수 있어 둘 다 찾는다.
remove_kite_longhorn_disks_from_node() {
  local node="$1"
  local disks
  local disk
  local tags

  # spec.disks 전체를 이름|태그 형식으로 뽑아 shell에서 간단히 판별한다.
  disks="$(kubectl -n longhorn-system get "${node}" -o 'go-template={{ range $name, $disk := .spec.disks }}{{ $name }}|{{ range $disk.tags }}{{ . }} {{ end }}{{ "\n" }}{{ end }}' 2>/dev/null || true)"
  while IFS="|" read -r disk tags; do
    [[ -z "${disk}" ]] && continue

    if [[ "${disk}" == "${KITE_LONGHORN_DISK_NAME}" ]]; then
      remove_longhorn_disk_from_node "${node}" "${disk}" "configured Kite disk name"
      continue
    fi

    if [[ " ${tags} " == *" ${KITE_LONGHORN_DISK_TAG} "* ]]; then
      remove_longhorn_disk_tag_from_node "${node}" "${disk}" "${tags}"
    fi
  done <<< "${disks}"
}

# Longhorn PV가 남아 있으면 host 데이터 삭제나 Longhorn uninstall이 데이터 손실로 이어질 수 있다.
longhorn_pv_count() {
  kubectl get pv -o jsonpath='{range .items[?(@.spec.csi.driver=="driver.longhorn.io")]}{.metadata.name}{"\n"}{end}' 2>/dev/null \
    | sed '/^[[:space:]]*$/d' \
    | wc -l \
    | tr -d ' '
}

longhorn_installed_by_kite() {
  local value

  value="$(kubectl get namespace longhorn-system -o "go-template={{ index .metadata.labels \"${KITE_LONGHORN_OWNER_LABEL_KEY}\" }}" 2>/dev/null || true)"
  [[ "${value}" == "${KITE_LONGHORN_OWNER_LABEL_VALUE}" ]]
}

longhorn_pv_lines() {
  kubectl get pv -o 'go-template={{ range .items }}{{ if .spec.csi }}{{ if eq .spec.csi.driver "driver.longhorn.io" }}{{ .metadata.name }}|{{ if .spec.claimRef }}{{ .spec.claimRef.namespace }}|{{ .spec.claimRef.name }}{{ else }}|{{ end }}{{ "\n" }}{{ end }}{{ end }}{{ end }}' 2>/dev/null || true
}

namespace_managed_by_kite() {
  local namespace="$1"
  local value

  value="$(kubectl get namespace "${namespace}" -o 'go-template={{ index .metadata.labels "hy3ons.github.io/managed-by" }}' 2>/dev/null || true)"
  [[ "${value}" == "kite-controller" ]]
}

pvc_managed_by_kite() {
  local namespace="$1"
  local pvc="$2"
  local value

  value="$(kubectl -n "${namespace}" get pvc "${pvc}" -o 'go-template={{ index .metadata.labels "hy3ons.github.io/managed-by" }}' 2>/dev/null || true)"
  [[ "${value}" == "kite-controller" ]]
}

longhorn_pv_is_kite_owned() {
  local namespace="$1"
  local pvc="$2"

  [[ -z "${namespace}" || -z "${pvc}" ]] && return 1
  if [[ "${namespace}" == "${KITE_NAMESPACE}" ]]; then
    pvc_managed_by_kite "${namespace}" "${pvc}" && return 0
    [[ "${pvc}" == "ubuntu-22.04" ]] && return 0
    return 1
  fi
  namespace_managed_by_kite "${namespace}" && return 0
  pvc_managed_by_kite "${namespace}" "${pvc}" && return 0
  return 1
}

external_longhorn_pv_count() {
  local pv
  local namespace
  local pvc
  local count=0

  while IFS="|" read -r pv namespace pvc; do
    [[ -z "${pv}" ]] && continue
    if ! longhorn_pv_is_kite_owned "${namespace}" "${pvc}"; then
      count=$((count + 1))
    fi
  done < <(longhorn_pv_lines)

  printf '%s\n' "${count}"
}

kite_namespace_has_external_pvcs() {
  local pvc

  kubectl get namespace "${KITE_NAMESPACE}" >/dev/null 2>&1 || return 1
  while read -r pvc; do
    [[ -z "${pvc}" ]] && continue
    pvc_managed_by_kite "${KITE_NAMESPACE}" "${pvc}" && continue
    [[ "${pvc}" == "ubuntu-22.04" ]] && continue
    return 0
  done < <(kubectl -n "${KITE_NAMESPACE}" get pvc -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)

  return 1
}

delete_kite_manifests_preserving_namespace() {
  log "deleting Kite manifests without deleting namespace/${KITE_NAMESPACE} because external PVCs exist"
  kubectl delete -f "${ROOT_DIR}/build/kite/gateway.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/controller.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/api.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/frontend.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/config.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/rbac.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/serviceaccount.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/crds.yaml" --ignore-not-found=true --wait=false || true
}

delete_kite_manifests() {
  if kite_namespace_has_external_pvcs; then
    delete_kite_manifests_preserving_namespace
    return
  fi

  kubectl delete -k "${ROOT_DIR}/build/kite" --ignore-not-found=true || true
}

delete_longhorn_webhook_configurations() {
  kubectl get validatingwebhookconfigurations.admissionregistration.k8s.io -o name 2>/dev/null \
    | grep 'longhorn' \
    | xargs -r kubectl delete --ignore-not-found=true || true
  kubectl get mutatingwebhookconfigurations.admissionregistration.k8s.io -o name 2>/dev/null \
    | grep 'longhorn' \
    | xargs -r kubectl delete --ignore-not-found=true || true
}

clear_longhorn_finalizers() {
  kubectl api-resources --api-group=longhorn.io --verbs=list --namespaced=true -o name 2>/dev/null \
    | while read -r resource; do
        [[ -z "${resource}" ]] && continue
        kubectl -n longhorn-system get "${resource}" -o name 2>/dev/null \
          | xargs -r -n 1 kubectl -n longhorn-system patch --type=merge -p '{"metadata":{"finalizers":[]}}' || true
      done

  kubectl api-resources --api-group=longhorn.io --verbs=list --namespaced=false -o name 2>/dev/null \
    | while read -r resource; do
        [[ -z "${resource}" ]] && continue
        kubectl get "${resource}" -o name 2>/dev/null \
          | xargs -r -n 1 kubectl patch --type=merge -p '{"metadata":{"finalizers":[]}}' || true
      done
}

delete_longhorn_crds() {
  kubectl get crd -o name 2>/dev/null \
    | grep 'longhorn.io' \
    | xargs -r kubectl delete --wait=false || true
}

# 명시 확인이 있을 때만 host path의 Kite Longhorn 데이터를 DaemonSet으로 삭제한다.
delete_kite_longhorn_host_data() {
  if [[ "${DELETE_LONGHORN_DATA}" != "true" ]]; then
    log "skipping Longhorn host data cleanup because DELETE_LONGHORN_DATA=${DELETE_LONGHORN_DATA}"
    return
  fi
  if [[ "${DELETE_LONGHORN_DATA_CONFIRM}" != "true" ]]; then
    echo "[kite-deploy] refusing Longhorn host data deletion without DELETE_LONGHORN_DATA_CONFIRM=true" >&2
    exit 1
  fi
  local external_pv_count
  external_pv_count="$(external_longhorn_pv_count)"
  if [[ "${external_pv_count}" != "0" ]]; then
    log "skipping Kite Longhorn host data cleanup because ${external_pv_count} external Longhorn PV(s) still exist"
    log "delete or migrate non-Kite Longhorn PVC/PV resources before deleting host data"
    return
  fi
  if [[ "${DELETE_LONGHORN_FORCE}" != "true" ]]; then
    local pv_count
    pv_count="$(longhorn_pv_count)"
    if [[ "${pv_count}" != "0" ]]; then
      log "skipping Kite Longhorn host data cleanup because ${pv_count} Longhorn PV(s) still exist"
      log "delete remaining Longhorn PVC/PV resources first, or set DELETE_LONGHORN_FORCE=true"
      return
    fi
  fi

  log "deleting Kite Longhorn host data on every node"
  # cleanup DaemonSet은 각 node에서 host path를 지운 뒤 제거된다.
  kubectl apply -f "${ROOT_DIR}/build/kite-storage/longhorn-cleanup"
  kubectl -n longhorn-system rollout status daemonset/kite-longhorn-disk-cleanup --timeout=180s || true
  kubectl delete -f "${ROOT_DIR}/build/kite-storage/longhorn-cleanup" --ignore-not-found=true || true
}

# DELETE_LONGHORN=true일 때 Longhorn 자체 리소스까지 제거한다. PV가 남으면 기본적으로 중단한다.
delete_longhorn_resources() {
  if [[ "${DELETE_LONGHORN}" != "true" ]]; then
    return
  fi

  local external_pv_count
  external_pv_count="$(external_longhorn_pv_count)"
  if [[ "${external_pv_count}" != "0" ]]; then
    log "skipping Longhorn disk cleanup and uninstall because ${external_pv_count} external Longhorn PV(s) still exist"
    log "DELETE_LONGHORN_FORCE does not override non-Kite Longhorn PVC/PV protection"
    return
  fi

  remove_kite_longhorn_disks

  if ! longhorn_installed_by_kite; then
    log "skipping Longhorn uninstall because longhorn-system is not marked as Kite-installed"
    log "Kite cleanup will not delete shared Longhorn without ${KITE_LONGHORN_OWNER_LABEL_KEY}=${KITE_LONGHORN_OWNER_LABEL_VALUE}"
    return
  fi

  if [[ "${DELETE_LONGHORN_FORCE}" != "true" ]]; then
    local pv_count
    pv_count="$(longhorn_pv_count)"
    if [[ "${pv_count}" != "0" ]]; then
      log "skipping Longhorn uninstall because ${pv_count} Longhorn PV(s) still exist"
      log "delete remaining Longhorn PVC/PV resources first, or set DELETE_LONGHORN_FORCE=true"
      return
    fi
  fi

  log "deleting Longhorn resources"
  kubectl delete storageclass longhorn --ignore-not-found=true || true
  delete_longhorn_webhook_configurations
  kubectl delete namespace longhorn-system --ignore-not-found=true --wait=false || true

  # namespace terminating에 걸린 Longhorn CR은 finalizer 때문에 남을 수 있어 강제로 비운다.
  clear_longhorn_finalizers
  delete_longhorn_crds
}

# 배포 제거의 전체 순서다. Kite 리소스, 선택적 storage 데이터, Longhorn, host sshd 복원을 처리한다.
main() {
  configure_interactive_uninstall_options
  normalize_uninstall_options
  schedule_host_sshd_restore_before_gateway_delete

  if [[ "${DELETE_GOLDEN_IMAGE}" == "true" ]]; then
    log "deleting golden image DataVolumes and PVCs"
    kubectl delete -f "${ROOT_DIR}/build/kite-storage/golden-images" --ignore-not-found=true || true
    kubectl -n "${KITE_NAMESPACE}" delete datavolume ubuntu-22.04 --ignore-not-found=true
    kubectl -n "${KITE_NAMESPACE}" delete pvc ubuntu-22.04 --ignore-not-found=true
  fi

  log "deleting Kite manifests"
  delete_kite_manifests
  kubectl delete -f "${ROOT_DIR}/build/kite-storage/longhorn" --ignore-not-found=true || true
  delete_kite_longhorn_host_data

  log "deleting remaining Kite namespace and cluster-scoped resources"
  if kite_namespace_has_external_pvcs; then
    log "skipping namespace/${KITE_NAMESPACE} deletion because it contains non-Kite PVCs"
  else
    kubectl delete namespace "${KITE_NAMESPACE}" --ignore-not-found=true
  fi
  kubectl delete crd kiteusers.hy3ons.github.io kitevirtualmachines.hy3ons.github.io --ignore-not-found=true
  kubectl delete clusterrole kite-api-role kite-controller-role kite-gateway-role --ignore-not-found=true
  kubectl delete clusterrolebinding kite-api-binding kite-controller-binding kite-gateway-binding --ignore-not-found=true
  kubectl delete clusterrole kite-control-plane-role --ignore-not-found=true
  kubectl delete clusterrolebinding kite-control-plane-binding --ignore-not-found=true

  delete_longhorn_resources

  finish_host_sshd_restore
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
