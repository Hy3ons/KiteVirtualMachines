# Kite Backend TODO List

이 문서는 프론트엔드 요구사항에 맞추어 백엔드(API Server 및 Controller) 쪽에서 추가/수정해야 할 작업 내역을 관리합니다.

## 1. CRD (CustomResourceDefinition) 업데이트
- [x] **KiteVirtualMachine 스펙 확장**:
  - `spec.domainPrefix`: 인그레스용 도메인 프리픽스 필드 추가
  - `spec.sshId`, `spec.sshPassword`: Cloud-init용 SSH 접속 정보 필드 추가 (또는 Secret 연동)
  - `status.domain`: 최종 조합된 전체 도메인 주소를 내려주기 위한 필드 추가
  - `status.observedGeneration`: controller가 처리한 CRD generation 기록

## 2. Global Configuration & Initial Setup (초기 셋업 플로우)
- [ ] **초기 설정(Initial Setup) API 구현**:
  - 관리자가 플랫폼 설치 직후 최초 접속 또는 설정 메뉴에서 입력한 **베이스 도메인**과 **HTTPS 인증서** 데이터를 수신하는 엔드포인트.
  - 베이스 도메인은 `etcd` (또는 시스템 전역 ConfigMap)에 영구 저장.
  - HTTPS 인증서 데이터는 쿠버네티스 클러스터 내 `kube-system/global-tls-secret` (타입: `kubernetes.io/tls`) Secret으로 자동 생성/저장.
- [ ] **전역 도메인 변경에 따른 대규모 Reconcile 로직 (중요)**:
  - 도메인이 변경될 경우, 기존에 배포된 수많은 `KiteVirtualMachine` 리소스들의 Ingress 설정값도 새로운 도메인에 맞게 모조리 바뀌어야 합니다. 
  - 이를 위해 도메인 변경 이벤트 발생 시 컨트롤러가 기존 CRD들을 싹 긁어와 Reconcile 큐에 넣고 순차적으로 Ingress Host를 업데이트하도록 아키텍처를 잡아야 합니다.
- [ ] **`kite/internal/render` Ingress 동적 렌더링 및 HTTPS 고도화**:
  - [x] `kite/kite-runtime-config` ConfigMap의 `baseDomain`을 읽어 `spec.domainPrefix + 베이스도메인` 형태로 Ingress Host를 매핑.
  - Ingress controller의 default certificate가 `kube-system/global-tls-secret`을 보도록 구성하고, VM Ingress는 `websecure` TLS 라우팅을 사용하도록 흐름 개선.

## 3. Controller 로직 수정 (`kite-controller`)
- [x] **CRD -> KubeVirt 생성 로직**: `KiteVirtualMachine` reconcile에서 DataVolume, cloud-init Secret, KubeVirt VirtualMachine, Service, 선택적 Ingress를 server-side apply.
- [x] **DataVolume import 방식**: `kite/ubuntu-22.04` golden DataVolume/PVC를 먼저 만들고, VM별 DataVolume은 해당 PVC를 source로 참조한다. storageClassName은 클러스터 기본값을 쓰도록 생략.
- [x] **SSH Service ClusterIP 전환**: VM SSH Service는 `vps-access-<vmName>` 고정 이름의 `ClusterIP` Service로 생성하고 NodePort는 사용하지 않는다.
- [x] **Cloud-init 설정 주입**: `spec.sshId`와 `spec.sshPassword`를 Cloud-init Secret 템플릿에 주입하여 VM 구동 시 SSHD가 정상 동작하도록 처리.
- [x] **KubeVirt 상태 -> CRD status 갱신**: KubeVirt `VirtualMachine` informer가 실제 VM 상태를 읽어 `status.phase`, `status.currentPowerState`, `status.domain`을 갱신.
- [x] **host 계정 reconcile**: `cmd/kite-host-agent`가 `KiteVirtualMachine`, VM SSH key Secret, `vps-access-<vmName>` Service DNS를 보고 host Linux 계정과 proxy shell을 맞춘다.
- [x] **상태 drift 재조정**: KubeVirt VM의 `spec.running`이 `KiteVirtualMachine.spec.powerState`와 다르면 CRD 기준으로 다시 reconcile.
- [x] **고아 리소스 방지 삭제 흐름**: `spec.delete=true` 또는 CRD 직접 삭제 시 KubeVirt VM, Service, Ingress, Secret, DataVolume을 정리.
- [x] **DataVolume 상태 반영**: DataVolume ready/progress/failure를 별도 informer로 감지하고 `KiteVirtualMachine.status.dataVolumePhase`, `status.dataVolumeProgress`, `status.dataVolumeMessage`에 반영.

