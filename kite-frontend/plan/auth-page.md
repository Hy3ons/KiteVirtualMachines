# Auth Pages Planning

## 1. 개요
서비스의 진입점이 되는 로그인 및 회원가입 페이지 규격입니다.
Ant Design의 폼 컴포넌트를 사용하여 깔끔하고 정돈된 화면을 구성합니다. (0px 라운딩 원칙 적용)

## 2. 라우트 경로
- `/login`: 로그인 페이지
- `/signup`: 회원가입 페이지

## 3. 주요 기능 및 UI 스펙

### A. 로그인 (`/login`)
- **폼 필드**: Username, Password
- **동작**: 
  - `kite-api`로 인증 요청. 
  - 성공 시 세션/토큰을 저장하고, 유저 정보(`access_level` 포함)를 브라우저 상태에 저장.
  - `access_level`을 검사하여 관리자면 `/admin`으로, 일반 유저면 `/dashboard`로 리다이렉트.

### B. 회원가입 (`/signup`)
- **폼 필드**: Username, Password, Password Confirm, Email
- **동작**: 
  - `kite-api`로 KiteUser 생성 요청. 
  - 기본 가입 시 `access_level`은 일반 유저 등급(예: 1)으로 설정됨.
  - 생성 후 로그인 페이지로 이동하거나 즉시 로그인 처리.

### C. Auth Guard (라우팅 보안)
- 라우터 레벨에서 사용자의 토큰/세션 존재 여부를 확인합니다.
- 토큰이 없는데 `/admin`이나 `/dashboard` 접근 시 `/login`으로 튕겨냅니다.
