#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_CLUSTER="${1:-}"

KITE_NAMESPACE_WAS_SET="${KITE_NAMESPACE+x}"
TEST_IMAGE_REGISTRY_WAS_SET="${TEST_IMAGE_REGISTRY+x}"
TEST_IMAGE_TAG_WAS_SET="${TEST_IMAGE_TAG+x}"
TEST_INSTALL_DEPS_WAS_SET="${TEST_INSTALL_DEPS+x}"
TEST_CLEANUP_WAS_SET="${TEST_CLEANUP+x}"
TEST_CLEANUP_TIMEOUT_WAS_SET="${TEST_CLEANUP_TIMEOUT+x}"
TEST_VM_TIMEOUT_WAS_SET="${TEST_VM_TIMEOUT+x}"
TEST_DRY_RUN_WAS_SET="${TEST_DRY_RUN+x}"
TEST_API_LOCAL_PORT_WAS_SET="${TEST_API_LOCAL_PORT+x}"
TEST_FRONTEND_LOCAL_PORT_WAS_SET="${TEST_FRONTEND_LOCAL_PORT+x}"
TEST_GATEWAY_LOCAL_PORT_WAS_SET="${TEST_GATEWAY_LOCAL_PORT+x}"
TEST_GATEWAY_HOST_KEY_SOURCE_WAS_SET="${TEST_GATEWAY_HOST_KEY_SOURCE+x}"
TEST_GATEWAY_HOST_KEY_REFRESH_WAS_SET="${TEST_GATEWAY_HOST_KEY_REFRESH+x}"
TEST_GATEWAY_HOST_KEY_FILE_NAME_WAS_SET="${TEST_GATEWAY_HOST_KEY_FILE_NAME+x}"
TEST_USERNAME_WAS_SET="${TEST_USERNAME+x}"
TEST_EMAIL_WAS_SET="${TEST_EMAIL+x}"
TEST_PASSWORD_WAS_SET="${TEST_PASSWORD+x}"
TEST_VM_NAME_WAS_SET="${TEST_VM_NAME+x}"
TEST_VM_DOMAIN_PREFIX_WAS_SET="${TEST_VM_DOMAIN_PREFIX+x}"
TEST_VM_DISK_WAS_SET="${TEST_VM_DISK+x}"
TEST_VM_SSH_ID_WAS_SET="${TEST_VM_SSH_ID+x}"
TEST_VM_SSH_PASSWORD_WAS_SET="${TEST_VM_SSH_PASSWORD+x}"
K3S_CTR_CMD_WAS_SET="${K3S_CTR_CMD+x}"
MINIKUBE_PROFILE_WAS_SET="${MINIKUBE_PROFILE+x}"
MINIKUBE_DRIVER_WAS_SET="${MINIKUBE_DRIVER+x}"
MINIKUBE_CPUS_WAS_SET="${MINIKUBE_CPUS+x}"
MINIKUBE_MEMORY_WAS_SET="${MINIKUBE_MEMORY+x}"
MINIKUBE_START_WAS_SET="${MINIKUBE_START+x}"

default_gateway_host_key_source() {
  case "${TEST_CLUSTER}" in
    k3s)
      echo "host"
      ;;
    *)
      echo "generate"
      ;;
  esac
}

default_gateway_host_key_refresh() {
  case "${TEST_GATEWAY_HOST_KEY_SOURCE}" in
    host)
      echo "true"
      ;;
    *)
      echo "false"
      ;;
  esac
}

KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
TEST_IMAGE_TAG="${TEST_IMAGE_TAG:-test-$(date +%Y%m%d%H%M%S)}"
TEST_INSTALL_DEPS="${TEST_INSTALL_DEPS:-true}"
TEST_CLEANUP="${TEST_CLEANUP:-true}"
TEST_CLEANUP_TIMEOUT="${TEST_CLEANUP_TIMEOUT:-5m}"
TEST_VM_TIMEOUT="${TEST_VM_TIMEOUT:-20m}"
TEST_DRY_RUN="${TEST_DRY_RUN:-false}"
TEST_API_LOCAL_PORT="${TEST_API_LOCAL_PORT:-18080}"
TEST_FRONTEND_LOCAL_PORT="${TEST_FRONTEND_LOCAL_PORT:-18081}"
TEST_GATEWAY_LOCAL_PORT="${TEST_GATEWAY_LOCAL_PORT:-10022}"
TEST_GATEWAY_HOST_KEY_SOURCE="${TEST_GATEWAY_HOST_KEY_SOURCE:-$(default_gateway_host_key_source)}"
TEST_GATEWAY_HOST_KEY_REFRESH="${TEST_GATEWAY_HOST_KEY_REFRESH:-$(default_gateway_host_key_refresh)}"
TEST_GATEWAY_HOST_KEY_FILE_NAME="${TEST_GATEWAY_HOST_KEY_FILE_NAME:-ssh_host_rsa_key}"
TEST_USERNAME="${TEST_USERNAME:-e2e-$(date +%s)}"
TEST_EMAIL="${TEST_EMAIL:-${TEST_USERNAME}@example.com}"
TEST_PASSWORD="${TEST_PASSWORD:-Kite-e2e-password-1}"
TEST_VM_NAME="${TEST_VM_NAME:-kite-e2e-vm-$(date +%s)}"
TEST_VM_DOMAIN_PREFIX="${TEST_VM_DOMAIN_PREFIX:-${TEST_VM_NAME}}"
TEST_VM_DISK="${TEST_VM_DISK:-20Gi}"
TEST_VM_SSH_ID="${TEST_VM_SSH_ID:-kitee2e}"
TEST_VM_SSH_PASSWORD="${TEST_VM_SSH_PASSWORD:-Kite-e2e-vm-password-1}"
K3S_CTR_CMD="${K3S_CTR_CMD:-sudo k3s ctr -n k8s.io}"
MINIKUBE_PROFILE="${MINIKUBE_PROFILE:-minikube}"
MINIKUBE_DRIVER="${MINIKUBE_DRIVER:-}"
MINIKUBE_CPUS="${MINIKUBE_CPUS:-4}"
MINIKUBE_MEMORY="${MINIKUBE_MEMORY:-8192}"
MINIKUBE_START="${MINIKUBE_START:-true}"

if [[ "${TEST_CLUSTER}" == "k8s" ]]; then
  TEST_IMAGE_REGISTRY="${TEST_IMAGE_REGISTRY:-}"
else
  TEST_IMAGE_REGISTRY="${TEST_IMAGE_REGISTRY:-kite-test}"
fi

