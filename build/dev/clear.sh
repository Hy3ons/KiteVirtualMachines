#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Script: build/dev/clear.sh
# Description: 개발 클러스터에서 Kite 리소스, user namespace, 선택적 Longhorn 데이터와 이미지 캐시를 정리한다.
#
# Usage:
#   build/dev/clear.sh
#
# Environment Variables:
#   KITE_CLUSTER: default auto
#   KITE_NAMESPACE: default kite
#   MINIKUBE_PROFILE: default minikube
#   MINIKUBE_PURGE: default false
#   K3S_CTR_CMD: default sudo k3s ctr -n k8s.io
#   CLEAR_IMAGES: default true
#   CLEAR_LONGHORN: default false
#   CLEAR_LONGHORN_FORCE: default false
#   CLEAR_LONGHORN_DATA: default false
#   CLEAR_LONGHORN_DATA_CONFIRM: default false
#   KITE_LONGHORN_DISK_NAME: default kite-longhorn
#   KITE_LONGHORN_DISK_TAG: default kite
#   KITE_LONGHORN_OWNER_LABEL_KEY: default hy3ons.github.io/kite-installed-longhorn
#   KITE_LONGHORN_OWNER_LABEL_VALUE: default true
#   KITE_LONGHORN_DISK_REMOVE_TIMEOUT_SECONDS: default 180
#   KITE_LONGHORN_DISK_REMOVE_RETRY_SECONDS: default 5
#   KITE_LOG_COLOR: default auto
#   NO_COLOR: default (unset)
#
# Interactive:
#   TTY에서 직접 실행하면 명시적으로 지정하지 않은 위험 옵션을 번호 선택으로 묻는다.
#
# Side Effects:
#   Kubernetes 리소스, 이미지 캐시, 선택적 Longhorn 상태를 변경하거나 삭제할 수 있다.
# ==============================================================================

MINIKUBE_PURGE_WAS_SET="${MINIKUBE_PURGE+x}"
CLEAR_IMAGES_WAS_SET="${CLEAR_IMAGES+x}"
CLEAR_LONGHORN_WAS_SET="${CLEAR_LONGHORN+x}"
CLEAR_LONGHORN_FORCE_WAS_SET="${CLEAR_LONGHORN_FORCE+x}"
CLEAR_LONGHORN_DATA_WAS_SET="${CLEAR_LONGHORN_DATA+x}"
CLEAR_LONGHORN_DATA_CONFIRM_WAS_SET="${CLEAR_LONGHORN_DATA_CONFIRM+x}"

KITE_CLUSTER="${KITE_CLUSTER:-auto}"
KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MINIKUBE_PROFILE="${MINIKUBE_PROFILE:-minikube}"
MINIKUBE_PURGE="${MINIKUBE_PURGE:-false}"
K3S_CTR_CMD="${K3S_CTR_CMD:-sudo k3s ctr -n k8s.io}"
CLEAR_IMAGES="${CLEAR_IMAGES:-true}"
CLEAR_LONGHORN="${CLEAR_LONGHORN:-false}"
CLEAR_LONGHORN_FORCE="${CLEAR_LONGHORN_FORCE:-false}"
CLEAR_LONGHORN_DATA="${CLEAR_LONGHORN_DATA:-false}"
CLEAR_LONGHORN_DATA_CONFIRM="${CLEAR_LONGHORN_DATA_CONFIRM:-false}"
KITE_LONGHORN_DISK_NAME="${KITE_LONGHORN_DISK_NAME:-kite-longhorn}"
KITE_LONGHORN_DISK_TAG="${KITE_LONGHORN_DISK_TAG:-kite}"
KITE_LONGHORN_OWNER_LABEL_KEY="${KITE_LONGHORN_OWNER_LABEL_KEY:-hy3ons.github.io/kite-installed-longhorn}"
KITE_LONGHORN_OWNER_LABEL_VALUE="${KITE_LONGHORN_OWNER_LABEL_VALUE:-true}"
KITE_LONGHORN_DISK_REMOVE_TIMEOUT_SECONDS="${KITE_LONGHORN_DISK_REMOVE_TIMEOUT_SECONDS:-180}"
KITE_LONGHORN_DISK_REMOVE_RETRY_SECONDS="${KITE_LONGHORN_DISK_REMOVE_RETRY_SECONDS:-5}"
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

