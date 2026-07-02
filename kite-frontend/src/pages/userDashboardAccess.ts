export const LEVEL_1_FIXED_CPU = 2;
export const LEVEL_1_FIXED_MEMORY = '4Gi';
export const LEVEL_1_FIXED_DISK_GI = 25;
export const MIN_DISK_GI = 20;
export const LEVEL_1_VM_QUOTA = 2;

export const getAccessLevelDescription = (level: number): string => {
  if (level === 0) return '로그인과 대시보드 확인만 가능하며, VM 직접 생성과 제어는 막혀 있습니다.';
  if (level === 1) return `일반 계정입니다. 직접 생성 VM은 최대 ${LEVEL_1_VM_QUOTA}개이며 CPU ${LEVEL_1_FIXED_CPU}, RAM ${LEVEL_1_FIXED_MEMORY}, Disk ${LEVEL_1_FIXED_DISK_GI}Gi로 고정됩니다.`;
  if (level === 2) return '팀 운영자 계정입니다. 자기 VM은 자유 스펙으로 만들 수 있고, 관리자 화면에서는 Level 0/1 권한만 조정할 수 있습니다.';
  if (level >= 3) return '전체 관리자 계정입니다. 사용자, VM, 시스템 설정, VM offer를 모두 관리할 수 있습니다.';
  return '';
};
