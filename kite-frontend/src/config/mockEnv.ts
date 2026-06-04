export const MOCK_ENV = {
  // 베이스 도메인 (관리자가 AdminSettings에서 설정하면 백엔드에서 내려와야 하는 값)
  BASE_DOMAIN: 'anacnu.com',
  
  // 일반 유저(accessLevel === 1)의 최대 VM 갯수 제한 (백엔드 쿼타 정책에서 관리되어야 하는 값)
  MAX_VM_QUOTA_LEVEL_1: 3,

  // 더미 토큰 프리픽스
  DUMMY_TOKEN_ADMIN: 'dummy-admin-token',
  DUMMY_TOKEN_USER: 'dummy-user-token',
};
