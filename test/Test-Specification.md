# Kite Test Specification

이 문서는 Kite 프로젝트에서 "무엇을 테스트해야 하는가"를 정리한 기준서다.
특정 파일 단위 테스트 목록이 아니라, 서비스가 실제로 잘 동작하기 위해 반드시
검증해야 하는 사용자 목적, 운영 목적, 클러스터 동작, 실패 복구 흐름을 기준으로
중분류와 소분류를 나눈다.

새 기능을 만들거나 기존 기능을 바꿀 때는 이 문서에서 영향을 받는 항목을 먼저
찾고, 가장 가까운 단위 테스트와 실제 클러스터 E2E 테스트 중 무엇을 추가해야
하는지 결정한다.

## 테스트 작성 원칙

- 테스트는 "코드가 실행된다"가 아니라 "사용자 또는 운영자가 기대한 목적을 달성한다"를 증명해야 한다.
- mock 테스트는 로직 단위 확신을 주는 용도이고, 배포 전 최종 확신은 실제 클러스터 E2E가 담당한다.
- API가 CRD를 쓰고 controller가 실제 리소스를 만들며 status를 갱신하는 전체 경로를 끊어서 보지 않는다.
- shell script 테스트는 성공 경로뿐 아니라 prompt, noninteractive, 실패 중단, rollback, cleanup을 같이 본다.
- frontend mock 데이터는 UI 개발 보조일 뿐이고, production 동작 증거로 단독 인정하지 않는다.
- 모든 wait는 timeout이 있어야 하고, 실패 메시지는 무엇이 기대 상태가 아니었는지 말해야 한다.
- cleanup 테스트는 공유 인프라(Longhorn, KubeVirt, CDI)를 지우지 않는 기본 안전 경로와 명시 opt-in 위험 경로를 분리해야 한다.

## 루트 실행 진입점

목적: 사용자가 저장소 루트에서 기억하기 쉬운 명령만 실행해 설치, 개발 배포, 삭제를 수행한다.

최종 기준: `./ghcr-install.sh`, `./build-install.sh`, `./uninstall.sh`, `./build-clear.sh`가 각각 올바른 하위 스크립트로 위임되고, 사용자의 선택 없이 위험 작업을 수행하지 않는다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| `./ghcr-install.sh` pull 기반 설치 | git checkout이 있는 경우와 curl pipe 실행 모두에서 deploy install 흐름으로 진입하는지 확인한다. |
| `./build-install.sh` 개발 설치 | 로컬 checkout 이미지를 빌드하고 현재 클러스터에 배포하는 dev all-in-one 흐름으로 진입하는지 확인한다. |
| `./uninstall.sh` 배포 제거 | pull 기반 배포 제거 경로로 진입하고, git checkout이 없어도 archive download cleanup이 가능한지 확인한다. |
| `./build-clear.sh` 개발 제거 | 개발 배포 제거 경로로 진입하고, local image 삭제와 클러스터 리소스 삭제 선택이 분리되는지 확인한다. |
| 환경변수 전달 | 루트 wrapper가 하위 스크립트에 사용자가 지정한 환경변수를 손실 없이 전달하는지 확인한다. |
| prompt-first 정책 | 터미널 실행 시 필요한 질문이 실행 초반에 모두 끝나고, 하위 스크립트 실행 중 같은 질문이 다시 나오지 않는지 확인한다. |
| interactive prompt | 터미널 실행 시 위험 옵션을 설명하고 묻는지, noninteractive 실행 시 기본값 또는 명시 환경변수만 쓰는지 확인한다. |
| 실패 전파 | 하위 스크립트 실패가 wrapper 성공으로 숨겨지지 않는지 확인한다. |

## GitHub Workflow와 이미지 배포

목적: production 이미지 publish와 로컬 E2E 이미지 빌드가 같은 컴포넌트와 build argument를 검증한다.

최종 기준: API, controller, gateway, frontend 네 이미지가 같은 소스 기준으로 빌드되고, 태그 혼동 없이 대상 클러스터가 정확한 이미지를 실행한다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| GHCR publish matrix | workflow의 이미지 목록과 로컬 E2E build matrix가 서로 어긋나지 않는지 확인한다. |
| API 이미지 | `kite-api` 바이너리가 production Dockerfile로 빌드되고 배포 pod가 해당 태그를 사용하는지 확인한다. |
| Controller 이미지 | `kite-controller` 바이너리가 production Dockerfile로 빌드되고 informer/reconcile 실행 pod로 뜨는지 확인한다. |
| Gateway 이미지 | `kite-gateway` 바이너리가 production Dockerfile로 빌드되고 SSH server로 listen하는지 확인한다. |
| Frontend 이미지 | Vite production build가 mock API 없이 생성되고 nginx container에서 index를 서빙하는지 확인한다. |
| build args | frontend `VITE_API_BASE_URL`, `VITE_USE_MOCK`, build mode가 배포 목적에 맞게 들어가는지 확인한다. |
| registry push | 일반 k8s에서는 registry 설정이 없으면 build 전에 실패하고, 설정이 있으면 push 후 cluster pull이 되는지 확인한다. |
| local image load | k3s, minikube, k3d, kind별 로컬 image load/import가 실제 runtime image store에 반영되는지 확인한다. |

