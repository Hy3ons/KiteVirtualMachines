# Kite TODO

## 현재 결정

SSH 진입 라우팅은 Kubernetes 내부에서 동작하는 `kite-gateway` 방식으로 고정한다.
기존 host Linux 계정 생성 방식은 기본 런타임에서 제거한다.

최종 목표 구조는 다음과 같다.

```text
사용자 SSH client
  -> 노드 public IP:22
  -> Kubernetes Service / TCP ingress
  -> Kubernetes 안의 kite-gateway
  -> Kite CRD/Secret 상태를 보고 인증과 라우팅 결정
  -> vps-access-<vmName>.<namespace>.svc.cluster.local:22 로 SSH 프록시
  -> VM 내부 sshd
```

이 구조에서는 인증, 라우팅, DNS, VM 조회가 모두 Kubernetes 내부에서 처리된다.
호스트 OS에는 사용자 Linux 계정, `/home/<sshId>`, custom shell, resolver 설정을 만들지 않는다.
호스트 OS는 관리용 sshd 포트만 2222 같은 별도 포트로 옮기고, 외부 22번은 Kubernetes Service가 `kite-gateway`로 전달한다.

## 목표 아키텍처: kite-gateway

### 컴포넌트 이름

새 컴포넌트 이름은 `kite-gateway`로 한다.

예상 경로:

- `kite/cmd/kite-gateway`
- `kite/internal/gateway`
- `kite/Dockerfile.gateway`
- `build/kite/gateway.yaml`

이전 node/host agent 방식, host account reconcile, host DNS mutation은 기본 설계에서 제외한다.

## 배포 방식

기본 목표는 `kite-gateway`을 일반 Deployment로 실행하는 것이다.
가능하면 control plane 또는 Kite 관리 노드에만 배치한다.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kite-gateway
  namespace: kite
spec:
  replicas: 1
  template:
    spec:
      serviceAccountName: kite-control-plane
      nodeSelector:
        node-role.kubernetes.io/control-plane: "true"
      containers:
        - name: kite-gateway
          image: ghcr.io/hy3ons/kite-gateway:latest
          ports:
            - name: ssh
              containerPort: 2222
              protocol: TCP
```

앞단에는 Service를 둔다.

기본값은 LoadBalancer Service로 외부 22번을 받고, pod 내부 `targetPort: 2222`로 넘기는 구조다.
host OS의 기존 `sshd`는 운영자가 미리 2222 같은 별도 포트로 옮긴다.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: kite-gateway
  namespace: kite
spec:
  type: LoadBalancer
  selector:
    app: kite-gateway
  ports:
    - name: ssh
      port: 22
      targetPort: 2222
```

### 기본 선택: K3s ServiceLB / LoadBalancer

K3s의 ServiceLB 또는 MetalLB를 사용해 `type: LoadBalancer` Service를 만든다.
이 방식을 기본으로 둔다.

이유:

- `kite-gateway` pod가 호스트 네트워크와 계정에 접근하지 않는다.
- Kubernetes Service가 진입점을 고정한다.
- pod 재시작과 rolling update가 쉬워진다.
- `*.svc.cluster.local` DNS를 기본 kube-dns로 바로 사용할 수 있다.

예상 Service:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: kite-gateway
  namespace: kite
spec:
  type: LoadBalancer
  selector:
    app: kite-gateway
  ports:
    - name: ssh
      port: 22
      targetPort: 2222
```

### 대안 A: NodePort 22

Kubernetes 기본 NodePort 범위는 보통 `30000-32767`이다.
NodePort로 22번을 직접 쓰려면 K3s API server 옵션을 바꿔야 한다.

```text
--service-node-port-range=1-65535
```

이 방식은 가능하지만 기본값으로 두지 않는다.
클러스터 설정 변경이 필요하고, 호스트 SSHD의 22번 포트와 충돌할 수 있다.

### 대안 B: hostPort fallback

Service 기반 진입이 막히면 마지막 fallback으로 `hostPort`를 검토한다.

```yaml
ports:
  - name: ssh
    containerPort: 2222
    hostPort: 22
    protocol: TCP
