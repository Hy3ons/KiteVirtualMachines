# Kite 기여 가이드

Kite에 관심을 가져 주셔서 감사합니다. 작은 UI/UX 개선, 문서 수정, 테스트
추가, 배포 스크립트 개선, API/controller/gateway/frontend 기능까지 모두
환영합니다. 프로젝트의 코드 설계 철학을 크게 흔들지 않는 기여라면 가능한 한
함께 받아들이는 방향으로 검토합니다.

## 먼저 issue를 열어 주세요

모든 기여는 먼저 issue로 시작합니다.

Kite는 개발이 빠르게 진행되는 프로젝트라서, 아직 `main`에 merge되지 않은
`stage` 작업이나 원격 서버에서 검증 중인 `dev` 작업이 있을 수 있습니다.
따라서 PR을 바로 열기 전에 issue에서 아래 내용을 먼저 공유해 주세요.

- 바꾸고 싶은 문제 또는 개선 아이디어
- 직접 수정할 의사가 있는지 여부
- 예상되는 영향 범위
- 테스트할 수 있는 환경: k3s, minikube, generic k8s, 없음

maintainer는 현재 개발 상황과 브랜치 상태를 확인한 뒤, 몇 시간 안에 가능한
한 다음 정보를 답변합니다.

- 작업을 시작해도 되는지
- PR base branch가 `stage`인지 `main`인지
- 참고해야 할 설계 방향
- 필요한 테스트 범위
- maintainer가 대신 실행할 E2E 테스트가 있는지

## 브랜치 기준

- `main`: production publish 기준이다. 전체 E2E와 호환성 검증이 끝난 변경만
  들어간다.
- `stage`: main 직전 통합 브랜치다. stage 기준으로 빌드/설치해도 바로 문제가
  없어야 한다.
- `dev`: 빠른 통합과 원격 검증을 위한 브랜치다. 실험과 중간 산출물을 허용한다.
- 작업 브랜치: `feature/*`, `fix/*`, `docs/*`, `chore/*`, `refactor/*`를 사용한다.

자세한 정책은 [브랜치 운영 정책](docs/branch-policy.md)을 따릅니다.

## 설계 철학

Kite의 중심 설계는 Kubernetes CRD와 controller reconcile입니다.

- API 서버는 `KiteUser`, `KiteVirtualMachine` 같은 CRD의 desired state를 쓴다.
- controller는 CRD를 watch하고 실제 Kubernetes/KubeVirt/CDI 리소스를 reconcile한다.
- 관측된 runtime 상태는 CRD `status`에 쓴다.
- API가 controller에 직접 명령하는 protobuf/gRPC 흐름은 현재 설계가 아니다.
- 기존 helper, renderer, store, shell prompt, test harness를 먼저 재사용한다.
- 전역 변수와 전역 상태를 과하게 늘리지 않는다.
- 비슷한 코드를 대량으로 복붙하지 않는다.
- frontend mock 데이터는 개발 보조일 뿐 production 검증 증거가 아니다.

이 철학과 어긋나는 변경은 바로 거절하기보다, issue에서 더 맞는 구조로 같이
정리합니다.

## PR에 적어야 할 내용

PR 설명에는 다음 내용을 적어 주세요.

```text
Purpose
- 무엇을 왜 바꿨는지 적습니다.

Changes
- 주요 변경 내용을 짧게 적습니다.

Impact
- API, CRD, controller, frontend, gateway, deploy/test script 중 영향받는 영역을 적습니다.

Tests
- 실행한 테스트 명령과 결과를 적습니다.
- 실행하지 못한 테스트가 있으면 이유와 필요한 환경을 적습니다.
```

커밋 메시지는 [커밋 컨벤션](docs/commit-convention.md)을 따릅니다.

## Test Contract

기여자가 모든 클러스터 환경을 갖추고 있지 않아도 PR은 가능합니다. 대신 issue나
PR에서 테스트 책임을 명확히 나눕니다.

maintainer는 필요한 경우 아래 형식의 Test Contract를 제공합니다.

```text
Test Contract
- Base branch:
- Change type:
- Required local checks:
- Required cluster E2E:
- Optional checks:
- Maintainer-run checks:
- Tests the contributor could not run:
```

기여자는 자신이 실행할 수 있는 테스트를 실행하고, 실행하지 못한 테스트는 PR에
명확히 적습니다. k3s/minikube/generic k8s E2E가 merge gate에 필요한 경우,
기여자가 실행하지 못한 환경은 maintainer가 병합 전에 실행합니다.

변경 유형별 최소 테스트 기준은 [테스트 기준서](test/Test-Specification.md)를
따릅니다. release 성격의 변경은 [테스트 표준](test/Readme.md)의 E2E gate를
기준으로 합니다.

## 환영하는 기여 예시

- 로그인, VM 생성, 관리자 화면 같은 UI/UX 개선
- 문서 보강과 오타 수정
- 테스트 케이스 추가
- shell script prompt와 실패 메시지 개선
- CRD schema와 controller reconcile 안정화
- gateway SSH 인증, fallback, banner 개선
- k3s, minikube, generic k8s 설치/삭제 안정화

작은 기여도 환영합니다. 먼저 issue로 이야기하면, 현재 브랜치 상황에 맞는 가장
좋은 작업 경로를 함께 정하겠습니다.