## Kubernetes 매니페스트와 CRD

목적: Kite runtime 리소스와 CRD 스키마가 Kubernetes에서 원하는 범위와 권한으로 적용된다.

최종 기준: `build/kite` 매니페스트가 원본 수정 없이 적용되고, API/controller/gateway/frontend가 필요한 권한과 설정으로 rollout된다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| Namespace | Kite runtime 리소스가 `kite` namespace에만 생성되는지 확인한다. |
| CRD 적용 | `KiteUser`, `KiteVirtualMachine`, `KiteVirtualMachineOffer` CRD가 생성되고 read/write 가능한지 확인한다. |
| CRD scope | `KiteUser`는 cluster-scoped, VM과 offer는 의도한 namespace scope로 동작하는지 확인한다. |
| CRD schema | spec/status 필드가 API와 controller가 쓰는 값들을 거부하지 않고, 잘못된 타입은 거부하는지 확인한다. |
| RBAC | API/controller ServiceAccount가 필요한 CRD, Secret, Service, DataVolume, KubeVirt 리소스만 다룰 수 있는지 확인한다. |
| Runtime ConfigMap | base domain, TLS secret, gateway 관련 설정이 pod env로 반영되는지 확인한다. |
| Service | API/frontend/gateway Service가 targetPort와 type을 올바르게 노출하는지 확인한다. |
| Deployment rollout | 네 workload가 새 이미지 태그로 rollout되고 CrashLoop 없이 Ready가 되는지 확인한다. |
| kustomize overlay | E2E overlay가 image, tag, pullPolicy만 바꾸고 원본 manifest를 변경하지 않는지 확인한다. |

## Storage, Longhorn, CDI, Golden Image

목적: VM 디스크가 Longhorn StorageClass와 CDI DataVolume을 통해 실제 PVC로 준비된다.

최종 기준: golden image가 `Succeeded`가 되고, 사용자 VM DataVolume이 golden image에서 clone되어 VM boot disk로 연결된다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| Longhorn 설치 대기 | Longhorn namespace, manager, CSI pod들이 Ready가 될 때까지 기다리고 timeout 시 원인을 출력하는지 확인한다. |
| Longhorn disk tag | 기존 Ready disk에 `kite` tag를 붙이는 기본 경로가 scheduling 가능한 disk만 대상으로 하는지 확인한다. |
| 전용 disk entry | opt-in 시 `/mnt/kite-longhorn` host path disk entry가 생성되고 tag가 붙는지 확인한다. |
| 점유/동기화 재시도 | Longhorn node disk sync 중 patch가 거절되면 제한 시간 안에서 재시도하는지 확인한다. |
| StorageClass | `kite-vm-storage`가 Longhorn provisioner, diskSelector, expansion 설정을 올바르게 갖는지 확인한다. |
| CDI 설치 대기 | CDI operator/apiserver/uploadproxy가 Ready가 될 때까지 기다리는지 확인한다. |
| Golden image import | `ubuntu-22.04` DataVolume이 import 진행률을 보이고 최종 `Succeeded`가 되는지 확인한다. |
| Golden image idempotency | 이미 존재하는 golden image를 재적용해도 기존 성공 상태를 깨지 않는지 확인한다. |
| VM disk clone | 사용자 VM DataVolume이 golden image PVC에서 clone되고 Bound PVC로 이어지는지 확인한다. |
| host data cleanup | `DELETE_LONGHORN_DATA=true`일 때 Longhorn PV가 남아 있으면 host data 삭제를 건너뛰는지 확인한다. |
| Longhorn uninstall | `DELETE_LONGHORN=true`에서도 Longhorn PV가 남아 있으면 uninstall을 중단하거나 skip하는지 확인한다. |

## KubeVirt VM Lifecycle

목적: 사용자가 VM을 만들고 켜면 실제 KubeVirt VM이 생성되어 Running 상태가 된다.

