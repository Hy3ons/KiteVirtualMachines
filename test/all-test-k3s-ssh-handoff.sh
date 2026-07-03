#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

KITE_NAMESPACE_WAS_SET="${KITE_NAMESPACE+x}"
TEST_SSH_HOST_WAS_SET="${TEST_SSH_HOST+x}"
TEST_HOST_SSH_USER_WAS_SET="${TEST_HOST_SSH_USER+x}"
TEST_HOST_SSH_PASSWORD_WAS_SET="${TEST_HOST_SSH_PASSWORD+x}"
TEST_GATEWAY_PORT_WAS_SET="${TEST_GATEWAY_PORT+x}"
TEST_HOST_SSHD_PORT_WAS_SET="${TEST_HOST_SSHD_PORT+x}"
TEST_MANAGE_HOST_SSHD_WAS_SET="${TEST_MANAGE_HOST_SSHD+x}"
TEST_APPLY_GATEWAY_LOADBALANCER_WAS_SET="${TEST_APPLY_GATEWAY_LOADBALANCER+x}"
TEST_SSH_PROBE_RUNNER_WAS_SET="${TEST_SSH_PROBE_RUNNER+x}"
TEST_DRY_RUN_WAS_SET="${TEST_DRY_RUN+x}"

KITE_NAMESPACE="${KITE_NAMESPACE:-kite}"
TEST_SSH_HOST="${TEST_SSH_HOST:-}"
TEST_HOST_SSH_USER="${TEST_HOST_SSH_USER:-${USER:-}}"
TEST_HOST_SSH_PASSWORD="${TEST_HOST_SSH_PASSWORD:-}"
TEST_GATEWAY_PORT="${TEST_GATEWAY_PORT:-22}"
TEST_HOST_SSHD_PORT="${TEST_HOST_SSHD_PORT:-2022}"
TEST_MANAGE_HOST_SSHD="${TEST_MANAGE_HOST_SSHD:-true}"
TEST_APPLY_GATEWAY_LOADBALANCER="${TEST_APPLY_GATEWAY_LOADBALANCER:-true}"
TEST_SSH_PROBE_RUNNER="${TEST_SSH_PROBE_RUNNER:-auto}"
TEST_DRY_RUN="${TEST_DRY_RUN:-false}"

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
    printf "\033[0;32m[kite-ssh-handoff] %s - %s\033[0m\n" "${timestamp}" "$*"
  else
    printf "[kite-ssh-handoff] %s - %s\n" "${timestamp}" "$*"
  fi
}

warn() {
  local timestamp

  timestamp="$(log_timestamp)"
  if log_color_enabled; then
    printf "\033[1;33m[kite-ssh-handoff] WARNING: %s - %s\033[0m\n" "${timestamp}" "$*" >&2
  else
    printf "[kite-ssh-handoff] WARNING: %s - %s\n" "${timestamp}" "$*" >&2
  fi
}

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    warn "missing required command: ${name}"
    exit 1
  fi
}

run_cmd() {
  if [[ "${TEST_DRY_RUN}" == "true" ]]; then
    printf '[kite-ssh-handoff] dry-run:'
    printf ' %q' "$@"
    printf '\n'
    return 0
  fi

  "$@"
}

prompt_value() {
  local variable_name="$1"
  local was_set="$2"
  local prompt="$3"
  local description="$4"
  local secret="${5:-false}"
  local current_value
  local answer

  eval "current_value=\"\${${variable_name}:-}\""
  if [[ -n "${was_set}" ]] || ! kite_prompt_interactive; then
    return 0
  fi

  printf '%s\n' "${prompt}" >&2
  printf '  %s\n' "${description}" >&2
  if [[ "${secret}" == "true" ]]; then
    read -r -s -p "입력 [필수, 화면에 표시하지 않음] " answer
    printf '\n' >&2
  else
    read -r -p "입력 [기본: ${current_value:-없음}] " answer
    answer="${answer:-${current_value}}"
  fi
  printf -v "${variable_name}" '%s' "${answer}"
  export "${variable_name}"
}

# valid_tcp_port validates user-entered SSH ports before they are used in sshd and Kubernetes patches.
valid_tcp_port() {
  local port="$1"

  [[ "${port}" =~ ^[0-9]+$ ]] || return 1
  (( port >= 1 && port <= 65535 ))
}

default_node_ip() {
  kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null || true
}

