# Admin Dashboard Planning

## 1. 개요
관리자(`access_level: 2` 또는 `3`) 전용 페이지입니다.
개별 사용자의 네임스페이스 범위를 넘어, 클러스터 전체의 리소스(KiteUser, KiteVirtualMachine)를 조회하고 시스템 전역 설정을 통제하는 역할을 수행합니다.

## 2. 라우트 경로 (Admin Guard 적용)
- `/admin/setup`: 초기 설치 시 셋업 마법사 (도메인, HTTPS 인증서 설정)
- `/admin/dashboard`: 클러스터 전체 리소스 요약
- `/admin/settings`: 전역 시스템 설정 (이후 수정용)
- `/admin/users`: 사용자 관리 메뉴
- `/admin/vms`: 전체 가상머신 관리 메뉴

## 3. 주요 화면 및 기능 스펙

### A. 초기 셋업 위저드 및 전역 설정 (`/admin/setup`)
- **초기 접속 플로우**: 
  - 플랫폼 최초 배포 시, 관리자가 로컬로 포워딩된 `localhost:80`으로 접속하면 자동으로 이 초기 셋업 화면으로 유도됩니다.
- **베이스 도메인 입력**:
  - 클러스터 전역 Ingress 라우팅에 사용될 베이스 도메인(예: `hy3ons.github.io`)을 입력받습니다.
- **HTTPS 인증서 (TLS) 등록**:
  - `domain.com` 및 `*.domain.com`을 커버할 수 있는 와일드카드 인증서 파일(`Fullchain Cert`, `Private Key`)을 직접 업로드하거나 복사/붙여넣기 할 수 있는 폼을 제공합니다.
- **저장 동작**:
  - "설정 완료" 시 백엔드로 전송되며, 백엔드는 도메인을 `etcd`에, 인증서를 K8s `global-secret`에 저장합니다. 이후부터 생성되는 모든 VM은 완벽한 HTTPS 통신이 지원되는 도메인을 부여받습니다.

### B. 관리자 홈 (`/admin/dashboard`)
- **통계 요약 지표**: 전체 등록된 사용자 수, 클러스터에 배포된 총 VM 수, 전체 CPU/Memory 할당량 등 현황판 구성.

### C. 사용자 관리 (`/admin/users`)
- **모든 사용자 목록 테이블**:
  - 컬럼: Username, Namespace, Access Level, Status, Actions.
- **기능 (Actions)**:
  - **권한 수정 (Promote/Demote)**: 특정 사용자의 `access_level`을 일반(1) ↔ 관리자(3) 등으로 변경.
  - **상태 확인**: 네임스페이스 및 Quota 정상 생성 여부 디버깅용 메시지 확인.

### D. 전체 가상머신 관리 (`/admin/vms`)
- **클러스터 전체 VM 테이블**:
  - 일반 사용자는 자기 것만 보지만, 관리자는 **소유자(Namespace)** 필드가 포함된 전체 목록을 조회.
  - 컬럼: Namespace(Owner), VM Name, Status, CPU, RAM, Actions.
- **글로벌 통제 권한**:
  - 악성 사용자 또는 리소스를 과도하게 점유 중인 VM을 관리자 권한으로 **강제 Power Off** 하거나 **Delete** 할 수 있는 강력한 기능 제공.
