# Kite API

Kite API는 사용자와 프론트엔드가 Kite 기능을 사용하기 위해 접근하는 HTTP 서버입니다. 현재는 Gin 기반 서버, 임시 관리자 로그인, JWT 발급, KiteUser 목록 조회가 들어가 있습니다.

앞으로는 단순한 HTTP 엔드포인트 모음이 아니라, 사용자 요청을 검증하고 권한을 확인한 뒤 Kite CRD를 생성/수정/삭제하는 입구 역할을 해야 합니다. API 서버는 KubeVirt 리소스를 직접 만들지 않습니다.

## API 서버의 역할

- 로그인 요청을 받고 access token을 발급합니다.
- 사용자의 권한을 확인합니다.
- 사용자 관리 API를 제공합니다.
- VM 관리 API를 제공합니다.
- 요청 값이 Kite CRD 규칙에 맞는지 검증합니다.
- Kubernetes API server에 KiteUser와 KiteVirtualMachine CRD를 기록합니다.
- 프론트엔드가 이해하기 쉬운 응답과 에러 메시지를 반환합니다.

## 현재 구현된 것

- `GET /health`: 서버 상태 확인
- `POST /api/login`: 임시 관리자 계정으로 로그인
- `GET /api/users`: manager 이상 권한으로 KiteUser 목록 조회
- JWT access token 발급과 검증
- Kubernetes dynamic client 연결

## 먼저 구현할 API

### 인증

- [ ] KiteUser 기반 로그인으로 전환한다.
- [ ] 비밀번호 해시 검증 방식을 정한다.
- [ ] 토큰 만료, 재발급, 로그아웃 정책을 정한다.
- [ ] 쿠키와 Authorization 헤더 사용 방식을 정리한다.

### 사용자 API

- [ ] 사용자 생성 API에서 KiteUser CRD를 생성한다.
- [ ] 사용자 단건 조회 API에서 KiteUser CRD를 조회한다.
- [ ] 사용자 수정 API에서 KiteUser CRD spec을 수정한다.
- [ ] 사용자 삭제 API에서 KiteUser CRD를 삭제한다.
- [ ] 사용자 삭제 시 네임스페이스와 VM 정리 정책을 API 응답에 반영한다.

### VM API

- [ ] VM 생성 API에서 KiteVirtualMachine CRD를 생성한다.
- [ ] VM 목록 조회 API에서 KiteVirtualMachine CRD를 조회한다.
- [ ] VM 상세 조회 API에서 KiteVirtualMachine CRD와 status를 반환한다.
- [ ] VM 수정 API에서 KiteVirtualMachine CRD spec을 수정한다.
- [ ] VM 삭제 API에서 KiteVirtualMachine CRD를 삭제한다.
- [ ] VM 시작/중지 API에서 `spec.powerState`를 수정한다.
- [ ] VM 접속 정보 조회 API를 만든다.

## 권한 기준

권한 단계는 현재 코드와 CRD 기준으로 아래 값을 사용합니다.

- `0`: read-only
- `1`: user
- `2`: manager
- `3`: admin

정리해야 할 기준은 다음과 같습니다.

- [ ] read-only가 볼 수 있는 범위를 정한다.
- [ ] user가 자기 VM만 다룰 수 있게 한다.
- [ ] manager가 사용자와 VM을 어디까지 관리할 수 있는지 정한다.
- [ ] admin 전용 작업을 분리한다.
- [ ] 권한 부족, 인증 실패, 토큰 만료 응답을 통일한다.

## 컨트롤러와의 연결 방향

Kite는 Kubernetes의 선언형 API와 controller reconcile 패턴을 따릅니다. 따라서 API 서버는 컨트롤러에 gRPC 명령을 보내지 않고, Kubernetes API server에 Kite CRD를 기록합니다. 컨트롤러는 그 CRD를 watch하고 실제 클러스터 상태를 맞춥니다.

기본 흐름:

1. 프론트엔드가 kite-api에 HTTP 요청을 보낸다.
2. kite-api가 인증, 권한, 요청 validation을 처리한다.
3. kite-api가 KiteUser 또는 KiteVirtualMachine CRD를 생성/수정/삭제한다.
4. kite-controller가 CRD 변경을 감지한다.
5. kite-controller가 Namespace, KubeVirt VM, DataVolume, Service, Ingress 같은 실제 리소스를 reconcile한다.
6. kite-controller가 처리 결과를 CRD status에 기록한다.

이 구조에서 API 서버가 직접 만들 수 있는 것은 Kite CRD까지입니다. Namespace, KubeVirt VM, DataVolume, Service, Ingress, NetworkPolicy, QuotaPolicy는 컨트롤러가 만듭니다.

gRPC 서버는 현재 핵심 경로에서 제외합니다. 나중에 Kubernetes CRD로 표현하기 어려운 내부 명령이 생길 때 별도 검토합니다.

## API 서버 TODO

- [ ] 요청/응답 구조체를 API별로 정리한다.
- [ ] validation 함수를 추가한다.
- [ ] 에러 응답 형식을 통일한다.
- [ ] 사용자 비밀번호가 목록 응답에 노출되지 않게 한다.
- [ ] KiteUser CRD 생성/조회/수정/삭제 helper를 만든다.
- [ ] KiteVirtualMachine CRD 생성/조회/수정/삭제 helper를 만든다.
- [ ] VM 전원 제어는 gRPC 호출이 아니라 `spec.powerState` 수정으로 처리한다.
- [ ] Kubernetes API 실패를 사용자에게 보여줄 메시지와 내부 로그로 나눈다.
- [ ] 프론트엔드가 사용할 API 명세를 README 또는 별도 문서로 정리한다.
- [ ] 인증/권한 테스트를 보강한다.