configure_options() {
  if [[ -z "${TEST_SSH_HOST}" ]]; then
    TEST_SSH_HOST="$(default_node_ip)"
  fi
  if [[ -z "${TEST_SSH_HOST}" && "${TEST_DRY_RUN}" == "true" ]]; then
    TEST_SSH_HOST="127.0.0.1"
  fi

  if kite_prompt_interactive; then
    log "interactive SSH handoff test options"
    prompt_value KITE_NAMESPACE "${KITE_NAMESPACE_WAS_SET}" "KITE_NAMESPACE 값을 정합니다." "kite-gateway Service와 Deployment가 있는 Kubernetes namespace입니다."
    prompt_value TEST_SSH_HOST "${TEST_SSH_HOST_WAS_SET}" "TEST_SSH_HOST 값을 정합니다." "테스트가 SSH로 접속할 노드 주소입니다. 보통 k3s 노드의 내부 IP 또는 외부 도메인입니다."
    prompt_value TEST_HOST_SSH_USER "${TEST_HOST_SSH_USER_WAS_SET}" "TEST_HOST_SSH_USER 값을 정합니다." "VM route가 없는 일반 Linux host 계정입니다. 이 사용자로 gateway fallback 로그인을 검증합니다."
    prompt_value TEST_HOST_SSH_PASSWORD "${TEST_HOST_SSH_PASSWORD_WAS_SET}" "TEST_HOST_SSH_PASSWORD 값을 정합니다." "위 host 계정의 SSH 비밀번호입니다. host sshd 직접 로그인과 gateway fallback 로그인을 검증할 때만 사용합니다." true
    prompt_value TEST_GATEWAY_PORT "${TEST_GATEWAY_PORT_WAS_SET}" "TEST_GATEWAY_PORT 값을 정합니다." "외부에서 kite-gateway가 받아야 하는 SSH 포트입니다. 운영 기본값은 22입니다."
    prompt_value TEST_HOST_SSHD_PORT "${TEST_HOST_SSHD_PORT_WAS_SET}" "TEST_HOST_SSHD_PORT 값을 정합니다." "host sshd가 handoff 후 직접 들을 포트입니다. 테스트 기본값은 2022입니다."
    prompt_value TEST_SSH_PROBE_RUNNER "${TEST_SSH_PROBE_RUNNER_WAS_SET}" "TEST_SSH_PROBE_RUNNER 값을 정합니다." "SSH password probe 실행 방식입니다. auto는 go가 있으면 go run, 없으면 Docker golang image를 사용합니다. 값: auto, go, docker."
    kite_prompt_configure_bool TEST_MANAGE_HOST_SSHD "${TEST_MANAGE_HOST_SSHD_WAS_SET}" $'TEST_MANAGE_HOST_SSHD 값을 정합니다.\n  예를 고르면 /etc/ssh/sshd_config를 백업하고 host sshd를 선택한 포트로 옮깁니다.'
    kite_prompt_configure_bool TEST_APPLY_GATEWAY_LOADBALANCER "${TEST_APPLY_GATEWAY_LOADBALANCER_WAS_SET}" $'TEST_APPLY_GATEWAY_LOADBALANCER 값을 정합니다.\n  예를 고르면 kite-gateway Service를 LoadBalancer로 바꿔 22번을 gateway가 받게 합니다.'
    kite_prompt_configure_bool TEST_DRY_RUN "${TEST_DRY_RUN_WAS_SET}" $'TEST_DRY_RUN 값을 정합니다.\n  예를 고르면 host sshd와 Kubernetes Service를 바꾸지 않고 실행 계획만 출력합니다.'
  fi

  if [[ -z "${TEST_SSH_HOST}" ]]; then
    warn "TEST_SSH_HOST is required because the script must check ports ${TEST_GATEWAY_PORT} and ${TEST_HOST_SSHD_PORT}"
    exit 1
  fi
  if [[ -z "${TEST_HOST_SSH_USER}" ]]; then
    warn "TEST_HOST_SSH_USER is required for host fallback login"
    exit 1
  fi
  if [[ -z "${TEST_HOST_SSH_PASSWORD}" && "${TEST_DRY_RUN}" != "true" ]]; then
    warn "TEST_HOST_SSH_PASSWORD is required for password login checks"
    exit 1
  fi
  if ! valid_tcp_port "${TEST_GATEWAY_PORT}"; then
    warn "TEST_GATEWAY_PORT must be a number between 1 and 65535"
    exit 1
  fi
  if ! valid_tcp_port "${TEST_HOST_SSHD_PORT}"; then
    warn "TEST_HOST_SSHD_PORT must be a number between 1 and 65535"
    exit 1
  fi
  if [[ "${TEST_GATEWAY_PORT}" == "${TEST_HOST_SSHD_PORT}" ]]; then
    warn "TEST_GATEWAY_PORT and TEST_HOST_SSHD_PORT must be different"
    exit 1
  fi
}

