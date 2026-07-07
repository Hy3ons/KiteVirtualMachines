import type { SSHGatewaySettings } from '../api/types';

export type SSHCommandState = {
  readonly ready: boolean;
  readonly command: string;
  readonly configPort: string;
  readonly message: string;
};

export const buildSSHCommandState = (sshId: string, host: string, gateway?: SSHGatewaySettings): SSHCommandState => {
  const trimmedHost = host.trim();
  if (!trimmedHost) {
    return {
      ready: false,
      command: '',
      configPort: '',
      message: '관리자 설정의 baseDomain을 불러오지 못했거나 아직 설정되지 않았습니다.',
    };
  }

  const phase = gateway?.status?.phase || gateway?.phase || 'Disabled';
  if (!gateway?.externalEnabled || phase !== 'Ready') {
    return {
      ready: false,
      command: '',
      configPort: gateway?.publicPort?.trim() || gateway?.externalPort?.trim() || '',
      message: gateway?.status?.message || gateway?.message || '운영자가 외부 VM SSH gateway를 아직 활성화하지 않았습니다.',
    };
  }

  const port = (gateway.publicPort || gateway.externalPort).trim();
  if (!validSSHPort(port)) {
    return {
      ready: false,
      command: '',
      configPort: port,
      message: '운영자가 설정한 사용자 안내 SSH 포트가 올바르지 않습니다.',
    };
  }

  const command = port === '22' ? `ssh ${sshId}@${trimmedHost}` : `ssh -p ${port} ${sshId}@${trimmedHost}`;
  return {
    ready: true,
    command,
    configPort: port,
    message: '',
  };
};

const validSSHPort = (value: string): boolean => {
  const port = Number(value);
  return Number.isInteger(port) && port >= 1 && port <= 65535;
};