ask_numbered_bool() {
  local prompt="$1"
  local default_value="$2"
  local default_label
  local answer

  if [[ "${default_value}" == "true" ]]; then
    default_label="1"
  else
    default_label="2"
  fi

  while true; do
    printf '%s\n' "${prompt}" >&2
    printf '  1) 예\n' >&2
    printf '  2) 아니오\n' >&2
    read -r -p "선택 [1/2, 기본: ${default_label}] " answer
    answer="${answer:-${default_label}}"

    case "${answer}" in
      1|y|Y|yes|YES)
        return 0
        ;;
      2|n|N|no|NO)
        return 1
        ;;
      *)
        warn "1 또는 2를 입력하세요"
        ;;
    esac
  done
}

interactive_clear_enabled() {
  [[ -t 0 && "${KITE_ASSUME_DEFAULTS:-false}" != "true" ]]
}

configure_interactive_clear_options() {
  local cluster="$1"

  interactive_clear_enabled || return 0

  log "interactive cleanup options"
  if [[ -z "${CLEAR_IMAGES_WAS_SET}" ]]; then
    if ask_numbered_bool "로컬/k3s에 남은 Kite 개발 이미지도 삭제할까요?" "${CLEAR_IMAGES}"; then
      CLEAR_IMAGES=true
    else
      CLEAR_IMAGES=false
    fi
  fi

  if [[ "${cluster}" == "minikube" ]]; then
    if [[ -z "${MINIKUBE_PURGE_WAS_SET}" ]]; then
      if ask_numbered_bool "minikube profile만이 아니라 전체 minikube 상태를 purge할까요?" "${MINIKUBE_PURGE}"; then
        MINIKUBE_PURGE=true
      else
        MINIKUBE_PURGE=false
      fi
    fi
    log "cleanup choices: CLEAR_IMAGES=${CLEAR_IMAGES}, MINIKUBE_PURGE=${MINIKUBE_PURGE}"
    return 0
  fi

  if [[ -z "${CLEAR_LONGHORN_WAS_SET}" ]]; then
    if ask_numbered_bool "Longhorn 설치 자체와 Kite Longhorn disk entry까지 제거할까요?" "${CLEAR_LONGHORN}"; then
      CLEAR_LONGHORN=true
    else
      CLEAR_LONGHORN=false
    fi
  fi
  if [[ "${CLEAR_LONGHORN}" == "true" && -z "${CLEAR_LONGHORN_FORCE_WAS_SET}" ]]; then
    if ask_numbered_bool "Longhorn PV가 남아 있어도 Longhorn 제거를 강제로 진행할까요?" "${CLEAR_LONGHORN_FORCE}"; then
      CLEAR_LONGHORN_FORCE=true
    else
      CLEAR_LONGHORN_FORCE=false
    fi
  fi

  if [[ -z "${CLEAR_LONGHORN_DATA_WAS_SET}" ]]; then
    if ask_numbered_bool "노드의 Kite Longhorn host data까지 삭제할까요? VM 디스크 데이터가 사라질 수 있습니다." "${CLEAR_LONGHORN_DATA}"; then
      CLEAR_LONGHORN_DATA=true
      if [[ -z "${CLEAR_LONGHORN_DATA_CONFIRM_WAS_SET}" ]]; then
        CLEAR_LONGHORN_DATA_CONFIRM=true
      fi
    else
      CLEAR_LONGHORN_DATA=false
    fi
  fi
  if [[ "${CLEAR_LONGHORN_DATA}" == "true" && -z "${CLEAR_LONGHORN_FORCE_WAS_SET}" ]]; then
    if ask_numbered_bool "Longhorn PV가 남아 있어도 host data 삭제를 강제로 진행할까요?" "${CLEAR_LONGHORN_FORCE}"; then
      CLEAR_LONGHORN_FORCE=true
    fi
  fi

  log "cleanup choices: namespace=${KITE_NAMESPACE}, CLEAR_IMAGES=${CLEAR_IMAGES}, CLEAR_LONGHORN=${CLEAR_LONGHORN}, CLEAR_LONGHORN_DATA=${CLEAR_LONGHORN_DATA}, CLEAR_LONGHORN_FORCE=${CLEAR_LONGHORN_FORCE}"
}

