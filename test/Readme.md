# Kite Test Standard

`test/` is the release gate for Kite.

This directory exists to answer one question before production publishing:

```text
Can a real cluster build this checkout, deploy Kite, accept API requests,
write CRDs, reconcile controller output, and boot a Kite VM successfully?
```

If the answer is not proven by an automated command, the change is not ready to
ship.

The full test inventory lives in [Test-Specification.md](./Test-Specification.md).
The deferred VMware SSH gateway verification plan lives in
[VMware-SSH-Gateway-Checklist.md](./VMware-SSH-Gateway-Checklist.md).

## Testing Philosophy

Kite is a Kubernetes control plane, so mock-only confidence is not enough.
Unit tests can prove that one function behaves correctly, but they cannot prove
that API, CRDs, RBAC, controller informers, KubeVirt, CDI, storage, frontend,
and gateway wiring all still work together.

Every meaningful change must therefore have two layers of proof:

1. Local correctness proof close to the code.
2. Cluster behavior proof through one of the E2E scripts in this directory.

The test standard is intentionally strict:

- Test real user paths, not only internal helpers.
- Prefer the same build and manifest path used by production.
- Fail early when required tools or registry settings are missing.
- Make every test idempotent enough to rerun after failure.
- Clean test resources by default, but allow cleanup to be disabled for
  debugging.
- Do not hide failures behind `|| true` except in cleanup.
- Do not replace a real E2E check with a mock just because the real check is
  slow.
- If a check cannot be run, record the exact reason and the next command that
  must be run.

## Directory Contract

Root-level `test/` scripts are for end-to-end release validation.

```text
test/
  all-test-k3s.sh        Full E2E gate for a k3s cluster.
  all-test-k8s.sh        Full E2E gate for a generic Kubernetes cluster.
  all-test-minikube.sh   Full E2E gate for a minikube cluster.
  lib/e2e.sh             Shared implementation used by the root scripts.
  Readme.md              This testing policy.
```

`build/test/` is not the release gate. It may keep small manual smoke helpers,
but new cluster-wide release validation belongs in `test/`.

## Required Gate Before Release

Before merging staged work toward production, run the E2E script that matches
the target cluster:

```sh
./test/all-test-k3s.sh
```

```sh
./test/all-test-minikube.sh
```

```sh
TEST_IMAGE_REGISTRY=registry.example.com/kite ./test/all-test-k8s.sh
```

For generic Kubernetes, `TEST_IMAGE_REGISTRY` is required because the cluster
must pull images from a registry. k3s and minikube can use locally loaded
images.

The E2E gate must prove all of the following:

- Docker buildx can build `kite-api`, `kite-controller`, `kite-gateway`, and
  `kite-frontend` from the current checkout.
- The target cluster receives or can pull those exact images.
- Kite manifests apply without editing the source manifests.
- `kite-api`, `kite-controller`, `kite-gateway`, and `kite-frontend` roll out.
- `/api/v1/health` reports healthy CRD read paths.
- Signup creates a real `KiteUser`.
- Login issues an authenticated session.
- The controller reconciles the user namespace and network policies.
- VM creation writes a real `KiteVirtualMachine`.
- The controller creates DataVolume, KubeVirt VirtualMachine, Secrets, and
  Services for that VM.
- The VM reaches `Running`.
- `KiteVirtualMachine.status.currentPowerState` becomes `On`.
- The frontend serves HTML.
- The gateway responds with an SSH banner.
- On k3s, the gateway host key Secret and the running gateway SSH port both
  reuse the host sshd fingerprint.

## Interactive Usage

The scripts are designed for humans who do not remember every environment
variable. In an interactive terminal they ask for important values and explain
what each value means.

Use the defaults when unsure. The defaults are chosen for a normal local test
cluster.

Use non-interactive mode only in automation:

```sh
KITE_ASSUME_DEFAULTS=true ./test/all-test-k3s.sh
```

Preview the commands without touching the cluster:

```sh
KITE_ASSUME_DEFAULTS=true TEST_DRY_RUN=true ./test/all-test-k3s.sh
```

## Important Environment Variables

`KITE_NAMESPACE`

The namespace where Kite runtime workloads are deployed. Default: `kite`.

`TEST_IMAGE_TAG`

The image tag used for this test run. Keep it unique so old images do not hide
new build problems. Default: `test-<timestamp>`.

`TEST_IMAGE_REGISTRY`

The image prefix. For k3s and minikube this can be a local prefix such as
`kite-test`. For generic Kubernetes it must be a pullable registry/repository
prefix such as `registry.example.com/kite`.

`TEST_INSTALL_DEPS`

When `true`, the script prepares Longhorn, KubeVirt, CDI, the Kite StorageClass,
and the golden image using the existing deployment helpers. Default: `true`.

`TEST_GATEWAY_HOST_KEY_SOURCE`

