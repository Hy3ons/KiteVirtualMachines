import React, { useEffect, useRef, useState } from 'react';
import { Alert, Button, Space, Tag, Typography } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { FitAddon } from '@xterm/addon-fit';
import { Terminal } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';
import { buildConsoleWebSocketUrl } from '../api/console';
import { vmApi } from '../api';

const { Text } = Typography;

type ConsoleStatus = 'idle' | 'connecting' | 'connected' | 'closed' | 'failed';

type VmConsoleTerminalProps = {
  readonly vmName: string;
  readonly enabled: boolean;
  readonly mock: boolean;
};

const statusColor: Record<ConsoleStatus, string> = {
  idle: 'default',
  connecting: 'processing',
  connected: 'success',
  closed: 'default',
  failed: 'error',
};

export const VmConsoleTerminal: React.FC<VmConsoleTerminalProps> = ({ vmName, enabled, mock }) => {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [status, setStatus] = useState<ConsoleStatus>('idle');
  const [message, setMessage] = useState('Console is waiting for a running VM.');
  const [reconnectKey, setReconnectKey] = useState(0);

  useEffect(() => {
    const container = containerRef.current;
    if (!container || !enabled) {
      const timer = window.setTimeout(() => {
        setStatus('idle');
        setMessage('Start the VM before opening the serial console.');
      }, 0);
      return () => window.clearTimeout(timer);
    }

    let active = true;
    const updateConsoleState = (nextStatus: ConsoleStatus, nextMessage: string) => {
      if (!active) return;
      setStatus(nextStatus);
      setMessage(nextMessage);
    };

    const terminal = new Terminal({
      cursorBlink: true,
      convertEol: true,
      fontFamily: '"SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace',
      fontSize: 13,
      theme: {
        background: '#111111',
        foreground: '#F4EFE8',
        cursor: '#C9B59C',
        selectionBackground: '#594E46',
      },
    });
    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(container);
    fitAddon.fit();

    const resizeObserver = new ResizeObserver(() => fitAddon.fit());
    resizeObserver.observe(container);

    if (mock) {
      queueMicrotask(() => updateConsoleState('connected', 'Debug console is connected.'));
      terminal.writeln('Kite debug serial console');
      terminal.writeln(`Connected to ${vmName}`);
      terminal.write('$ ');
      const input = terminal.onData((data) => {
        if (data === '\r') {
          terminal.write('\r\n$ ');
          return;
        }
        terminal.write(data);
      });
      return () => {
        active = false;
        input.dispose();
        resizeObserver.disconnect();
        terminal.dispose();
      };
    }

    let socket: WebSocket | null = null;
    const decoder = new TextDecoder();
    queueMicrotask(() => updateConsoleState('connecting', 'Requesting console ticket...'));
    terminal.writeln('Connecting to Kite VM serial console...');

    void vmApi.createConsoleTicket(vmName)
      .then((ticket) => {
        updateConsoleState('connecting', 'Opening console WebSocket...');
        socket = new WebSocket(buildConsoleWebSocketUrl(vmName, ticket.ticket), ['plain.kubevirt.io']);
        socket.binaryType = 'arraybuffer';
        socket.onopen = () => {
          updateConsoleState('connected', 'Console is connected.');
          terminal.writeln('Connected. Serial console output will appear below.');
        };
        socket.onmessage = (event) => {
          if (event.data instanceof ArrayBuffer) {
            terminal.write(decoder.decode(event.data));
            return;
          }
          if (typeof event.data === 'string') {
            terminal.write(event.data);
          }
        };
        socket.onerror = () => {
          updateConsoleState('failed', 'Console connection failed.');
        };
        socket.onclose = () => {
          if (!active) return;
          setStatus((current) => current === 'failed' ? 'failed' : 'closed');
          setMessage('Console connection closed.');
        };
      })
      .catch((error: unknown) => {
        updateConsoleState('failed', error instanceof Error ? error.message : 'Could not create console ticket.');
      });

    const input = terminal.onData((data) => {
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(data);
      }
    });

    return () => {
      active = false;
      input.dispose();
      resizeObserver.disconnect();
      socket?.close();
      terminal.dispose();
    };
  }, [enabled, mock, reconnectKey, vmName]);

  return (
    <div className="console-panel">
      <div className="console-panel-header">
        <Space>
          <Text strong>Serial console</Text>
          <Tag color={statusColor[status]}>{status}</Tag>
        </Space>
        <Button icon={<ReloadOutlined />} onClick={() => setReconnectKey((value) => value + 1)}>
          Reconnect
        </Button>
      </div>
      <Alert message={message} type={status === 'failed' ? 'error' : 'info'} showIcon className="console-status" />
      <div ref={containerRef} className="console-terminal" />
    </div>
  );
};
