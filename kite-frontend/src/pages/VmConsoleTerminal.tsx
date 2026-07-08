import React, { useEffect, useRef, useState } from 'react';
import { Alert, Button, Space, Tag, Typography } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { FitAddon } from '@xterm/addon-fit';
import { Terminal } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';
import { buildConsoleWebSocketUrl, createConsoleTicket } from '../api/console';

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
      scrollback: 1000,
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

    const writeToTerminal = (data: string | Uint8Array) => {
      terminal.write(data);
    };

    const writeLineToTerminal = (data: string | Uint8Array) => {
      terminal.writeln(data);
    };

    let fitFrame: number | null = null;
    const scheduleFit = () => {
      if (fitFrame !== null) return;
      fitFrame = window.requestAnimationFrame(() => {
        fitFrame = null;
        fitAddon.fit();
      });
    };

    window.addEventListener('resize', scheduleFit);

    if (mock) {
      queueMicrotask(() => updateConsoleState('connected', 'Debug console is connected.'));
      writeLineToTerminal('Kite debug serial console');
      writeLineToTerminal(`Connected to ${vmName}`);
      writeToTerminal('$ ');
      const input = terminal.onData((data) => {
        if (data === '\r') {
          writeToTerminal('\r\n$ ');
          return;
        }
        writeToTerminal(data);
      });
      return () => {
        active = false;
        input.dispose();
        if (fitFrame !== null) {
          window.cancelAnimationFrame(fitFrame);
        }
        window.removeEventListener('resize', scheduleFit);
        terminal.dispose();
      };
    }

    let socket: WebSocket | null = null;
    const ticketAbortController = new AbortController();
    const decoder = new TextDecoder();
    queueMicrotask(() => updateConsoleState('connecting', 'Requesting console ticket...'));
    writeLineToTerminal('Connecting to Kite VM serial console...');

    void createConsoleTicket(vmName, { signal: ticketAbortController.signal })
      .then((ticket) => {
        if (!active || ticketAbortController.signal.aborted) return;
        updateConsoleState('connecting', 'Opening console WebSocket...');
        socket = new WebSocket(buildConsoleWebSocketUrl(vmName, ticket.ticket), ['plain.kubevirt.io']);
        socket.binaryType = 'arraybuffer';
        socket.onopen = () => {
          if (!active) return;
          updateConsoleState('connected', 'Console is connected.');
          writeLineToTerminal('Connected. Serial console output will appear below.');
        };
        socket.onmessage = (event) => {
          if (!active) return;
          if (event.data instanceof ArrayBuffer) {
            writeToTerminal(decoder.decode(event.data));
            return;
          }
          if (typeof event.data === 'string') {
            writeToTerminal(event.data);
          }
        };
        socket.onerror = () => {
          if (!active) return;
          updateConsoleState('failed', 'Console connection failed.');
        };
        socket.onclose = () => {
          if (!active) return;
          setStatus((current) => current === 'failed' ? 'failed' : 'closed');
          setMessage('Console connection closed.');
        };
      })
      .catch((error: unknown) => {
        if (!active || ticketAbortController.signal.aborted) return;
        updateConsoleState('failed', error instanceof Error ? error.message : 'Could not create console ticket.');
      });

    const input = terminal.onData((data) => {
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(data);
      }
    });

    return () => {
      active = false;
      ticketAbortController.abort();
      input.dispose();
      if (fitFrame !== null) {
        window.cancelAnimationFrame(fitFrame);
      }
      window.removeEventListener('resize', scheduleFit);
      if (socket) {
        socket.onopen = null;
        socket.onmessage = null;
        socket.onerror = null;
        socket.onclose = null;
        socket.close();
      }
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
      <Alert title={message} type={status === 'failed' ? 'error' : 'info'} showIcon className="console-status" />
      <div ref={containerRef} className="console-terminal" />
    </div>
  );
};