## 4. API Server 로직 수정 (`kite-api`)
- [ ] **디스크(Storage) Quota 검증 로직**: VM 생성 요청 시 프론트에서 넘어온 `disk` 용량이 해당 유저의 `access_level` 한도를 초과하지 않는지 백엔드 단에서 방어하는 검증 로직 추가.
- [ ] **VM 생성 API**: 변경된 스펙(Domain Prefix, SSH 정보 등)을 모두 받아 CRD를 생성하도록 수정.
- [x] **VM 목록 반환 API**: 프론트엔드에서 VM 목록을 그릴 때 사용할 VM spec/status와 최종 도메인 정보를 함께 리턴하도록 DTO를 구성한다. NodePort는 사용하지 않는다.

## 5. 🏗️ Kite-Node-Agent (DaemonSet) 아키텍처 및 데이터 흐름

단순한 TODO 리스트가 아닌, 실제 프로덕션 레벨의 SSH 무혈입성 아키텍처 핵심 설계도입니다.
이 문서는 향후 구현 시 설계 지침(Blueprint)으로 사용됩니다.

### 5.1. 전제 조건 및 Controller의 역할
1. 사용자가 웹 UI에서 VM을 신청하면, Go 백엔드(`kite-controller`)는 `KiteVirtualMachine` CRD를 생성합니다.
2. CRD `spec`에는 발급할 유저 ID(예: `apple`)와 암호화된 비밀번호(해시값)가 보관됩니다.
3. KubeVirt VM이 특정 워커 노드(예: `node-1`)에 배치(Scheduling)되면, CRD의 `status.nodeName`에 `node-1`이 기록됩니다.

### 5.2. Node-Agent의 감시 및 리컨사일(Reconcile)
각 워커 노드마다 데몬셋(DaemonSet) 형태로 떠 있는 `kite-node-agent`는 쿠버네티스 Informer를 통해 `KiteVirtualMachine` CRD의 상태 변화를 실시간으로 구독합니다.

- **조건 검사 필터링:** 이벤트가 들어왔을 때, `vm.Status.NodeName`이 **현재 에이전트 자신이 돌고 있는 노드의 이름**과 일치할 때만 작업을 시작합니다.
- **권한:** 호스트 OS 수준의 제어를 위해 `privileged: true` 및 호스트 볼륨 마운트(`/host/etc`, `/host/home`) 권한을 가진 채로 동작합니다.

### 5.3. 호스트 OS 내 격리된 파일 시스템 및 계정 구조 (핵심)
보안과 격리를 최우선으로 하여, 호스트 OS에 아래와 같은 구조를 동적으로 생성합니다. 관리자 공용 폴더가 아닌, 철저히 유저별 홈 디렉토리 안에 비밀키와 쉘 스크립트를 숨겨둡니다.

```text
/host/ (실제 Host OS의 루트)
├── etc/
│   └── passwd  -> 에이전트가 유저(apple)를 추가하고, 기본 쉘을 /home/apple/custom-shell.sh 로 지정함
└── home/
    └── apple/  -> 생성된 유저의 전용 홈 디렉토리
        ├── .ssh/
        │   └── id_rsa  -> [Private Key] 에이전트가 직접 생성. 권한 600으로 소유자 외 절대 접근 불가
        └── custom-shell.sh  -> [핵심 프록시 쉘] 유저 로그인 시 강제 실행되는 통로 스크립트
```

### 5.4. 핵심 메커니즘 및 스크립트 렌더링

#### A. Custom Shell Script (`custom-shell.sh`)
에이전트는 계정을 파자마자 아래 내용의 스크립트를 유저의 홈 폴더에 렌더링하고 `chmod +x`를 부여합니다. 유저가 호스트에 SSH로 접근해 비밀번호를 치면, bash/zsh 대신 이 스크립트가 실행되며 유저를 VM 안으로 빨려 들어가게 만듭니다.

```bash
#!/bin/bash
# /home/{username}/custom-shell.sh

# 1. 자신만의 격리된 Private Key 경로
PRIVATE_KEY="/home/{username}/.ssh/id_rsa"

# 2. 쿠버네티스 내부망의 VM 타겟 주소 (ClusterIP Service DNS)
# 서비스 이름은 vps-access-{vmName}, 네임스페이스는 {vmNamespace} 형태가 됩니다.
VM_TARGET="ubuntu@vps-access-{vmName}.{vmNamespace}.svc.cluster.local"

# 3. 호스트에 떨어지자마자 키를 쥐고 VM으로 패스스루 (StrictHostKeyChecking=no로 프롬프트 무시)
exec ssh -i "$PRIVATE_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "$VM_TARGET"
```

