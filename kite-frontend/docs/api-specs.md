# Kite API Specifications

프론트엔드와 백엔드(`kite-api`) 간의 REST API 엔드포인트 명세서입니다.
기본 경로는 `/api/v1`입니다.

## 1. 인증 및 사용자 (Auth & User)

### 1-1. 로그인 (`POST /api/v1/auth/login`)
- **Request Body**: `{ "email": "admin@example.com", "password": "password123" }`
- **Response**: JWT access token, token metadata, 해당 유저의 `accessLevel` 및 `namespace` 반환.
- **Cookie**: `accessToken` HttpOnly/Secure cookie도 함께 설정합니다.

### 1-2. 회원가입 (`POST /api/v1/auth/signup`)
- **Request Body**: `{ "username": "...", "email": "...", "password": "..." }`
- **Description**: 첫 가입자는 `access_level: 3`, 이후 가입자는 `access_level: 0`으로 `KiteUser` CRD를 생성합니다.
- **Response**: password hash를 제외한 `KiteUser` public response를 반환합니다.

## 2. 사용자 대시보드 (User Dashboard)

### 2-1. 내 가상머신 목록 조회 (`GET /api/v1/vms`)
- **Description**: 접속한 유저의 네임스페이스에 속한 `KiteVirtualMachine` 목록과 상세 정보를 반환합니다.
- **Response**: 
  ```json
  [
    {
      "id": "uuid",
      "name": "dev-vm-1",
      "domain": "dev.hy3ons.github.io",
      "phase": "Running",
      "cpu": 2,
      "memory": "4Gi",
      "disk": "25Gi",
      "sshId": "ubuntu"
    }
  ]
  ```

### 2-2. 가상머신 생성 (`POST /api/v1/vms`)
- **Request Body**: `{ "name": "...", "domainPrefix": "...", "sshId": "...", "sshPassword": "...", "disk": 25 }`
- **Description**: 백엔드는 디스크 용량과 전역 유일 `sshId`를 검증한 뒤 `KiteVirtualMachine` CRD를 생성합니다. 요청의 `sshPassword`는 CRD에 평문 저장하지 않고 `spec.sshPasswordHash`로 저장합니다.

### 2-3. 가상머신 수정 (`PATCH /api/v1/vms/:name`)
- **Request Body**: `{ "domainPrefix": "...", "sshPassword": "...", "powerState": "On" | "Off" }`
- **Description**: 전달된 필드만 `KiteVirtualMachine.spec`에 반영합니다. `sshPassword`는 전달된 경우 새 hash로 저장합니다.

### 2-4. 가상머신 시작 (`POST /api/v1/vms/:name/start`)
- **Description**: `KiteVirtualMachine.spec.powerState`를 `On`으로 수정합니다.

### 2-5. 가상머신 중지 (`POST /api/v1/vms/:name/stop`)
- **Description**: `KiteVirtualMachine.spec.powerState`를 `Off`로 수정합니다.

### 2-6. 가상머신 삭제 (`DELETE /api/v1/vms/:name`)
- **Description**: `KiteVirtualMachine.spec.delete=true`를 설정합니다. 실제 KubeVirt VM, DataVolume, Service, Secret 정리는 controller finalizer 흐름이 처리합니다.

## 3. 관리자 대시보드 (Admin Dashboard)

### 3-1. 전역 도메인 설정 (`POST /api/v1/admin/domain`)
- **Request Body**: `{ "baseDomain": "hy3ons.github.io" }`
- **Description**: `kite/kite-runtime-config` ConfigMap에 클러스터 베이스 도메인을 저장/수정합니다.

### 3-2. 와일드카드 인증서 갱신 (`POST /api/v1/admin/cert`)
- **Request Body**: `{ "tlsCert": "...", "tlsKey": "..." }`
- **Description**: `kite/global-tls-secret` TLS Secret을 즉시 생성/수정합니다.

### 3-3. 전체 사용자 조회 (`GET /api/v1/admin/users`)
- **Description**: 전체 `KiteUser` 목록을 반환합니다.

### 3-4. 사용자 권한 변경 (`PATCH /api/v1/admin/users/:name/access-level`)
- **Request Body**: `{ "access_level": 3 }`
- **Description**: 유저의 레벨을 강등시키거나 관리자로 승급시킵니다.

### 3-5. 사용자 영구 삭제 (`DELETE /api/v1/admin/users/:name`)
- **Description**: 사용자의 namespace에 있는 `KiteVirtualMachine` CRD를 먼저 삭제한 뒤 `KiteUser` CRD를 삭제합니다.

### 3-6. 전체 가상머신 조회 (`GET /api/v1/admin/vms`)
- **Description**: 네임스페이스 구분 없이 클러스터에 존재하는 모든 가상머신을 반환합니다. 소유자(Namespace) 정보가 포함되어야 합니다.

### 3-7. 가상머신 전원 제어 (`PATCH /api/v1/admin/vms/:namespace/:name/power`)
- **Request Body**: `{ "powerState": "On" | "Off" }`
- **Description**: manager 이상 권한으로 타 유저 네임스페이스의 VM 전원 desired state를 수정합니다.

### 3-8. 가상머신 삭제 (`DELETE /api/v1/admin/vms/:namespace/:name`)
- **Description**: manager 이상 권한으로 타 유저 네임스페이스의 VM에 `spec.delete=true`를 설정합니다.
