# Kite TODO

이 문서는 2026-06-04 기준으로 README, CRD, API 서버, 컨트롤러, store, renderer 코드를 읽고 다시 정리한 진행 상태와 작업 목록이다.

## 확정된 방향

Kite의 상태 저장소는 별도 etcd 클라이언트가 아니라 Kubernetes CRD다.

즉 사용자와 VM 요청은 `kite-api`가 Kubernetes API server에 Kite CRD로 기록하고, Kubernetes API server가 그 CRD를 etcd에 저장한다. `kite-controller`는 이 CRD를 watch/reconcile해서 실제 Kubernetes와 KubeVirt 리소스를 원하는 상태로 맞춘다.

Kite 자체 운영 리소스는 `kite` namespace에 둔다. `kite-api`, `kite-controller`, `kite-frontend`, Service, ServiceAccount는 `-n kite` 기준이고, CRD와 cluster-scoped `KiteUser`는 namespace 없이 cluster-wide로 관리한다. API/controller Pod는 in-cluster config와 `kite` namespace의 service account 권한으로 Kubernetes API server에 접근한다.

핵심 제어 흐름은 다음과 같다.

1. 프론트엔드 또는 API 클라이언트가 `kite-api`에 요청한다.
2. `kite-api`가 인증, 권한, validation을 처리한다.
3. `kite-api`가 `KiteUser` 또는 `KiteVirtualMachine` CRD를 생성/수정/삭제한다.
4. `kite-controller`가 Kite CRD 변경을 감지한다.
5. `kite-controller`가 Namespace, policy, KubeVirt VM, DataVolume, Service, Ingress 같은 실제 리소스를 reconcile한다.
6. 실제 KubeVirt 상태가 바뀌면 controller가 다시 상태를 읽고 `KiteVirtualMachine.status`에 반영한다.
7. 실제 상태가 사용자가 원하는 `spec`과 다르면 controller가 다시 조정한다.
8. `kite-account`가 `KiteVirtualMachine`과 `vps-access-<vmName>` ClusterIP Service를 watch해서 single-node host Linux 계정과 proxy shell을 reconcile한다.

## 현재 구현 상태

### 저장소와 CRD

- [x] Kite 자체 운영 namespace는 `kite`로 정한다.
- [x] `install.yaml`의 API/controller/frontend 배포 namespace를 `kite`로 맞춘다.
- [x] `install.yaml`에 API/controller용 `kite-account` ServiceAccount와 cluster-wide RBAC를 둔다.
- [x] CRD와 cluster-scoped `KiteUser`는 namespace 없이 관리한다.
- [x] 사용자 저장 방식은 `KiteUser` CRD를 통해 Kubernetes API server와 etcd에 저장하는 방향이다.
- [x] VM 요청 저장 방식은 `KiteVirtualMachine` CRD를 통해 Kubernetes API server와 etcd에 저장하는 방향이다.
- [x] `KiteUser` CRD는 cluster-scoped다.
- [x] `KiteVirtualMachine` CRD는 namespaced다.
- [x] `custom/kite-user-crd.yaml`에 user spec과 namespace reconcile용 status가 있다.
- [x] `custom/kite-machine-crd.yaml`에 VM desired state용 spec과 runtime status 필드가 있다.
- [x] `internal/store/UserStore`가 `KiteUser` CRD CRUD를 제공한다.
- [x] `internal/store/VirtualMachineStore`가 `KiteVirtualMachine` CRD CRUD를 제공한다.
- [x] `kite/api/v1/types.go`의 `KiteVirtualMachineSpec`에 CRD의 `powerState`, `domainPrefix`, `sshId`, `sshPassword`, `delete`가 있다.
- [x] `kite/api/v1/types.go`의 `KiteVirtualMachineStatus`에 CRD의 `currentPowerState`, `observedGeneration`, `domain`, `dataVolumePhase`, `dataVolumeProgress`, `dataVolumeMessage`, `conditions`가 있다.
- [ ] CRD schema에 required field, validation rule, 기본값을 더 엄격히 둘지 결정해야 한다.

### kite-api