최종 기준: API 요청 하나가 `KiteVirtualMachine` CRD, DataVolume, Secret, Service, KubeVirt VirtualMachine, status 갱신으로 이어지고 VM이 접속 가능한 상태가 된다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| VM create request | `powerState=On`, image, disk, password/sshId 요청이 유효성 검증을 통과하고 CRD spec에 기록되는지 확인한다. |
| VM name/namespace | VM이 사용자 namespace 안에 생성되고 다른 사용자 namespace와 충돌하지 않는지 확인한다. |
| DataVolume 생성 | controller가 `<vm>-disk` DataVolume을 만들고 source PVC와 storageClass를 올바르게 설정하는지 확인한다. |
| cloud-init Secret | VM 초기 계정, SSH public key, password login 정책이 cloud-init Secret에 기대대로 들어가는지 확인한다. |
| SSH key Secret | gateway가 VM 내부 접속에 사용할 private key Secret이 생성되고 owner/reference가 맞는지 확인한다. |
| KubeVirt VM 생성 | VirtualMachine spec의 disk, interface, runStrategy 또는 running 의도가 CRD spec과 일치하는지 확인한다. |
| Access Service | `vps-access-*` Service가 VM SSH target으로 연결되는 selector/port를 갖는지 확인한다. |
| Web Service | `vps-web-*` Service가 VM web forwarding 목적에 맞는 port와 selector를 갖는지 확인한다. |
| Power on reconcile | `spec.powerState=On`이면 KubeVirt VM이 시작되고 status가 `Running`/`On`으로 수렴하는지 확인한다. |
| Power off reconcile | `spec.powerState=Off`이면 KubeVirt VM 정지 의도가 반영되고 status가 꺼진 상태로 수렴하는지 확인한다. |
| Status watch | KubeVirt printableStatus, DataVolume phase, nodeName 변화가 Kite CRD status에 반영되는지 확인한다. |
| Drift recovery | controller가 소유한 Service, Secret, VM spec이 외부 변경으로 drift되면 다시 원하는 상태로 복구되는지 확인한다. |
| Delete cleanup | VM CR 삭제 시 VM 관련 DataVolume, Secret, Service가 고아 리소스로 남지 않는지 확인한다. |

## API Server

목적: HTTP API가 인증, 권한, 유효성 검증을 수행하고 Kubernetes API server에 올바른 desired state를 기록한다.

최종 기준: frontend와 외부 사용자가 API를 통해 계정, 관리자 기능, VM, offer, console 기능을 안전하게 사용할 수 있다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| Health | `/api/v1/health`가 서버 상태와 CRD read 가능 여부를 함께 검증하는지 확인한다. |
| Signup | 신규 사용자가 생성되고 `KiteUser.spec`과 초기 status가 기대 형태로 만들어지는지 확인한다. |
| Login/logout | password 검증, cookie/session 발급, logout 후 인증 실패가 올바르게 동작하는지 확인한다. |
| Auth middleware | 인증 필요 route에서 cookie 누락/위조/만료가 거부되고 정상 cookie는 통과하는지 확인한다. |
| Access level | 일반 사용자와 관리자 권한이 route별로 정확히 제한되는지 확인한다. |
| User read/update | 사용자 본인 정보와 관리자 사용자 관리가 CRD spec/status 기준으로 일관되게 동작하는지 확인한다. |
| VM list/detail | 사용자별 VM 목록과 상세가 자기 namespace의 CRD/status만 반환하는지 확인한다. |
| VM create | disk/image/offer/password/sshId 검증 후 VM CRD spec이 정확히 작성되는지 확인한다. |
| VM power action | 전원 변경 API가 `spec.powerState`만 갱신하고 실제 상태는 controller status로 관측하는지 확인한다. |
| VM delete | 삭제 API가 자기 VM만 삭제하고 controller cleanup을 유도하는지 확인한다. |
| VM offer admin | 관리자가 offer를 만들고 수정/삭제하면 일반 사용자 VM 생성 옵션에 반영되는지 확인한다. |
| Platform settings | base domain, TLS secret 같은 설정 변경이 CRD/ConfigMap과 ingress reconcile에 반영되는지 확인한다. |
| Console ticket | console 접속 티켓이 인증된 사용자와 해당 VM에만 발급되고 만료/재사용 제한이 동작하는지 확인한다. |
| Console proxy | websocket/HTTP proxy가 권한 없는 VM 접근을 막고 정상 VM console로 연결되는지 확인한다. |
| Error mapping | Kubernetes NotFound/Conflict/Forbidden과 validation error가 사용자에게 의미 있는 HTTP status로 변환되는지 확인한다. |

## Controller

목적: CRD spec을 원하는 상태로 보고 실제 Kubernetes/KubeVirt 리소스를 idempotent하게 맞춘다.

