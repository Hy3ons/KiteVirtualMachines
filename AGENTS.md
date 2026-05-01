# AGENTS.md

## Project Overview

This repository contains the Kite Kubernetes control plane prototype.

- `kite/cmd/kite-api`: HTTP API server based on Gin.
- `kite/cmd/kite-controller`: Kubernetes controller code and future gRPC server code.
- `kite/api/proto`: protobuf definitions for controller-facing APIs.
- `kite/api/v1`: Go structs for Kite custom resources.
- `kite/internal/kube`: Kubernetes client helpers.
- `kite/internal/render`: YAML template renderers that return `unstructured.Unstructured`.
- `custom`: Kite CRD definitions and example custom resources.

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
handlers, validators, renderers, controller reconcilers, and gRPC methods.

Avoid comments that only repeat the code. Comments should explain intent,
parameters, project usage, or non-obvious behavior.

## Protobuf Rules

Keep protobuf files focused on API contracts.

For `kite/api/proto/resource/resource.proto`, define only the information needed
to create the Kite custom resources in `custom/` unless a task explicitly asks
for more.

Current custom resources:

- `KiteUser`
  - CRD file: `custom/kite-user-crd.yaml`
  - scope: `Cluster`
  - resource: `kiteusers`
  - group/version: `anacnu.com/v1`
- `KiteVirtualMachine`
  - CRD file: `custom/kite-machine-crd.yaml`
  - scope: `Namespaced`
  - resource: `kitevirtualmachines`
  - group/version: `anacnu.com/v1`

Add English field comments in proto files when the field maps to Kubernetes
metadata or a CRD `spec` field.

Do not run `protoc` or commit generated protobuf files unless the user
explicitly asks for code generation.

## Kubernetes Rules

Use `dynamic.Interface` and `unstructured.Unstructured` for Kite custom
resources unless the task explicitly asks for typed clients.

When creating Kite custom resources, use these GVR values:

```go
schema.GroupVersionResource{
	Group:    "anacnu.com",
	Version:  "v1",
	Resource: "kiteusers",
}
```

```go
schema.GroupVersionResource{
	Group:    "anacnu.com",
	Version:  "v1",
	Resource: "kitevirtualmachines",
}
```

Do not call `.Namespace(...)` for `KiteUser` because it is cluster-scoped.

Call `.Namespace(req.Namespace)` for `KiteVirtualMachine` because it is
namespaced.

## Controller gRPC Rules

The controller gRPC server belongs under:

- `kite/cmd/kite-controller/apps/gRPC-server.go`

Do not add generated-code imports until protobuf Go files actually exist.

When implementing the gRPC server later:

- inject `*kube.ClientManager` or `dynamic.Interface` into the server struct,
- validate request fields before creating Kubernetes objects,
- map invalid requests to `codes.InvalidArgument`,
- map existing resources to `codes.AlreadyExists`,
- map Kubernetes API failures to `codes.Internal`,
- return created resource metadata in the response.

## Commit Rules

When creating a git commit, follow this commit message convention.

Write the first line as `type : summary`.

Examples of valid first lines:

- `feat : gRPC 서버 기본 포트 추가`
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
feat : gRPC 서버 기본 포트 추가

Problem
- 컨트롤러가 API 서버의 gRPC 요청을 받을 기본 진입점이 없었다.

Changes
- 50051 포트를 기본값으로 사용하는 gRPC 서버 실행 함수를 추가했다.
- context가 종료되면 GracefulStop으로 서버를 종료하도록 구성했다.

Implementation
- 아직 protobuf generated service가 없으므로 서비스 등록은 추가하지 않았다.
- 기존 apps 패키지 구조에 맞춰 서버 골격만 작성했다.

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
