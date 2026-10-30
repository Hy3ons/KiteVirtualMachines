import type { ReactNode } from 'react';
import { Button } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { StatusBadge } from './StatusBadge';

type SerialConsoleFrameProps = {
  readonly status: string;
  readonly message: string;
  readonly reconnectDisabled: boolean;
  readonly onReconnect: () => void;
  readonly children: ReactNode;
};

// SerialConsoleFrame shares console chrome while children owns the actual terminal or inert demo output.
// Reconnection and connection state are supplied by the caller, never opened by the frame.
export function SerialConsoleFrame({ status, message, reconnectDisabled, onReconnect, children }: SerialConsoleFrameProps) {
  return <div className="console-panel">
    <div className="console-panel-header">
      <div className="console-connection-state" role="status">
        <StatusBadge phase={status} label={`Serial console ${status}`} />
        {status !== 'connected' && <span className={`console-connection-detail${status === 'failed' ? ' is-error' : ''}`}>{message}</span>}
      </div>
      <Button icon={<ReloadOutlined />} disabled={reconnectDisabled} onClick={onReconnect}>다시 연결</Button>
    </div>
    <div className="console-terminal-frame">{children}</div>
  </div>;
}