normalize_clear_options() {
  export CLEAR_IMAGES
  export CLEAR_LONGHORN
  export CLEAR_LONGHORN_FORCE
  export CLEAR_LONGHORN_DATA
  export CLEAR_LONGHORN_DATA_CONFIRM
}


# 정리 과정에 꼭 필요한 CLI가 없으면 삭제가 절반만 진행되기 전에 중단한다.
require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

# KITE_CLUSTER=auto일 때 kubectl context와 로컬 k3s 명령 존재 여부로 대상 클러스터를 추정한다.
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

# Kite app/storage manifest와 user namespace에 흩어진 VM 관련 리소스를 삭제한다.
delete_kite_resources() {
  require_command kubectl

  log "deleting Kite storage manifests"
  # golden image와 Longhorn StorageClass부터 지워 새 VM disk 생성이 더 진행되지 않게 한다.
  kubectl delete -f "${ROOT_DIR}/build/kite-storage/golden-images" --ignore-not-found=true || true
  kubectl delete -f "${ROOT_DIR}/build/kite-storage/longhorn" --ignore-not-found=true || true

  log "stopping Kite workloads"
  # workload 삭제는 wait=false로 걸어 두고, 아래에서 CR/namespace 잔여 리소스를 따로 정리한다.
  kubectl delete -f "${ROOT_DIR}/build/kite/gateway.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/controller.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/api.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/frontend.yaml" --ignore-not-found=true --wait=false || true

  log "deleting Kite allocated resources from user namespaces"
  delete_kite_allocated_resources
  delete_kite_user_namespaces

  log "deleting Kite custom resources"
  delete_kite_custom_resources

  log "deleting Kite shared manifests"
  kubectl delete -f "${ROOT_DIR}/build/kite/config.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/rbac.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete -f "${ROOT_DIR}/build/kite/serviceaccount.yaml" --ignore-not-found=true --wait=false || true

  log "deleting Kite namespace-scoped resources from namespace/${KITE_NAMESPACE}"
  if kite_namespace_has_external_pvcs; then
    log "skipping namespace/${KITE_NAMESPACE} deletion because it contains non-Kite PVCs"
  else
    kubectl delete namespace "${KITE_NAMESPACE}" --ignore-not-found=true --wait=false || true
  fi

  log "deleting remaining Kite cluster-scoped resources"
  kubectl delete -f "${ROOT_DIR}/build/kite/crds.yaml" --ignore-not-found=true --wait=false || true
  kubectl delete clusterrole kite-api-role kite-controller-role kite-gateway-role --ignore-not-found=true --wait=false || true
  kubectl delete clusterrolebinding kite-api-binding kite-controller-binding kite-gateway-binding --ignore-not-found=true --wait=false || true
  kubectl delete clusterrole kite-control-plane-role --ignore-not-found=true --wait=false || true
  kubectl delete clusterrolebinding kite-control-plane-binding --ignore-not-found=true --wait=false || true

}

# 특정 API resource가 현재 클러스터에서 조회 가능한지 확인한다.
resource_available() {
  local resource="$1"

  kubectl api-resources -o name 2>/dev/null | grep -qx "${resource}"
}

