#!/usr/bin/env bash
set -euo pipefail

# Exercise the updater's process-level failure contract without a cluster.
# The kubectl fixture models independent deployment failures and image rollback.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
fixture="$(mktemp -d "${TMPDIR:-/tmp}/kite-update-test.XXXXXX")"
trap 'rm -rf "${fixture}"' EXIT
mkdir -p "${fixture}/bin"
cat > "${fixture}/bin/kubectl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "${FAKE_STATE}/calls"
if [[ "${1:-}" == '-n' ]]; then shift 2; fi
case "$1 $2" in
  'get deployment')
    [[ "${FAKE_MISSING_DEPLOYMENT:-}" != "$3" ]] || exit 1
    printf '%s|%s\n' "$(cat "${FAKE_STATE}/$3.image")" "$(cat "${FAKE_STATE}/$3.policy")"
    ;;
  'get pods')
    for argument in "$@"; do
      if [[ "${argument}" == app=* ]]; then
        printf 'true %s\n' "$(cat "${FAKE_STATE}/${argument#app=}.digest")"
      fi
    done
    ;;
  'get nodes') printf '%s\n' "${FAKE_NODES:-node-1 arm64}" ;;
  'get configmap') : ;;
  'apply -f'|'apply -k')
    [[ "${FAKE_REJECT_APPLY:-false}" != true ]] || exit 9
    ;;
  'patch deployment/'*)
    if [[ "${FAKE_REQUIRE_IMAGES:-false}" == true ]]; then
      [[ -f "${FAKE_STATE}/imported" ]] || exit 99
    fi
    deployment="${2#deployment/}"
    image_pattern='"image":"([^"]+)"'
    [[ "$*" =~ ${image_pattern} ]] || exit 2
    image="${BASH_REMATCH[1]}"
    if [[ "${FAKE_FAIL_APPLY:-}" == "${deployment}" && "${image}" == *:new ]]; then exit 9; fi
    printf '%s\n' "${image}" > "${FAKE_STATE}/${deployment}.image"
    policy_pattern='"imagePullPolicy":"([^"]+)"'
    [[ "$*" =~ ${policy_pattern} ]] || exit 2
    printf '%s\n' "${BASH_REMATCH[1]}" > "${FAKE_STATE}/${deployment}.policy"
    if [[ "$*" == *'kite.hy3ons.github.io/update-id'* ]]; then
      touch "${FAKE_STATE}/restarted"
      if [[ "$(cat "${FAKE_STATE}/${deployment}.policy")" == Always ]]; then
        cp "${FAKE_STATE}/${deployment}.image" "${FAKE_STATE}/${deployment}.resolved"
      fi
    fi
    ;;
  'rollout status')
    deployment="${3#deployment/}"
    if [[ "${FAKE_FAIL_ROLLOUT:-}" == "${deployment}" ]] && [[ "$(cat "${FAKE_STATE}/${deployment}.image")" == *:new ]]; then exit 7; fi
    ;;
  'port-forward svc/'*) exec /bin/sleep 60 ;;
  *) printf 'Unexpected kubectl arguments: %s\n' "$*" >&2; exit 2 ;;
esac
EOF
cat > "${fixture}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'docker %s\n' "$*" >> "${FAKE_STATE}/calls"
[[ "${FAKE_REJECT_DOCKER:-false}" != true ]] || exit 99
case "$1" in
  build)
    [[ -z "${FAKE_FAIL_BUILD:-}" || "$*" != *"kite-${FAKE_FAIL_BUILD}:"* ]] || exit 8
    printf 'built\n' >> "${FAKE_STATE}/built"
    ;;
  save) printf 'image-archive' ;;
  *) exit 2 ;;
esac
EOF
cat > "${fixture}/bin/k3s" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'k3s %s\n' "$*" >> "${FAKE_STATE}/calls"
case "$1" in
  kubectl) printf '%s\n' "${FAKE_LOCAL_NODES:-node-1 arm64}" ;;
  ctr)
    [[ "$(wc -l < "${FAKE_STATE}/built" | tr -d ' ')" == 4 ]] || exit 99
    cat >/dev/null
    [[ "${FAKE_FAIL_IMPORT:-false}" != true ]] || exit 6
    touch "${FAKE_STATE}/imported"
    ;;
  *) exit 2 ;;
