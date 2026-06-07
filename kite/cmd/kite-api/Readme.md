# Kite API

Kite API는 사용자와 프론트엔드가 Kite 기능을 사용하기 위해 접근하는 HTTP 서버입니다. 현재는 Gin 기반 서버, KiteUser 기반 로그인, JWT 발급, KiteUser 생성/조회/수정/삭제 흐름이 들어가 있습니다.

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

- `GET /health`, `GET /api/v1/health`: API process, Kubernetes dynamic client, Kite CRD/etcd read path 상태 확인
- `GET /api/v1/config`: 프론트엔드용 base domain, runtime config 존재 여부, TLS 등록 여부 조회
- `POST /api/v1/auth/login`: KiteUser CRD에 저장된 email/password로 로그인
- `POST /api/v1/auth/signup`: 공개 가입. 첫 가입자는 admin, 이후 가입자는 read-only로 생성
- `GET /api/v1/me`: 현재 로그인한 KiteUser 프로필 조회
- `GET /api/v1/vms`: 현재 사용자 namespace의 KiteVirtualMachine 목록 조회
- `POST /api/v1/vms`: 현재 사용자 namespace에 KiteVirtualMachine CRD 생성
- `GET /api/v1/vms/:name`: 현재 사용자 namespace의 KiteVirtualMachine 단건 조회
- `PATCH /api/v1/vms/:name`: 현재 사용자 namespace의 VM desired state 수정
- `POST /api/v1/vms/:name/start`: 현재 사용자 namespace의 VM 희망 전원 상태를 `On`으로 수정
- `POST /api/v1/vms/:name/stop`: 현재 사용자 namespace의 VM 희망 전원 상태를 `Off`로 수정
- `DELETE /api/v1/vms/:name`: 현재 사용자 namespace의 VM에 `spec.delete=true` 설정
- `GET /api/v1/admin/users`: manager 이상 권한으로 KiteUser 목록 조회
- `PATCH /api/v1/admin/users/:name/access-level`: admin 권한으로 사용자 access level 수정
- `DELETE /api/v1/admin/users/:name`: admin 권한으로 사용자 namespace의 KiteVirtualMachine CRD를 먼저 삭제한 뒤 KiteUser 삭제
- `GET /api/v1/admin/vms`: manager 이상 권한으로 모든 KiteVirtualMachine 목록 조회
- `PATCH /api/v1/admin/vms/:namespace/:name/power`: manager 이상 권한으로 모든 VM 희망 전원 상태 수정
- `DELETE /api/v1/admin/vms/:namespace/:name`: manager 이상 권한으로 모든 VM에 `spec.delete=true` 설정
- `GET /api/v1/admin/settings`: admin 권한으로 전역 설정 조회
- `POST /api/v1/admin/domain`: admin 권한으로 `kite/kite-runtime-config`의 base domain 수정
- `POST /api/v1/admin/runtime-secrets/rotate`: admin 권한으로 `jwtSecret`, `passwordSalt` 재생성
- `POST /api/v1/admin/cert`: admin 권한으로 `kube-system/global-tls-secret` TLS Secret 생성/수정
- JWT access token 발급과 검증
- Kubernetes dynamic client 연결
- `kite/kite-runtime-config` ConfigMap 자동 생성과 `jwtSecret`, `passwordSalt` 자동 생성

VM 생성 요청의 `sshPassword`는 HTTP request body에서만 평문으로 다룹니다.
`vm.Service.Create`는 즉시 `auth.HashPassword`와 runtime `passwordSalt`로
`KiteVirtualMachine.spec.sshPasswordHash`를 만들고, CRD에는 `spec.sshPassword`
평문 필드를 저장하지 않습니다. `kite-gateway`는 같은 salt로
`spec.sshPasswordHash`를 검증합니다.

## 로컬 실행

`cmd/kite-api`는 `main.go`와 `kube.go`가 같은 `main` package를 구성합니다. 단일 파일 실행인 `go run main.go`를 사용하면 `kube.go`의 `getDynamicClient()`가 포함되지 않아 `undefined: getDynamicClient` 오류가 납니다.