최종 기준: API가 controller에 직접 명령하지 않아도 CRD 변경만으로 사용자 namespace, VM, ingress, service, status가 수렴한다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| Informer startup | KiteUser, KiteVirtualMachine, offer, KubeVirt/CDI 리소스 watch가 시작되고 이벤트를 reconcile로 전달하는지 확인한다. |
| User reconcile | `KiteUser` 생성 시 사용자 namespace, ResourceQuota, NetworkPolicy, 기본 Secret/label이 생성되는지 확인한다. |
| User status | 사용자 준비 성공/실패가 `KiteUser.status.phase`와 reason에 기록되는지 확인한다. |
| Namespace idempotency | 동일 user reconcile을 여러 번 실행해도 quota/policy/namespace가 중복되거나 손상되지 않는지 확인한다. |
| VM reconcile | VM spec을 기준으로 DataVolume, Secret, VirtualMachine, Service가 생성/갱신되는지 확인한다. |
| VM status reconcile | KubeVirt/CDI 상태 변화가 Kite VM status phase, power state, IP/node 정보로 반영되는지 확인한다. |
| Optional resource | optional ingress/service 설정이 꺼져 있거나 값이 비어 있을 때 불필요한 리소스를 만들지 않는지 확인한다. |
| Platform ingress | base domain과 TLS 설정에 따라 사용자/VM ingress가 올바른 host/path로 생성되는지 확인한다. |
| VM offer cleanup | 삭제되거나 비활성화된 offer가 기존/신규 VM 생성 정책에 기대대로 반영되는지 확인한다. |
| Failure status | Kubernetes API error, invalid spec, missing dependency가 status failure reason으로 남는지 확인한다. |
| Ownership | controller가 만든 리소스에 label/owner 정보가 붙어 cleanup과 조회가 가능한지 확인한다. |
| No gRPC command path | API-to-controller gRPC 명령 없이 CRD watch/reconcile만으로 동작하는지 확인한다. |

## SSH Gateway

목적: 외부 SSH 22번 연결을 받아 VM route 또는 host fallback으로 안전하게 전달한다.

최종 기준: VM `sshId` 사용자는 자기 VM으로 접속하고, VM route가 없는 host 계정은 host sshd fallback으로 로그인할 수 있다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| SSH banner | gateway가 `SSH-2.0-kite-gateway` 서버 식별자와 로그인 안내 banner를 노출하는지 확인한다. |
| Route lookup | login username이 `KiteVirtualMachine.spec.sshId`와 매칭되면 해당 VM route를 선택하는지 확인한다. |
| Password auth | VM password가 `spec.sshPasswordHash`와 맞을 때만 인증되는지 확인한다. |
| Public key auth | 지원되는 key 인증 흐름이 있다면 VM route와 Secret 기준으로만 허용되는지 확인한다. |
| Backend dial | gateway가 `vps-access-<vm>` Service의 22번으로 SSH session을 프록시하는지 확인한다. |
| VM not ready | VM route는 있지만 Service/VM이 준비되지 않았을 때 명확히 실패하고 연결이 매달리지 않는지 확인한다. |
| Host fallback | VM route가 없는 username은 configured host sshd 주소로 password/key 인증을 전달하는지 확인한다. |
| Fallback priority | host username과 VM sshId가 충돌하면 VM route가 우선되는지 확인한다. |
| Host key Secret | gateway host key Secret이 없으면 생성되고, k3s E2E에서는 실제 host sshd key fingerprint와 Secret 및 실행 중 gateway fingerprint가 일치하는지 확인한다. |
| Host sshd address env | `ghcr-install.sh`, `build-install.sh`, `uninstall.sh`, `build-clear.sh` 흐름에서 `KITE_GATEWAY_HOST_SSHD_ADDRESS`가 실제 host sshd 포트와 일치하는지 확인한다. |
| Timeout | backend VM 또는 host sshd가 응답하지 않을 때 제한 시간 안에 실패하는지 확인한다. |
| External port handoff | gateway Service가 외부 22번을 받고 host sshd는 선택 포트로 직접 접속 가능한지 확인한다. |

## Host SSHD Handoff

목적: gateway가 22번을 사용해야 할 때 기존 host sshd 접속 경로를 안전하게 이동하고 복원한다.

최종 기준: 설정 변경 전 검증, 사용자 재확인, 포트 점유 방지, sshd restart 검증, rollback, uninstall/build-clear 복원이 모두 작동한다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| 이미 22번이 아닌 sshd | host sshd가 이미 다른 포트를 쓰면 이동하지 않고 해당 포트를 gateway fallback으로 감지하는지 확인한다. |
| 포트 입력 | interactive 실행에서 선택 포트 의미를 설명하고 같은 포트를 한 번 더 입력해야 적용되는지 확인한다. |
| 포트 검증 | 숫자가 아니거나 1-65535 범위를 벗어난 포트를 거부하는지 확인한다. |
| 22번 거부 | gateway external port와 같은 포트를 host sshd 이동 포트로 선택하지 못하게 하는지 확인한다. |
| 점유 확인 | `ss`, `lsof`, `netstat` 순서로 포트 점유를 확인하고 점유 중이면 config를 바꾸지 않는지 확인한다. |
| 확인 불가 실패 | 점유 확인 도구가 없으면 안전하게 실패하는지 확인한다. |
| sshd syntax | 새 config를 적용하기 전 `sshd -t`가 실패하면 원본 config/state가 바뀌지 않는지 확인한다. |
| restart rollback | sshd restart 실패 시 이전 config와 state로 되돌리고 gateway fallback을 바꾸지 않는지 확인한다. |
| listen 확인 | restart 후 선택 포트가 실제 listen 중이 아니면 rollback되는지 확인한다. |
| socket activation | ssh.socket/sshd.socket이 active이면 자동 handoff를 거부하거나 skip하는지 확인한다. |
| restore worker | `uninstall.sh`/`build-clear.sh`가 gateway 삭제 전에 root 권한 restore worker를 예약하고 완료까지 확인하는지 확인한다. |
| 복원 결과 | `uninstall.sh` 또는 `build-clear.sh` 후 host sshd가 22번으로 돌아오고 선택 포트와 `/etc/kite/host-sshd` state가 사라지는지 확인한다. |