How the general cluster E2E scripts create the `kite-gateway` SSH host key
Secret. k3s default: `host`, so the gateway reuses the real host sshd key and
the test verifies that the Secret fingerprint matches `/etc/ssh`. minikube and
generic k8s default: `generate`, because those environments do not necessarily
share the node host key with the test runner. `host` mode may need sudo access
to read `/etc/ssh/ssh_host_*_key`; for noninteractive runs, cache sudo first or
run as root. Some SSH non-TTY sudo configurations do not share the cached sudo
timestamp with later `sudo -n` reads, so root execution is the more reliable
automation path.

`TEST_GATEWAY_HOST_KEY_REFRESH`

Whether the E2E script replaces an existing `kite-gateway` host key Secret.
Default: `true` when `TEST_GATEWAY_HOST_KEY_SOURCE=host`, otherwise `false`.
This is important for k3s because an older generated Secret would keep the
gateway working while silently breaking fingerprint preservation.

`TEST_GATEWAY_HOST_KEY_FILE_NAME`

The file name stored inside the gateway host key Secret and mounted into the
gateway container. Default: `ssh_host_rsa_key`, matching the production
manifest path `/etc/kite-gateway/ssh/ssh_host_rsa_key`.

The old k3s host-sshd mutation test has been removed. Kite installers no longer
move, restore, or otherwise manage the host sshd port. Host SSH is treated as
operator-owned infrastructure and must remain reachable through the operator's
own path. Gateway external exposure is tested through Admin Settings/runtime
ConfigMap reconcile and routes only Kite VM `sshId` users.

The general cluster E2E scripts do not perform this check. They patch
`kite-gateway` to `ClusterIP` and verify the gateway through `kubectl
port-forward`, because normal deploy-before-release testing must not steal the
operator's SSH session.

`TEST_CLEANUP`

When `true`, the script deletes the test VM, test KiteUser, and generated user
namespace after the run. Default: `true`. Set it to `false` when you need to
inspect a failed cluster state.

`TEST_CLEANUP_TIMEOUT`

How long the E2E gate waits for the test namespace and KiteUser to actually
disappear after cleanup starts. Default: `5m`. A successful E2E run fails if
cleanup does not converge within this timeout.

`TEST_VM_TIMEOUT`

How long the E2E gate waits for the VM to reach `Running`. Default: `20m`.
Use a larger value for slow storage or first-time image imports.

`TEST_VM_DOMAIN_PREFIX`

The HTTP hostname prefix used by the test VM. The VM create API requires
`domainPrefix`, so the E2E script sends this field explicitly instead of
depending on an API default. Default: the same value as `TEST_VM_NAME`.

`TEST_VM_DISK`

The root disk size requested for the test VM. Default: `20Gi`. Use a smaller
value, such as `8Gi`, on small single-node Longhorn test servers where CDI clone
creates temporary PVCs and Longhorn must keep its reserved free-space threshold.

`TEST_DRY_RUN`

When `true`, prints the high-level commands without building, deploying, or
creating resources. Default: `false`.

`K3S_CTR_CMD`

k3s image import command. Default: `sudo k3s ctr -n k8s.io`.

`MINIKUBE_PROFILE`

minikube profile used by the minikube E2E gate. Default: `minikube`.

## What To Add For Each Kind Of Change

Every change must update tests at the lowest meaningful layer and then pass the
cluster E2E gate.

### API Changes

When adding or changing an HTTP route:

- Add or update Go handler tests near `kite/cmd/kite-api/apis`.
- Test success, validation failure, authorization failure, and Kubernetes error
  mapping when relevant.
- Add an E2E assertion if the route changes a real user workflow.
- For write routes, prove the expected CRD field changed in the cluster.

Examples:

- New signup field: handler test plus E2E check on `KiteUser.spec`.
- New VM create option: handler/service test plus E2E check on
  `KiteVirtualMachine.spec` and the reconciled KubeVirt object.
- New admin route: auth/authorization test plus at least one E2E admin path if
  it affects production behavior.

### Controller Changes

When changing reconciliation:

- Add or update controller tests near `kite/cmd/kite-controller/apps`.
- Test idempotency: running the same reconcile twice must not corrupt state.
- Test drift recovery when the controller owns the drifted resource.
- Test status updates for both success and failure.
- Add E2E checks for every new real Kubernetes/KubeVirt resource the controller
  creates or mutates.

Controller tests must not only check that a function returns nil. They must
inspect the resulting object shape and status.

### CRD or Manifest Changes

When changing CRDs or manifests:

- Verify schema shape in a focused test when possible.
- Update E2E checks so the new field or resource is observed in a real cluster.
- Confirm cluster-scoped resources stay namespace-free.
- Confirm namespaced resources are applied to the intended namespace.
- Do not add protobuf or gRPC tests unless the gRPC design is explicitly
  reopened.

### Image, Dockerfile, or Frontend Build Changes

When changing any Dockerfile, frontend build arg, or runtime image behavior:

- Run the matching E2E gate because buildx and image loading are part of the
  release surface.