KUSTOMIZE_OVERLAY_DIR="$(mktemp -d "${TMPDIR:-/tmp}/kite-e2e-kustomize.XXXXXX")"
RENDERED_MANIFEST="$(mktemp "${TMPDIR:-/tmp}/kite-e2e-manifest.XXXXXX")"
COOKIE_JAR="$(mktemp "${TMPDIR:-/tmp}/kite-e2e-cookie.XXXXXX")"
API_PORT_FORWARD_LOG="$(mktemp "${TMPDIR:-/tmp}/kite-api-port-forward.XXXXXX")"
FRONTEND_PORT_FORWARD_LOG="$(mktemp "${TMPDIR:-/tmp}/kite-frontend-port-forward.XXXXXX")"
GATEWAY_PORT_FORWARD_LOG="$(mktemp "${TMPDIR:-/tmp}/kite-gateway-port-forward.XXXXXX")"
TEST_USER_NAME=""
TEST_USER_NAMESPACE=""
API_PORT_FORWARD_PID=""
FRONTEND_PORT_FORWARD_PID=""
GATEWAY_PORT_FORWARD_PID=""
HOST_SSHD_PORTS_BEFORE=""

source "${ROOT_DIR}/build/lib/prompt.sh"

usage() {
  cat >&2 <<'EOF'
usage: test/lib/e2e.sh <k3s|k8s|minikube>

Use the root wrappers instead:
  test/all-test-k3s.sh
  test/all-test-k8s.sh
  test/all-test-minikube.sh
EOF
}

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
    printf "\033[0;32m[kite-e2e] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-e2e] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-e2e] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-e2e] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

prompt_value() {
  local variable_name="$1"
  local was_set="$2"
  local prompt="$3"
  local description="${4:-}"
  local current_value
  local answer

  eval "current_value=\"\${${variable_name}:-}\""
  if [[ -n "${was_set}" ]] || ! kite_prompt_interactive; then
    return 0
  fi

  printf '%s\n' "${prompt}" >&2
  if [[ -n "${description}" ]]; then
    printf '  %s\n' "${description}" >&2
  fi
  read -r -p "입력 [기본: ${current_value:-없음}] " answer
  answer="${answer:-${current_value}}"
  printf -v "${variable_name}" '%s' "${answer}"
  export "${variable_name}"
}