## Frontend

목적: 사용자가 웹 UI로 가입, 로그인, VM 생성/조회/접속 정보 확인, 관리자 설정을 수행한다.

최종 기준: mock 없이 production API 계약과 맞는 화면이 빌드되고, 주요 사용자 흐름이 깨지지 않는다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| Build/typecheck | `npm run typecheck`와 production build가 mock 없이 성공하는지 확인한다. |
| Routing | landing, signup, user dashboard, VM detail, console, admin route가 직접 URL 진입에서도 렌더링되는지 확인한다. |
| Auth state | login cookie/session 상태에 따라 protected route 접근, logout, redirect가 올바른지 확인한다. |
| Signup/login UI | form validation, API error 표시, 성공 후 이동이 실제 API 응답과 맞는지 확인한다. |
| User dashboard | VM 목록, 요약, toolbar, empty/loading/error 상태가 API 데이터 기준으로 표시되는지 확인한다. |
| VM offer 선택 | 관리자 offer 변경이 VM 생성 modal 선택지와 제한값에 반영되는지 확인한다. |
| VM create modal | disk/image/password/sshId 입력 검증과 API payload가 backend 계약과 일치하는지 확인한다. |
| VM detail | power state, status phase, 접속 정보, service/domain 정보가 CRD status와 맞게 표시되는지 확인한다. |
| Connection drawer | SSH 명령, host/domain, port, password/key 안내가 gateway 정책과 충돌하지 않는지 확인한다. |
| Console UI | console ticket 발급, 연결 실패, 권한 실패, 재시도 상태가 사용자에게 명확히 보이는지 확인한다. |
| Admin dashboard | 사용자/VM/offer/platform 설정 화면이 관리자 권한에서만 보이고 동작하는지 확인한다. |
| Mock mode | mock debug route와 mock store가 production build에 섞이지 않는지 확인한다. |
| Responsive/visual QA | 모바일과 데스크톱에서 주요 버튼, modal, table, drawer text가 겹치지 않는지 확인한다. |

## API Type, Store, Service Layer

목적: API handler와 controller 사이의 domain 변환이 CRD spec/status 계약을 깨지 않는다.

최종 기준: service/store/render 계층이 Kubernetes object shape를 정확히 만들고, edge case에서도 잘못된 CRD를 쓰지 않는다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| CRD Go types | Go struct field 이름과 json tag가 CRD schema 및 frontend API 타입과 일치하는지 확인한다. |
| Store GVR | `KiteUser`, `KiteVirtualMachine`, offer GVR이 group/version/resource와 scope를 정확히 쓰는지 확인한다. |
| Store create/update | fake dynamic client와 실제 cluster E2E에서 create, get, list, update status가 기대 object를 남기는지 확인한다. |
| Account service | password hash, access level, namespace naming, duplicate user 처리가 기대대로 동작하는지 확인한다. |
| Auth service | JWT/cookie 생성, 검증, 만료, 잘못된 secret 처리, password hash 비교가 정확한지 확인한다. |
| VM service | VM 생성 요청을 CRD spec으로 변환할 때 disk/image/offer/power/ssh 값이 손실되지 않는지 확인한다. |
| Offer service | offer record 변환, 기본 offer, 비활성 offer 제외, admin 변경 반영이 정확한지 확인한다. |
| Platform settings | base domain, TLS secret, redirect 설정의 default와 update merge가 정확한지 확인한다. |
| Guest login | VM 내부 로그인용 password/key 재료가 policy에 맞게 생성되고 노출 범위가 제한되는지 확인한다. |
| SSH key generation | keypair 생성 결과가 OpenSSH 호환이고 Secret/render/gateway에서 재사용 가능한지 확인한다. |

## Renderers

목적: controller가 만드는 Kubernetes YAML이 입력 spec에 맞는 `unstructured.Unstructured`로 렌더링된다.

