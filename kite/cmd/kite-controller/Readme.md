# Kite Controller

Kite Controller는 Kite CRD를 감시하고 실제 Kubernetes/KubeVirt 리소스를 원하는 상태로 맞추는 프로세스입니다.

컨트롤러의 핵심 책임은 Kubernetes API server에 기록된 Kite CRD를 보고 실제 클러스터 상태를 맞추고, 그 결과를 다시 Kite CRD status에 기록하는 것입니다. 사용자가 직접 KubeVirt 리소스를 알 필요 없이 KiteUser와 KiteVirtualMachine만 다루면 되게 만드는 것이 목표입니다.

## 컨트롤러의 역할

- KiteUser CRD를 감시합니다.
- KiteVirtualMachine CRD를 감시합니다.
- 사용자별 네임스페이스와 기본 정책을 생성합니다.
- VM에 필요한 KubeVirt, 스토리지, 네트워크 리소스를 생성합니다.
- 실제 VM 상태를 읽고 KiteVirtualMachine status를 업데이트합니다.
- 삭제된 Kite CRD에 맞춰 연결 리소스를 정리합니다.
- API 서버의 직접 명령을 처리하지 않고, CRD 변경을 기준으로 reconcile합니다.

## 현재 상태

- Kubernetes 클러스터 연결 코드가 있습니다.
- `kitevirtualmachines.anacnu.com/v1` 리소스를 감시하는 informer 초안이 있습니다.
- gRPC 서버 초안이 있지만 현재 핵심 흐름에서는 사용하지 않습니다.
- 실제 reconcile 로직은 아직 대부분 구현해야 합니다.

## 기본 동작 방향

Kite Controller는 Kubernetes controller답게 동작해야 합니다. API 서버가 컨트롤러에 “VM을 만들어라”라고 직접 명령하지 않습니다. API 서버는 KiteVirtualMachine CRD를 만들고, 컨트롤러는 그 desired state를 보고 실제 리소스를 맞춥니다.

기본 흐름:

1. kite-api가 KiteUser 또는 KiteVirtualMachine CRD를 생성/수정/삭제한다.
2. kite-controller가 informer로 CRD 변경을 감지한다.
3. kite-controller가 현재 클러스터 상태를 조회한다.
4. 원하는 상태와 실제 상태의 차이를 계산한다.
5. 필요한 Kubernetes/KubeVirt 리소스를 생성, 수정, 삭제한다.
6. 처리 결과를 Kite CRD status에 기록한다.

이 흐름은 컨트롤러가 재시작되거나 이벤트를 놓쳐도 다시 list/watch를 통해 상태를 복구할 수 있게 해줍니다.

## 구현해야 할 reconcile 흐름

### KiteUser reconcile

- [ ] KiteUser가 생성되면 `spec.namespace` 네임스페이스를 만든다.
- [ ] 네임스페이스에 quota policy를 적용한다.
- [ ] 네임스페이스에 network policy를 적용한다.
- [ ] 사용자 기본 secret/config가 필요하면 생성한다.
- [ ] KiteUser가 수정되면 권한, 이미지, 사용자 정보 변경을 반영한다.
- [ ] KiteUser가 삭제되면 연결된 리소스 정리 정책을 실행한다.

### KiteVirtualMachine reconcile

- [ ] KiteVirtualMachine이 생성되면 KubeVirt VirtualMachine을 만든다.
- [ ] VM 디스크를 위한 DataVolume을 만든다.
- [ ] VM 접속을 위한 Service를 만든다.
- [ ] 외부 접근이 필요하면 Ingress를 만든다.
- [ ] `spec.powerState`가 바뀌면 VM 전원 상태를 맞춘다.
- [ ] 실제 KubeVirt 상태를 읽어 `status.phase`에 반영한다.
- [ ] 실패 원인을 `status.conditions`에 남긴다.
- [ ] KiteVirtualMachine이 삭제되면 관련 리소스를 정리한다.

## gRPC 서버 보류

컨트롤러 gRPC 서버는 현재 핵심 구현 대상에서 제외합니다. Kite의 기본 흐름은 `HTTP API -> Kite CRD -> controller reconcile`입니다.

따라서 지금은 아래 작업을 하지 않습니다.

- [ ] API 서버에서 컨트롤러 gRPC 서버로 CRD 생성 요청을 보내지 않는다.
- [ ] 컨트롤러를 명령형 RPC 서버처럼 사용하지 않는다.
- [ ] protobuf Go 코드 생성은 필요해질 때까지 보류한다.

나중에 CRD spec/status로 표현하기 어려운 내부 명령이 생기면 gRPC 사용을 다시 검토합니다.

## 렌더러 사용 계획

`kite/internal/render`에는 실제 Kubernetes 리소스를 만들기 위한 YAML 템플릿 렌더러가 있습니다. 컨트롤러는 reconcile 과정에서 이 렌더러들을 사용해야 합니다.

- Namespace 렌더러: 사용자별 네임스페이스 생성
- QuotaPolicy 렌더러: 사용자 네임스페이스 자원 제한
- NetworkPolicy 렌더러: 사용자 네임스페이스 네트워크 정책
- KubeVirtMachine 렌더러: 실제 VM 생성
- DataVolume 렌더러: VM 디스크 생성
- VM Service 렌더러: VM 접속용 Service 생성
- Ingress 렌더러: 외부 접속 경로 생성
- Cloud-init 렌더러: VM 초기 설정 생성

## 컨트롤러 TODO

- [ ] informer 이벤트 처리 코드를 reconcile 함수 중심으로 정리한다.
- [ ] Add, Update, Delete 이벤트에서 같은 reconcile 경로를 타도록 만든다.
- [ ] 이미 존재하는 리소스는 업데이트하거나 그대로 둔다.
- [ ] KiteUser와 KiteVirtualMachine informer를 모두 구성한다.
- [ ] reconcile 입력은 HTTP 요청이 아니라 CRD의 namespace/name으로 받는다.
- [ ] API 서버가 만든 Kite CRD를 기준으로만 실제 리소스를 생성한다.
- [ ] owner reference 또는 label 전략을 정해 관련 리소스를 찾기 쉽게 만든다.
- [ ] status update와 spec 처리 로직을 분리한다.
- [ ] 삭제 처리용 finalizer가 필요한지 결정한다.
- [ ] 로그 메시지를 한국어/영어 중 하나로 통일한다.
- [ ] 컨트롤러 재시작 후에도 기존 리소스를 다시 맞출 수 있게 한다.

## 주의할 점

- KiteUser는 cluster-scoped 리소스이므로 namespace를 붙여 만들면 안 됩니다.
- KiteVirtualMachine은 namespaced 리소스이므로 반드시 요청 namespace 안에서 다뤄야 합니다.
- KubeVirt 리소스 생성은 중복 실행되어도 같은 결과가 나오도록 만들어야 합니다.
- status는 사용자가 입력하는 값이 아니라 컨트롤러가 관리하는 값입니다.