```

`hostNetwork: true`와 `privileged: true`는 기본 설계에서 제외한다.

## 호스트 SSHD 처리

노드의 기존 `sshd`도 보통 `0.0.0.0:22`를 사용한다.
`kite-gateway`이 외부 22번을 받으려면 host sshd를 2222 같은 다른 포트로 옮겨야 한다.

설치 스크립트가 무조건 `/etc/ssh/sshd_config`를 수정하면 위험하다. 관리 SSH가 끊길 수 있기 때문이다.

따라서 기본 방침:

- install/dev 스크립트는 host sshd 변경 전 반드시 확인한다.
- `KITE_MANAGE_HOST_SSHD=true`일 때만 비대화식으로 host sshd를 변경한다.
- `KITE_RESTORE_HOST_SSHD=true`일 때만 비대화식으로 clear/uninstall 원복을 수행한다.
- 개발과 운영 모두 `ssh <sshId>@host`를 목표 진입 방식으로 한다.
- 22번 노출은 LoadBalancer/ServiceLB/TCP ingress를 먼저 검토한다.
- Linux/systemd/OpenSSH가 없으면 host sshd 변경은 건너뛴다.
- host sshd가 이미 22번을 쓰지 않으면 변경하지 않는다.
- 변경 시 `/etc/kite/host-sshd`에 원본 config와 state를 남기고, clear/uninstall에서 이 백업으로 원복한다.

수동으로도 같은 작업을 할 수 있다.

```sh
sudo sed -i 's/^#\?Port .*/Port 2222/' /etc/ssh/sshd_config
sudo systemctl restart ssh
```

초기 개발 단계에서도 host sshd를 2222로 옮긴 뒤 `kite-gateway`을 22번에 붙여 테스트한다.

```sh
ssh asdf@hy3on.site
```

## 인증 계획

### 1차 구현: Kite 상태 기반 password 인증

stock `sshd`를 컨테이너에서 실행하기보다 Go SSH server를 직접 구현하는 방향이 좋다.

사용 후보:

- `golang.org/x/crypto/ssh`

기본 흐름:

```mermaid
sequenceDiagram
    participant Client as ssh 사용자
    participant SVC0 as Service/TCP ingress
    participant Gateway as kite-gateway
    participant API as Kubernetes API
    participant Secret as VM SSH Secret
    participant SVC as vps-access Service
    participant VM as KubeVirt VM sshd

    Client->>SVC0: ssh <sshId>@host
    SVC0->>Gateway: TCP 22 -> targetPort 2222
    Gateway->>Gateway: SSH handshake + username/password/publickey
    Gateway->>API: informer cache에서 spec.sshId 기준 KiteVirtualMachine 조회
    Gateway->>API: VM 상태와 삭제 여부 확인
    Gateway->>API: VM SSH key Secret 조회
    Gateway->>Gateway: password 검증
    Gateway->>SVC: vps-access-<vm>.<namespace>.svc.cluster.local:22 연결
    Gateway->>VM: Kite-managed key로 VM 내부 SSH 연결
    Gateway->>Client: SSH session/exec/pty 스트림 프록시