print_plan() {
  cat <<EOF

[kite-ssh-handoff] test plan
  namespace:          ${KITE_NAMESPACE}
  ssh host:           ${TEST_SSH_HOST}
  host user:          ${TEST_HOST_SSH_USER}
  gateway port:       ${TEST_GATEWAY_PORT}
  host sshd port:     ${TEST_HOST_SSHD_PORT}
  manage host sshd:   ${TEST_MANAGE_HOST_SSHD}
  apply LoadBalancer: ${TEST_APPLY_GATEWAY_LOADBALANCER}
  probe runner:       ${TEST_SSH_PROBE_RUNNER}
  dry run:            ${TEST_DRY_RUN}

EOF
}

selected_probe_runner() {
  case "${TEST_SSH_PROBE_RUNNER}" in
    go|docker)
      echo "${TEST_SSH_PROBE_RUNNER}"
      ;;
    auto)
      if command -v go >/dev/null 2>&1; then
        echo go
      else
        echo docker
      fi
      ;;
    *)
      warn "TEST_SSH_PROBE_RUNNER must be auto, go, or docker"
      exit 1
      ;;
  esac
}

preflight() {
  local runner

  require_command kubectl
  require_command ssh
  require_command ssh-keygen
  require_command ssh-keyscan
  require_command python3
  runner="$(selected_probe_runner)"
  require_command "${runner}"

  if [[ "${TEST_DRY_RUN}" == "true" ]]; then
    return
  fi

  kubectl -n "${KITE_NAMESPACE}" get deployment/kite-gateway >/dev/null
  kubectl -n "${KITE_NAMESPACE}" get service/kite-gateway >/dev/null
}

ensure_host_sshd_handoff() {
  local manage_mode

  if [[ "${TEST_MANAGE_HOST_SSHD}" != "true" ]]; then
    log "skipping host sshd handoff because TEST_MANAGE_HOST_SSHD=${TEST_MANAGE_HOST_SSHD}"
    return
  fi

  manage_mode=true
  if kite_prompt_interactive; then
    manage_mode=ask
  fi

  log "moving host sshd to port ${TEST_HOST_SSHD_PORT}"
  run_cmd env \
    KITE_MANAGE_HOST_SSHD="${manage_mode}" \
    KITE_HOST_SSHD_PORT="${TEST_HOST_SSHD_PORT}" \
    KITE_GATEWAY_EXTERNAL_PORT="${TEST_GATEWAY_PORT}" \
    "${ROOT_DIR}/build/deploy/scripts/manage-host-sshd.sh" ensure
}

patch_gateway_host_sshd_address() {
  log "configuring kite-gateway host fallback address with host sshd port ${TEST_HOST_SSHD_PORT}"
  run_cmd kubectl -n "${KITE_NAMESPACE}" set env deployment/kite-gateway "KITE_GATEWAY_HOST_SSHD_ADDRESS=\$(KITE_NODE_IP):${TEST_HOST_SSHD_PORT}"
}

apply_gateway_loadbalancer() {
  if [[ "${TEST_APPLY_GATEWAY_LOADBALANCER}" != "true" ]]; then
    log "skipping gateway Service LoadBalancer patch because TEST_APPLY_GATEWAY_LOADBALANCER=${TEST_APPLY_GATEWAY_LOADBALANCER}"
    return
  fi

  log "exposing kite-gateway Service on external port ${TEST_GATEWAY_PORT}"
  patch_gateway_host_sshd_address
  run_cmd kubectl -n "${KITE_NAMESPACE}" patch service/kite-gateway --type=merge -p "{
    \"spec\": {
      \"type\": \"LoadBalancer\",
      \"ports\": [
        {
          \"name\": \"ssh\",
          \"port\": ${TEST_GATEWAY_PORT},
          \"targetPort\": \"ssh\",
          \"protocol\": \"TCP\"
        }
      ]
    }
  }"
  run_cmd kubectl -n "${KITE_NAMESPACE}" rollout status deployment/kite-gateway --timeout=180s
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