configure_interactive_test_options() {
  kite_prompt_interactive || return 0

  log "interactive e2e test options"
  prompt_value KITE_NAMESPACE "${KITE_NAMESPACE_WAS_SET}" "KITE_NAMESPACE 값을 정합니다." "Kite API/controller/frontend/gateway가 배포될 Kubernetes namespace입니다. 기본값 kite를 쓰면 기존 manifest와 가장 잘 맞습니다."
  prompt_value TEST_IMAGE_TAG "${TEST_IMAGE_TAG_WAS_SET}" "TEST_IMAGE_TAG 값을 정합니다." "이번 테스트에서 빌드한 네 개 이미지에 붙일 tag입니다. 같은 서버에서 여러 번 돌릴 때 이전 이미지와 섞이지 않게 해줍니다."
  prompt_value TEST_IMAGE_REGISTRY "${TEST_IMAGE_REGISTRY_WAS_SET}" "TEST_IMAGE_REGISTRY 값을 정합니다." "k3s/minikube에서는 로컬 이미지 이름 prefix이고, 일반 k8s에서는 클러스터가 pull할 수 있는 registry/repository prefix입니다."
  prompt_value TEST_VM_TIMEOUT "${TEST_VM_TIMEOUT_WAS_SET}" "TEST_VM_TIMEOUT 값을 정합니다." "테스트 VM이 실제 Running 상태가 될 때까지 기다리는 최대 시간입니다. golden image clone과 부팅까지 포함하므로 느린 서버는 30m 이상이 나을 수 있습니다."
  prompt_value TEST_API_LOCAL_PORT "${TEST_API_LOCAL_PORT_WAS_SET}" "TEST_API_LOCAL_PORT 값을 정합니다." "로컬 curl이 cluster 내부 kite-api Service에 접속하도록 port-forward할 내 PC/서버의 포트입니다. 이미 쓰는 포트면 다른 값을 고르세요."
  prompt_value TEST_FRONTEND_LOCAL_PORT "${TEST_FRONTEND_LOCAL_PORT_WAS_SET}" "TEST_FRONTEND_LOCAL_PORT 값을 정합니다." "frontend Service가 HTML을 내는지 확인할 때 port-forward할 로컬 포트입니다."
  prompt_value TEST_GATEWAY_LOCAL_PORT "${TEST_GATEWAY_LOCAL_PORT_WAS_SET}" "TEST_GATEWAY_LOCAL_PORT 값을 정합니다." "gateway Service의 SSH banner를 확인하기 위해 port-forward할 로컬 포트입니다. 22번 대신 높은 포트를 쓰는 게 안전합니다."
  prompt_value TEST_GATEWAY_HOST_KEY_SOURCE "${TEST_GATEWAY_HOST_KEY_SOURCE_WAS_SET}" "TEST_GATEWAY_HOST_KEY_SOURCE 값을 정합니다." "gateway SSH host key Secret을 어디서 만들지 정합니다. k3s 기본값 host는 실제 /etc/ssh host key를 재사용해 fingerprint 유지까지 검증합니다. minikube/k8s 기본값 generate는 클러스터 외부 host key를 읽지 않습니다."
  if [[ -z "${TEST_GATEWAY_HOST_KEY_REFRESH_WAS_SET}" ]]; then
    TEST_GATEWAY_HOST_KEY_REFRESH="$(default_gateway_host_key_refresh)"
  fi
  prompt_value TEST_GATEWAY_HOST_KEY_FILE_NAME "${TEST_GATEWAY_HOST_KEY_FILE_NAME_WAS_SET}" "TEST_GATEWAY_HOST_KEY_FILE_NAME 값을 정합니다." "gateway Deployment가 Secret에서 읽는 key 파일 이름입니다. 기본 manifest는 /etc/kite-gateway/ssh/ssh_host_rsa_key를 읽으므로 기본값을 유지하는 것이 안전합니다."
  prompt_value TEST_USERNAME "${TEST_USERNAME_WAS_SET}" "TEST_USERNAME 값을 정합니다." "테스트용 KiteUser의 로그인 username입니다. 테스트가 끝나면 TEST_CLEANUP=true일 때 이 사용자를 삭제합니다."
  TEST_EMAIL="${TEST_EMAIL:-${TEST_USERNAME}@example.com}"
  prompt_value TEST_EMAIL "${TEST_EMAIL_WAS_SET}" "TEST_EMAIL 값을 정합니다." "테스트 사용자가 API login에 사용할 email입니다. signup 후 이 email/password로 실제 API 로그인을 검증합니다."
  prompt_value TEST_PASSWORD "${TEST_PASSWORD_WAS_SET}" "TEST_PASSWORD 값을 정합니다." "테스트 사용자의 API login password입니다. CRD에는 hash로 저장되고, 테스트 중 cookie 발급 확인에 사용됩니다."
  prompt_value TEST_VM_NAME "${TEST_VM_NAME_WAS_SET}" "TEST_VM_NAME 값을 정합니다." "테스트가 생성할 KiteVirtualMachine 이름입니다. 같은 namespace 안에서 겹치면 안 됩니다."
  TEST_VM_DOMAIN_PREFIX="${TEST_VM_DOMAIN_PREFIX:-${TEST_VM_NAME}}"
  prompt_value TEST_VM_DOMAIN_PREFIX "${TEST_VM_DOMAIN_PREFIX_WAS_SET}" "TEST_VM_DOMAIN_PREFIX 값을 정합니다." "테스트 VM의 HTTP hostname prefix입니다. API가 domainPrefix를 필수로 요구하므로 비워둘 수 없고, 기본값은 TEST_VM_NAME과 같습니다."
  prompt_value TEST_VM_DISK "${TEST_VM_DISK_WAS_SET}" "TEST_VM_DISK 값을 정합니다." "테스트 VM의 root disk 요청량입니다. 표준 기본값은 20Gi이고, 작은 단일 노드 Longhorn 환경에서는 8Gi처럼 낮춰 실행할 수 있습니다."
  prompt_value TEST_VM_SSH_ID "${TEST_VM_SSH_ID_WAS_SET}" "TEST_VM_SSH_ID 값을 정합니다." "VM guest OS와 gateway 라우팅에 쓰는 Linux login id입니다. 소문자/숫자/밑줄/하이픈 규칙을 따라야 합니다."
  prompt_value TEST_VM_SSH_PASSWORD "${TEST_VM_SSH_PASSWORD_WAS_SET}" "TEST_VM_SSH_PASSWORD 값을 정합니다." "VM 생성 API가 guest login Secret과 cloud-init password hash를 만들 때 사용하는 테스트용 비밀번호입니다."

  case "${TEST_CLUSTER}" in
    k3s)
      prompt_value K3S_CTR_CMD "${K3S_CTR_CMD_WAS_SET}" "K3S_CTR_CMD 값을 정합니다." "빌드한 Docker 이미지를 k3s의 containerd k8s.io namespace로 넣는 명령입니다. sudo가 필요 없으면 'k3s ctr -n k8s.io'로 바꾸면 됩니다."
      ;;
    minikube)
      prompt_value MINIKUBE_PROFILE "${MINIKUBE_PROFILE_WAS_SET}" "MINIKUBE_PROFILE 값을 정합니다." "테스트가 사용할 minikube profile 이름입니다. 기본 profile을 쓰면 minikube입니다."
      prompt_value MINIKUBE_DRIVER "${MINIKUBE_DRIVER_WAS_SET}" "MINIKUBE_DRIVER 값을 정합니다." "minikube start에 넘길 driver입니다. 비워두면 현재 minikube 기본 driver를 그대로 사용합니다."
      prompt_value MINIKUBE_CPUS "${MINIKUBE_CPUS_WAS_SET}" "MINIKUBE_CPUS 값을 정합니다." "VM 부팅 테스트에는 CPU 여유가 필요합니다. 기본 4로 부족하면 더 올리세요."
      prompt_value MINIKUBE_MEMORY "${MINIKUBE_MEMORY_WAS_SET}" "MINIKUBE_MEMORY 값을 정합니다." "minikube에 줄 메모리(MB)입니다. KubeVirt/CDI/VM까지 띄우므로 기본 8192보다 작으면 불안정할 수 있습니다."
      kite_prompt_configure_bool MINIKUBE_START "${MINIKUBE_START_WAS_SET}" $'MINIKUBE_START 값을 정합니다.\n  예를 고르면 테스트 전에 minikube start/update-context를 실행합니다. 이미 준비된 profile을 그대로 쓰려면 아니오를 고르세요.'
      ;;
  esac

  kite_prompt_configure_bool TEST_INSTALL_DEPS "${TEST_INSTALL_DEPS_WAS_SET}" $'TEST_INSTALL_DEPS 값을 정합니다.\n  예를 고르면 Longhorn/KubeVirt/CDI/StorageClass/golden image를 기존 설치 스크립트로 준비합니다. 이미 완전히 준비된 클러스터만 확인하려면 아니오를 고르세요.'
  kite_prompt_configure_bool TEST_GATEWAY_HOST_KEY_REFRESH "${TEST_GATEWAY_HOST_KEY_REFRESH_WAS_SET}" $'TEST_GATEWAY_HOST_KEY_REFRESH 값을 정합니다.\n  예를 고르면 기존 gateway host key Secret이 있어도 다시 만듭니다. host key 재사용 테스트에서는 예여야 예전 generate key가 남아 fingerprint 검증을 속이지 않습니다.'
  kite_prompt_configure_bool TEST_CLEANUP "${TEST_CLEANUP_WAS_SET}" $'TEST_CLEANUP 값을 정합니다.\n  예를 고르면 테스트가 만든 VM, KiteUser, 사용자 namespace를 끝나고 삭제합니다. 실패 상태를 직접 조사하려면 아니오가 좋습니다.'
  prompt_value TEST_CLEANUP_TIMEOUT "${TEST_CLEANUP_TIMEOUT_WAS_SET}" "TEST_CLEANUP_TIMEOUT 값을 정합니다." "TEST_CLEANUP=true일 때 테스트 VM/User/namespace가 실제로 삭제될 때까지 기다리는 최대 시간입니다."
  kite_prompt_configure_bool TEST_DRY_RUN "${TEST_DRY_RUN_WAS_SET}" $'TEST_DRY_RUN 값을 정합니다.\n  예를 고르면 실제 빌드/배포/VM 생성 없이 어떤 명령을 실행할지 계획만 출력합니다.'

  log "e2e choices: cluster=${TEST_CLUSTER}, namespace=${KITE_NAMESPACE}, image=${TEST_IMAGE_REGISTRY}/<component>:${TEST_IMAGE_TAG}, install_deps=${TEST_INSTALL_DEPS}, gateway_host_key_source=${TEST_GATEWAY_HOST_KEY_SOURCE}, gateway_host_key_refresh=${TEST_GATEWAY_HOST_KEY_REFRESH}, cleanup=${TEST_CLEANUP}, cleanup_timeout=${TEST_CLEANUP_TIMEOUT}, vm_name=${TEST_VM_NAME}, vm_domain_prefix=${TEST_VM_DOMAIN_PREFIX}, vm_disk=${TEST_VM_DISK}, vm_timeout=${TEST_VM_TIMEOUT}, dry_run=${TEST_DRY_RUN}"
}

