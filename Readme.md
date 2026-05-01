# Kite

Kite는 단일 Kubernetes 클러스터 위에서 사용자별 가상 머신을 쉽게 제공하기 위한 컨트롤 플레인 프로토타입입니다.

최종 목표는 사용자가 웹 화면이나 API를 통해 VM을 요청하면 Kite가 사용자, 네임스페이스, 권한, VM, 네트워크, 스토리지 리소스를 일관되게 만들고 관리하는 것입니다. 내부적으로는 Kite 전용 CRD를 중심으로 상태를 기록하고, 컨트롤러가 Kubernetes와 KubeVirt 리소스를 실제 상태로 맞춥니다.

## 현재 방향

이 저장소는 아직 완성된 제품이 아니라 구현 방향을 잡아가는 단계입니다. 지금 중요한 것은 기능을 많이 붙이는 것보다, 아래 흐름을 안정적으로 완성하는 것입니다.

1. 사용자를 KiteUser CRD로 표현합니다.
2. 사용자의 VM 요청을 KiteVirtualMachine CRD로 표현합니다.
3. API 서버는 인증과 사용자 요청을 받아 CRD 생성을 요청합니다.
4. 컨트롤러는 CRD를 감시하고 필요한 Kubernetes/KubeVirt 리소스를 만듭니다.
5. 컨트롤러는 실제 리소스 상태를 다시 CRD status에 반영합니다.
6. 프론트엔드는 사용자와 관리자가 위 흐름을 쉽게 사용할 수 있게 만듭니다.

## 저장소 구성

- `kite/cmd/kite-api`: Gin 기반 HTTP API 서버입니다.
- `kite/cmd/kite-controller`: Kite CRD를 감시하고 실제 클러스터 리소스를 맞추는 컨트롤러입니다.
- `kite/api/proto`: API 서버와 컨트롤러 사이에서 사용할 gRPC 계약입니다.
- `kite/api/v1`: Kite CRD를 Go 구조체로 다루기 위한 타입입니다.
- `kite/internal/render`: Namespace, KubeVirt VM, DataVolume, Service, Ingress, NetworkPolicy, QuotaPolicy 등을 만들기 위한 YAML 렌더러입니다.
- `custom`: Kite CRD 정의와 예시 리소스입니다.
- `kite-yaml`: 클러스터에 올릴 기본 매니페스트와 골든 이미지 정의입니다.
- `kite-frontend`: 앞으로 구현할 웹 프론트엔드 영역입니다.

## 구현해야 할 큰 흐름

### 1. 사용자 관리

- KiteUser CRD 생성 흐름을 확정합니다.
- 사용자가 생성되면 사용자 전용 네임스페이스를 만듭니다.
- 사용자 네임스페이스에 quota, network policy, 기본 secret/config를 적용합니다.
- 사용자가 삭제되면 연결된 네임스페이스와 VM 리소스를 어떻게 정리할지 정책을 정합니다.
- 비밀번호 저장 방식, 프로필 이미지 저장 방식, 접근 권한 단계를 확정합니다.

### 2. VM 관리

- KiteVirtualMachine CRD 생성 흐름을 확정합니다.
- VM 요청이 들어오면 KubeVirt VirtualMachine, DataVolume, Service, Ingress를 생성합니다.
- VM 시작/중지 요청을 `spec.powerState` 중심으로 정리합니다.
- 실제 KubeVirt 상태를 읽어 `status.phase`, `status.currentPowerState`, `status.conditions`에 반영합니다.
- VM 삭제 시 연결된 스토리지와 네트워크 리소스를 함께 정리합니다.

### 3. API 서버

- 임시 관리자 로그인 방식을 실제 사용자 인증 흐름으로 바꿉니다.
- 사용자 생성, 조회, 수정, 삭제 API를 구현합니다.
- VM 생성, 조회, 수정, 삭제, 시작, 중지 API를 구현합니다.
- API 서버가 직접 CRD를 만들지, 컨트롤러 gRPC 서버에 요청할지 경계를 정리합니다.
- 에러 응답 형식과 권한 검사를 일관되게 만듭니다.

