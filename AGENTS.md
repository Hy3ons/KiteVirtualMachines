# AGENTS.md

## Project Overview

This repository contains the Kite Kubernetes control plane prototype.

- `kite/cmd/kite-api`: HTTP API server based on Gin.
- `kite/cmd/kite-controller`: Kubernetes controller code that reconciles Kite CRDs.
- `kite/api/proto`: retired protobuf draft from the old API-to-controller gRPC plan.
- `kite/api/v1`: Go structs for Kite custom resources.
- `kite/internal/kube`: Kubernetes client helpers.
- `kite/internal/render`: YAML template renderers that return `unstructured.Unstructured`.
- `build/kite`: Kite CRD definitions and shared install manifests.
- `build/examples`: example Kite custom resources for manual testing.

## Coding Rules

Write readable Go code that follows the existing project layout. Prefer small
functions with clear names over large functions with many responsibilities.

When adding or changing a function, write an English comment that explains:

- what the function does,
- what each parameter is used for,
- what the return value means when it is not obvious,
- where the function is expected to be used in this project.

Use this style for exported functions:

```go
// NewClientManager creates a Kubernetes client manager.
// kubeClient is used for typed Kubernetes API calls.
// dynamicClient is used for unstructured custom resource calls.
// This function is used by API and controller startup code.
func NewClientManager(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface) *ClientManager {
	return &ClientManager{
		KubeClient:    kubeClient,
		DynamicClient: dynamicClient,
	}
}
```

Use the same level of explanation for important unexported functions, especially
handlers, validators, renderers, controller reconcilers, and status helpers.

Avoid comments that only repeat the code. Comments should explain intent,
parameters, project usage, or non-obvious behavior.

## Retired gRPC and Protobuf Rules

The previous API-to-controller gRPC direction has been retired.

Do not add new protobuf definitions, generated protobuf files, gRPC service
implementations, or API-to-controller RPC calls unless the user explicitly
reopens the gRPC design.

The API server should write `KiteUser` and `KiteVirtualMachine` CRDs through the
Kubernetes API server. The controller should watch those CRDs and reconcile real
Kubernetes/KubeVirt resources from CRD `spec`, then write observed state back to
CRD `status`.

Current custom resources:

- `KiteUser`
  - CRD file: `build/kite/crds.yaml`
  - scope: `Cluster`
  - resource: `kiteusers`
  - group/version: `hy3ons.github.io/v1`
- `KiteVirtualMachine`
  - CRD file: `build/kite/crds.yaml`
  - scope: `Namespaced`
  - resource: `kitevirtualmachines`
  - group/version: `hy3ons.github.io/v1`

Do not run `protoc` or commit generated protobuf files.

## Kubernetes Rules

Use `dynamic.Interface` and `unstructured.Unstructured` for Kite custom
resources unless the task explicitly asks for typed clients.

Deploy Kite's own runtime resources in the `kite` namespace. This includes
`kite-api`, `kite-controller`, `kite-frontend`, Services, and the ServiceAccount
used by API/controller pods.

Keep cluster-wide resources namespace-free. `CustomResourceDefinition` objects
do not have a namespace, and `KiteUser` custom resources are cluster-scoped.

When API/controller code runs in a Pod, use Kubernetes in-cluster config through
the mounted service account. Local kubeconfig fallback is only for development.

When creating Kite custom resources, use these GVR values:

```go
schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kiteusers",
}
```

```go
schema.GroupVersionResource{
	Group:    "hy3ons.github.io",
	Version:  "v1",
	Resource: "kitevirtualmachines",
}
```

Do not call `.Namespace(...)` for `KiteUser` because it is cluster-scoped.

Call `.Namespace(req.Namespace)` for `KiteVirtualMachine` because it is
namespaced.

## Controller Reconcile Rules

Do not implement controller commands through gRPC. The controller should be a
Kubernetes-style reconciler:

- watch `KiteUser` and `KiteVirtualMachine` CRDs,
- treat CRD `spec` as the desired state,
- create or update Kubernetes/KubeVirt resources with idempotent apply logic,
- watch real KubeVirt/DataVolume state when needed,
- write observed state and failure reasons to CRD `status`,
- avoid changing CRD `spec` from KubeVirt state watchers.

For `KiteVirtualMachine`, user power intent belongs in
`spec.powerState`. The controller should translate it to KubeVirt VM state, and
if the actual KubeVirt state drifts from the desired power state, reconcile it
back.

## Branch Rules

`main` is the production publishing branch. A push to `main` triggers the GHCR
image publishing workflow and updates production Docker image tags.

Before starting user-requested code or documentation work, check the current
branch with `git status -sb` or `git branch --show-current`.

Do not work directly on `main` unless the user explicitly asks for a direct
`main` change. Use this branch flow instead:

1. Make sure a `stage` branch exists and is based on the current `main`.
2. Start each task from `stage`.
3. Create a task branch from `stage` using a scoped prefix:
   - `feature/<what>` for new behavior,
   - `fix/<what>` for bug fixes,
   - `refactor/<what>` for structure-only changes,
   - `docs/<what>` for documentation-only changes,
   - `chore/<what>` for scripts, CI, or maintenance changes.
4. Do the work on that task branch.
5. When the task is complete and verified, merge the task branch back into
   `stage`.
6. Push `stage` when the user asks to publish the staged work.

Only merge `stage` into `main`, or push `main`, when the user explicitly says it
is time to put `stage` into `main`.

If the working tree already has uncommitted user changes, do not overwrite or
move them silently. Inspect the status, keep unrelated changes intact, and ask
for direction only when the branch operation cannot be done safely.

## Commit Rules

When creating a git commit, follow this commit message convention.

Write the first line as `type : summary`.

Examples of valid first lines:

- `feat : KiteVirtualMachine reconcile 골격 추가`
- `fix : KiteVirtualMachine disk 필드 매핑 수정`
- `docs : 커밋 메시지 규칙 추가`

After the first line, add a detailed commit body in an outline format. Use
short section labels and bullet points instead of a single long paragraph.

The body should explain:

- what problem the change solves,
- which files or flows were changed,
- how the change was implemented,
- how the change follows the project coding style or existing structure,
- which tests were run, or why tests were not run.

Example:

```text
feat : KiteVirtualMachine reconcile 골격 추가

Problem
- KiteVirtualMachine CRD 변경을 실제 KubeVirt 리소스로 조정하는 흐름이 없었다.

Changes
- KiteVirtualMachine informer 이벤트에서 reconcile 함수를 호출하도록 정리했다.
- VM 관련 리소스를 idempotent하게 적용하기 위한 GVR 매핑을 추가했다.

Implementation
- API 서버는 CRD만 기록하고, 컨트롤러가 CRD spec을 기준으로 실제 상태를 맞춘다.
- gRPC 명령형 흐름은 사용하지 않았다.

Tests
- 테스트는 요청 범위에 포함되지 않아 실행하지 않았다.
```

## Editing Rules

Keep changes scoped to the user request.

Do not run Go compile, tests, `protoc`, dependency installation, or Kubernetes
commands unless the user asks for them or they are required for the requested
task.

Do not rewrite unrelated files or refactor unrelated code while making a focused
change.