최종 기준: renderer output이 Kubernetes에 적용 가능한 구조이고, controller E2E에서 실제 리소스로 만들어진다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| Namespace renderer | 사용자 namespace 이름, label, annotation이 정책과 일치하는지 확인한다. |
| QuotaPolicy renderer | 사용자 access level/offer 기준 ResourceQuota가 기대 limit을 갖는지 확인한다. |
| NetworkPolicy renderer | 사용자 namespace 기본 isolation과 필요한 ingress/egress 허용이 적용되는지 확인한다. |
| DataVolume renderer | golden image clone source, PVC size, storageClass, namespace가 VM spec과 맞는지 확인한다. |
| KubeVirt VM renderer | CPU/memory/disk/cloud-init/network/interface 설정이 offer와 VM spec을 반영하는지 확인한다. |
| VM Service renderer | SSH/web service 이름, selector, port가 gateway/frontend가 기대하는 값과 맞는지 확인한다. |
| Cloud-init renderer | user, authorized_keys, password policy, package/script 내용이 VM 접속 정책과 맞는지 확인한다. |
| Platform ingress renderer | base domain, TLS secret, service backend, host/path 규칙이 사용자 지정 domain 정책과 맞는지 확인한다. |
| HTTP/HTTPS redirect renderer | redirect middleware/ingress가 TLS 사용 여부와 충돌하지 않는지 확인한다. |
| Invalid input | 필수 값이 비어 있을 때 잘못된 Kubernetes object를 만들지 않고 명확히 실패하는지 확인한다. |

## Install, Deploy, Verify Scripts

목적: 운영자가 한 번의 명령으로 의존성 준비, 앱 배포, 검증을 반복 가능하게 수행한다.

최종 기준: 설치 스크립트는 이미 설치된 의존성을 안전하게 재사용하고, 누락된 의존성은 명확히 준비하거나 실패한다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| Longhorn install | opt-in일 때 Longhorn manifest 설치와 Ready 대기가 끝까지 수행되는지 확인한다. |
| KubeVirt install | KubeVirt operator/CR 설치와 Ready 대기가 idempotent하게 동작하는지 확인한다. |
| CDI install | CDI operator/CR 설치와 Ready 대기가 idempotent하게 동작하는지 확인한다. |
| all-in-one 순서 | host sshd handoff, storage, KubeVirt, CDI, golden image, app deploy, verify 순서가 지켜지는지 확인한다. |
| `ghcr-install.sh` install | GHCR 이미지를 pull하는 설치 흐름에서 이미지 태그와 manifest가 production default와 맞는지 확인한다. |
| `build-install.sh` install | 로컬 build/import 후 같은 manifest가 새 image tag로 배포되는지 확인한다. |
| component deploy | api/controller/gateway/frontend 단일 rebuild가 해당 Deployment만 갱신하고 다른 workload를 깨지 않는지 확인한다. |
| verify script | storage, dependencies, CRD, workload, golden image 검사가 실제 실패를 성공으로 숨기지 않는지 확인한다. |
| prompt helper | bool prompt가 기본값, 명시 env, interactive/noninteractive 모드를 일관되게 처리하는지 확인한다. |
| dry-run | dry-run이 위험 리소스를 바꾸지 않고 실행 계획만 출력하는지 확인한다. |

## Uninstall and Build Clear

목적: 설치된 Kite 리소스를 안전하게 제거하고, 필요하면 host sshd와 Longhorn host data를 복원/정리한다.

최종 기준: 기본 cleanup은 Kite 리소스만 제거하고 공유 인프라는 보존하며, 위험 삭제는 명시 확인이 있을 때만 수행한다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| 기본 `uninstall.sh` | Kite namespace, CRD, RBAC, Service, Deployment, StorageClass가 제거되는지 확인한다. |
| 기본 `build-clear.sh` | 개발 배포 리소스와 local image cleanup 선택이 분리되어 동작하는지 확인한다. |
| Golden image 삭제 | `DELETE_GOLDEN_IMAGE=true`일 때 DataVolume/PVC가 namespace 삭제 전에 제거되는지 확인한다. |
| User namespace cleanup | 테스트/사용자 namespace와 VM 관련 리소스가 고아로 남지 않는지 확인한다. |
| Longhorn data skip | Longhorn PV가 남아 있으면 host data cleanup을 건너뛰고 이유를 출력하는지 확인한다. |
| Longhorn data delete | 명시 확인과 PV 없음 조건에서 cleanup DaemonSet이 생성/완료/삭제되는지 확인한다. |
| Longhorn disk entry 제거 | Kite 전용 disk entry만 제거하고 공유 기본 disk를 통째로 삭제하지 않는지 확인한다. |
| Longhorn uninstall | `DELETE_LONGHORN=true`와 force 조합별로 Longhorn namespace/CR finalizer 처리가 기대대로 동작하는지 확인한다. |
| Host sshd restore | gateway 삭제 후 host sshd가 22번으로 돌아오고 restore worker 실패가 숨겨지지 않는지 확인한다. |
| Idempotent rerun | 이미 삭제된 상태에서 `uninstall.sh`/`build-clear.sh`를 다시 실행해도 성공하거나 안전하게 skip하는지 확인한다. |
| Remote cleanup | curl pipe `uninstall.sh`가 git checkout 없이 같은 cleanup 정책을 수행하는지 확인한다. |