esac
EOF
cat > "${fixture}/bin/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${FAKE_FAIL_HEALTH:-false}" == true && "$*" == *api/v1/health* ]]; then exit 22; fi
EOF
chmod +x "${fixture}/bin/"*
export PATH="${fixture}/bin:${PATH}"
export KITE_ASSUME_DEFAULTS=true KITE_UPDATE_TAG=new KITE_UPDATE_COMPONENTS=api,frontend
export KITE_UPDATE_APPLY_CRDS=false KITE_UPDATE_APPLY_RBAC=false KITE_UPDATE_HEALTH_CHECK=false
failures=0

run_case() {
  local name="$1"
  shift
  export FAKE_STATE="${fixture}/${name}"
  mkdir "${FAKE_STATE}"
  printf 'registry/kite-api:old\n' > "${FAKE_STATE}/kite-api.image"
  printf 'registry/kite-frontend:old\n' > "${FAKE_STATE}/kite-frontend.image"
  for component in kite-api kite-controller kite-gateway kite-frontend; do
    printf 'registry/%s:old\n' "${component}" > "${FAKE_STATE}/${component}.image"
    printf 'IfNotPresent\n' > "${FAKE_STATE}/${component}.policy"
    printf 'cached-old-image\n' > "${FAKE_STATE}/${component}.resolved"
    printf 'registry/%s@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n' "${component}" > "${FAKE_STATE}/${component}.digest"
  done
  : > "${FAKE_STATE}/calls"
  case_exit_code=0
  env "$@" bash "${TEST_UPDATER:-${ROOT_DIR}/build/deploy/scripts/update-ghcr.sh}" > "${FAKE_STATE}/output" 2>&1 || case_exit_code=$?
}

expect_failure_and_rollback() {
  if [[ "${case_exit_code}" -eq 0 ]] || [[ "$(cat "${FAKE_STATE}/kite-api.image")" != "$(cat "${FAKE_STATE}/kite-api.digest")" ]] || [[ "$(cat "${FAKE_STATE}/kite-frontend.image")" != "$(cat "${FAKE_STATE}/kite-frontend.digest")" ]] || [[ "$(cat "${FAKE_STATE}/kite-api.policy")" != IfNotPresent ]]; then
    printf 'FAIL %s: exit=%s; images=%s / %s\n' "$1" "${case_exit_code}" "$(cat "${FAKE_STATE}/kite-api.image")" "$(cat "${FAKE_STATE}/kite-frontend.image")"
    failures=$((failures + 1))
  else printf 'PASS %s\n' "$1"; fi
}

run_case first-rollout-failure FAKE_FAIL_ROLLOUT=kite-api
expect_failure_and_rollback 'earlier rollout failure cannot be hidden by later success'

run_case first-health-failure KITE_UPDATE_HEALTH_CHECK=true FAKE_FAIL_HEALTH=true
expect_failure_and_rollback 'API health failure cannot be hidden by frontend success'

run_case partial-apply-failure FAKE_FAIL_APPLY=kite-frontend
expect_failure_and_rollback 'partial image mutation rolls back previous components'

run_case invalid-components KITE_UPDATE_COMPONENTS=api,unknown
if [[ "${case_exit_code}" -eq 0 ]] || grep -q '^apply ' "${FAKE_STATE}/calls"; then
  printf 'FAIL invalid components must stop before applying manifests\n'
  failures=$((failures + 1))
else printf 'PASS invalid components rejected before mutation\n'; fi

run_case refresh
if [[ "${case_exit_code}" -ne 0 ]] || [[ ! -f "${FAKE_STATE}/restarted" ]] || [[ "$(cat "${FAKE_STATE}/kite-api.resolved")" != ghcr.io/hy3ons/kite-api:new ]]; then
  printf 'FAIL repeated tag updates must replace Pods and resolve the remote image\n'
  failures=$((failures + 1))
else printf 'PASS repeated tags request fresh Pods\n'; fi