# KiteUser status.namespace와 kite-user-* 이름 패턴을 합쳐 사용자 namespace 후보를 찾는다.
kite_user_namespaces() {
  if kubectl get crd kiteusers.hy3ons.github.io >/dev/null 2>&1; then
    kubectl get kiteusers.hy3ons.github.io -o jsonpath='{range .items[*]}{.spec.namespace}{"\n"}{end}' 2>/dev/null || true
  fi

  kubectl get namespace -l hy3ons.github.io/managed-by=kite-controller -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true
}

# selector로 찾은 리소스가 terminating에 걸리지 않게 finalizer를 비운다.
patch_finalizers_by_selector() {
  local namespace="$1"
  local resource="$2"
  local selector="$3"

  resource_available "${resource}" || return 0
  kubectl -n "${namespace}" get "${resource}" -l "${selector}" -o name 2>/dev/null \
    | xargs -r -n 1 kubectl -n "${namespace}" patch --type=merge -p '{"metadata":{"finalizers":[]}}' || true
}

# selector로 선택한 namespaced 리소스를 삭제한다.
delete_by_selector() {
  local namespace="$1"
  local resource="$2"
  local selector="$3"

  resource_available "${resource}" || return 0
  if [[ -n "${selector}" ]]; then
    kubectl -n "${namespace}" delete "${resource}" -l "${selector}" --ignore-not-found=true --wait=false 2>/dev/null || true
  else
    kubectl -n "${namespace}" delete "${resource}" --all --ignore-not-found=true --wait=false 2>/dev/null || true
  fi
}

# 이름 prefix로 선택한 리소스를 삭제한다. service/vps-*처럼 label이 없을 수 있는 리소스에 쓴다.
delete_by_name_prefix() {
  local namespace="$1"
  local resource="$2"
  local prefix="$3"

  resource_available "${resource}" || return 0
  kubectl -n "${namespace}" get "${resource}" -o name 2>/dev/null \
    | awk -F/ -v prefix="${prefix}" '$2 ~ "^" prefix { print }' \
    | xargs -r -n 1 kubectl -n "${namespace}" delete --ignore-not-found=true --wait=false || true
}

# Kite DataVolume과 연결된 PVC를 찾아 삭제한다. CDI가 만든 PVC는 label이 항상 일정하지 않을 수 있다.
delete_pvcs_for_kite_datavolumes() {
  local namespace="$1"

  resource_available datavolumes.cdi.kubevirt.io || return 0
  kubectl -n "${namespace}" get datavolumes.cdi.kubevirt.io -l hy3ons.github.io/managed-by=kite-controller -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null \
    | while read -r name; do
        [[ -z "${name}" ]] && continue
        kubectl -n "${namespace}" patch pvc "${name}" --type=merge -p '{"metadata":{"finalizers":[]}}' 2>/dev/null || true
        kubectl -n "${namespace}" delete pvc "${name}" --ignore-not-found=true --wait=false 2>/dev/null || true
      done
}

