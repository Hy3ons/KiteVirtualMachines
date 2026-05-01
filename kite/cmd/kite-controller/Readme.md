# Kite Controller

Kite Controller는 Kite CRD를 감시하고 실제 Kubernetes/KubeVirt 리소스를 원하는 상태로 맞추는 프로세스입니다.

컨트롤러의 핵심 책임은 API 서버가 만든 요청을 실제 클러스터 상태로 바꾸고, 그 결과를 다시 Kite CRD status에 기록하는 것입니다. 사용자가 직접 KubeVirt 리소스를 알 필요 없이 KiteUser와 KiteVirtualMachine만 다루면 되게 만드는 것이 목표입니다.

## 컨트롤러의 역할

- KiteUser CRD를 감시합니다.
- KiteVirtualMachine CRD를 감시합니다.
- 사용자별 네임스페이스와 기본 정책을 생성합니다.
- VM에 필요한 KubeVirt, 스토리지, 네트워크 리소스를 생성합니다.
- 실제 VM 상태를 읽고 KiteVirtualMachine status를 업데이트합니다.
- 삭제된 Kite CRD에 맞춰 연결 리소스를 정리합니다.
- API 서버가 사용할 gRPC 서버를 제공합니다.

## 현재 상태

- Kubernetes 클러스터 연결 코드가 있습니다.
- `kitevirtualmachines.anacnu.com/v1` 리소스를 감시하는 informer 초안이 있습니다.
- gRPC 서버를 띄우기 위한 기본 서버 코드가 있습니다.
- 실제 reconcile 로직은 아직 대부분 구현해야 합니다.

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

## gRPC 서버

컨트롤러 gRPC 서버는 API 서버가 CRD 생성 요청을 보낼 수 있는 내부 인터페이스입니다.

구현해야 할 것:

- [ ] protobuf Go 코드가 생성된 뒤 서비스 구현체를 추가한다.
- [ ] `CreateKiteUser` 요청을 검증한다.
- [ ] `CreateKiteVirtualMachine` 요청을 검증한다.
- [ ] KiteUser는 cluster-scoped 리소스로 생성한다.
- [ ] KiteVirtualMachine은 요청 namespace 안에 생성한다.
- [ ] 이미 존재하는 리소스는 AlreadyExists로 응답한다.
- [ ] 잘못된 요청은 InvalidArgument로 응답한다.
- [ ] Kubernetes API 오류는 Internal로 응답한다.
- [ ] 생성된 리소스의 apiVersion, kind, name, namespace를 응답한다.

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