- Confirm frontend is built with `VITE_USE_MOCK=false` for E2E.
- Confirm the deployed image tag in the cluster matches `TEST_IMAGE_TAG`.

### Frontend Workflow Changes

When changing frontend behavior:

- Add or update frontend tests for route/component behavior where practical.
- Add E2E API checks when the frontend change depends on backend contract
  changes.
- Never rely on mock API behavior as the only proof for production-facing
  frontend behavior.

### Storage, VM, Gateway, or Console Changes

When changing VM disk, KubeVirt, CDI, gateway, SSH, or console behavior:

- The E2E gate must create a real VM.
- The VM must reach `Running` unless the change is explicitly about a failure
  state.
- Check the real Kubernetes objects, not only the Kite CRD.
- For gateway changes, keep the SSH banner check and add deeper checks if the
  changed behavior requires it.

## Required Developer Workflow

Use this workflow for every non-trivial change:

1. Identify the user-facing or cluster-facing behavior that could break.
2. Add or update the closest unit/service/controller test.
3. Add or update the E2E assertion when real cluster behavior changes.
4. Run focused tests first.
5. Run the matching `./test/all-test-*.sh` gate.
6. Record the exact command and result in the final change notes or commit body.

Skipping the E2E gate is allowed only when no suitable cluster is available.
When skipped, the change notes must say:

```text
E2E not run.
Reason: <specific reason>
Required command: <exact ./test/all-test-*.sh command>
Risk: <what remains unproven>
```

Do not write "not run" without a reason.

## Test Design Rules

Tests in this repository must be boring, explicit, and hard to fake.

- Name test resources with a unique prefix or timestamp.
- Prefer JSONPath or API response assertions over log-only checks.
- Wait for observed state, not arbitrary sleep.
- Use timeouts on every wait.
- Print enough context to debug failure.
- Keep cleanup idempotent.
- Do not delete shared infrastructure in normal cleanup.
- Do not require a human to copy/paste intermediate values.
- Do not depend on frontend mock data for E2E.
- Do not mutate production `main` images during test runs.
- Stage QA may pull `stage` images through `./ghcr-stage-install.sh`; E2E
  scripts must still avoid pushing `latest`, `main`, or `production` tags.
- GHCR cleanup must protect moving tags and unknown tags. Only `sha-*` images
  from `main` older than 30 days and `stage-sha-*` images from `stage` older
  than 10 days are cleanup candidates.

## Acceptance Criteria For New E2E Checks

A new E2E check is acceptable only if it has:

- A clear resource or endpoint being checked.
- A timeout.
- A failure message that names what did not become true.
- Cleanup behavior when it creates resources.
- No dependency on a previous manual step that the script could perform.

Bad:

```sh
sleep 60
kubectl get pods
```

Good:

```sh
kubectl -n "$namespace" wait \
  --for=jsonpath='{.status.phase}'=Running \
  "kitevirtualmachines.hy3ons.github.io/$vm_name" \
  --timeout="$TEST_VM_TIMEOUT"
```

## Failure Policy

When E2E fails, treat the failure as real until proven otherwise.

Triage order:

1. Check whether the script built the expected image tag.
2. Check workload rollout and pod events.
3. Check `/api/v1/health`.
4. Check the `KiteUser` and `KiteVirtualMachine` CRDs.
5. Check controller logs.
6. Check DataVolume and KubeVirt VM status.
7. Check frontend and gateway port-forward logs.

Do not patch the test to pass before understanding the failed behavior. If the
test expectation is wrong, fix the expectation and explain why the old
expectation was wrong.

## When To Add A New Script

Do not add a new root E2E script for every feature. Add feature checks to
`test/lib/e2e.sh` unless the target environment is genuinely different.

Add a new root script only when:

- The cluster type is different.
- The image distribution mechanism is different.
- The setup or cleanup safety rules are different.
- The script is a separate release gate with a different operator audience.

## Relationship To Other Tests

`go test` remains mandatory for Go behavior near the changed code.

Frontend tests remain mandatory for frontend behavior near the changed UI.

`test/all-test-*.sh` is mandatory for release confidence because it proves the
whole system through Kubernetes.

`test/shared-infra-protection.sh` is mandatory before promoting cleanup changes.
It creates an external KubeVirt VM and Longhorn PVC, runs Kite cleanup with
dangerous Longhorn flags enabled, and then proves the external infrastructure is
still present.

No single layer replaces the others.

## Current Release Gate Commands

k3s:

```sh
./test/all-test-k3s.sh
./test/shared-infra-protection.sh
```

minikube:

```sh
./test/all-test-minikube.sh
```

generic Kubernetes:

```sh
TEST_IMAGE_REGISTRY=registry.example.com/kite ./test/all-test-k8s.sh
```

Dry-run preview:

```sh
KITE_ASSUME_DEFAULTS=true TEST_DRY_RUN=true ./test/all-test-k3s.sh
```