run_case dry-run KITE_UPDATE_DRY_RUN=true
if [[ "${case_exit_code}" -ne 0 ]] || [[ -s "${FAKE_STATE}/calls" ]]; then
  printf 'FAIL dry run must not access or mutate the cluster\n'
  failures=$((failures + 1))
else printf 'PASS dry run has no cluster side effects\n'; fi

run_case runtime-only FAKE_REJECT_APPLY=true
if [[ "${case_exit_code}" -ne 0 ]] || grep -Eq '^(apply|set image|create|delete) ' "${FAKE_STATE}/calls"; then
  printf 'FAIL default update must only patch existing runtime containers\n'
  failures=$((failures + 1))
else printf 'PASS default update does not reapply installation manifests\n'; fi

run_case missing-deployment FAKE_MISSING_DEPLOYMENT=kite-frontend
if [[ "${case_exit_code}" -eq 0 ]] || grep -q '^patch ' "${FAKE_STATE}/calls"; then
  printf 'FAIL missing deployment must stop before any component is changed\n'
  failures=$((failures + 1))
else printf 'PASS preflight checks all deployments before mutation\n'; fi

run_case explicit-migrations KITE_UPDATE_APPLY_CRDS=true KITE_UPDATE_APPLY_RBAC=true
if [[ "${case_exit_code}" -ne 0 ]] || [[ "$(grep -c '^apply -f ' "${FAKE_STATE}/calls")" -ne 3 ]]; then
  printf 'FAIL explicit migrations must apply only CRD, ServiceAccount and RBAC manifests\n'
  failures=$((failures + 1))
else printf 'PASS schema and RBAC migrations require explicit opt-in\n'; fi

TEST_UPDATER="${ROOT_DIR}/update.sh" run_case general-update FAKE_REJECT_DOCKER=true
if [[ "${case_exit_code}" -ne 0 ]] || grep -q '^docker ' "${FAKE_STATE}/calls"; then
  printf 'FAIL general update must never build source\n'; failures=$((failures + 1))
else printf 'PASS general update pulls images without a builder\n'; fi

export KITE_UPDATE_SOURCE_DIR="${fixture}/source" KITE_UPDATE_SOURCE_REVISION=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
mkdir -p "${KITE_UPDATE_SOURCE_DIR}"
for mode in success build-failure import-failure wrong-cluster; do
  options=(KITE_UPDATE_COMPONENTS=api,controller,gateway,frontend FAKE_REQUIRE_IMAGES=true)
  case "${mode}" in
    build-failure) options+=(FAKE_FAIL_BUILD=controller) ;;
    import-failure) options+=(FAKE_FAIL_IMPORT=true) ;;
    wrong-cluster) options+=(FAKE_LOCAL_NODES=other-node) ;;
  esac
  TEST_UPDATER="${ROOT_DIR}/build/dev/update-local.sh" run_case "source-${mode}" "${options[@]}"
  if [[ "${mode}" == success ]]; then
    if [[ "${case_exit_code}" -ne 0 ]] || [[ ! -f "${FAKE_STATE}/imported" ]] || [[ "$(cat "${FAKE_STATE}/kite-api.policy")" != Never ]]; then
      printf 'FAIL all source images must build/import before rollout\n'; failures=$((failures + 1))
    else printf 'PASS source builds/imports all images before patching with Never policy\n'; fi
  else
    if [[ "${case_exit_code}" -eq 0 ]] || grep -q '^patch ' "${FAKE_STATE}/calls"; then
      printf 'FAIL source %s must preserve deployments\n' "${mode}"; failures=$((failures + 1))
    else printf 'PASS source %s stops before deployment mutation\n' "${mode}"; fi
  fi
done

TEST_UPDATER="${ROOT_DIR}/build/dev/update-local.sh" run_case source-dry-run KITE_UPDATE_DRY_RUN=true FAKE_REJECT_DOCKER=true
if [[ "${case_exit_code}" -ne 0 ]] || [[ -s "${FAKE_STATE}/calls" ]]; then
  printf 'FAIL source dry run must not contact the builder or cluster\n'; failures=$((failures + 1))
else printf 'PASS source dry run has no builder or cluster side effects\n'; fi

printf 'Updater checks: %s failure(s)\n' "${failures}"
test "${failures}" -eq 0
