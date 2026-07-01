import { Button, Popconfirm, Space, Tag } from 'antd';
import { CaretRightOutlined, CodeOutlined, DeleteOutlined, PoweroffOutlined } from '@ant-design/icons';
import { VmConsoleButton } from './VmConsoleButton';
import type { TableColumnsType } from 'antd';
import type { DashboardVm } from './UserDashboardTypes';

type UserDashboardColumnsOptions = {
  onNavigateToDetail: (name: string) => void;
  onConnect: (vm: DashboardVm) => void;
  onOpenConsole: (name: string) => void;
  onStart: (name: string) => void;
  onStop: (name: string) => void;
  onDelete: (name: string) => void;
};

export const createUserDashboardColumns = ({
  onNavigateToDetail,
  onConnect,
  onOpenConsole,
  onStart,
  onStop,
  onDelete
}: UserDashboardColumnsOptions): TableColumnsType<DashboardVm> => [
  {
    title: 'Name',
    dataIndex: 'name',
    key: 'name',
    width: 180,
    ellipsis: true,
    render: (text: string) => (
      <a
        href={`/dashboard/kite-machine/${text}`}
        onClick={(event) => {
          event.preventDefault();
          onNavigateToDetail(text);
        }}
        style={{ fontWeight: 'bold', color: '#8B7355', textDecoration: 'underline' }}
      >
        {text}
      </a>
    )
  },
  { title: 'Domain', dataIndex: 'domain', key: 'domain', width: 260, ellipsis: true },
  { title: 'SSH ID', dataIndex: 'sshId', key: 'sshId', width: 140, ellipsis: true },
  {
    title: 'Status',
    dataIndex: 'phase',
    key: 'phase',
    render: (phase: string) => {
      let color = 'default';
      if (phase === 'Running') color = 'success';
      if (phase === 'Stopped') color = 'error';
      if (phase === 'Creating' || phase === 'Terminating') color = 'processing';
      return <Tag color={color}>{phase}</Tag>;
    },
    width: 130
  },
  { title: 'CPU', dataIndex: 'cpu', key: 'cpu', width: 90 },
  { title: 'Memory', dataIndex: 'memory', key: 'memory', width: 110 },
  { title: 'Disk', dataIndex: 'disk', key: 'disk', width: 100 },
  {
    title: 'Actions',
    key: 'actions',
    width: 420,
    fixed: 'right',
    render: (_, record) => (
      <Space size="small" wrap={false} className="dashboard-table-actions">
        <Button type="text" icon={<CodeOutlined />} onClick={() => onConnect(record)}>
          Connect
        </Button>
        <VmConsoleButton vm={record} onOpen={onOpenConsole} />
        {record.phase !== 'Running' && (
          <Button
            type="text"
            icon={<CaretRightOutlined style={{ color: '#52c41a' }} />}
            onClick={() => onStart(record.name)}
          />
        )}
        {record.phase === 'Running' && (
          <Button
            type="text"
            icon={<PoweroffOutlined style={{ color: '#d9363e' }} />}
            onClick={() => onStop(record.name)}
          >
            Stop
          </Button>
        )}
        <Popconfirm title="정말 이 VM을 삭제하시겠습니까? 데이터가 모두 날아갑니다." onConfirm={() => onDelete(record.name)}>
          <Button type="text" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      </Space>
    )
  }
];