## Cluster E2E Release Gates

목적: 배포 전에 실제 클러스터에서 build, deploy, API, CRD, controller, VM, frontend, gateway가 한 번에 통과한다.

최종 기준: 대상 클러스터별 `./test/all-test-*.sh`가 사람이 중간 값을 외우지 않아도 실행되고, VM Running까지 검증한다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| k3s E2E | buildx build, docker load, k3s import, host key fingerprint 재사용, overlay deploy, API, VM Running, frontend, gateway port-forward가 통과하는지 확인한다. |
| minikube E2E | profile 시작/컨텍스트 설정, local image load, overlay deploy, 같은 API/VM 검증이 통과하는지 확인한다. |
| generic k8s E2E | registry 미설정 시 빌드 전 실패하고, 설정 시 buildx push 후 cluster pull/deploy가 통과하는지 확인한다. |
| Dependency setup | `TEST_INSTALL_DEPS=true`일 때 Longhorn/KubeVirt/CDI/golden image 준비가 누락 환경에서 동작하는지 확인한다. |
| API health | `/api/v1/health`가 ok이고 CRD read check가 성공하는지 확인한다. |
| Signup/login | 테스트 사용자를 만들고 access level 설정 후 cookie jar login이 되는지 확인한다. |
| User reconcile | `KiteUser.status.phase=Ready`, namespace, quota, network policy가 생성되는지 확인한다. |
| VM create | `/api/v1/vms` 요청 후 VM CRD spec과 controller 생성 리소스가 모두 생기는지 확인한다. |
| VM Running | `KiteVirtualMachine.status.phase=Running`, `currentPowerState=On`, KubeVirt printableStatus `Running`을 확인한다. |
| Frontend response | `svc/kite-frontend` port-forward 후 index HTML이 응답하는지 확인한다. |
| Gateway response | `svc/kite-gateway` port-forward 또는 외부 LB에서 SSH banner/handshake가 되는지 확인한다. |
| Cleanup | E2E 종료 후 test user, VM, namespace, temporary overlay가 정리되는지 확인한다. |

## SSH Handoff Acceptance Gate

목적: 실제 k3s host에서 port 22를 gateway가 받고 host sshd가 선택 포트로 이동하는 위험 경로를 별도 검증한다.

최종 기준: `./test/all-test-k3s-ssh-handoff.sh`가 host 접속 경로를 잃지 않고 gateway와 host fallback을 모두 검증한다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| Direct host port | 선택한 host sshd 포트가 OpenSSH banner를 반환하고 host 계정 password login이 되는지 확인한다. |
| Gateway port 22 | 22번이 kite-gateway banner를 반환하고 host OpenSSH banner가 직접 노출되지 않는지 확인한다. |
| Host key fingerprint | gateway 22번과 이동한 host sshd 포트가 같은 SSH host key fingerprint를 노출하는지 확인한다. |
| Fallback login | VM route가 없는 host 계정이 gateway 22번을 통해 password login 되는지 확인한다. |
| Domain path | node IP뿐 아니라 운영 도메인이 같은 22번 gateway/fallback 경로로 동작하는지 확인한다. |
| Occupied port | 선택 포트가 점유되어 있으면 host sshd config, state, gateway env를 바꾸지 않는지 확인한다. |
| Recovery | 테스트 실패 후에도 선택 포트 또는 22번 중 하나로 host에 접속 가능한지 확인한다. |

## Frontend E2E와 Visual QA

목적: 실제 브라우저에서 사용자 화면과 API 계약이 함께 깨지지 않는지 확인한다.

최종 기준: Playwright 또는 브라우저 수동 QA가 주요 route와 상태를 보고, responsive layout 문제가 없음을 확인한다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| Direct routes | 새로고침/직접 URL 진입 시 React router와 nginx fallback이 정상 동작하는지 확인한다. |
| Auth redirects | 로그인 전/후 route 접근 제어와 redirect가 정상인지 확인한다. |
| VM creation flow | UI에서 offer 선택, VM 생성, 목록 반영, detail 진입까지 실제 API 또는 staging API로 확인한다. |
| Admin flow | 관리자 route, offer 관리, platform setting 변경이 일반 사용자에게 노출되지 않는지 확인한다. |
| Console flow | console 버튼, ticket 발급, 연결 실패/성공 표시를 확인한다. |
| Mobile layout | 모바일 폭에서 header, dashboard, modal, drawer, terminal UI가 겹치지 않는지 확인한다. |
| Production build serve | nginx image 안의 built asset이 `/api/v1` base path와 함께 동작하는지 확인한다. |

## Documentation and Examples

목적: 운영자와 개발자가 문서와 예제를 따라 실제 설치/테스트/삭제를 수행할 수 있다.