validate_static_options() {
  TEST_IMAGE_REGISTRY="${TEST_IMAGE_REGISTRY%/}"

  case "${TEST_CLUSTER}" in
    k3s|minikube)
      if [[ -z "${TEST_IMAGE_REGISTRY}" ]]; then
        warn "TEST_IMAGE_REGISTRY must not be empty because it becomes the local image name prefix"
        exit 1
      fi
      ;;
    k8s)
      if [[ -z "${TEST_IMAGE_REGISTRY}" ]]; then
        warn "TEST_IMAGE_REGISTRY is required for test/all-test-k8s.sh because a generic Kubernetes cluster must pull pushed images"
        warn "Run it in a terminal to be prompted, or pass TEST_IMAGE_REGISTRY=registry.example.com/kite"
        exit 1
      fi
      ;;
    *)
      usage
      exit 1
      ;;
  esac
}

host_sshd_port_snapshot() {
  local sshd_bin=""
  local output
  local ports

  [[ "${TEST_CLUSTER}" == "k3s" ]] || return 1
  [[ "$(uname -s 2>/dev/null || true)" == "Linux" ]] || return 1
  for candidate in /usr/sbin/sshd /usr/local/sbin/sshd sshd; do
    if command -v "${candidate}" >/dev/null 2>&1 || [[ -x "${candidate}" ]]; then
      sshd_bin="${candidate}"
      break
    fi
  done
  [[ -n "${sshd_bin}" ]] || return 1

  output="$(sudo -n "${sshd_bin}" -T 2>/dev/null || "${sshd_bin}" -T 2>/dev/null || true)"
  ports="$(printf '%s\n' "${output}" | awk 'tolower($1) == "port" { print $2 }' | sort -n | paste -sd, -)"
  if [[ -n "${ports}" ]]; then
    printf '%s\n' "${ports}"
    return 0
  fi

  ports="$(grep -hE '^[[:space:]]*Port[[:space:]]+' /etc/ssh/sshd_config /etc/ssh/sshd_config.d/*.conf 2>/dev/null \
    | awk '{ print $2 }' \
    | sort -n \
    | paste -sd, -)"
  if [[ -n "${ports}" ]]; then
    printf '%s\n' "${ports}"
    return 0
  fi

  printf '22\n'
}

record_host_sshd_ports() {
  HOST_SSHD_PORTS_BEFORE="$(host_sshd_port_snapshot || true)"
  if [[ -z "${HOST_SSHD_PORTS_BEFORE}" ]]; then
    warn "skipping host sshd port preservation check because sshd effective config could not be read"
    return 0
  fi
  log "host sshd ports before E2E: ${HOST_SSHD_PORTS_BEFORE}"
}

verify_host_sshd_ports_unchanged() {
  local after

  [[ -n "${HOST_SSHD_PORTS_BEFORE}" ]] || return 0
  after="$(host_sshd_port_snapshot || true)"
  if [[ -z "${after}" ]]; then
    warn "host sshd ports could not be read after E2E"
    return 1
  fi
  if [[ "${after}" != "${HOST_SSHD_PORTS_BEFORE}" ]]; then
    warn "host sshd ports changed during E2E: before=${HOST_SSHD_PORTS_BEFORE}, after=${after}"
    return 1
  fi
  log "host sshd ports unchanged after E2E: ${after}"
}

run_cmd() {
  if [[ "${TEST_DRY_RUN}" == "true" ]]; then
    printf '[kite-e2e] dry-run:'
    printf ' %q' "$@"
    printf '\n'
    return 0
  fi

  "$@"
}

image_name() {
  local component="$1"
  echo "${TEST_IMAGE_REGISTRY}/${component}:${TEST_IMAGE_TAG}"
}

build_image() {
  local component="$1"
  local dockerfile="$2"
  local context="$3"
  shift 3
  local image

  image="$(image_name "${component}")"
  log "building ${image}"
  run_cmd docker buildx build --platform linux/amd64 --progress=plain --load "$@" -t "${image}" -f "${dockerfile}" "${context}"
}

build_and_distribute_images() {
  local -a frontend_args
  frontend_args=(
    --build-arg "VITE_BUILD_MODE=production"
    --build-arg "VITE_API_BASE_URL=/api/v1"
    --build-arg "VITE_USE_MOCK=false"
  )

  case "${TEST_CLUSTER}" in
    k8s)
      build_and_push_image kite-api "${ROOT_DIR}/kite/Dockerfile.api" "${ROOT_DIR}/kite"
      build_and_push_image kite-controller "${ROOT_DIR}/kite/Dockerfile.controller" "${ROOT_DIR}/kite"
      build_and_push_image kite-gateway "${ROOT_DIR}/kite/Dockerfile.gateway" "${ROOT_DIR}/kite"
      build_and_push_image kite-frontend "${ROOT_DIR}/kite-frontend/Dockerfile" "${ROOT_DIR}/kite-frontend" "${frontend_args[@]}"
      ;;
    k3s|minikube)
      build_image kite-api "${ROOT_DIR}/kite/Dockerfile.api" "${ROOT_DIR}/kite"
      build_image kite-controller "${ROOT_DIR}/kite/Dockerfile.controller" "${ROOT_DIR}/kite"
      build_image kite-gateway "${ROOT_DIR}/kite/Dockerfile.gateway" "${ROOT_DIR}/kite"
      build_image kite-frontend "${ROOT_DIR}/kite-frontend/Dockerfile" "${ROOT_DIR}/kite-frontend" "${frontend_args[@]}"
      distribute_local_image "$(image_name kite-api)"
      distribute_local_image "$(image_name kite-controller)"
      distribute_local_image "$(image_name kite-gateway)"
      distribute_local_image "$(image_name kite-frontend)"
      ;;
  esac
}

build_and_push_image() {
  local component="$1"
  local dockerfile="$2"
  local context="$3"
  shift 3
  local image

  image="$(image_name "${component}")"
  log "building and pushing ${image}"
  run_cmd docker buildx build --platform linux/amd64 --progress=plain --push "$@" -t "${image}" -f "${dockerfile}" "${context}"
}

distribute_local_image() {
  local image="$1"

  case "${TEST_CLUSTER}" in
    k3s)
      log "importing ${image} into k3s containerd"
      if [[ "${TEST_DRY_RUN}" == "true" ]]; then
        printf '[kite-e2e] dry-run: docker save %q | %s images import -\n' "${image}" "${K3S_CTR_CMD}"
      else
        docker save "${image}" | ${K3S_CTR_CMD} images import -
      fi
      ;;
    minikube)
      log "loading ${image} into minikube profile=${MINIKUBE_PROFILE}"
      run_cmd minikube -p "${MINIKUBE_PROFILE}" image load "${image}"
      ;;
  esac
}

