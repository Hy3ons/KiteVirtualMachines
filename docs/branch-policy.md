# 브랜치 운영 정책

이 문서는 Kite 프로젝트에서 `main`, `stage`, `dev`, 작업 브랜치를 어떤
목적으로 쓰고, 어떤 조건에서 다음 단계로 올릴 수 있는지 정의한다.

## 기본 원칙

- `main`은 production publish 기준이다. 여기에 올라간 커밋은 GHCR image
  publish와 실제 사용자 설치 경로의 기준이 된다.
- `stage`는 main 직전 통합 기준이다. stage checkout을 빌드해서 바로 써도
  큰 문제가 없어야 한다.
- `dev`는 빠른 개발 통합과 원격 환경 검증을 위한 브랜치다. 실험과 중간
  산출물은 허용하지만, stage로 올리기 전에는 반드시 정리한다.
- 기능 작업은 가능하면 `feature/*`, 버그 수정은 `fix/*`, 문서는 `docs/*`,
  스크립트와 유지보수는 `chore/*`, 구조 정리는 `refactor/*` 브랜치를 쓴다.
- PR base branch는 작업자가 임의로 고정하지 않고, issue에서 maintainer와
  현재 개발 상황을 확인한 뒤 결정한다.

## `main`

`main`은 production publishing branch다.

`main`에 push 또는 merge할 수 있는 조건:

- 분기 또는 개발 주기에서 정한 모든 개발 모듈의 개별 구현이 끝났다.
- API, controller, gateway, frontend, manifest, storage 흐름이 하나의 제품으로
  유기적으로 동작한다.
- k3s, minikube, generic k8s에서 release E2E 테스트가 끝났다.
- 레거시 etcd 데이터 경로, 기존 CRD object, 이전 schema/status와의 호환성이
  필요한 변경이라면 migration 또는 compatibility 테스트가 끝났다.
- `stage` 기준 빌드/설치/삭제가 정상 동작하고, production GHCR publish에
  올려도 되는 이미지와 문서 상태다.

`main`에 올릴 수 없는 상태:

- 특정 환경에서만 우연히 동작하고 k3s/minikube/k8s 중 하나의 검증이 빠진 경우.
- CRD 설계, reconcile 철학, API 계약이 문서와 다르게 흘러간 경우.
- mock 또는 unit test만 통과했고 실제 cluster E2E 증거가 없는 경우.
- 이전 CRD/status/data와 호환되어야 하는 변경인데 호환성 검증이 없는 경우.

## `stage`

`stage`는 main 전의 통합 브랜치다.

`stage`에 올릴 수 있는 조건:

- 현재 checkout 기준으로 `build-install.sh` 또는 `ghcr-install.sh` 흐름을 따라
  빌드/설치해도 즉시 치명적인 문제가 없어야 한다.
- 전체 Kite 코드 설계 철학과 맞아야 한다.
  - API는 CRD desired state를 기록한다.
  - controller는 CRD를 watch하고 reconcile한다.
  - runtime 관측값은 CRD `status`에 기록한다.
  - protobuf/gRPC command path는 재도입하지 않는다.
- 기존 helper, renderer, store, prompt, test harness를 우선 재사용한다.
- 전역 변수, 전역 상태, 임시 복붙 코드, 한 기능만을 위한 대량 신규 코드는
  stage에 올리기 전에 정리한다.
- k3s, minikube, generic k8s에서 필요한 테스트가 끝났거나, maintainer가
  남은 환경 테스트를 병합 전 책임지고 실행하기로 issue/PR에 명시해야 한다.

`stage`에서 거절되는 상태:

- 코드가 너무 전역에 퍼져 있어 변경 영향 범위를 추적하기 어려운 경우.
- 재사용 가능한 기존 구조를 무시하고 유사 코드가 대량 추가된 경우.
- CRD 사용 목적, 권한 모델, controller reconcile 규칙이 프로젝트 방향과 맞지
  않는 경우.
- 테스트 없이 “로컬에서만 되는 것 같다”는 상태로 올라온 경우.

## `dev`

`dev`는 개발 중간 통합과 원격 검증을 위한 브랜치다.

허용되는 작업:

- 아직 테스트가 안정화되지 않은 기능을 원격 서버에서 빠르게 검증한다.
- 여러 작업이 임시로 섞인 상태를 모아 실제 cluster에서 문제를 발견한다.
- 설계가 완전히 닫히기 전이라도 prototype을 올려 토론한다.

주의할 점:

- `dev`는 난장판이 될 수 있지만, 그 상태를 `stage`로 그대로 올리지 않는다.
- stage 승격 전에는 중복 코드 제거, helper 재사용, 테스트 정리, 문서 갱신을
  반드시 수행한다.
- dev에서 발견한 실패는 issue 또는 PR에 남겨 다음 작업자가 같은 문제를 다시
  밟지 않게 한다.

## 승격 규칙

`dev -> stage`:

- 기능 목적과 사용자 흐름이 정리되어 있다.
- CRD/API/controller/frontend/storage/gateway 중 영향받는 영역의 테스트가
  정의되어 있다.
- 중복 구현과 임시 전역 상태가 정리되어 있다.
- 최소 하나 이상의 실제 cluster 검증 또는 maintainer 대리 검증 계획이 있다.

`stage -> main`:

- k3s, minikube, generic k8s release E2E가 완료되어야 한다.
- legacy etcd/CRD compatibility가 필요한 변경은 호환성 검증이 완료되어야 한다.
- install, uninstall, cleanup, host sshd handoff 같은 운영 흐름이 깨지지 않아야 한다.
- 문서, 테스트 명세, 실제 스크립트 이름이 서로 일치해야 한다.

## 관련 문서

- [한국어 기여 가이드](../CONTRIBUTE-KO.md)
- [English contribution guide](../CONTRIBUTE-EN.md)
- [테스트 표준](../test/Readme.md)
- [테스트 기준서](../test/Test-Specification.md)
- [커밋 컨벤션](commit-convention.md)