- [x] Gin 기반 HTTP 서버가 있다.
- [x] `GET /health`가 있다.
- [x] `POST /api/v1/auth/login`이 있다.
- [x] access token 발급과 검증 코드가 있다.
- [x] access level 기반 middleware가 있다.
- [x] `GET /api/users`가 `KiteUser` 목록을 조회한다.
- [x] `POST /api/users`가 admin 권한으로 `KiteUser` CRD를 생성한다.
- [x] `POST /api/user`가 인증 없이 `KiteUser` CRD를 생성한다.
- [x] 사용자 생성 시 password를 hash해서 `KiteUser.spec.password`에 저장한다.
- [x] 사용자 목록 응답에서 password hash는 제외한다.
- [x] 로그인은 `KiteUser` 기반으로 동작한다.
- [x] 사용자 단건 조회, 수정, 삭제 API가 있다.
- [ ] VM 생성, 조회, 수정, 삭제, 전원 제어 API가 없다.
- [ ] API request validation과 error response 형식이 아직 통일되지 않았다.
- [x] `POST /api/user` 공개 가입을 유지하고 첫 가입자만 admin으로 만든다.

### kite-controller

- [x] controller main이 client manager를 만들고 reconciler goroutine을 실행한다.
- [x] `KiteUser` informer가 있다.
- [x] `KiteUser` add/update 이벤트가 `ReconcileKiteUser`를 호출한다.
- [x] `KiteUser` reconcile이 사용자 namespace를 만든다.
- [x] `KiteUser` reconcile이 network policy와 quota policy를 적용한다.
- [x] `KiteUser` reconcile이 status를 `Ready` 또는 `Failed`로 갱신한다.
- [x] Kite-managed namespace orphan 정리 흐름이 있다.
- [x] `KiteVirtualMachine` informer가 있다.
- [x] `KiteVirtualMachine` add/update/delete 이벤트가 reconcile 함수를 호출한다.
- [x] `KiteVirtualMachine.spec.delete=true`이면 KubeVirt VirtualMachine을 먼저 삭제하고, KubeVirt VM이 없으면 KiteVirtualMachine CRD도 삭제한다.
- [x] `KiteVirtualMachine` cleanup finalizer가 있어 CRD 직접 삭제 시에도 KubeVirt VM 삭제를 먼저 시도한다.
- [x] `KiteVirtualMachine` reconcile이 DataVolume, cloud-init Secret, KubeVirt VM, Service, 선택적 Ingress를 생성/수정한다.
- [x] KubeVirt `VirtualMachine` 상태 변화 watch가 있다.
- [x] DataVolume 상태 변화 watch가 있다.
- [x] controller가 KubeVirt 실제 상태를 `KiteVirtualMachine.status`에 반영한다.
- [x] controller가 실제 KubeVirt 상태가 사용자의 desired state와 달라졌을 때 다시 조정한다.

### renderer와 KubeVirt manifest

- [x] Namespace renderer가 있다.
- [x] NetworkPolicy renderer가 있다.
- [x] QuotaPolicy renderer가 있다.
- [x] KubeVirt `VirtualMachine` renderer가 있다.
- [x] CDI `DataVolume` renderer가 있다.
- [x] VM용 Service renderer가 있다.
- [x] Ingress renderer가 있다.
- [x] Ubuntu 22.04 cloud-init Secret renderer가 있다.
- [x] VM renderer들이 `KiteVirtualMachine` controller에 연결되어 있다.
- [x] `kubevirt-machine.yaml`의 `spec.running`이 `KiteVirtualMachine.spec.powerState`를 반영한다.
- [x] DataVolume template의 이름과 KubeVirt VM template의 `dataVolume.name`이 일치한다.
- [x] cloud-init Secret template 이름과 KubeVirt VM template의 `secretRef.name`이 일치한다.
- [x] VM SSH Service는 NodePort 없이 `ClusterIP`로 만들고, `kite-account`가 고정 이름 `vps-access-<vmName>` Service를 직접 조회한다.
- [x] Ingress domain은 `spec.domainPrefix`와 `kite/kite-runtime-config.data.baseDomain` 조합으로 만든다.
- [x] 모든 VM 관련 리소스에 Kite 관리 label과 VM owner label을 붙인다.

### frontend

- [x] `kite-frontend/` 디렉터리는 있다.
- [ ] Vite 프로젝트는 아직 생성되지 않았다.
- [ ] 로그인, 사용자 관리, VM 관리 화면은 아직 없다.

### 폐기 예정 gRPC 코드

- [x] `api/proto/resource/resource.proto`에 이전 gRPC 계약 초안이 남아 있다.
- [x] controller gRPC 서버 골격이 남아 있다.
- [x] generated protobuf Go 파일은 없다.
- [x] 새 설계에서는 gRPC를 사용하지 않고 CRD watch/reconcile만 사용한다.
- [ ] `api/proto/resource/resource.proto`를 삭제한다.
- [ ] `kite/cmd/kite-controller/apps/gRPC-server.go`를 삭제한다.
- [ ] gRPC 의존성이 더 이상 필요 없는지 `go.mod`를 확인하고 정리한다.
- [x] README와 하위 README에서 gRPC 사용/보류 표현을 제거하고 폐기 방향으로 정리한다.