#### B. 호스트 계정 세팅 (`useradd`)
에이전트는 내부적으로 다음 명령어(또는 동일한 CGO/Syscall)를 실행하여 OS에 계정을 박아 넣습니다.
- `sudo groupadd -f kite-users` (최초 1회 그룹 생성)
- `sudo useradd -m -g kite-users -s /home/{username}/custom-shell.sh {username}`
- 비밀번호는 CRD에 저장된 해시 패스워드를 주입합니다.

#### C. SSH 키 쌍(KeyPair) 동적 생성과 인젝션
1. 에이전트(Go 로직)는 계정 생성 시 `crypto/rsa` 패키지를 이용해 키 쌍을 동적으로 생성합니다.
2. `Private Key`는 `/home/{username}/.ssh/id_rsa`에 떨굽니다.
3. `Public Key`는 쿠버네티스의 CRD Status(또는 Secret)로 던져서, `kite-controller`가 해당 값을 받아 KubeVirt VM의 `cloud-init` user-data에 주입하도록 핑퐁(Ping-Pong)을 구현합니다.

### 5.5. 호스트 탈출 방지 및 보안 하드닝 (Security Hardening)
사용자가 호스트 계정을 통해 악의적인 행위나 쉘 탈출을 시도하는 것을 원천 차단하기 위해 다음 조치를 병행합니다.

1. **`exec` 프로세스 교체 기법:**
   `custom-shell.sh`의 마지막 줄인 `exec ssh ...`는 자식 프로세스를 띄우는 것이 아니라, 현재 로그인한 쉘 프로세스 자체를 `ssh` 클라이언트로 덮어씌웁니다. 따라서 VM에서 로그아웃(exit)하는 순간 호스트 쉘로 돌아오지 못하고 그대로 접속이 뚝 끊기게 됩니다.
2. **명령어 인젝션 방지:**
   사용자가 `ssh apple@HostIP "/bin/bash"` 처럼 임의의 명령어를 강제 주입하려 해도, 우리의 커스텀 쉘은 인자(`$@`)를 아예 무시하고 무조건 고정된 VM 연결 명령어만 실행하므로 쉘 탈출이 불가능합니다.
3. **권한 최소화 (Unprivileged User):**
   `useradd`로 생성된 계정은 `sudo` 권한이 전혀 없으며 어떠한 관리자 그룹(`wheel`, `sudo` 등)에도 속하지 않는 깡통 계정입니다. 만에 하나 탈출하더라도 호스트 시스템을 건드릴 수 없습니다.
4. **호스트 `sshd_config` 제한 (권장 추가 사항):**
   호스트 장비의 관리자(현석님) 계정은 영향을 받지 않도록, `kite-users`라는 그룹에 대해서만 포트 포워딩(터널링)을 통한 내부망 스캐닝을 차단하는 것이 맞습니다. `/etc/ssh/sshd_config` 파일 맨 아래에 다음 설정을 추가합니다.
   ```text
   Match Group kite-users
       AllowTcpForwarding no
       AllowAgentForwarding no
       X11Forwarding no
   ```

### 5.6. 왜 이 설계가 압도적인가 (성과 포인트)
1. **완벽한 보안 격리 (Isolation):** 유저의 Private Key가 공용 저장소에 노출되지 않고, 본인의 리눅스 홈 디렉토리 밑에 `chmod 600`으로 갇혀 있습니다. 본인조차도 `custom-shell.sh`에 의해 즉시 튕겨 넘어가므로 키를 유출할 수 없습니다.
2. **고아 계정 방지 (Garbage Collection):** 사용자가 대시보드에서 VM을 삭제(delete)하면, 에이전트의 Informer가 이를 즉각 감지하고 `userdel {username}` 및 `/home/{username}` 디렉토리를 통째로 폭파시킵니다. 호스트 OS는 항상 깨끗한 상태를 유지합니다.
3. **선언적 무한 복구 (Self-Healing):** 만에 하나 시스템 오류나 관리자 실수로 `custom-shell.sh`가 지워져도, 데몬셋이 리컨사일 루프를 돌며 상태를 비교하고 즉시 파일을 다시 그려냅니다. 장애 복원력이 100% 보장됩니다.
