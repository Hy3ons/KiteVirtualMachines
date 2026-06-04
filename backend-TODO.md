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
- [x] **host 계정 reconcile**: `cmd/kite-account`가 `KiteVirtualMachine.spec.sshId/sshPassword`와 `vps-access-<vmName>` Service를 보고 host Linux 계정과 proxy shell을 맞춘다.
- [x] **상태 drift 재조정**: KubeVirt VM의 `spec.running`이 `KiteVirtualMachine.spec.powerState`와 다르면 CRD 기준으로 다시 reconcile.
- [x] **고아 리소스 방지 삭제 흐름**: `spec.delete=true` 또는 CRD 직접 삭제 시 KubeVirt VM, Service, Ingress, Secret, DataVolume을 정리.
- [x] **DataVolume 상태 반영**: DataVolume ready/progress/failure를 별도 informer로 감지하고 `KiteVirtualMachine.status.dataVolumePhase`, `status.dataVolumeProgress`, `status.dataVolumeMessage`에 반영.

## 4. API Server 로직 수정 (`kite-api`)
- [ ] **디스크(Storage) Quota 검증 로직**: VM 생성 요청 시 프론트에서 넘어온 `disk` 용량이 해당 유저의 `access_level` 한도를 초과하지 않는지 백엔드 단에서 방어하는 검증 로직 추가.
- [ ] **VM 생성 API**: 변경된 스펙(Domain Prefix, SSH 정보 등)을 모두 받아 CRD를 생성하도록 수정.
- [x] **VM 목록 반환 API**: 프론트엔드에서 VM 목록을 그릴 때 사용할 VM spec/status와 최종 도메인 정보를 함께 리턴하도록 DTO를 구성한다. NodePort는 사용하지 않는다.
