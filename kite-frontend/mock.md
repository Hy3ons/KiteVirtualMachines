# Kite Frontend API Contract

프론트엔드는 더 이상 `setTimeout` 기반 mock 로직에 의존하지 않고, 아래 `/api/v1/...` 엔드포인트를 호출하면 된다.

공통 규칙:

- 인증이 필요한 요청은 `Authorization: Bearer <accessToken>` 헤더를 붙인다.
- Access level `0`, `1` 사용자는 `/api/v1/vms...`로 자기 namespace의 VM만 조회/생성/수정/삭제할 수 있다.
- Access level `2`, `3` 사용자는 `/api/v1/admin/vms...`로 다른 사용자의 VM까지 조회/전원 변경/삭제할 수 있다.
- 사용자 권한 변경, 사용자 삭제, 시스템 전역 설정은 access level `3`만 가능하다.
- 응답은 JSON이다.
- 에러 응답은 기본적으로 `{ "message": "..." }` 형식이다.
- VM 삭제는 CRD를 즉시 삭제하지 않고 `spec.delete=true`로 변경한다. 실제 KubeVirt VM, Service, Ingress, Secret, DataVolume 정리는 `kite-controller`가 수행한다.

## Health

### `GET /api/v1/health`

Kubernetes dynamic client, Kite CRD 접근 상태를 확인한다.

Response:

```json
{
  "status": "ok",
  "checks": []
}
```

## Auth

### `POST /api/v1/auth/login`

`LandingPage.tsx`의 로그인 mock을 이 호출로 교체한다.

Request:

```json
{
  "email": "admin@example.com",
  "password": "admin"
}
```

Response `200`:

```json
{
  "accessToken": "jwt-token",
  "tokenType": "Bearer",
  "expiresIn": 3600,
  "expiresAt": "2026-06-04T12:00:00Z",
  "user": {
    "name": "ku-...",
    "username": "admin",
    "email": "admin@example.com",
    "namespace": "kite-user-ku-...",
    "profile_image": "base64encodedimage",
    "access_level": 3
  }
}
```

프론트 저장 매핑:

- `token`: `accessToken`
- `accessLevel`: `user.access_level`
- `username`: `user.username`
- `namespace`: `user.namespace`

### `POST /api/v1/auth/signup`

`SignupPage.tsx`의 회원가입 mock을 이 호출로 교체한다. 
**[중요]** 프론트엔드뿐만 아니라 백엔드의 `POST /api/v1/auth/signup` 컨트롤러에서도 `username`이 영문자(`^[a-zA-Z]+$`)로만 이루어져 있는지 반드시 2차 검증(Server-side Validation)을 수행해야 하며, 형식에 어긋날 경우 `400 Bad Request` 에러를 반환해야 한다.

Request:

```json
{
  "username": "test",
  "email": "test@gmail.com",
  "password": "password"
}
```

Response `201`:

```json
{
  "message": "kite user created successfully",
  "user": {
    "name": "ku-...",
    "username": "test",
    "email": "test@gmail.com",
    "namespace": "kite-user-ku-...",
    "profile_image": "base64encodedimage",
    "access_level": 0
  }
}
```

첫 가입자는 `access_level=3`, 이후 가입자는 `access_level=0`으로 생성된다.

### `GET /api/v1/me`

로그인 후 현재 사용자 프로필을 다시 읽는다.

Response `200`:

```json
{
  "user": {
    "name": "ku-...",
    "username": "test",
    "email": "test@gmail.com",
    "namespace": "kite-user-ku-...",
    "profile_image": "base64encodedimage",
    "access_level": 0
  }
}
```

## Config

### `GET /api/v1/config`

`MOCK_ENV.BASE_DOMAIN` 같은 전역 설정 mock을 이 응답으로 교체한다.

Response `200`:

```json
{
  "config": {
    "baseDomain": "apps.example.com",
    "hasJWTSecret": true,
    "hasPasswordSalt": true,
    "hasTLSCertificate": true
  }
}
```

## User VM Dashboard

### `GET /api/v1/vms`

`UserDashboard.tsx`의 `fetchVms` mock을 이 호출로 교체한다. 현재 로그인한 사용자의 namespace 안에 있는 VM만 반환한다.

Response `200`:

```json
{
  "vms": [
    {
      "id": "kite-user-ku-.../dev-vm-1",
      "name": "dev-vm-1",
      "namespace": "kite-user-ku-...",
      "owner": "kite-user-ku-...",
      "domain": "dev.apps.example.com",
      "phase": "Running",
      "powerState": "On",
      "currentPowerState": "On",
      "cpu": 2,
      "memory": "4Gi",
      "image": "ubuntu-22.04",
      "disk": "25Gi",
      "domainPrefix": "dev",
      "sshId": "ubuntu",
      "delete": false,
      "dataVolumePhase": "Succeeded",
      "dataVolumeProgress": "100.0%",
      "dataVolumeMessage": "DataVolume phase is Succeeded"
    }
  ]
}
```