## 1차 목표: CRD 기반 user 흐름 정리

- [ ] README에 “Kite는 CRD를 통해 Kubernetes etcd에 상태를 저장한다”라고 명확히 적는다.
- [ ] `KiteUser.metadata.name`과 `KiteUser.spec.username`의 관계를 정한다.
- [ ] username 중복을 어떻게 막을지 정한다.
- [ ] namespace 이름을 사용자가 입력할지, API 서버가 생성할지 결정한다.
- [x] 공개 가입 API인 `POST /api/user`를 유지한다.
- [x] `POST /api/v1/auth/login`을 `KiteUser` email 조회 기반으로 바꾼다.
- [x] `KiteUser.spec.password`의 hash 검증 함수를 추가한다.
- [x] 첫 가입자는 admin, 이후 가입자는 read-only로 생성한다.
- [x] 사용자 단건 조회 API를 구현한다.
- [x] 사용자 수정 API를 구현한다.
- [x] 사용자 삭제 API를 구현한다.
- [x] 사용자 삭제 API는 해당 namespace의 KiteVirtualMachine CRD를 먼저 삭제한 뒤 KiteUser를 삭제한다.

## 2차 목표: VM CRD와 API 완성

- [x] `KiteVirtualMachine` Go type을 CRD 스키마와 맞춘다.
- [x] `KiteVirtualMachine.spec.powerState`를 Go type에 추가한다.
- [x] `KiteVirtualMachine.spec.delete`를 CRD schema와 Go type에 추가한다.
- [x] `KiteVirtualMachine.status.currentPowerState`를 Go type에 추가한다.
- [x] `KiteVirtualMachine.status.observedGeneration`을 Go type에 추가한다.
- [x] VM SSH 접근은 NodePort status 없이 `vps-access-<vmName>` ClusterIP Service와 host proxy shell을 사용한다.
- [x] `KiteVirtualMachine.status.domain`을 Go type에 추가한다.
- [x] `KiteVirtualMachine.status.conditions`를 Go type에 추가한다.
- [ ] VM 생성 API를 추가한다.
- [ ] VM 목록 조회 API를 추가한다.
- [ ] VM 상세 조회 API를 추가한다.
- [ ] VM 수정 API를 추가한다.
- [ ] VM 삭제 API를 추가한다.
- [ ] VM 시작 API는 KubeVirt를 직접 호출하지 않고 `KiteVirtualMachine.spec.powerState = "On"`으로 수정한다.
- [ ] VM 중지 API는 KubeVirt를 직접 호출하지 않고 `KiteVirtualMachine.spec.powerState = "Off"`로 수정한다.
- [ ] user 권한은 자기 namespace 안의 VM만 조회/수정/삭제할 수 있게 한다.
- [ ] manager/admin 권한이 전체 namespace VM을 볼 수 있는지 결정한다.
- [ ] VM API 응답에는 CRD spec과 status를 함께 내려준다.

## 3차 목표: KiteVirtualMachine desired state reconcile

이 단계의 목표는 `KiteVirtualMachine.spec`을 사용자가 원하는 상태로 보고, 실제 KubeVirt 관련 리소스를 거기에 맞추는 것이다.

