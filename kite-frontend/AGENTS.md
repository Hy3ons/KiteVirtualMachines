# Kite Frontend - Agent Context Guide

이 문서는 AI 에이전트 및 새로운 개발자가 이 프로젝트(Kite Frontend)의 컨텍스트를 빠르고 정확하게 파악할 수 있도록 작성된 가이드입니다. 프로젝트에 코드를 기여하기 전에 반드시 이 문서를 숙지하세요.

## 1. Project Overview
Kite Frontend는 쿠버네티스 기반의 가상머신(VM) 및 인프라 프로비저닝 플랫폼인 **Kite**의 사용자 대시보드입니다.
사용자는 이 대시보드를 통해 자신의 정보(`KiteUser`)와 가상머신(`KiteVirtualMachine`) 리소스를 생성, 조회, 제어, 삭제할 수 있습니다. 
모든 백엔드 통신은 클러스터 내에 함께 배포된 `kite-api` 서버와 이루어집니다.

## 2. Tech Stack & Architecture
- **Build Tool**: Vite
- **Core Library**: React + TypeScript
- **UI Framework**: Ant Design (AntD)
- **Deployment & Serving**: Docker Multi-stage Build 적용
  - 빌드 단계: `node:20-alpine` 환경에서 정적(Static) 파일 빌드
  - 서빙 단계: `nginx:alpine`을 이용하여 빌드된 정적 파일을 80번 포트로 서빙
- 이 프론트엔드는 쿠버네티스에서 `ghcr.io/hy3ons/kite-frontend:latest` 이미지 기반의 Deployment(`replicas: 1`)로 실행됩니다.

## 3. Directory Structure & `docs`
프로젝트의 구조 및 특히 `docs` 폴더가 어떤 역할을 하는지 알아두어야 합니다.

- `src/`: React 앱의 핵심 소스코드가 위치합니다. (컴포넌트, 훅, API 연동 등)
- `Dockerfile`: 다단계 빌드(Multi-stage Build)를 위한 설정 파일
- `README.md`: 사용자가 개발해야 할 기능 스펙이 간략히 정리된 개요 문서
- **`docs/`**: 프론트엔드 프로젝트의 주요 정책, 규약, 디자인 시스템 등이 정리된 문서 폴더입니다.
  - `docs/design-convention.md`: 디자인 시스템 규약. 테마 색상(Bright/Cream 계열), 프레임워크 철학, 타협 없는 직각(0px Border Radius) 규칙 등을 상세히 정의하고 있습니다. **UI 컴포넌트를 작업할 때 반드시 참고해야 합니다.**
- **`mock.md`**: 아직 백엔드(`kite-api`)가 연동되지 않아 프론트엔드 내부에 임시로 만들어둔 Mock API(`setTimeout` 등) 코드의 위치(파일 경로 및 라인 번호)를 추적하는 문서입니다. **실제 프로덕션 배포 전에 반드시 이 문서를 확인하여 해당 코드들을 실제 API 호출로 교체해야 합니다.**

## 4. Coding Rules & Guidelines
1. **Design System 준수**: `docs/design-convention.md`에 명시된 색상 팔레트와 형태 규칙(border-radius: 0 등)을 AntD의 `ConfigProvider`나 CSS를 통해 일관성 있게 적용해야 합니다.
2. **API 연동**: `kite-api`와의 통신 규격을 준수하며, 에러 처리(Error Handling)를 꼼꼼하게 작성하여 사용자에게 직관적인 실패 원인을 알려야 합니다.
3. **상태 관리**: 복잡성을 줄이기 위해 꼭 필요한 전역 상태만 최소한으로 관리하며, React의 기본 훅(useState, useEffect)을 최대한 활용합니다.
4. **코딩 스타일**: 컴포넌트는 단일 책임 원칙에 따라 작게 분리하고, 함수와 변수명은 의미를 쉽게 유추할 수 있도록 명확하게 작성합니다.
