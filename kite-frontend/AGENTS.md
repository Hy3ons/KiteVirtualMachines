# Kite Frontend - Agent Context Guide

이 문서는 AI 에이전트 및 새로운 개발자가 이 프로젝트(Kite Frontend)의 컨텍스트를 빠르고 정확하게 파악할 수 있도록 작성된 가이드입니다. 프로젝트에 코드를 기여하기 전에 반드시 이 문서를 숙지하세요.

## 1. Project Overview
Kite Frontend는 쿠버네티스 기반의 가상머신(VM) 및 인프라 프로비저닝 플랫폼인 **Kite**의 사용자 대시보드입니다.
사용자는 이 대시보드를 통해 자신의 정보(`KiteUser`)와 가상머신(`KiteVirtualMachine`) 리소스를 생성, 조회, 제어, 삭제할 수 있습니다. 
모든 백엔드 통신은 클러스터 내에 함께 배포된 `kite-api` 서버와 이루어집니다.

## 2. Tech Stack & Architecture
- **Framework**: Next.js App Router
- **Core Library**: React + TypeScript
- **UI Framework**: Ant Design (AntD)
- **Deployment & Serving**: Docker Multi-stage Build 적용
  - 빌드 단계: `node:22-alpine` 환경에서 Next.js standalone 빌드
  - 실행 단계: 비루트 Node.js 프로세스가 3000번 포트에서 SSR 제공
  - Kubernetes Service는 외부 포트 80을 컨테이너 포트 3000으로 연결
  - `/api` 요청은 Ingress에서 Go API로 직접 전달하며, Next.js는 세션 확인과 SSR 조회만 담당
- 이 프론트엔드는 쿠버네티스에서 `ghcr.io/hy3ons/kite-frontend:latest` 이미지 기반의 Deployment(`replicas: 1`)로 실행됩니다.

## 3. Directory Structure & `docs`
프로젝트의 구조 및 특히 `docs` 폴더가 어떤 역할을 하는지 알아두어야 합니다.

- `src/app/`: Next.js 서버 페이지, 레이아웃, 메타데이터가 위치합니다.
- `src/screens/`: 기존 React 화면과 인터랙션을 유지합니다.
- `src/server/`: 서버 전용 세션 검증과 Go API 연결을 담당합니다.
- `src/api/`: 브라우저/SSR HTTP 클라이언트와 응답 계약을 관리합니다.
- `Dockerfile`: 다단계 빌드(Multi-stage Build)를 위한 설정 파일
- `README.md`: 사용자가 개발해야 할 기능 스펙이 간략히 정리된 개요 문서
- **`docs/`**: 프론트엔드 프로젝트의 주요 정책, 규약, 디자인 시스템 등이 정리된 문서 폴더입니다.
  - `docs/design-convention.md`: 디자인 시스템 규약. DESIGN.md 기반의 white/ink 색상, 공통 콘솔 셸, iOS 스타일 squircle 규칙 등을 상세히 정의하고 있습니다. **UI 컴포넌트를 작업할 때 반드시 참고해야 합니다.**
- **`mock.md`**: Go API와 연결되는 엔드포인트 계약을 설명합니다. 별도 개발용 Mock은 `src/api/debugMock*.ts`에 있으며 명시적인 debug 모드에서만 사용합니다.

## 4. Coding Rules & Guidelines
1. **Design System 준수**: `docs/design-convention.md`에 명시된 색상 팔레트와 iOS 스타일 squircle 형태 규칙을 AntD의 `ConfigProvider`나 CSS를 통해 일관성 있게 적용해야 합니다.
2. **API 연동**: `kite-api`와의 통신 규격을 준수하며, 에러 처리(Error Handling)를 꼼꼼하게 작성하여 사용자에게 직관적인 실패 원인을 알려야 합니다.
3. **상태 관리**: 복잡성을 줄이기 위해 꼭 필요한 전역 상태만 최소한으로 관리하며, React의 기본 훅(useState, useEffect)을 최대한 활용합니다.
4. **코딩 스타일**: 컴포넌트는 단일 책임 원칙에 따라 작게 분리하고, 함수와 변수명은 의미를 쉽게 유추할 수 있도록 명확하게 작성합니다.

## Session and QA
- JWT 발급·검증·쿠키 삭제는 Go API가 담당합니다. Next.js는 HttpOnly `accessToken` 쿠키를 `/me`에 전달해 세션을 검증합니다.
- 사용자/권한/JWT를 localStorage에 저장하거나 SSR 인증 근거로 사용하지 않습니다.
- 서버 데이터는 요청별로 격리하고 공유 캐시에 저장하지 않습니다.
- `npm test`는 브라우저 없이 HTTP 요청·응답 테스트만 실행합니다. 화면과 동선 QA는 사용자가 담당하며 Playwright/E2E를 실행하지 않습니다.

<!-- BEGIN:nextjs-agent-rules -->

## This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->