올바른 실행 명령:

```bash
cd kite
go run ./cmd/kite-api
```

또는:

```bash
cd kite/cmd/kite-api
go run .
```

`kite-api`는 `.env` 파일 없이 실행된다. 시작 시 Kubernetes dynamic client로 `kite` namespace와 `kite/kite-runtime-config` ConfigMap을 확인하고, 없으면 생성한다. `jwtSecret`과 `passwordSalt`는 최초 시작 시 한 번 생성되어 ConfigMap에 저장되고 이후 재사용된다.

AdminSettings에서 base domain과 runtime secret을 관리할 수 있다. base domain은 `kite-platform` Ingress host와 VM별 Ingress host 계산에 사용된다. secret 원문은 API 응답이나 화면에 노출하지 않고, rotate 요청으로 새 값을 생성해 ConfigMap에 저장한다. `kite-api` process는 시작 시 ConfigMap을 읽어 `config.Config` 객체로 보관하므로, secret rotate는 다음 `kite-api` 재시작부터 적용된다.

## 현재 endpoint와 호출 흐름

### `GET /health`

- 등록 위치: `cmd/kite-api/main.go`
- 호출 흐름:
  1. `newRouter`가 Gin router에 `/health`를 등록한다.
  2. handler가 `health.Run(c.Request.Context(), dynamicClient)`를 호출한다.
  3. `health.Run`이 API process 자체는 `api` check로 기록한다.
  4. dynamic client가 없으면 `kubernetes.dynamicClient` check를 failed로 반환한다.
  5. dynamic client가 있으면 `kiteusers` CRD를 list해서 Kubernetes API server와 etcd read path를 확인한다.
  6. 이어서 `kitevirtualmachines` CRD를 list해서 VM CRD read path를 확인한다.
  7. 모든 check가 성공하면 `status: "ok"`와 HTTP 200을 반환한다.
  8. 하나라도 실패하면 `status: "degraded"`와 HTTP 503을 반환한다.

`/health`는 etcd에 직접 접속하지 않는다. Kubernetes CRD는 Kubernetes API server를 통해 etcd에 저장되므로, health check는 Kite CRD list 성공 여부로 API server와 etcd read path가 정상인지 간접 확인한다.

### `POST /api/v1/auth/login`

- 등록 위치: `RegisterV1`
- handler: `loginHandler`
- 내부 호출 흐름:
  1. `loginHandler`가 email/password request body를 읽는다.
  2. `account.NewService(deps.DynamicClient, deps.Config.PasswordSalt)`로 account service를 만든다.
  3. `account.Service.Authenticate`를 호출한다.
  4. `Authenticate`가 `Service.FindByEmail`을 호출한다.
  5. `FindByEmail`이 `store.UserStore.List`로 `KiteUser` CRD 목록을 읽고 `spec.email`이 같은 사용자를 찾는다.
  6. `Authenticate`가 `auth.VerifyPassword`로 요청 password와 `KiteUser.spec.password` hash를 비교한다.
  7. 인증에 성공하면 `auth.TokenService.IssueAccessToken`으로 JWT access token을 만든다.
  8. 응답 body와 `accessToken` HttpOnly cookie에 Bearer token을 내려준다.

로그인은 CRD를 만들지 않는다. 로그인은 이미 존재하는 `KiteUser` CRD를 읽어서 password hash와 access level을 확인하는 작업이다. `KiteUser` CRD 생성은 회원가입 또는 admin 사용자 생성 API에서만 한다.

### `POST /api/v1/auth/signup`