pull_policy() {
  case "${TEST_CLUSTER}" in
    k8s)
      echo "IfNotPresent"
      ;;
    *)
      echo "Never"
      ;;
  esac
}

render_manifest() {
  local policy

  policy="$(pull_policy)"
  log "rendering test manifest with tag ${TEST_IMAGE_TAG}"
  rm -rf "${KUSTOMIZE_OVERLAY_DIR}/kite"
  cp -R "${ROOT_DIR}/build/kite" "${KUSTOMIZE_OVERLAY_DIR}/kite"
  cat > "${KUSTOMIZE_OVERLAY_DIR}/kustomization.yaml" <<EOF
resources:
  - kite

images:
  - name: ghcr.io/hy3ons/kite-api
    newName: ${TEST_IMAGE_REGISTRY}/kite-api
    newTag: ${TEST_IMAGE_TAG}
  - name: ghcr.io/hy3ons/kite-controller
    newName: ${TEST_IMAGE_REGISTRY}/kite-controller
    newTag: ${TEST_IMAGE_TAG}
  - name: ghcr.io/hy3ons/kite-gateway
    newName: ${TEST_IMAGE_REGISTRY}/kite-gateway
    newTag: ${TEST_IMAGE_TAG}
  - name: ghcr.io/hy3ons/kite-frontend
    newName: ${TEST_IMAGE_REGISTRY}/kite-frontend
    newTag: ${TEST_IMAGE_TAG}

patches:
  - target:
      version: v1
      kind: Service
      name: kite-gateway
    patch: |-
      apiVersion: v1
      kind: Service
      metadata:
        name: kite-gateway
      spec:
        type: ClusterIP
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
                imagePullPolicy: ${policy}
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
                imagePullPolicy: ${policy}
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
                imagePullPolicy: ${policy}
                env:
                  - name: KITE_GATEWAY_HOST_KEY_PATH
                    value: /etc/kite-gateway/ssh/${TEST_GATEWAY_HOST_KEY_FILE_NAME}
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
                imagePullPolicy: ${policy}
EOF

  if [[ "${TEST_DRY_RUN}" == "true" ]]; then
    run_cmd kubectl kustomize "${KUSTOMIZE_OVERLAY_DIR}"
    printf 'dry-run manifest placeholder\n' > "${RENDERED_MANIFEST}"
    return
  fi

  kubectl kustomize "${KUSTOMIZE_OVERLAY_DIR}" > "${RENDERED_MANIFEST}"
}

prepare_minikube() {
  local args

  if [[ "${TEST_CLUSTER}" != "minikube" ]]; then
    return
  fi

  if [[ "${MINIKUBE_START}" != "true" ]]; then
    log "updating minikube context for profile=${MINIKUBE_PROFILE}"
    run_cmd minikube -p "${MINIKUBE_PROFILE}" update-context
    return
  fi

  args=(start -p "${MINIKUBE_PROFILE}" --cpus="${MINIKUBE_CPUS}" --memory="${MINIKUBE_MEMORY}")
  if [[ -n "${MINIKUBE_DRIVER}" ]]; then
    args+=(--driver="${MINIKUBE_DRIVER}")
  fi

  log "starting minikube profile=${MINIKUBE_PROFILE}"
  run_cmd minikube "${args[@]}"
  run_cmd minikube -p "${MINIKUBE_PROFILE}" update-context
}

prepare_dependencies() {
  if [[ "${TEST_INSTALL_DEPS}" != "true" ]]; then
    log "skipping dependency install because TEST_INSTALL_DEPS=${TEST_INSTALL_DEPS}"
    return
  fi

  log "preparing Longhorn, KubeVirt, CDI, storage class, and golden image"
  run_cmd env \
    INSTALL_LONGHORN=true \
    INSTALL_KUBEVIRT=true \
    INSTALL_CDI=true \
    CONFIGURE_LONGHORN=true \
    APPLY_STORAGECLASS=true \
    APPLY_GOLDEN_IMAGE=true \
    DEPLOY_KITE=false \
    RUN_VERIFY=false \
    KITE_ASSUME_DEFAULTS=true \
    KITE_CLUSTER="${TEST_CLUSTER}" \
    KITE_NAMESPACE="${KITE_NAMESPACE}" \
    "${ROOT_DIR}/build/dev/all-in-one.sh"
}

sudo_cmd() {
  if kite_prompt_interactive; then
    sudo "$@"
  else
    sudo -n "$@"
  fi
}

host_key_candidates() {
  printf '%s\n' \
    /etc/ssh/ssh_host_ed25519_key \
    /etc/ssh/ssh_host_ecdsa_key \
    /etc/ssh/ssh_host_rsa_key
}

copy_host_key_file() {
  local source="$1"
  local target="$2"

  if [[ -r "${source}" ]]; then
    cp "${source}" "${target}"
  elif command -v sudo >/dev/null 2>&1; then
    sudo_cmd cat "${source}" > "${target}"
  else
    return 1
  fi
  chmod 0600 "${target}"
}

copy_selected_host_key_file() {
  local target="$1"
  local candidate

  if [[ "$(uname -s 2>/dev/null || true)" != "Linux" ]]; then
    warn "TEST_GATEWAY_HOST_KEY_SOURCE=host requires Linux host SSH keys under /etc/ssh"
    return 1
  fi

  while IFS= read -r candidate; do
    if [[ -f "${candidate}" ]] || { command -v sudo >/dev/null 2>&1 && sudo_cmd test -f "${candidate}" 2>/dev/null; }; then
      log "using host SSH key ${candidate} as fingerprint reference"
      copy_host_key_file "${candidate}" "${target}"
      return 0
    fi
  done < <(host_key_candidates)

  warn "no host SSH private key was found under /etc/ssh"
  return 1
}

private_key_fingerprint() {
  local key_path="$1"

  ssh-keygen -y -f "${key_path}" | ssh-keygen -lf - | awk 'NR == 1 { print $2 }'
}

scan_ssh_fingerprints() {
  local host="$1"
  local port="$2"

  ssh-keyscan -T 10 -p "${port}" "${host}" 2>/dev/null \
    | ssh-keygen -lf - 2>/dev/null \
    | awk '{ print $2 }' \
    | sort -u \
    || true
}

decode_base64_to_file() {
  local payload="$1"
  local target="$2"

  KITE_E2E_BASE64_PAYLOAD="${payload}" python3 - "${target}" <<'PY'
import base64
import os
import sys

target = sys.argv[1]
payload = os.environ["KITE_E2E_BASE64_PAYLOAD"].strip()
with open(target, "wb") as handle:
    handle.write(base64.b64decode(payload))
PY
}