```

HTTP API는 요청 body로 `sshPassword`를 받지만, CRD에는 평문을 저장하지 않는다.
API는 즉시 `spec.sshPasswordHash`로 변환하고, gateway는 runtime password salt로 hash를 검증한다.
`kite-gateway`은 etcd를 직접 읽지 않는다. Kubernetes API와 informer cache를 통해 CRD/Secret/Service 상태를 읽는다.

운영 전 개선 방향:

- password는 단방향 hash로만 저장한다.
- `kite-gateway`은 hash 검증만 수행한다.
- CRD status에는 password나 private key를 절대 넣지 않는다.

### 2차 구현: Public key 인증

password 인증이 안정화되면 public key 인증도 지원한다.

목표 상태:

- `KiteVirtualMachine.spec.sshId`
- `KiteVirtualMachine.status.sshKeySecretName`
- Secret data:
  - `id_rsa`: `kite-gateway`이 VM 내부로 접속할 때 사용하는 private key
  - `id_rsa.pub`: VM cloud-init에 주입한 public key
  - `authorized_keys`: 외부 사용자가 `kite-gateway`에 접속할 때 허용할 public key 목록

외부 사용자는 `kite-gateway`에 public key로 인증하고, `kite-gateway`은 Kite가 관리하는 private key로 VM 내부에 접속한다.

## 라우팅 계획

### etcd 조회 원칙

`kite-gateway`이 "etcd를 뒤진다"는 것은 구현상 Kubernetes API를 통해 desired/current state를 읽는다는 뜻으로 고정한다.

직접 etcd client를 쓰지 않는 이유:

- Kubernetes API server의 인증, 인가, watch cache, validation을 활용할 수 있다.
- CRD schema와 RBAC 권한 모델을 그대로 따른다.
- etcd endpoint와 인증서를 애플리케이션에 배포하지 않아도 된다.
- 장애 시 Kubernetes client-go informer 재동기화 모델을 사용할 수 있다.

따라서 라우팅 테이블은 informer가 메모리에 유지한다.

```text
sshId -> {
  vmNamespace,
  vmName,
  serviceName: vps-access-<vmName>,
  secretName: status.sshKeySecretName,
  phase,
  deletionTimestamp
}
```

요청이 들어올 때마다 API server를 매번 직접 조회하지 않고, informer cache를 우선 사용한다.
Secret/private key처럼 민감한 데이터는 필요 시 get하거나 별도 informer cache에 넣되, 로그에는 절대 출력하지 않는다.

### VM 조회 규칙

`kite-gateway`은 SSH login username을 기준으로 VM을 찾는다.

v1 규칙:

```text
spec.sshId == SSH login username
```

예:

```sh
ssh asdf@hy3on.site
```

`kite-gateway`은 모든 live `KiteVirtualMachine` 중 `spec.sshId == "asdf"`인 VM을 찾는다.

필수 제약:

- live VM 전체에서 `sshId`는 전역 유일해야 한다.
- `kite-api`는 VM 생성 시 같은 `sshId`를 사용하는 live VM이 있으면 생성 요청을 거절해야 한다.

나중에 확장할 수 있는 형식:

- `ssh <vmName>@host`
- `ssh <sshId>.<vmName>@host`
- `ssh <vmName>+<sshId>@host`

하지만 v1은 단순하게 `sshId` 전역 유일 모델로 간다.

### 노드 라우팅

싱글 노드 기준:

- `kite-gateway` pod는 어느 VM Service에도 접근 가능하다.
- `status.nodeName`을 강하게 사용할 필요가 없다.

멀티 노드 확장 시:

- 우선은 ClusterIP Service 라우팅을 계속 사용한다.
- 스토리지/네트워크 구조 때문에 같은 노드 라우팅이 필요해지면 `status.nodeName`을 사용한다.
- 정말 노드 로컬 라우팅이 필요해지면 DaemonSet 모드로 바꿔 자기 노드의 VM만 담당하게 만들 수 있다.
- 하지만 v1 기본값은 Deployment + Service 구조로 둔다.

멀티 노드 복잡도는 v1에서 넣지 않는다.

## SSH 프록시 요구사항

VS Code Remote SSH까지 지원하려면 단순 TCP proxy만으로는 부족할 수 있다.

`kite-gateway`이 외부 SSH를 terminate하고 내부 VM으로 새 SSH 연결을 여는 구조라면 다음 SSH channel/request를 처리해야 한다.

- `session`
- `exec`
- `shell`
- `pty-req`
- `env`
- `window-change`
- `signal`
- exit status 전달
- stdout/stderr 전달
- keepalive/global request 처리

필수 테스트:

```sh
ssh asdf@hy3on.site
ssh asdf@hy3on.site whoami
ssh asdf@hy3on.site 'echo hello'
```

VS Code Remote SSH도 별도로 테스트해야 한다.

## 구현 방식 후보

### 후보 A: Go Gateway

`kite-gateway`이 직접 SSH server 역할을 한다.

장점:

- Kubernetes CRD/Secret 기반 인증을 직접 구현하기 쉽다.
- host Linux 계정이 필요 없다.
- stock sshd 설정 파일을 다루지 않아도 된다.
- audit log, admin 정책, 권한 체크를 나중에 붙이기 좋다.

단점:

- SSH protocol channel proxy 구현이 필요하다.
- VS Code Remote SSH 호환성을 꼼꼼히 확인해야 한다.

우선 추천은 후보 A다.

### 후보 B: 컨테이너 내부 stock sshd

pod 안에서 Linux `sshd`를 실행하고, pod 내부 계정을 동기화한다.

장점:

- OpenSSH 호환성이 좋다.
- VS Code Remote SSH 호환성은 상대적으로 안전하다.

단점:

- pod 내부 계정 관리가 다시 필요하다.
- AuthorizedKeysCommand, ForceCommand, PAM 설정 등 운영 복잡도가 생긴다.
- 결국 `sshd_config` 템플릿과 계정 reconcile을 또 만들어야 한다.

후보 B는 Go gateway이 VS Code 호환성에서 막힐 때 fallback으로 검토한다.

## Kubernetes 리소스 변경

새로 추가할 파일:

- `kite/cmd/kite-gateway`
- `kite/internal/gateway`
- `kite/Dockerfile.gateway`
- `build/kite/gateway.yaml`

수정할 파일:

- `build/kite/kustomization.yaml`
- `build/kite/rbac.yaml`
- `build/dev/dev.sh`
- `build/deploy/scripts/install-all.sh`
- `build/deploy/scripts/verify.sh`
- `Readme.md`
- `build/dev/README.md`
- `build/deploy/README.md`

## RBAC 원칙

gateway은 가능하면 read-only 권한만 갖는다.

필요 권한:

```yaml
- apiGroups: ["hy3ons.github.io"]
  resources: ["kitevirtualmachines"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["secrets", "services"]
  verbs: ["get", "list", "watch"]
```

`kite-gateway`이 CRD를 생성/수정/삭제하지 않게 한다.
SSH 접속 시 필요한 Secret 읽기 권한은 namespace 범위를 좁힐 수 있으면 좁힌다.

## 설치 스크립트 계획

### dev.sh

추가할 일:

- `kite-gateway` 이미지 build
- k3s/containerd 또는 minikube/kind/k3d에 이미지 load
- kustomize image override 추가
- Deployment rollout wait 추가
- Service 생성 확인 추가

초기 기본값:

```text
KITE_GATEWAY_ENABLED=true
KITE_GATEWAY_SERVICE_TYPE=LoadBalancer
KITE_GATEWAY_EXTERNAL_PORT=22
KITE_GATEWAY_CONTAINER_PORT=2222
```

22번 포트는 host sshd와 충돌 가능성이 크므로 설치 전 host sshd를 2222로 옮긴다.

### install.sh

운영 설치 전 체크:

- Service 타입이 LoadBalancer면 ServiceLB/MetalLB 사용 가능 여부 확인
- NodePort 22를 대안으로 요청하면 `service-node-port-range`가 22를 포함하는지 안내
- hostPort 22 fallback을 요청하면 노드에서 22번 포트 사용 여부 확인
- 22번이 사용 중이면 설치 중단
- host sshd 포트 변경 안내 출력하되 자동 변경은 하지 않음

## cleanup 계획

`kite-gateway` 방식으로 전환하면 cleanup은 단순해진다.

제거 대상:

- `kite-gateway` Deployment
- `kite-gateway` Service
- gateway image
- Kite CRD
- Kite namespace
- 사용자 namespace
- Kite Longhorn tag 또는 dedicated disk entry

더 이상 필요 없어지는 것:

- host Linux 계정 삭제
- `/home/<sshId>` 삭제
- `/var/lib/kite/accounts` 삭제
- host DNS resolver 원복
- custom shell 삭제

이 점이 `kite-gateway` 방식의 가장 큰 장점이다.

## 기존 host agent 처리

이전 node/host agent 기반 SSH 경로는 기본 런타임에서 제거한다.
이전 방식이 만들던 host Linux 계정, proxy shell, host DNS resolver 변경은 더 이상 Kite 설치/삭제 흐름의 책임이 아니다.

정리 완료 기준:

- `kite-gateway`이 password auth를 처리한다.
- `kite-gateway`이 VM 내부 SSH key Secret으로 VM sshd에 접속한다.
- `ssh <sshId>@host` 진입점은 Kubernetes Service가 받는다.
- clear/uninstall이 host 계정, `/home/<sshId>`, `/var/lib/kite/accounts`, host DNS를 만지지 않는다.
- `build/kite` 기본 manifest에 host agent DaemonSet이 없다.
- 코드와 문서에 이전 node/host agent 기본 경로가 남아 있지 않다.

## 구현 마일스톤

### M1: 설계 고정

- [x] `kite-gateway` 이름 확정
- [x] gateway 기본 포트와 Service 타입 결정
- [x] LoadBalancer/ServiceLB 22번 기본값 확정
- [x] NodePort는 대안으로만 유지
- [x] host sshd 포트 변경 정책 문서화
- [ ] `sshId` 전역 유일 정책 확정
- [x] host agent 기본 런타임 제거

### M2: Gateway 기본 서버

- [x] `kite/cmd/kite-gateway` 생성
- [x] Go SSH server 실행
- [x] Kubernetes dynamic client 연결
- [x] KiteVirtualMachine informer 추가
- [x] `spec.sshId` 기준 route table 생성

### M3: 인증

- [x] password auth callback 구현
- [x] KiteVM 상태 기반 인증 구현
- [x] 삭제 중인 VM 거절
- [x] Secret/Service 없는 VM 거절
- [x] duplicate `sshId` 감지
- [x] `spec.sshPassword` 평문 저장 제거
- [x] `spec.sshPasswordHash` 기반 gateway 인증 적용

### M4: VM 내부 연결

- [x] `status.sshKeySecretName` 조회
- [x] private key Secret 조회
- [x] `vps-access-<vm>.<namespace>.svc.cluster.local:22` dial
- [x] VM 내부 sshd에 `spec.sshId`로 접속
- [x] Service/Secret/VM sshd 미준비 시 명확한 에러 출력

### M5: SSH channel proxy

- [x] interactive shell proxy 기본 channel forwarding
- [x] exec command proxy 기본 channel forwarding
- [x] PTY 요청 proxy 기본 request forwarding
- [x] env/window-change/signal 기본 request forwarding
- [ ] exit status 전달 검증
- [ ] VS Code Remote SSH 테스트
- [ ] direct-tcpip, keepalive, agent forwarding 필요 여부 검증

### M6: Manifest와 설치 통합

- [x] `build/kite/gateway.yaml` 추가
- [x] Deployment + Service 리소스 추가
- [x] `build/kite/kustomization.yaml`에 추가
- [x] `build/dev/dev.sh`에 image build/load 추가
- [x] `install-all.sh` rollout wait 추가
- [x] `verify.sh` 검증 추가
- [x] README 업데이트
- [x] 운영용 host key Secret mount 추가
- [x] install/dev 스크립트에서 gateway host key Secret 자동 생성
- [ ] Service type/port를 install/dev 환경변수로 패치 가능하게 개선

### M7: host agent 제거

- [x] host agent DaemonSet manifest 제거
- [x] host account reconcile 코드 제거
- [x] custom shell renderer 제거
- [x] Linux user 생성/삭제 로직 제거
- [x] dev/clear/uninstall에서 host DNS mutation 제거

## 열려 있는 질문

- SSH login username을 `sshId` 하나로 고정할 것인가?
- 관리자 SSH 접속은 어떻게 설계할 것인가?
- VM password hash salt 회전 시 기존 VM 접속 password를 어떻게 재설정할 것인가?
- public key auth를 v1에 포함할 것인가?
- 세션 감사 로그를 남길 것인가?
- VS Code Remote SSH 호환성을 어느 수준까지 보장할 것인가?
- ServiceLB/LoadBalancer 22번 노출이 안 되는 환경에서 어떤 fallback을 허용할 것인가?
- `kite-gateway`을 control plane에 고정할 것인가, 일반 Kite namespace Deployment로 둘 것인가?
- TCP ingress를 쓸 경우 nginx ingress stream 설정을 Kite 설치 범위에 포함할 것인가?

## 현재 알려진 문제

- VMware nested virtualization이 꺼져 있으면 KubeVirt가 `/dev/kvm`을 못 보고 `ErrorUnschedulable`이 된다.
- 테스트만 할 때는 KubeVirt `useEmulation=true`로 우회할 수 있지만 매우 느리다.
- Longhorn 단일 디스크 환경에서는 `/mnt/kite-longhorn`을 같은 파일시스템의 두 번째 Longhorn disk로 등록하면 안 된다.
- host 계정 생성 방식은 제거했지만, 기존 테스트 노드에 수동으로 남은 Linux 계정은 별도 수동 정리가 필요할 수 있다.
- `kite-gateway` 방식은 host OS 오염을 크게 줄인다.
- Go SSH server가 VS Code Remote SSH와 완전히 호환되는지 검증이 필요하다.
- Service 기반 22번 노출은 클러스터 네트워크 구현체에 따라 설정 방식이 달라질 수 있다.