- 등록 위치: `RegisterV1`
- handler: `userSignUpHandler`
- 권한: 공개 endpoint
- 내부 호출 흐름:
  1. `userSignUpHandler`가 signup request body를 읽는다.
  2. `account.NewService`로 account service를 만든다.
  3. `account.Service.SignUp`을 호출한다.
  4. `SignUp`이 `newSignUpRecord`에서 request를 `store.KiteUserRecord`로 바꾼다.
  5. `newSignUpRecord`가 `FindByUsername`으로 username 중복을 확인한다.
  6. `accessLevelForNewUser`가 `store.UserStore.List`로 기존 `KiteUser` 개수를 확인한다.
  7. 기존 사용자가 없으면 `AccessLevelAdmin(3)`, 한 명이라도 있으면 `AccessLevelReadOnly(0)`를 넣는다.
  8. `metadata.name`은 API가 `ku-<uuid>` 형식으로 생성한다. 이 값은 KiteUser의 내부 PK로 본다.
  9. `spec.namespace`는 `kite-user-<metadata.name>`으로 생성한다. public signup 요청자가 namespace를 직접 고르지 않는다.
  10. `spec.profile_image` 기본값은 `base64encodedimage`로 저장한다.
  11. password는 `auth.HashPassword`로 hash해서 `KiteUser.spec.password`에 넣는다.
  12. `store.UserStore.Create`가 cluster-scoped `KiteUser` CRD를 생성한다.
  13. API 응답은 password를 제외한 `account.PublicUser`만 반환한다.

생성되는 CRD spec 예시는 다음과 같다.

```yaml
metadata:
  name: ku-<uuid>
spec:
  username: test
  email: test@gmail.com
  password: <one-way-hash>
  namespace: kite-user-ku-<uuid>
  profile_image: base64encodedimage
  access_level: 0
```

`/api/signup`과 `/api/user`는 기존 호출 호환을 위해 남긴 alias다. 새 프론트엔드는 `/api/v1/auth/signup`을 사용한다.

### `GET /api/v1/me`

- 등록 위치: `RegisterUsers`
- middleware: `RequireAccessLevel(..., AccessLevelReadOnly)`
- handler: `currentUserHandler`
- 내부 호출 흐름:
  1. `RequireAccessLevel`이 Authorization header 또는 `accessToken` cookie에서 Bearer token을 읽는다.
  2. `auth.TokenService.VerifyAccessToken`으로 token을 검증하고 claims를 Gin context에 저장한다.
  3. `currentUserHandler`가 `currentClaims`로 token subject를 읽는다.
  4. `account.Service.FindByUsername`이 `KiteUser.spec.username` 기준으로 현재 사용자를 찾는다.
  5. `account.Service.Get`이 `metadata.name` 기준으로 CRD를 다시 읽고 public response로 변환한다.

### `GET /api/v1/admin/users`

- 등록 위치: `RegisterUsers`
- middleware: `RequireAccessLevel(..., AccessLevelManager)`
- handler: `userListHandler`
- 내부 호출 흐름:
  1. middleware가 manager 이상 권한인지 확인한다.
  2. `userListHandler`가 `account.Service.List`를 호출한다.
  3. `List`가 `store.UserStore.List`로 모든 cluster-scoped `KiteUser` CRD를 읽는다.
  4. 각 CRD를 `account.PublicUser`로 바꾸고 password hash는 제외한다.

### `GET /api/users/:name`

- 등록 위치: `RegisterUsers`
- middleware: `RequireAccessLevel(..., AccessLevelReadOnly)`
- handler: `userGetHandler`
- 내부 호출 흐름:
  1. `account.Service.Get`이 `store.UserStore.Get`으로 `metadata.name`에 해당하는 `KiteUser` CRD를 읽는다.
  2. `canReadUser`가 token claims와 user response를 비교한다.
  3. manager 이상은 모든 사용자를 조회할 수 있고, read-only/user는 자기 username만 조회할 수 있다.

### `PATCH /api/v1/admin/users/:name/access-level`

- 등록 위치: `RegisterUsers`
- middleware: `RequireAccessLevel(..., AccessLevelAdmin)`
- handler: `adminUserAccessLevelHandler`
- 내부 호출 흐름:
  1. admin 권한을 확인한다.
  2. `userUpdateHandler`가 수정 request body를 읽는다.
  3. `account.Service.Update`를 호출한다.
  4. `Update`가 `store.UserStore.Get`으로 현재 CRD를 읽는다.
  5. `applyUpdate`가 email, password, namespace, profile image, access level 중 전달된 field만 반영한다.
  6. password가 전달되면 `auth.HashPassword`로 다시 hash한다.
  7. access level은 `0`부터 `3`까지만 허용한다.
  8. `store.UserStore.Update`가 `KiteUser.spec`을 갱신한다.