- [x] `RegisterKiteVirtualMachineReconciler`가 로그만 남기지 않고 `ReconcileKiteVirtualMachine`을 호출하게 한다.
- [x] `ReconcileKiteVirtualMachine(ctx, dynamicClient, eventObj)`를 추가한다.
- [x] informer event object를 `kitev1.KiteVirtualMachine`으로 변환하는 helper를 추가한다.
- [x] status-only update event는 reconcile loop를 만들지 않도록 generation을 비교한다.
- [x] VM namespace가 비어 있으면 reconcile을 건너뛰고 로그를 남긴다.
- [x] VM name, CPU, memory, image, disk, sshId, sshPassword, powerState validation을 controller에도 둔다.
- [x] `powerState` 기본값은 빈 값이면 `"Off"`로 처리한다.
- [x] `powerState` 값은 `"On"`과 `"Off"`만 허용한다.
- [x] VM 관련 리소스 GVR을 명시적으로 정의한다.
- [x] KubeVirt `VirtualMachine` GVR: `kubevirt.io/v1`, resource `virtualmachines`.
- [x] CDI `DataVolume` GVR: `cdi.kubevirt.io/v1beta1`, resource `datavolumes`.
- [x] core `Secret` GVR: `v1`, resource `secrets`.
- [x] core `Service` GVR: `v1`, resource `services`.
- [x] networking `Ingress` GVR: `networking.k8s.io/v1`, resource `ingresses`.
- [x] generic kind 소문자 + `s` 방식의 GVR 추론을 VM reconcile에는 쓰지 않는다.
- [x] DataVolume을 render하고 server-side apply한다.
- [x] cloud-init Secret을 render하고 server-side apply한다.
- [x] KubeVirt VirtualMachine을 render하고 server-side apply한다.
- [x] Service를 render하고 server-side apply한다.
- [x] Ingress가 필요한 경우 render하고 server-side apply한다.
- [x] `KiteVirtualMachine.spec.powerState = "On"`이면 KubeVirt `VirtualMachine.spec.running`을 `true`로 맞춘다.
- [x] `KiteVirtualMachine.spec.powerState = "Off"`이면 KubeVirt `VirtualMachine.spec.running`을 `false`로 맞춘다.
- [x] 현재 `kubevirt-machine.yaml`의 `spec.running: true` 고정을 template 입력값으로 바꾼다.
- [x] repeated reconcile이 같은 결과를 만들도록 모든 apply를 idempotent하게 만든다.
- [x] reconcile 성공 시 `KiteVirtualMachine.status.phase`와 condition을 갱신한다.
- [x] reconcile 실패 시 실패 reason과 message를 `status.conditions`에 남긴다.

## 4차 목표: KubeVirt 상태 watch와 재조정

이 단계의 목표는 KubeVirt 실제 상태가 바뀌었을 때 `KiteVirtualMachine.status`를 갱신하고, 사용자가 원하는 상태와 다르면 다시 맞추는 것이다.

- [x] KubeVirt `VirtualMachine` informer를 추가한다.
- [x] CDI `DataVolume` informer를 추가한다.
- [ ] 필요하면 KubeVirt `VirtualMachineInstance` informer를 추가한다.
- [x] KubeVirt VM 이벤트에서 관련 `KiteVirtualMachine`을 찾는 label 전략을 정한다.
- [x] DataVolume 이벤트에서 관련 `KiteVirtualMachine`을 찾는 label 전략을 정한다.
- [x] 관련 실제 리소스에는 `kite.anacnu.com/managed-by: kite-controller` label을 붙인다.
- [x] 관련 실제 리소스에는 `kite.anacnu.com/kite-vm-name` label을 붙인다.
- [x] 관련 실제 리소스에는 `kite.anacnu.com/kite-vm-namespace` label을 붙인다.
- [x] KubeVirt VM의 실제 running/ready/phase 정보를 읽는 helper를 만든다.
- [x] DataVolume의 ready/progress/failure 정보를 읽는 helper를 만든다.
- [x] 실제 상태를 `KiteVirtualMachine.status.currentPowerState`에 반영한다.
- [x] 실제 상태를 `KiteVirtualMachine.status.phase`에 반영한다.
- [x] KubeVirt 상태 변화는 spec을 직접 바꾸지 않고 status만 갱신한다.
- [x] 실제 KubeVirt VM이 사용자가 원하는 `spec.powerState`와 다르면 다시 reconcile한다.
- [x] 누군가 KubeVirt VM을 직접 꺼도 `spec.powerState = "On"`이면 controller가 다시 켜도록 한다.
- [x] 누군가 KubeVirt VM을 직접 켜도 `spec.powerState = "Off"`이면 controller가 다시 끄도록 한다.
- [x] DataVolume이 실패하면 `status.phase=Failed`와 `status.dataVolumeMessage`에 실패 원인을 남긴다.
- [x] KubeVirt VM이 사라졌지만 `KiteVirtualMachine` CRD가 남아 있으면 다시 생성한다.
- [x] Service나 Ingress가 사라졌지만 `KiteVirtualMachine` CRD가 남아 있으면 다음 CRD reconcile에서 다시 생성된다.
- [x] status update가 다시 VM reconcile을 무한 반복하지 않도록 observedGeneration과 condition 비교를 둔다.

## 5차 목표: 삭제, finalizer, 소유권

