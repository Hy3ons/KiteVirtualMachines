# Kite API Specifications (Draft)

프론트엔드와 백엔드(`kite-api`) 간의 통신을 위해 구현되어야 할 REST API 엔드포인트 명세서입니다.
현재 프론트엔드의 Mock 데이터 로직은 실제 개발 시 이 명세서의 API들로 교체되어야 합니다.

## 1. 인증 및 사용자 (Auth & User)

### 1-1. 로그인 (`POST /api/v1/auth/login`)
- **Request Body**: `{ "username": "admin", "password": "password123" }`
- **Response**: JWT Token, 해당 유저의 `access_level` 및 `namespace` 반환.

### 1-2. 회원가입 (`POST /api/v1/auth/signup`)
- **Request Body**: `{ "username": "...", "email": "...", "password": "...", "profile_image": "data:image/png;base64,..." }`
- **Description**: `profile_image`는 Base64 인코딩된 문자열을 받으며 필수가 아닙니다 (빈 값일 경우 기본 프로필 적용).
- **Response**: `KiteUser` 리소스 생성 (기본 `access_level: 1` 할당).

## 2. 사용자 대시보드 (User Dashboard)

### 2-1. 내 가상머신 목록 조회 (`GET /api/v1/vms`)
- **Description**: 접속한 유저의 네임스페이스에 속한 `KiteVirtualMachine` 목록과 상세 정보를 반환합니다.
- **Response**: 
  ```json
  [
    {
      "id": "uuid",
      "name": "dev-vm-1",
      "domain": "dev.anacnu.com",
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
- **Description**: 백엔드는 디스크 용량이 Quota 내에 있는지 검증 후 CR을 배포합니다.

### 2-3. 가상머신 상태 제어 (`PATCH /api/v1/vms/:name/power`)
- **Request Body**: `{ "powerState": "On" | "Off" }`
- **Description**: `KiteVirtualMachine`의 spec.powerState를 수정하여 컨트롤러가 VM을 정지/시작하도록 유도합니다.

### 2-4. 가상머신 삭제 (`DELETE /api/v1/vms/:name`)
- **Description**: 가상머신 CR을 영구 삭제합니다.

## 3. 관리자 대시보드 (Admin Dashboard)

### 3-1. 전역 도메인 설정 (`POST /api/v1/admin/domain`)
- **Request Body**: `{ "baseDomain": "anacnu.com" }`
- **Description**: `etcd` 또는 K8s ConfigMap에 클러스터 베이스 도메인을 저장/수정합니다.

### 3-2. 와일드카드 인증서 갱신 (`POST /api/v1/admin/cert`)
- **Request Body**: `{ "tlsCert": "...", "tlsKey": "..." }`
- **Description**: 클러스터 내의 `global-secret` (TLS Secret)을 즉시 갱신합니다.

### 3-3. 전체 사용자 조회 (`GET /api/v1/admin/users`)
- **Description**: 전체 `KiteUser` 목록을 반환합니다.

### 3-4. 사용자 권한 변경 (`PATCH /api/v1/admin/users/:username/level`)
- **Request Body**: `{ "accessLevel": 3 }`
- **Description**: 유저의 레벨을 강등시키거나 관리자로 승급시킵니다.

### 3-5. 사용자 영구 삭제 (`DELETE /api/v1/admin/users/:username`)
- **Description**: 악성 유저의 네임스페이스와 모든 리소스(`KiteUser` 포함)를 완전히 날려버립니다.

### 3-6. 전체 가상머신 조회 (`GET /api/v1/admin/vms`)
- **Description**: 네임스페이스 구분 없이 클러스터에 존재하는 모든 가상머신을 반환합니다. 소유자(Namespace) 정보가 포함되어야 합니다.

### 3-7. 가상머신 강제 제어 (`PATCH & DELETE /api/v1/admin/vms/:namespace/:name`)
- **Description**: 관리자 권한으로 타 유저 네임스페이스의 VM을 강제 종료(Force Stop)하거나 강제 삭제(Force Delete)합니다.