### `DELETE /api/v1/admin/users/:name`

- 등록 위치: `RegisterUsers`
- middleware: `RequireAccessLevel(..., AccessLevelAdmin)`
- handler: `userDeleteHandler`
- 내부 호출 흐름:
  1. admin 권한을 확인한다.
  2. `account.Service.Delete`를 호출한다.
  3. `Delete`가 `store.UserStore.Get`으로 `KiteUser` CRD를 읽고 `spec.namespace`를 확인한다.
  4. `deleteVirtualMachinesInNamespace`가 `store.VirtualMachineStore.List`로 해당 namespace의 `KiteVirtualMachine` CRD를 모두 조회한다.
  5. 각 VM CRD에 대해 `store.VirtualMachineStore.Delete`를 호출한다.
  6. VM CRD 삭제가 끝나면 `store.UserStore.Delete`로 cluster-scoped `KiteUser` CRD를 삭제한다.
  7. Namespace, quota, network policy 같은 실제 Kubernetes 리소스 정리는 controller reconcile 정책에 맡긴다.

## internal 계층 역할

- `internal/account`: KiteUser 기반 가입, 로그인, 사용자 조회/수정/삭제 서비스 로직을 담당한다.
- `internal/store`: dynamic client로 Kite CRD를 create/get/list/update/delete한다.
- `internal/auth`: token 발급/검증과 password hash/verify를 담당한다.
- `cmd/kite-api/apis`: Gin request/response, middleware, HTTP status mapping만 담당한다.

이 구조에서는 HTTP handler가 직접 CRD object를 조립하지 않는다. handler는 request를 읽고 `internal/account`를 호출하며, `internal/account`가 `internal/store`를 통해 `KiteUser` CRD를 다룬다.

## internal/render 확인

`internal/render`는 Kubernetes에 직접 파일을 적용하는 코드가 아니다. 현재 구조는 YAML template을 Go binary에 embed하고, template data를 주입한 뒤 `unstructured.Unstructured` 객체로 변환하는 렌더러다.

현재 동작:

1. 각 renderer 패키지가 `//go:embed`으로 YAML template 파일을 문자열로 포함한다.
2. `render.NewRendererFromTemplate(name, content)`가 메모리 template renderer를 만든다.
3. `BaseRenderer.Render(data)`는 첫 번째 YAML document를 `unstructured.Unstructured`로 반환한다.
4. `BaseRenderer.RenderAll(data)`는 `---`로 나뉜 여러 YAML document를 모두 `unstructured.Unstructured` 목록으로 반환한다.
5. 실제 Kubernetes apply/create/update/delete는 renderer가 하지 않는다. controller가 반환된 `unstructured.Unstructured`를 dynamic client로 apply해야 한다.

현재 renderer 목록:

- `namespace.NamespaceData.Render`: Namespace object 생성
- `networkpolicy.NetworkPolicyData.RenderAll`: NetworkPolicy object 목록 생성
- `quotapolicy.QuotaPolicyData.RenderAll`: ResourceQuota/LimitRange object 목록 생성
- `kubevirtmachine.KubevirtMachineData.Render`: KubeVirt VirtualMachine object 생성
- `datavolume.DataVolumeData.Render`: CDI DataVolume object 생성
- `cloudinituserdata.Ubuntu2204CloudInit.Render`: cloud-init Secret object 생성
- `vmservice.ServiceData.Render`: Service object 생성. template은 multi-document YAML이지만 현재 method는 `Render`라 첫 Service만 반환한다.
- `ingress.IngressData.Render`: Ingress object 생성

주의할 점:

- `vm-service.yaml`은 SSH Service와 web Service 두 document가 있지만 `ServiceData.Render`는 첫 번째 document만 반환한다. 둘 다 적용하려면 `RenderAll` method가 필요하다.
- `kubevirt-machine.yaml`은 현재 `spec.running: true`가 고정되어 있어 `KiteVirtualMachine.spec.powerState`를 반영하지 못한다.
- `DataVolumeData.VmName` type이 현재 `VmName` enum처럼 정의되어 있어 사용자 VM 이름과 image 이름 역할이 섞일 수 있다. controller 구현 전에 template field 의미를 정리해야 한다.

