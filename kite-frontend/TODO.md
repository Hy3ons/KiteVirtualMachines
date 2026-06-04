# Kite Frontend TODO List

## 1. Project Setup & Routing
- [ ] Ant Design 테마(Bright/Cream, 0px border-radius) 전역 적용 (`ConfigProvider`)
- [ ] React Router 기반 라우팅 설계 (`/`, `/login`, `/admin`, `/dashboard`)
- [ ] Axios/Fetch API 클라이언트 세팅 (Kite-API 연동용)
- [ ] 전역 상태 관리 세팅 (인증 정보, 유저 access_level 유지)

## 2. Landing & Auth Pages
- [ ] `/` 랜딩 페이지 구현 (Kite 프로젝트 소개 및 로그인 유도)
- [ ] `/login` 페이지 구현 (사용자 인증 및 토큰 발급)
- [ ] `/signup` 페이지 구현 (KiteUser 생성 요청)
- [ ] Private Route / Auth Guard 구현 (로그인 안 된 유저는 `/login`으로 튕김)
- [ ] Admin Guard 구현 (`access_level` 2 이상 관리자만 `/admin` 접근 허용)

## 3. User Dashboard (`/dashboard`)
- [ ] 대시보드 홈 (내 가상머신 목록 조회 및 Quota 확인)
- [ ] 가상머신 생성 기능 (CPU 2, Mem 4Gi, Image 고정. Disk 25Gi 기본, 등급별 확장)
- [ ] 내 가상머신 제어 액션 (Power On / Off 상태 변경, 영구 삭제)

## 4. Admin Page (`/admin`)
- [ ] 전체 클러스터 요약 (총 유저 수, 전체 VM 동작 개수)
- [ ] 사용자 관리 테이블 (전체 KiteUser 조회 및 `access_level` 수정/승급)
- [ ] 전역 가상머신 관리 테이블 (모든 사용자의 VM 조회, 관리자 권한으로 강제 Power On/Off 및 삭제)

## 5. Polish
- [ ] API 에러 발생 시 알림창(Toast/Notification) 일괄 처리
- [ ] 로딩 상태(Skeleton, Spinner) UI 추가