verify_gateway_host_key_secret() {
  local tmpdir
  local host_key
  local secret_key
  local secret_payload
  local host_fingerprint
  local secret_fingerprint

  if [[ "${TEST_GATEWAY_HOST_KEY_SOURCE}" != "host" ]]; then
    return
  fi
  if [[ "${TEST_DRY_RUN}" == "true" ]]; then
    log "dry-run: would compare host /etc/ssh fingerprint with gateway host key Secret"
    return
  fi

  require_command ssh-keygen
  tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/kite-gateway-key-check.XXXXXX")"
  host_key="${tmpdir}/host_key"
  secret_key="${tmpdir}/secret_key"

  if ! copy_selected_host_key_file "${host_key}"; then
    rm -rf "${tmpdir}"
    exit 1
  fi

  if ! secret_payload="$(kubectl -n "${KITE_NAMESPACE}" get secret kite-gateway-host-key -o "go-template={{ index .data \"${TEST_GATEWAY_HOST_KEY_FILE_NAME}\" }}")"; then
    rm -rf "${tmpdir}"
    warn "gateway host key Secret ${KITE_NAMESPACE}/kite-gateway-host-key does not contain ${TEST_GATEWAY_HOST_KEY_FILE_NAME}"
    exit 1
  fi
  if [[ -z "${secret_payload}" ]]; then
    rm -rf "${tmpdir}"
    warn "gateway host key Secret ${KITE_NAMESPACE}/kite-gateway-host-key has empty ${TEST_GATEWAY_HOST_KEY_FILE_NAME}"
    exit 1
  fi
  decode_base64_to_file "${secret_payload}" "${secret_key}"
  chmod 0600 "${secret_key}"

  host_fingerprint="$(private_key_fingerprint "${host_key}")"
  secret_fingerprint="$(private_key_fingerprint "${secret_key}")"
  rm -rf "${tmpdir}"

  if [[ -z "${host_fingerprint}" || -z "${secret_fingerprint}" || "${host_fingerprint}" != "${secret_fingerprint}" ]]; then
    warn "gateway host key fingerprint does not match host sshd fingerprint: host=${host_fingerprint:-empty}, secret=${secret_fingerprint:-empty}"
    exit 1
  fi

  log "gateway host key Secret reuses host sshd fingerprint ${host_fingerprint}"
}

restart_gateway_when_host_key_changes() {
  if [[ "${TEST_GATEWAY_HOST_KEY_REFRESH}" != "true" ]]; then
    return
  fi

  log "restarting kite-gateway so the refreshed host key is loaded"
  run_cmd kubectl -n "${KITE_NAMESPACE}" rollout restart deployment/kite-gateway
}

apply_kite_runtime() {
  log "applying Kite runtime manifest"
  run_cmd env \
    KITE_ASSUME_DEFAULTS=true \
    KITE_NAMESPACE="${KITE_NAMESPACE}" \
    KITE_GATEWAY_HOST_KEY_SOURCE="${TEST_GATEWAY_HOST_KEY_SOURCE}" \
    KITE_GATEWAY_HOST_KEY_REFRESH="${TEST_GATEWAY_HOST_KEY_REFRESH}" \
    KITE_GATEWAY_HOST_KEY_FILE_NAME="${TEST_GATEWAY_HOST_KEY_FILE_NAME}" \
    "${ROOT_DIR}/build/deploy/scripts/ensure-gateway-host-key-secret.sh"
  verify_gateway_host_key_secret
  run_cmd kubectl apply -f "${RENDERED_MANIFEST}"
  restart_gateway_when_host_key_changes

  log "waiting for Kite rollouts"
  run_cmd kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-api --timeout=180s
  run_cmd kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-controller --timeout=180s
  run_cmd kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-gateway --timeout=180s
  run_cmd kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-frontend --timeout=180s
}

wait_for_http() {
  local url="$1"
  local tries="${2:-60}"

  for _ in $(seq 1 "${tries}"); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

start_port_forward() {
  local service="$1"
  local mapping="$2"
  local log_file="$3"
  local pid_variable="$4"

  log "port-forwarding ${service} ${mapping}"
  if [[ "${TEST_DRY_RUN}" == "true" ]]; then
    printf '[kite-e2e] dry-run: kubectl -n %q port-forward %q %q\n' "${KITE_NAMESPACE}" "${service}" "${mapping}"
    return
  fi

  kubectl -n "${KITE_NAMESPACE}" port-forward "${service}" "${mapping}" >"${log_file}" 2>&1 &
  printf -v "${pid_variable}" '%s' "$!"
}

require_json_field() {
  local json="$1"
  local path="$2"

  python3 - "${path}" "${json}" <<'PY'
import json
import sys

path = sys.argv[1].split(".")
data = json.loads(sys.argv[2])
for key in path:
    if isinstance(data, dict):
        data = data[key]
    else:
        raise KeyError(key)
print(data)
PY
}

signup_test_user() {
  local response

  log "signing up test user ${TEST_USERNAME}"
  response="$(curl -fsS -X POST "http://127.0.0.1:${TEST_API_LOCAL_PORT}/api/v1/auth/signup" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${TEST_USERNAME}\",\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\"}")"

  TEST_USER_NAME="$(require_json_field "${response}" "user.name")"
  TEST_USER_NAMESPACE="$(require_json_field "${response}" "user.namespace")"

  if [[ -z "${TEST_USER_NAME}" || -z "${TEST_USER_NAMESPACE}" ]]; then
    warn "signup response did not include user.name and user.namespace"
    exit 1
  fi

  log "test user created name=${TEST_USER_NAME}, namespace=${TEST_USER_NAMESPACE}"
}

promote_test_user() {
  log "promoting test user ${TEST_USER_NAME} to admin access for VM creation"
  kubectl patch "kiteusers.hy3ons.github.io/${TEST_USER_NAME}" --type=merge -p '{"spec":{"access_level":3}}' >/dev/null
}

login_test_user() {
  log "logging in test user"
  curl -fsS -c "${COOKIE_JAR}" -X POST "http://127.0.0.1:${TEST_API_LOCAL_PORT}/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\"}" >/dev/null
}

verify_health() {
  local health

  log "checking API health"
  health="$(curl -fsS "http://127.0.0.1:${TEST_API_LOCAL_PORT}/api/v1/health")"
  if [[ "$(require_json_field "${health}" "status")" != "ok" ]]; then
    warn "API health status is not ok: ${health}"
    exit 1
  fi
}

verify_user_reconcile() {
  wait_for_jsonpath "kiteusers.hy3ons.github.io/${TEST_USER_NAME}" "{.status.phase}" "Ready" "180s"
  kubectl get namespace "${TEST_USER_NAMESPACE}" >/dev/null
  kubectl -n "${TEST_USER_NAMESPACE}" get networkpolicy deny-from-other-namespaces >/dev/null
  kubectl -n "${TEST_USER_NAMESPACE}" get networkpolicy tenant-isolation-egress >/dev/null
}

create_test_vm() {
  local response

  log "creating test VM ${TEST_VM_NAME}"
  response="$(curl -fsS -b "${COOKIE_JAR}" -X POST "http://127.0.0.1:${TEST_API_LOCAL_PORT}/api/v1/vms" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"${TEST_VM_NAME}\",\"domainPrefix\":\"${TEST_VM_DOMAIN_PREFIX}\",\"cpu\":2,\"memory\":\"4Gi\",\"image\":\"ubuntu-22.04\",\"disk\":\"${TEST_VM_DISK}\",\"sshId\":\"${TEST_VM_SSH_ID}\",\"sshPassword\":\"${TEST_VM_SSH_PASSWORD}\",\"powerState\":\"On\"}")"

  if [[ "$(require_json_field "${response}" "vm.name")" != "${TEST_VM_NAME}" ]]; then
    warn "VM create response did not include expected vm.name: ${response}"
    exit 1
  fi
}

verify_vm_reconcile() {
  log "checking VM-owned resources"
  wait_for_resource "datavolumes.cdi.kubevirt.io/${TEST_VM_NAME}-disk" "${TEST_USER_NAMESPACE}"
  wait_for_resource "virtualmachines.kubevirt.io/${TEST_VM_NAME}" "${TEST_USER_NAMESPACE}"
  wait_for_resource "secret/${TEST_VM_NAME}-guest-login" "${TEST_USER_NAMESPACE}"
  wait_for_resource "secret/${TEST_VM_NAME}-ssh-key" "${TEST_USER_NAMESPACE}"
  wait_for_resource "secret/${TEST_VM_NAME}-cloud-init-userdata" "${TEST_USER_NAMESPACE}"
  wait_for_resource "service/vps-access-${TEST_VM_NAME}" "${TEST_USER_NAMESPACE}"
  wait_for_resource "service/vps-web-${TEST_VM_NAME}" "${TEST_USER_NAMESPACE}"

  wait_for_jsonpath "kitevirtualmachines.hy3ons.github.io/${TEST_VM_NAME}" "{.status.phase}" "Running" "${TEST_VM_TIMEOUT}" -n "${TEST_USER_NAMESPACE}"
}

wait_for_resource() {
  local resource="$1"
  local namespace="$2"
  local timeout_seconds="${3:-180}"
  local deadline

  log "waiting for ${namespace}/${resource} to exist"
  deadline=$((SECONDS + timeout_seconds))
  while true; do
    if kubectl -n "${namespace}" get "${resource}" >/dev/null 2>&1; then
      return 0
    fi
    if (( SECONDS >= deadline )); then
      kubectl -n "${namespace}" get "${resource}"
      return 1
    fi
    sleep 2
  done
}

wait_for_jsonpath() {
  local resource="$1"
  local jsonpath="$2"
  local expected="$3"
  local timeout="$4"
  shift 4

  log "waiting for ${resource} ${jsonpath}=${expected}"
  kubectl "$@" wait --for=jsonpath="${jsonpath}"="${expected}" "${resource}" --timeout="${timeout}" >/dev/null
}

verify_kubevirt_running() {
  local printable_status

  log "checking KubeVirt VM Running status"
  printable_status="$(kubectl -n "${TEST_USER_NAMESPACE}" get "virtualmachines.kubevirt.io/${TEST_VM_NAME}" -o jsonpath='{.status.printableStatus}')"
  if [[ "${printable_status}" != "Running" ]]; then
    warn "KubeVirt VM printableStatus is ${printable_status}, expected Running"
    exit 1
  fi

  wait_for_jsonpath "kitevirtualmachines.hy3ons.github.io/${TEST_VM_NAME}" "{.status.currentPowerState}" "On" "${TEST_VM_TIMEOUT}" -n "${TEST_USER_NAMESPACE}"
}

verify_frontend() {
  start_port_forward "svc/kite-frontend" "${TEST_FRONTEND_LOCAL_PORT}:80" "${FRONTEND_PORT_FORWARD_LOG}" FRONTEND_PORT_FORWARD_PID
  if ! wait_for_http "http://127.0.0.1:${TEST_FRONTEND_LOCAL_PORT}" 60; then
    warn "frontend port-forward did not become ready"
    cat "${FRONTEND_PORT_FORWARD_LOG}" >&2 || true
    exit 1
  fi

  curl -fsS "http://127.0.0.1:${TEST_FRONTEND_LOCAL_PORT}" | grep -qi '<html'
}

verify_gateway() {
  log "checking gateway SSH TCP handshake"
  start_port_forward "svc/kite-gateway" "${TEST_GATEWAY_LOCAL_PORT}:22" "${GATEWAY_PORT_FORWARD_LOG}" GATEWAY_PORT_FORWARD_PID
  python3 - "${TEST_GATEWAY_LOCAL_PORT}" <<'PY'
import socket
import sys
import time

port = int(sys.argv[1])
last_error = None
for _ in range(60):
    try:
        with socket.create_connection(("127.0.0.1", port), timeout=2) as conn:
            banner = conn.recv(64)
        break
    except OSError as exc:
        last_error = exc
        time.sleep(1)
else:
    raise SystemExit(f"gateway SSH port did not become ready: {last_error}")

if not banner.startswith(b"SSH-"):
    raise SystemExit(f"gateway did not return an SSH banner: {banner!r}")
PY
  verify_gateway_runtime_fingerprint
}

verify_gateway_runtime_fingerprint() {
  local tmpdir
  local host_key
  local host_fingerprint
  local gateway_fingerprints

  if [[ "${TEST_GATEWAY_HOST_KEY_SOURCE}" != "host" ]]; then
    return
  fi
  if [[ "${TEST_DRY_RUN}" == "true" ]]; then
    log "dry-run: would compare gateway port-forward fingerprint with host sshd fingerprint"
    return
  fi

  require_command ssh-keygen
  require_command ssh-keyscan
  tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/kite-gateway-runtime-key-check.XXXXXX")"
  host_key="${tmpdir}/host_key"

  if ! copy_selected_host_key_file "${host_key}"; then
    rm -rf "${tmpdir}"
    exit 1
  fi

  host_fingerprint="$(private_key_fingerprint "${host_key}")"
  gateway_fingerprints="$(scan_ssh_fingerprints 127.0.0.1 "${TEST_GATEWAY_LOCAL_PORT}")"
  rm -rf "${tmpdir}"

  if [[ -z "${gateway_fingerprints}" ]]; then
    warn "could not read gateway fingerprints from forwarded port 127.0.0.1:${TEST_GATEWAY_LOCAL_PORT}"
    exit 1
  fi
  if ! grep -Fxq "${host_fingerprint}" <(printf '%s\n' "${gateway_fingerprints}"); then
    warn "running gateway fingerprint does not match host sshd fingerprint"
    warn "host sshd fingerprint: ${host_fingerprint:-empty}"
    warn "gateway fingerprints: ${gateway_fingerprints//$'\n'/, }"
    exit 1
  fi

  log "running gateway exposes host sshd fingerprint ${host_fingerprint}"
}

run_e2e_checks() {
  start_port_forward "svc/kite-api" "${TEST_API_LOCAL_PORT}:8080" "${API_PORT_FORWARD_LOG}" API_PORT_FORWARD_PID
  if ! wait_for_http "http://127.0.0.1:${TEST_API_LOCAL_PORT}/api/v1/health" 60; then
    warn "API health check failed"
    cat "${API_PORT_FORWARD_LOG}" >&2 || true
    exit 1
  fi

  verify_health
  signup_test_user
  promote_test_user
  login_test_user
  verify_user_reconcile
  create_test_vm
  verify_vm_reconcile
  verify_kubevirt_running
  verify_frontend
  verify_gateway
}

wait_for_cleanup_delete() {
  local resource="$1"

  if ! kubectl get "${resource}" >/dev/null 2>&1; then
    return 0
  fi
  log "waiting for cleanup deletion of ${resource}"
  kubectl wait --for=delete "${resource}" --timeout="${TEST_CLEANUP_TIMEOUT}" >/dev/null
}

cleanup() {
  local status=$?
  local cleanup_status=0

  for pid in "${API_PORT_FORWARD_PID}" "${FRONTEND_PORT_FORWARD_PID}" "${GATEWAY_PORT_FORWARD_PID}"; do
    if [[ -n "${pid}" ]]; then
      kill "${pid}" >/dev/null 2>&1 || true
    fi
  done

  if [[ "${TEST_DRY_RUN}" != "true" && "${TEST_CLEANUP}" == "true" ]]; then
    if [[ -n "${TEST_USER_NAMESPACE}" && -n "${TEST_VM_NAME}" ]]; then
      kubectl -n "${TEST_USER_NAMESPACE}" delete "kitevirtualmachines.hy3ons.github.io/${TEST_VM_NAME}" --ignore-not-found=true --wait=false >/dev/null 2>&1 || true
      kubectl -n "${TEST_USER_NAMESPACE}" delete "virtualmachines.kubevirt.io/${TEST_VM_NAME}" --ignore-not-found=true --wait=false >/dev/null 2>&1 || true
      kubectl -n "${TEST_USER_NAMESPACE}" delete "datavolumes.cdi.kubevirt.io/${TEST_VM_NAME}-disk" --ignore-not-found=true --wait=false >/dev/null 2>&1 || true
    fi
    if [[ -n "${TEST_USER_NAME}" ]]; then
      kubectl delete "kiteusers.hy3ons.github.io/${TEST_USER_NAME}" --ignore-not-found=true --wait=false >/dev/null 2>&1 || true
    fi
    if [[ -n "${TEST_USER_NAMESPACE}" ]]; then
      kubectl delete namespace "${TEST_USER_NAMESPACE}" --ignore-not-found=true --wait=false >/dev/null 2>&1 || true
    fi
    if [[ -n "${TEST_USER_NAMESPACE}" ]]; then
      wait_for_cleanup_delete "namespace/${TEST_USER_NAMESPACE}" || cleanup_status=1
    fi
    if [[ -n "${TEST_USER_NAME}" ]]; then
      wait_for_cleanup_delete "kiteusers.hy3ons.github.io/${TEST_USER_NAME}" || cleanup_status=1
    fi
    if [[ "${cleanup_status}" -ne 0 ]]; then
      warn "cleanup did not finish within ${TEST_CLEANUP_TIMEOUT}"
      if [[ "${status}" -eq 0 ]]; then
        status=1
      fi
    fi
  fi

  rm -rf "${KUSTOMIZE_OVERLAY_DIR}"
  rm -f "${RENDERED_MANIFEST}" "${COOKIE_JAR}" "${API_PORT_FORWARD_LOG}" "${FRONTEND_PORT_FORWARD_LOG}" "${GATEWAY_PORT_FORWARD_LOG}"
  exit "${status}"
}

preflight() {
  require_command kubectl
  require_command docker
  require_command curl
  require_command python3
  if [[ "${TEST_GATEWAY_HOST_KEY_SOURCE}" == "host" ]]; then
    require_command ssh-keygen
    require_command ssh-keyscan
  fi
  docker buildx version >/dev/null

  case "${TEST_CLUSTER}" in
    k3s)
      if [[ "${K3S_CTR_CMD}" == *k3s* ]]; then
        require_command k3s
      fi
      ;;
    minikube)
      require_command minikube
      ;;
    k8s)
      ;;
    *)
      usage
      exit 1
      ;;
  esac

  kubectl get nodes >/dev/null
}