# 특정 user namespace 안에서 Kite VM이 만든 KubeVirt/CDI/Service 리소스를 정리한다.
delete_kite_namespace_allocations() {
  local namespace="$1"
  local selector="hy3ons.github.io/managed-by=kite-controller"

  [[ -z "${namespace}" || "${namespace}" == "${KITE_NAMESPACE}" ]] && return 0
  if ! kubectl get namespace "${namespace}" >/dev/null 2>&1; then
    return 0
  fi

  log "deleting Kite allocations in namespace/${namespace}"
  patch_finalizers_by_selector "${namespace}" virtualmachines.kubevirt.io "${selector}"
  patch_finalizers_by_selector "${namespace}" virtualmachineinstances.kubevirt.io "${selector}"
  patch_finalizers_by_selector "${namespace}" datavolumes.cdi.kubevirt.io "${selector}"
  patch_finalizers_by_selector "${namespace}" persistentvolumeclaims "${selector}"

  delete_pvcs_for_kite_datavolumes "${namespace}"
  delete_by_selector "${namespace}" virtualmachines.kubevirt.io "${selector}"
  delete_by_selector "${namespace}" virtualmachineinstances.kubevirt.io "${selector}"
  delete_by_selector "${namespace}" datavolumes.cdi.kubevirt.io "${selector}"
  delete_by_selector "${namespace}" persistentvolumeclaims "${selector}"
  delete_by_selector "${namespace}" pods "${selector}"
  delete_by_name_prefix "${namespace}" pods virt-launcher-
  delete_by_selector "${namespace}" services "${selector}"
  delete_by_selector "${namespace}" ingresses.networking.k8s.io "${selector}"
  delete_by_selector "${namespace}" secrets "${selector}"
  delete_by_selector "${namespace}" networkpolicies.networking.k8s.io ""
  kubectl -n "${namespace}" delete resourcequota kite-user-quota-policy --ignore-not-found >/dev/null 2>&1 || true
}

# KiteUser가 만든 모든 user namespace를 순회하며 VM 관련 리소스를 먼저 삭제한다.
delete_kite_allocated_resources() {
  kite_user_namespaces \
    | sed '/^[[:space:]]*$/d' \
    | sort -u \
    | while read -r namespace; do
        delete_kite_namespace_allocations "${namespace}"
      done
}

# kite-user-* namespace 자체를 비동기로 삭제한다. namespace finalization은 Kubernetes에 맡긴다.
delete_kite_user_namespaces() {
  kite_user_namespaces \
    | sed '/^[[:space:]]*$/d' \
    | sort -u \
    | while read -r namespace; do
        [[ -z "${namespace}" || "${namespace}" == "${KITE_NAMESPACE}" ]] && continue
        log "deleting Kite user namespace/${namespace}"
        kubectl delete namespace "${namespace}" --ignore-not-found=true --wait=false || true
      done
}

