# User Dashboard Planning

## 1. 개요
일반 사용자(`access_level: 1`)가 로그인했을 때 진입하는 메인 대시보드 화면입니다.
본인의 네임스페이스에 있는 가상머신 자원을 확인하고 통제합니다.

## 2. 라우트 경로
- `/dashboard`: 가상머신 목록 조회, 생성, 제어 등 모든 기능을 한 곳에서 처리합니다.

## 3. 주요 화면 및 기능 스펙

### A. 내 가상머신 목록
- **테이블 UI**:
  - 컬럼: VM Name, Status, CPU, Memory, Disk, **Domain**, **SSH ID**, Actions.
  - *참고*: SSH 접속은 `kite-gateway`가 사용자 인증 후 VM의 ClusterIP SSH Service로 프록시합니다.
- **제어(Actions)**:
  - **시작 (Power On)**: 꺼진 VM을 켬 (`powerState: "On"`으로 Patch 요청).
  - **종료 (Power Off)**: 켜진 VM을 끔 (`powerState: "Off"`로 Patch 요청).
  - **삭제**: 리소스를 영구 삭제 (모달로 재확인 필수).

### B. 가상머신 생성 (Modal 창 또는 서랍 UI)
- **입력 항목 (유저가 직접 작성)**:
  - **이름 (Name)**: 인스턴스 이름 (영문 소문자/숫자 조합)
  - **도메인 프리픽스 (Domain Prefix)**: 웹 접속용 서브도메인 프리픽스 (예: `my-dev` 입력 시 `my-dev.hy3ons.github.io`으로 맵핑됨)
  - **SSH 접속 정보**: VM 접속을 위한 계정 ID와 Password (또는 SSH Key) 입력
  - **Disk (Storage)**: 기본 제공 `25Gi`. `access_level`에 따라 유동적으로 입력 허용. (백엔드 검증 필수)
- **고정 항목 (화면에서 변경 불가/숨김)**:
  - **CPU**: `2`
  - **Memory**: `4Gi`
  - **Image**: `ubuntu-22.04`
- **동작**:
  - 폼 제출 시 백엔드(`kite-api`)로 생성 요청.
  - 성공 후 모달을 닫고 대시보드 리스트 자동 갱신.