### 4. 컨트롤러

- KiteUser reconcile을 구현합니다.
- KiteVirtualMachine reconcile을 구현합니다.
- gRPC 서버에서 CRD 생성 요청을 받을 수 있게 만듭니다.
- 이미 존재하는 리소스, 잘못된 요청, Kubernetes API 오류를 명확하게 처리합니다.
- 재시작해도 같은 리소스를 중복 생성하지 않도록 idempotent하게 만듭니다.

### 5. 프론트엔드

- 로그인 화면을 만듭니다.
- 관리자용 사용자 목록/생성/수정/삭제 화면을 만듭니다.
- 사용자용 VM 목록/생성/상세/전원 제어 화면을 만듭니다.
- VM 상태, 에러, 생성 진행 상황을 사람이 이해하기 쉽게 보여줍니다.
- API 권한 단계에 따라 보이는 메뉴와 가능한 동작을 나눕니다.

## 우선순위 TODO

### 1차 목표: CRD 기반 최소 동작 완성

- [ ] KiteUser 생성 시 네임스페이스를 자동 생성한다.
- [ ] KiteUser 삭제 시 정리 정책을 정하고 구현한다.
- [ ] KiteVirtualMachine 생성 시 KubeVirt VM 관련 리소스를 생성한다.
- [ ] KiteVirtualMachine status를 실제 KubeVirt 상태와 동기화한다.
- [ ] API 서버에서 사용자와 VM을 만들 수 있는 최소 API를 제공한다.
- [ ] 컨트롤러가 재시작되어도 기존 CRD를 다시 읽고 상태를 맞춘다.

### 2차 목표: API와 권한 정리

- [ ] 임시 관리자 계정 기반 로그인을 실제 KiteUser 기반 로그인으로 바꾼다.
- [ ] 사용자 권한 단계를 read-only, user, manager, admin 기준으로 정리한다.
- [ ] 모든 API에 권한 검사를 적용한다.
- [ ] API 응답 형식과 에러 메시지를 통일한다.
- [ ] gRPC 계약을 실제 컨트롤러 구현과 맞춘다.

### 3차 목표: 운영에 필요한 안정성

- [ ] VM 생성 실패 시 어떤 단계에서 실패했는지 status에 남긴다.
- [ ] 리소스 생성/삭제 작업을 재시도 가능하게 만든다.
- [ ] 로그 형식을 정리한다.
- [ ] 기본 테스트를 추가한다.
- [ ] 배포용 매니페스트를 정리한다.

### 4차 목표: 사용자 경험

- [ ] 프론트엔드 프로젝트 구조를 만든다.
- [ ] 로그인과 토큰 저장 흐름을 구현한다.
- [ ] 관리자 화면과 사용자 화면을 나눈다.
- [ ] VM 생성 폼에서 CPU, 메모리, 디스크, 이미지 선택을 지원한다.
- [ ] VM 상태와 접속 정보를 화면에서 확인할 수 있게 한다.

## 아직 결정해야 할 것

- API 서버가 CRD를 직접 생성할지, 컨트롤러 gRPC 서버를 통해서만 생성할지 결정해야 합니다.
- etcd를 직접 저장소로 사용할지, Kubernetes CRD를 사실상의 상태 저장소로 사용할지 역할을 나눠야 합니다.
- 사용자 삭제 시 VM과 디스크를 즉시 삭제할지, 보존 기간을 둘지 정해야 합니다.
- VM 접속 방식이 SSH, VNC, 웹 콘솔 중 무엇을 우선할지 정해야 합니다.
- 골든 이미지 관리 방식을 수동 YAML로 둘지, Kite API에서 관리할지 정해야 합니다.

## 문서 작성 기준

이 README는 세부 구현법보다 “무엇을 만들어야 하는지”를 정리하는 문서입니다. 실제 명령어, API 상세, 배포 방법은 각 하위 디렉터리 README에서 필요한 만큼만 관리합니다.