# Kite CRD 인스턴스를 삭제한다. controller가 이미 내려간 뒤에도 CR이 남을 수 있어 별도 처리한다.
delete_kite_custom_resources() {
  if kubectl get crd kitevirtualmachines.hy3ons.github.io >/dev/null 2>&1; then
    kubectl get kitevirtualmachines.hy3ons.github.io -A -o jsonpath='{range .items[*]}{.metadata.namespace}{" "}{.metadata.name}{"\n"}{end}' 2>/dev/null \
      | while read -r namespace name; do
          [[ -z "${namespace}" || -z "${name}" ]] && continue
          kubectl -n "${namespace}" patch kitevirtualmachines.hy3ons.github.io "${name}" --type=merge -p '{"metadata":{"finalizers":[]}}' || true
        done
    kubectl delete kitevirtualmachines.hy3ons.github.io -A --all --ignore-not-found=true --wait=false || true
  fi

  if kubectl get crd kiteusers.hy3ons.github.io >/dev/null 2>&1; then
    kubectl get kiteusers.hy3ons.github.io -o name 2>/dev/null \
      | xargs -r -n 1 kubectl patch --type=merge -p '{"metadata":{"finalizers":[]}}' || true
    kubectl delete kiteusers.hy3ons.github.io --all --ignore-not-found=true --wait=false || true
  fi
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

# Longhorn CRD가 있는 경우 Kite 전용/태그 disk entry를 node별로 제거한다.
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

# JSON Patch path에서 /와 ~는 escape해야 disk 이름을 안전하게 path로 쓸 수 있다.
json_pointer_escape() {
  local value="$1"

  value="${value//\~/~0}"
  value="${value//\//~1}"
  printf '%s' "${value}"
}

# Longhorn node CR의 특정 disk entry를 삭제한다. replica/backing image가 남으면 제한 시간 동안 재시도한다.
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

  # Longhorn은 replica가 남은 disk 삭제를 막으므로 새 배치를 막기 위해 scheduling부터 끈다.
  # 삭제 실패 시에는 아래에서 원래 allowScheduling 값으로 되돌린다.
  kubectl -n longhorn-system patch "${node}" --type=json -p "[{\"op\":\"replace\",\"path\":\"/spec/disks/${escaped_disk}/allowScheduling\",\"value\":false}]" >/dev/null 2>&1 || true

  # PVC/PV 삭제가 Longhorn controller에 비동기로 반영되므로 잠시 기다리며 remove patch를 재시도한다.
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

  # 끝까지 실패하면 cleanup 때문에 디스크가 scheduling 불가 상태로 남지 않게 원래 값을 복원한다.
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

# 설치 방식에 따라 전용 disk 이름이 있거나 기존 disk에 kite tag만 있을 수 있어 둘 다 찾는다.
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
  if [[ "${CLEAR_LONGHORN_DATA}" != "true" ]]; then
    log "skipping Longhorn host data cleanup because CLEAR_LONGHORN_DATA=${CLEAR_LONGHORN_DATA}"
    return
  fi
  if [[ "${CLEAR_LONGHORN_DATA_CONFIRM}" != "true" ]]; then
    echo "[kite] refusing Longhorn host data deletion without CLEAR_LONGHORN_DATA_CONFIRM=true" >&2
    exit 1
  fi
  local external_pv_count
  external_pv_count="$(external_longhorn_pv_count)"
  if [[ "${external_pv_count}" != "0" ]]; then
    log "skipping Kite Longhorn host data cleanup because ${external_pv_count} external Longhorn PV(s) still exist"
    log "delete or migrate non-Kite Longhorn PVC/PV resources before deleting host data"
    return
  fi
  if [[ "${CLEAR_LONGHORN_FORCE}" != "true" ]]; then
    local pv_count
    pv_count="$(longhorn_pv_count)"
    if [[ "${pv_count}" != "0" ]]; then
      log "skipping Kite Longhorn host data cleanup because ${pv_count} Longhorn PV(s) still exist"
      log "delete remaining Longhorn PVC/PV resources first, or set CLEAR_LONGHORN_FORCE=true"
      return
    fi
  fi

  log "deleting Kite Longhorn host data on every node"
  # cleanup DaemonSet은 각 node에서 host path를 삭제하고, rollout 확인 뒤 바로 제거된다.
  kubectl apply -f "${ROOT_DIR}/build/kite-storage/longhorn-cleanup"
  kubectl -n longhorn-system rollout status daemonset/kite-longhorn-disk-cleanup --timeout=180s || true
  kubectl delete -f "${ROOT_DIR}/build/kite-storage/longhorn-cleanup" --ignore-not-found=true || true
}

# CLEAR_LONGHORN=true일 때 Longhorn 자체 namespace/CR까지 제거한다. PV가 남으면 기본적으로 중단한다.
delete_longhorn_resources() {
  if [[ "${CLEAR_LONGHORN}" != "true" ]]; then
    log "skipping Longhorn cleanup because CLEAR_LONGHORN=${CLEAR_LONGHORN}"
    return
  fi

  require_command kubectl
  local external_pv_count
  external_pv_count="$(external_longhorn_pv_count)"
  if [[ "${external_pv_count}" != "0" ]]; then
    log "skipping Longhorn disk cleanup and uninstall because ${external_pv_count} external Longhorn PV(s) still exist"
    log "CLEAR_LONGHORN_FORCE does not override non-Kite Longhorn PVC/PV protection"
    return
  fi

  remove_kite_longhorn_disks

  if ! longhorn_installed_by_kite; then
    log "skipping Longhorn uninstall because longhorn-system is not marked as Kite-installed"
    log "Kite cleanup will not delete shared Longhorn without ${KITE_LONGHORN_OWNER_LABEL_KEY}=${KITE_LONGHORN_OWNER_LABEL_VALUE}"
    return
  fi

  if [[ "${CLEAR_LONGHORN_FORCE}" != "true" ]]; then
    local pv_count
    pv_count="$(longhorn_pv_count)"
    if [[ "${pv_count}" != "0" ]]; then
      log "skipping Longhorn uninstall because ${pv_count} Longhorn PV(s) still exist"
      log "delete remaining Longhorn PVC/PV resources first, or set CLEAR_LONGHORN_FORCE=true"
      return
    fi
  fi

  log "deleting Longhorn workloads and custom resources"
  kubectl delete storageclass longhorn --ignore-not-found=true || true
  delete_longhorn_webhook_configurations
  kubectl delete namespace longhorn-system --ignore-not-found=true --wait=false || true

  log "removing Longhorn finalizers from terminating resources when present"
  # Longhorn CR은 finalizer 때문에 namespace terminating에 남을 수 있어 마지막에 비워 준다.
  clear_longhorn_finalizers
  delete_longhorn_crds
}

