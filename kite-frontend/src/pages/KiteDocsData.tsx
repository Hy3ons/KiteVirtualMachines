import type { TimelineProps } from 'antd';
import { Typography } from 'antd';

const { Paragraph, Text } = Typography;

export type AccessLevelDoc = {
  readonly level: string;
  readonly title: string;
  readonly tone: string;
  readonly summary: string;
  readonly rules: readonly string[];
};

export const accessLevelDocs: readonly AccessLevelDoc[] = [
  {
    level: '0',
    title: 'No VM Create',
    tone: '권한 요청 단계',
    summary: '로그인은 허용하지만 VM 생성과 제어는 제한합니다.',
    rules: ['Dashboard는 계정 상태를 보여주고 VM 생성 흐름은 열지 않습니다.', 'API 직접 호출로 VM 생성/수정/전원/삭제를 시도하면 403을 반환합니다.'],
  },
  {
    level: '1',
    title: 'Fixed Self-service',
    tone: '일반 사용자',
    summary: '자기 namespace 안에서 고정 스펙 VM만 생성하고 관리할 수 있습니다.',
    rules: ['CPU 2, Memory 4Gi, Disk 25Gi로 고정합니다.', 'Frontend와 API는 직접 생성 VM을 최대 2개로 제한합니다.'],
  },
  {
    level: '2',
    title: 'Power User',
    tone: '팀 운영자',
    summary: '자기 VM은 자유 스펙으로 만들 수 있고, 관리자 화면에서는 Level 0/1 권한만 조정합니다.',
    rules: ['다른 사용자의 VM 조회, 전원 변경, 삭제는 할 수 없습니다.', '사용자 권한은 0과 1 사이에서만 바꿀 수 있고 Level 2/3 대상은 건드릴 수 없습니다.'],
  },
  {
    level: '3',
    title: 'Full Admin',
    tone: '전체 관리자',
    summary: 'Kite의 모든 관리 작업을 수행할 수 있습니다.',
    rules: ['전체 VM 제어, 사용자 삭제, 시스템 설정을 허용합니다.', '사용자에게 VM Offer를 만들어 스펙을 미리 배정할 수 있습니다.'],
  },
];

export const architectureTimelineItems: TimelineProps['items'] = [
  {
    color: '#8B7355',
    content: (
      <div>
        <Text strong>사용자가 화면에서 요청</Text>
        <Paragraph className="docs-timeline-copy">사용자는 Dashboard에서 로그인, VM 생성, 전원 변경 같은 작업을 요청합니다. 화면은 API 응답과 VM 상태를 보여줍니다.</Paragraph>
      </div>
    ),
  },
  {
    color: '#C9B59C',
    content: (
      <div>
        <Text strong>API가 권한과 입력 확인</Text>
        <Paragraph className="docs-timeline-copy">`kite-api`는 세션과 access level을 확인하고, 허용된 요청만 Kubernetes API 서버에 기록합니다.</Paragraph>
      </div>
    ),
  },
  {
    color: '#7A9C74',
    content: (
      <div>
        <Text strong>CRD가 원하는 상태 보관</Text>
        <Paragraph className="docs-timeline-copy">`KiteUser`, `KiteVirtualMachine`, `KiteVirtualMachineOffer` CRD의 spec에는 Kite가 추상화한 원하는 상태가 저장됩니다.</Paragraph>
      </div>
    ),
  },
  {
    color: '#D4A373',
    content: (
      <div>
        <Text strong>Controller가 실제 리소스 조정</Text>
        <Paragraph className="docs-timeline-copy">`kite-controller`는 CRD spec을 읽고 KubeVirt VM, DataVolume, 서비스 같은 실제 Kubernetes 리소스를 맞춥니다.</Paragraph>
      </div>
    ),
  },
  {
    color: '#8B7355',
    content: (
      <div>
        <Text strong>상태가 다시 화면으로 반영</Text>
        <Paragraph className="docs-timeline-copy">controller는 관측한 상태와 실패 이유를 CRD status에 기록하고, Frontend는 그 값을 사용자에게 보여줍니다.</Paragraph>
      </div>
    ),
  },
];