## 먼저 구현할 API

### 인증

- [x] KiteUser 기반 로그인으로 전환한다.
- [x] 비밀번호 해시 검증 방식을 정한다.
- [x] 첫 가입자는 admin, 이후 가입자는 read-only로 생성한다.
- [ ] 토큰 만료, 재발급, 로그아웃 정책을 정한다.
- [ ] 쿠키와 Authorization 헤더 사용 방식을 정리한다.

### 사용자 API

- [x] 사용자 생성 API에서 KiteUser CRD를 생성한다.
- [x] 사용자 단건 조회 API에서 KiteUser CRD를 조회한다.
- [x] 사용자 수정 API에서 KiteUser CRD spec을 수정한다.
- [x] 사용자 삭제 API에서 KiteUser CRD를 삭제한다.
- [ ] 사용자 삭제 시 네임스페이스와 VM 정리 정책을 API 응답에 반영한다.

### VM API

- [x] VM 생성 API에서 KiteVirtualMachine CRD를 생성한다.
- [x] VM 목록 조회 API에서 KiteVirtualMachine CRD를 조회한다.
- [x] VM 상세 조회 API에서 KiteVirtualMachine CRD와 status를 반환한다.
- [x] VM 수정 API에서 KiteVirtualMachine CRD spec을 수정한다.
- [x] VM 삭제 API에서 `spec.delete=true`를 설정한다.
- [x] VM 시작/중지 API에서 `spec.powerState`를 수정한다.
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

Kite는 Kubernetes의 선언형 API와 controller reconcile 패턴을 따릅니다. 따라서 API 서버는 컨트롤러에 gRPC 명령을 보내지 않고, Kubernetes API server에 Kite CRD를 기록합니다. Kubernetes API server는 CRD를 etcd에 저장하고, 컨트롤러는 그 CRD를 watch해서 실제 클러스터 상태를 맞춥니다.

기본 흐름:

1. 프론트엔드가 kite-api에 HTTP 요청을 보낸다.
2. kite-api가 인증, 권한, 요청 validation을 처리한다.
3. kite-api가 KiteUser 또는 KiteVirtualMachine CRD를 생성/수정/삭제한다.
4. kite-controller가 CRD 변경을 감지한다.
5. kite-controller가 Namespace, KubeVirt VM, DataVolume, Service, Ingress 같은 실제 리소스를 reconcile한다.
6. kite-controller가 처리 결과를 CRD status에 기록한다.

이 구조에서 API 서버가 직접 만들 수 있는 것은 Kite CRD까지입니다. Namespace, KubeVirt VM, DataVolume, Service, Ingress, NetworkPolicy, QuotaPolicy는 컨트롤러가 만듭니다.

gRPC 연결은 폐기합니다. API 서버와 컨트롤러 사이의 계약은 gRPC가 아니라 `KiteUser`, `KiteVirtualMachine` CRD의 spec/status입니다.

## API 서버 TODO

- [ ] 요청/응답 구조체를 API별로 정리한다.
- [ ] validation 함수를 추가한다.
- [ ] 에러 응답 형식을 통일한다.
- [ ] 사용자 비밀번호가 목록 응답에 노출되지 않게 한다.
- [ ] KiteUser CRD 생성/조회/수정/삭제 helper를 만든다.
- [ ] KiteVirtualMachine CRD 생성/조회/수정/삭제 helper를 만든다.
- [ ] VM 전원 제어는 `KiteVirtualMachine.spec.powerState` 수정으로 처리한다.
- [ ] Kubernetes API 실패를 사용자에게 보여줄 메시지와 내부 로그로 나눈다.
- [ ] 프론트엔드가 사용할 API 명세를 README 또는 별도 문서로 정리한다.
- [ ] 인증/권한 테스트를 보강한다.