verify_gateway_reuses_host_fingerprint() {
  local host_fingerprints
  local gateway_fingerprints
  local matched_fingerprint

  log "checking gateway host key fingerprint reuse"
  if [[ "${TEST_DRY_RUN}" == "true" ]]; then
    printf '[kite-ssh-handoff] dry-run: compare ssh-keyscan fingerprints for %q:%q and %q:%q\n' "${TEST_SSH_HOST}" "${TEST_HOST_SSHD_PORT}" "${TEST_SSH_HOST}" "${TEST_GATEWAY_PORT}"
    return
  fi

  host_fingerprints="$(scan_ssh_fingerprints "${TEST_SSH_HOST}" "${TEST_HOST_SSHD_PORT}")"
  gateway_fingerprints="$(scan_ssh_fingerprints "${TEST_SSH_HOST}" "${TEST_GATEWAY_PORT}")"
  if [[ -z "${host_fingerprints}" ]]; then
    warn "could not read host sshd fingerprints from ${TEST_SSH_HOST}:${TEST_HOST_SSHD_PORT}"
    exit 1
  fi
  if [[ -z "${gateway_fingerprints}" ]]; then
    warn "could not read gateway fingerprints from ${TEST_SSH_HOST}:${TEST_GATEWAY_PORT}"
    exit 1
  fi

  matched_fingerprint="$(grep -Fxf <(printf '%s\n' "${host_fingerprints}") <(printf '%s\n' "${gateway_fingerprints}") | head -n 1 || true)"
  if [[ -z "${matched_fingerprint}" ]]; then
    warn "gateway fingerprints do not match host sshd fingerprints"
    warn "host sshd fingerprints: ${host_fingerprints//$'\n'/, }"
    warn "gateway fingerprints: ${gateway_fingerprints//$'\n'/, }"
    exit 1
  fi

  log "gateway reuses host sshd fingerprint ${matched_fingerprint}"
}

probe() {
  local runner

  if [[ "${TEST_DRY_RUN}" == "true" ]]; then
    printf '[kite-ssh-handoff] dry-run: %s SSH probe' "$(selected_probe_runner)"
    printf ' %q' "$@"
    printf '\n'
    return 0
  fi

  runner="$(selected_probe_runner)"
  case "${runner}" in
    go)
      (cd "${ROOT_DIR}/kite" && TEST_HOST_SSH_PASSWORD="${TEST_HOST_SSH_PASSWORD}" go run "${ROOT_DIR}/test/tools/ssh-probe.go" "$@")
      ;;
    docker)
      docker run --rm --network host \
        -e TEST_HOST_SSH_PASSWORD="${TEST_HOST_SSH_PASSWORD}" \
        -v "${ROOT_DIR}:/workspace" \
        -w /workspace/kite \
        golang:1.25-alpine \
        go run ../test/tools/ssh-probe.go "$@"
      ;;
  esac
}

verify_ports_and_fallback() {
  log "checking host sshd banner on ${TEST_SSH_HOST}:${TEST_HOST_SSHD_PORT}"
  probe -mode=banner -host="${TEST_SSH_HOST}" -port="${TEST_HOST_SSHD_PORT}" -want-contains=SSH -want-not-contains=kite-gateway

  log "checking gateway banner on ${TEST_SSH_HOST}:${TEST_GATEWAY_PORT}"
  probe -mode=banner -host="${TEST_SSH_HOST}" -port="${TEST_GATEWAY_PORT}" -want-contains=kite-gateway

  verify_gateway_reuses_host_fingerprint

  log "checking direct host password login on ${TEST_SSH_HOST}:${TEST_HOST_SSHD_PORT}"
  probe -mode=password -host="${TEST_SSH_HOST}" -port="${TEST_HOST_SSHD_PORT}" -user="${TEST_HOST_SSH_USER}"

  log "checking gateway host fallback password login on ${TEST_SSH_HOST}:${TEST_GATEWAY_PORT}"
  probe -mode=password -host="${TEST_SSH_HOST}" -port="${TEST_GATEWAY_PORT}" -user="${TEST_HOST_SSH_USER}"
}

main() {
  configure_options
  print_plan
  preflight
  ensure_host_sshd_handoff
  apply_gateway_loadbalancer
  verify_ports_and_fallback
  if [[ "${TEST_DRY_RUN}" == "true" ]]; then
    log "SSH handoff dry-run complete"
  else
    log "SSH handoff acceptance passed"
  fi
}

main "$@"