# 로컬 Docker daemon에 남은 Kite 개발 이미지를 제거한다.
delete_local_docker_images() {
  if [[ "${CLEAR_IMAGES}" != "true" ]]; then
    log "skipping local Docker image cleanup because CLEAR_IMAGES=${CLEAR_IMAGES}"
    return
  fi
  if ! command -v docker >/dev/null 2>&1; then
    log "docker is not installed; skipping local Docker image cleanup"
    return
  fi

  log "removing local Docker Kite images"
  # repository 이름이 kite-*인 개발 이미지를 찾아 tag와 관계없이 제거한다.
  docker image ls --format '{{.Repository}}:{{.Tag}}' \
    | grep -E '(^|/)kite-(api|controller|gateway|frontend):' \
    | xargs -r docker rmi -f
}

# k3s containerd에 import했던 Kite 이미지를 제거한다.
delete_k3s_images() {
  if [[ "${CLEAR_IMAGES}" != "true" ]]; then
    log "skipping k3s image cleanup because CLEAR_IMAGES=${CLEAR_IMAGES}"
    return
  fi

  log "removing k3s containerd Kite images"
  # k3s는 Docker가 아니라 containerd k8s.io namespace에서 이미지를 사용한다.
  ${K3S_CTR_CMD} images ls -q \
    | grep -E '(^|/)kite-(api|controller|gateway|frontend):' \
    | xargs -r -n 1 ${K3S_CTR_CMD} images rm || true
}

# minikube 모드에서 profile 또는 전체 minikube 상태를 삭제한다.
delete_minikube_profile() {
  require_command minikube

  if [[ "${MINIKUBE_PURGE}" == "true" ]]; then
    log "deleting every minikube cluster and purging local minikube state"
    minikube delete --all --purge
    return
  fi

  log "deleting minikube profile=${MINIKUBE_PROFILE}"
  minikube -p "${MINIKUBE_PROFILE}" delete
}

# 대상 클러스터 종류에 맞춰 Kubernetes 리소스, storage, 이미지 캐시를 정리한다.
main() {
  local cluster

  cluster="$(detect_cluster)"
  log "target cluster=${cluster}"
  configure_interactive_clear_options "${cluster}"
  normalize_clear_options

  case "${cluster}" in
    minikube)
      delete_minikube_profile
      delete_local_docker_images
      ;;
    k3s)
      delete_kite_resources
      delete_kite_longhorn_host_data
      delete_longhorn_resources
      delete_k3s_images
      delete_local_docker_images
      ;;
    k3d|kind|current|k8s|kubernetes)
      delete_kite_resources
      delete_kite_longhorn_host_data
      delete_longhorn_resources
      delete_local_docker_images
      ;;
    *)
      echo "[kite] unknown KITE_CLUSTER=${cluster}; use auto, minikube, k3s, k3d, kind, k8s, kubernetes, or current" >&2
      exit 1
      ;;
  esac

  log "clear complete"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
