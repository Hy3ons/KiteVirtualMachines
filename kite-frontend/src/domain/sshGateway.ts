import type { SSHGatewaySettings } from '../api/types';
import { validTCPPort } from './network';

export type SSHCommandState = {
  readonly ready: true;
  readonly command: string;
  readonly configPort: string;
} | {
  readonly ready: false;
  readonly configPort: string;
  readonly message: string;
};

export const buildSSHCommandState = (sshId: string, host: string, gateway: SSHGatewaySettings | null = null): SSHCommandState => {
  const trimmedHost = host.trim();
  if (!trimmedHost) {
    return {
      ready: false,
      configPort: '',
      message: '관리자 설정의 baseDomain을 불러오지 못했거나 아직 설정되지 않았습니다.',
    };
  }

  const phase = gateway?.status.phase ?? 'Disabled';
  if (!gateway?.externalEnabled || phase !== 'Ready') {
    return {
      ready: false,
      configPort: gateway?.publicPort.trim() || gateway?.externalPort.trim() || '',
      message: gateway?.status.message || '운영자가 외부 VM SSH gateway를 아직 활성화하지 않았습니다.',
    };
  }

  const port = (gateway.publicPort || gateway.externalPort).trim();
  if (!validTCPPort(port)) {
    return {
      ready: false,
      configPort: port,
      message: '운영자가 설정한 사용자 안내 SSH 포트가 올바르지 않습니다.',
    };
  }

  const command = port === '22' ? `ssh ${sshId}@${trimmedHost}` : `ssh -p ${port} ${sshId}@${trimmedHost}`;
  return {
    ready: true,
    command,
    configPort: port,
  };
};