print_plan() {
  cat <<EOF

[kite-e2e] test plan
  cluster:        ${TEST_CLUSTER}
  namespace:      ${KITE_NAMESPACE}
  image prefix:   ${TEST_IMAGE_REGISTRY}
  image tag:      ${TEST_IMAGE_TAG}
  install deps:   ${TEST_INSTALL_DEPS}
  gateway key:    ${TEST_GATEWAY_HOST_KEY_SOURCE}
  key refresh:    ${TEST_GATEWAY_HOST_KEY_REFRESH}
  key file:       ${TEST_GATEWAY_HOST_KEY_FILE_NAME}
  cleanup:        ${TEST_CLEANUP}
  vm timeout:     ${TEST_VM_TIMEOUT}
  test user:      ${TEST_USERNAME} <${TEST_EMAIL}>
  test vm:        ${TEST_VM_NAME}
  test vm disk:   ${TEST_VM_DISK}
  dry run:        ${TEST_DRY_RUN}

EOF
}

main() {
  if [[ "$#" -ne 1 ]]; then
    usage
    exit 1
  fi

  trap cleanup EXIT

  configure_interactive_test_options
  validate_static_options
  print_plan

  if [[ "${TEST_DRY_RUN}" == "true" ]]; then
    prepare_minikube
    prepare_dependencies
    build_and_distribute_images
    render_manifest
    apply_kite_runtime
    log "dry-run complete"
    return
  fi

  preflight
  record_host_sshd_ports
  prepare_minikube
  prepare_dependencies
  build_and_distribute_images
  render_manifest
  apply_kite_runtime
  run_e2e_checks
  verify_host_sshd_ports_unchanged
  log "E2E test complete"
}

main "${TEST_CLUSTER}"