- [x] `KiteVirtualMachine.spec.delete=true`이면 관련 KubeVirt VM을 먼저 삭제한다.
- [x] `KiteVirtualMachine.spec.delete=true`이고 관련 KubeVirt VM이 없으면 KiteVirtualMachine CRD를 삭제한다.
- [x] `KiteVirtualMachine` CRD가 직접 삭제되면 관련 KubeVirt VM도 삭제한다.
- [x] `KiteVirtualMachine` cleanup finalizer를 추가해서 삭제 정리 전 CRD가 바로 사라지지 않게 한다.
- [x] `KiteVirtualMachine` 삭제 시 Service와 Ingress를 삭제한다.
- [x] `KiteVirtualMachine` 삭제 시 DataVolume과 cloud-init Secret을 삭제한다.
- [ ] disk 보존 정책을 CRD spec에 둘지 annotation으로 둘지 결정한다.
- [x] 삭제 중 상태를 `status.phase = "Deleting"`으로 표현한다.
- [x] 삭제 정리가 완료되기 전 CRD가 사라지지 않도록 cleanup finalizer를 사용한다.
- [x] finalizer add/remove 흐름을 구현한다.
- [x] `KiteUser` 삭제 API는 그 namespace의 `KiteVirtualMachine` CRD를 먼저 삭제한다.
- [ ] namespace 삭제와 VM/DataVolume 삭제 순서를 정한다.

## 6차 목표: 프론트엔드

- [x] `kite-frontend`에 Vite 프로젝트를 생성한다.
- [x] 로그인 화면을 만든다.
- [x] access token 저장 방식을 정한다.
- [x] 관리자 사용자 목록 화면을 만든다.
- [x] 관리자 사용자 생성/수정/삭제 화면을 만든다.
- [x] 사용자 VM 목록 화면을 만든다.
- [x] 사용자 VM 생성 화면을 만든다.
- [x] 사용자 VM 상세 화면을 만든다.
- [x] VM 전원 제어 버튼은 API를 통해 `KiteVirtualMachine.spec.powerState`를 바꾸게 한다.
- [x] VM 생성 중, DataVolume 준비 중, Running, Stopped, Failed 상태를 구분해서 표시한다.
- [ ] status.conditions의 reason/message를 사용자에게 읽기 좋게 표시한다.

## 7차 목표: 테스트와 검증

- [ ] API 사용자 생성 테스트를 보강한다.
- [ ] API 사용자 수정/삭제 테스트를 추가한다.
- [ ] VM API 생성/조회/수정/삭제 테스트를 추가한다.
- [ ] VM 전원 제어 API 테스트를 추가한다.
- [ ] `KiteUser` reconcile 단위 테스트를 추가한다.
- [ ] `KiteVirtualMachine` desired state reconcile 단위 테스트를 추가한다.
- [ ] KubeVirt VM 상태 변화 watch 단위 테스트를 추가한다.
- [ ] DataVolume 상태 변화 watch 단위 테스트를 보강한다.
- [ ] renderer template 테스트를 추가한다.
- [ ] 실제 클러스터에서 CRD 적용부터 namespace 생성까지 수동 검증 절차를 문서화한다.
- [ ] 실제 KubeVirt 클러스터에서 VM 생성, 시작, 중지, 삭제 수동 검증 절차를 문서화한다.
- [ ] controller 재시작 후 기존 `KiteUser`와 `KiteVirtualMachine`을 다시 reconcile하는지 확인한다.

## 바로 다음에 하면 좋은 작업 순서

1. VM API를 추가해서 `KiteVirtualMachine` CRD를 생성/조회/수정/삭제하게 한다.
2. VM 시작/중지 API를 추가해서 `KiteVirtualMachine.spec.powerState`만 변경하게 한다.
3. DataVolume informer를 추가해서 이미지 import, progress, failure를 `KiteVirtualMachine.status`에 반영한다.
4. 실제 KubeVirt 클러스터에서 VM 생성, 시작, 중지, 삭제 수동 검증 절차를 문서화한다.
5. gRPC/protobuf 폐기 파일과 의존성을 정리한다.

## 남은 결정 사항

- 공개 회원가입을 허용할 것인가, 관리자 생성만 허용할 것인가.
- 사용자 namespace 이름을 사용자가 입력하게 할 것인가, API 서버가 생성하게 할 것인가.
- VM 기본 `powerState`는 현재 `"Off"`로 처리한다. API 입력 기본값도 같은 방향으로 맞출지 결정해야 한다.
- VM 접속 방식은 V1에서 single-node host Linux 계정 + `kite-account` proxy shell로 한다.
- NodePort는 사용하지 않는다. VM SSH Service는 `ClusterIP`이고 이름은 `vps-access-<vmName>`로 고정한다.
- Ingress domain은 현재 `spec.domainPrefix + kite-runtime-config.data.baseDomain`으로 조합한다.
- VM 삭제 시 현재 DataVolume을 즉시 삭제한다. 장기적으로 disk 보존 옵션을 둘지 결정해야 한다.
- 폐기 대상 gRPC/protobuf 코드를 언제 삭제할 것인가.