`vps-access-<vmName>` SSH Service는 ClusterIP이며, `kite-gateway`가 VM SSH key Secret을 읽어 Kubernetes 내부에서 VM SSH 세션으로 프록시한다.

### `POST /api/v1/vms`

`UserDashboard.tsx`의 `handleCreate` mock을 이 호출로 교체한다.

Request:

```json
{
  "name": "my-ubuntu-vm",
  "domainPrefix": "my-web",
  "sshId": "ubuntu",
  "sshPassword": "Strong password",
  "disk": 25
}
```

`sshPassword`는 HTTP 요청에서만 사용한다. API는 이 값을 즉시 hash하여
`KiteVirtualMachine.spec.sshPasswordHash`에 저장하고, 응답과 VM 목록에는
password 계열 값을 포함하지 않는다.

선택 필드:

```json
{
  "cpu": 2,
  "memory": "4Gi",
  "image": "ubuntu-22.04",
  "powerState": "Off"
}
```

`disk`는 숫자면 API가 `"25Gi"`로 변환한다. 문자열 `"25Gi"`도 허용한다.

Response `201`:

```json
{
  "vm": {
    "id": "kite-user-ku-.../my-ubuntu-vm",
    "name": "my-ubuntu-vm",
    "namespace": "kite-user-ku-...",
    "owner": "kite-user-ku-...",
    "domain": "",
    "phase": "Creating",
    "powerState": "Off",
    "currentPowerState": "",
    "cpu": 2,
    "memory": "4Gi",
    "image": "ubuntu-22.04",
    "disk": "25Gi",
    "domainPrefix": "my-web",
    "sshId": "ubuntu",
    "delete": false,
    "dataVolumePhase": "",
    "dataVolumeProgress": "",
    "dataVolumeMessage": ""
  }
}
```

### `GET /api/v1/vms/{name}`

`VmDetail.tsx`에서 단일 VM 상세를 조회할 때 사용한다.

Response `200`:

```json
{
  "vm": {
    "id": "kite-user-ku-.../my-ubuntu-vm",
    "name": "my-ubuntu-vm",
    "namespace": "kite-user-ku-...",
    "owner": "kite-user-ku-...",
    "domain": "my-web.apps.example.com",
    "phase": "Running",
    "powerState": "On",
    "currentPowerState": "On",
    "cpu": 2,
    "memory": "4Gi",
    "image": "ubuntu-22.04",
    "disk": "25Gi",
    "domainPrefix": "my-web",
    "sshId": "ubuntu",
    "delete": false,
    "dataVolumePhase": "Succeeded",
    "dataVolumeProgress": "100.0%",
    "dataVolumeMessage": "DataVolume phase is Succeeded"
  }
}
```

### `PATCH /api/v1/vms/{name}`

VM spec 일부를 수정한다. 전원 제어도 `powerState` 패치로 가능하다.

Request:

```json
{
  "powerState": "On"
}
```

Response `200`:

```json
{
  "vm": {
    "name": "my-ubuntu-vm",
    "powerState": "On",
    "phase": "Stopped"
  }
}
```

### `POST /api/v1/vms/{name}/start`

`handleStart` mock을 이 호출로 교체한다. 내부적으로 `spec.powerState="On"`으로 바꾼다.

Response `200`:

```json
{
  "vm": {
    "name": "my-ubuntu-vm",
    "powerState": "On"
  }
}
```

### `POST /api/v1/vms/{name}/stop`

`handleStop` mock을 이 호출로 교체한다. 내부적으로 `spec.powerState="Off"`로 바꾼다.

Response `200`:

```json
{
  "vm": {
    "name": "my-ubuntu-vm",
    "powerState": "Off"
  }
}
```

### `DELETE /api/v1/vms/{name}`

`handleDelete` mock을 이 호출로 교체한다. 내부적으로 `spec.delete=true`를 기록한다.

Response `200`:

```json
{
  "message": "virtual machine delete requested",
  "vm": {
    "name": "my-ubuntu-vm",
    "delete": true
  }
}
```

## Admin Dashboard

### `GET /api/v1/admin/users`

`AdminDashboard.tsx`의 `fetchUsers` mock을 이 호출로 교체한다.

Required access level: `2` 이상.

Response `200`:

```json
{
  "users": [
    {
      "name": "ku-...",
      "username": "hyeonseok",
      "email": "hyeonseok@kite.com",
      "namespace": "kite-user-ku-...",
      "profile_image": "base64encodedimage",
      "access_level": 1
    }
  ]
}
```

프론트의 `status` 컬럼은 현재 별도 API 필드가 없으므로 우선 `Active`로 표시하면 된다. 추후 `KiteUser.status.phase`를 응답에 포함하도록 확장할 수 있다.

### `PATCH /api/v1/admin/users/{nameOrUsername}/access-level`

`handleChangeAccessLevel` mock을 이 호출로 교체한다.

Required access level: `3`.

Request:

```json
{
  "access_level": 2
}
```

Response `200`:

```json
{
  "user": {
    "name": "ku-...",
    "username": "hyeonseok",
    "email": "hyeonseok@kite.com",
    "namespace": "kite-user-ku-...",
    "profile_image": "base64encodedimage",
    "access_level": 2
  }
}
```

`{nameOrUsername}`에는 `KiteUser.metadata.name` 또는 `spec.username` 둘 다 넣을 수 있다.

### `DELETE /api/v1/admin/users/{nameOrUsername}`

`handleDeleteUser` mock을 이 호출로 교체한다.

Required access level: `3`.

Response `200`:

```json
{
  "message": "kite user deleted successfully"
}
```

삭제 시 해당 사용자 namespace의 `KiteVirtualMachine` CRD를 먼저 삭제 요청한다.

### `GET /api/v1/admin/vms`

`AdminDashboard.tsx`의 `fetchVms` mock을 이 호출로 교체한다. 모든 namespace의 VM을 반환한다.

Required access level: `2` 이상.

Response `200`:

```json
{
  "vms": [
    {
      "id": "kite-user-ku-.../dev-vm-1",
      "owner": "kite-user-ku-...",
      "namespace": "kite-user-ku-...",
      "name": "dev-vm-1",
      "domain": "dev.apps.example.com",
      "phase": "Running",
      "cpu": 2,
      "memory": "4Gi"
    }
  ]
}
```

### `PATCH /api/v1/admin/vms/{namespace}/{name}/power`

`handleForceStopVm` mock을 이 호출로 교체한다.

Required access level: `2` 이상.

Request:

```json
{
  "powerState": "Off"
}
```

Response `200`:

```json
{
  "vm": {
    "namespace": "kite-user-ku-...",
    "name": "dev-vm-1",
    "powerState": "Off"
  }
}
```

### `DELETE /api/v1/admin/vms/{namespace}/{name}`

`handleDeleteVm` mock을 이 호출로 교체한다.

Required access level: `2` 이상.

Response `200`:

```json
{
  "message": "virtual machine delete requested",
  "vm": {
    "namespace": "kite-user-ku-...",
    "name": "dev-vm-1",
    "delete": true
  }
}
```

## Admin Settings

### `GET /api/v1/admin/settings`

현재 base domain, runtime secret 생성 여부, TLS 등록 여부를 읽는다.

Required access level: `3`.

Response `200`:

```json
{
  "config": {
    "baseDomain": "apps.example.com",
    "hasJWTSecret": true,
    "hasPasswordSalt": true,
    "hasTLSCertificate": true
  }
}
```

### `POST /api/v1/admin/domain`

`AdminSettings.tsx`의 `handleSaveDomain` mock을 이 호출로 교체한다.

Required access level: `3`.

Request:

```json
{
  "baseDomain": "apps.example.com"
}
```

Response `200`:

```json
{
  "message": "base domain updated",
  "config": {
    "baseDomain": "apps.example.com",
    "hasJWTSecret": true,
    "hasPasswordSalt": true,
    "hasTLSCertificate": false
  }
}
```

현재 API는 `kite/kite-runtime-config` ConfigMap의 `data.baseDomain`을 갱신한다. controller는 이 값을 보고 `kite/kite-platform` Ingress host를 갱신하고, VM별 Ingress는 `spec.domainPrefix + baseDomain` 조합으로 만든다. baseDomain이 비어 있으면 hostless/catch-all platform Ingress와 VM별 Ingress를 만들지 않는다.

### `POST /api/v1/admin/runtime-secrets/rotate`

AdminSettings의 런타임 secret 재생성 버튼에서 호출한다. secret 원문은 request나 response에 넣지 않는다.

Required access level: `3`.

Request:

```json
{
  "rotateJWTSecret": true,
  "rotatePasswordSalt": true
}
```

둘 중 하나만 `true`로 보내도 된다.

Response `200`:

```json
{
  "message": "runtime secrets rotated",
  "config": {
    "baseDomain": "apps.example.com",
    "hasJWTSecret": true,
    "hasPasswordSalt": true,
    "hasTLSCertificate": false
  }
}
```

새 `jwtSecret`과 `passwordSalt`는 `kite/kite-runtime-secret`에 저장되며, 실제 JWT 발급/검증과 password hash에는 다음 `kite-api` 프로세스 시작 시 적용된다.

### `POST /api/v1/admin/cert`

`AdminSettings.tsx`의 `handleSaveCert` mock을 이 호출로 교체한다.

Required access level: `3`.

Request:

```json
{
  "tlsCert": "-----BEGIN CERTIFICATE-----...",
  "tlsKey": "-----BEGIN PRIVATE KEY-----..."
}
```

Response `200`:

```json
{
  "message": "TLS certificate updated",
  "config": {
    "baseDomain": "apps.example.com",
    "hasJWTSecret": true,
    "hasPasswordSalt": true,
    "hasTLSCertificate": true
  }
}
```

인증서 원문은 응답하지 않는다. API는 `kite/global-tls-secret` Secret에 `kubernetes.io/tls` 타입으로 저장한다.
