export const LEVEL_1_FIXED_CPU = 2;
export const LEVEL_1_FIXED_MEMORY = '4Gi';
export const LEVEL_1_FIXED_DISK_GI = 20;
export const MIN_DISK_GI = 20;
export const LEVEL_1_VM_QUOTA = 3;

export const getAccessLevelDescription = (level: number): string => {
  if (level === 0) return 'VM 생성 권한이 없는 계정입니다. 사용이 필요하면 관리자에게 권한을 요청하세요.';
  if (level === 1) return '일반 계정입니다. VM은 최대 3개까지 생성할 수 있고 스펙은 CPU 2, RAM 4Gi, Disk 20Gi로 고정됩니다.';
  if (level === 2) return '매니저 계정입니다. 일반 권한을 포함하며, 타 유저의 VM 상태 제어 및 관리가 가능합니다.';
  if (level >= 3) return '최고 관리자 계정입니다. 매니저 권한을 포함하며, Host Setup 및 클러스터 인프라 전역을 통제할 수 있습니다.';
  return '';
};