최종 기준: README, design docs, example CR이 현재 코드와 맞고, 문서의 명령은 실제로 실행 가능한 형태다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| Root README | 아키텍처, install, SSH gateway, cleanup 설명이 실제 스크립트 동작과 일치하는지 확인한다. |
| build README | `ghcr-install.sh`, `build-install.sh`, `uninstall.sh`, `build-clear.sh` 경계와 wrapper 위임 설명이 실제 경로와 일치하는지 확인한다. |
| deploy docs | production install, Longhorn/KubeVirt/CDI, host sshd handoff 설명이 최신 환경변수와 일치하는지 확인한다. |
| dev docs | component rebuild, local image loading, frontend build arg 설명이 실제 script option과 일치하는지 확인한다. |
| frontend docs | API spec, design convention, mock 설명이 현재 UI/API 타입과 어긋나지 않는지 확인한다. |
| examples | `build/examples`의 KiteUser/VM 예제가 현재 CRD schema로 apply 가능한지 확인한다. |
| commit convention | commit message 규칙이 AGENTS.md와 docs 문서에서 충돌하지 않는지 확인한다. |
| test docs | `test/Readme.md`와 이 문서가 서로 역할이 겹치지 않고 최신 gate 명령을 안내하는지 확인한다. |

## Retired Proto/gRPC Area

목적: 은퇴한 API-to-controller gRPC 설계가 실수로 production 경로에 재도입되지 않게 한다.

최종 기준: protobuf/gRPC 파일은 새 기능 테스트 대상이 아니며, 현재 API와 controller는 CRD 기반으로만 결합된다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| Proto drift guard | 새 protobuf 생성물이나 API-to-controller RPC가 추가되지 않았는지 확인한다. |
| Controller command path | VM 생성/전원 변경이 gRPC 호출 없이 CRD spec 변경과 reconcile로만 동작하는지 확인한다. |
| Documentation | 문서가 retired proto를 active 설계처럼 안내하지 않는지 확인한다. |

## Observability and Failure Triage

목적: 테스트가 실패했을 때 원인을 빠르게 찾을 수 있는 정보를 남긴다.

최종 기준: 실패한 테스트는 마지막 상태, 관련 Kubernetes object, pod logs, event, 사용한 image tag를 사람이 바로 확인할 수 있게 출력한다.

| 소분류 | 테스트해야 하는 것 |
| --- | --- |
| Image tag trace | E2E 로그에 build한 image tag와 배포된 pod image가 같이 남는지 확인한다. |
| GHCR build workflow | `stage` push와 PR에서 publish workflow와 같은 image matrix, Dockerfile, build args로 네 이미지를 `push:false` buildx build 하고 결과물을 버리는지 확인한다. |
| Pod diagnostics | rollout 실패 시 pod describe, logs, events를 출력하는지 확인한다. |
| CRD diagnostics | VM/User 실패 시 CRD spec/status와 관련 resource 목록을 출력하는지 확인한다. |
| Storage diagnostics | DataVolume/PVC/PV 실패 시 CDI event와 Longhorn volume 상태를 출력하는지 확인한다. |
| Gateway diagnostics | SSH 실패 시 gateway env, Service endpoint, route lookup 대상 정보를 출력하는지 확인한다. |
| Cleanup diagnostics | cleanup 실패 시 남은 namespace, CRD, PV, host sshd state를 출력하는지 확인한다. |
| Log location | 긴 원격 테스트는 `/tmp/kite-acceptance` 같은 고정 위치에 로그를 남기는지 확인한다. |

## 변경 유형별 최소 테스트 선택

| 변경 유형 | 반드시 고려할 테스트 |
| --- | --- |
| API route 변경 | handler/service 단위 테스트, auth/권한 테스트, 관련 E2E API assertion |
| Controller reconcile 변경 | fake client reconcile 테스트, idempotency 테스트, 실제 클러스터 resource/status E2E |
| CRD schema 변경 | schema apply 테스트, API create/update 테스트, controller status E2E |
| Renderer 변경 | renderer output shape 테스트, controller가 실제 apply하는 E2E |
| Gateway 변경 | gateway unit 테스트, SSH handoff acceptance, fallback/password/banner E2E |
| Host sshd script 변경 | bash syntax, occupied port, rollback, restore-after-clean 실제 host 테스트 |
| Storage 변경 | Longhorn/CDI wait, golden image import, VM disk clone, cleanup safety 테스트 |
| Frontend UI 변경 | typecheck/build, route/component E2E, visual QA, backend contract 확인 |
| Dockerfile/build 변경 | GHCR build workflow, buildx build, local load/push, deployed image tag 확인 |
| install/uninstall 변경 | interactive/noninteractive, idempotency, failure propagation, 실제 install-uninstall roundtrip |
| docs/example 변경 | 명령 실행 가능성, CRD apply 가능성, 최신 환경변수 일치 확인 |
