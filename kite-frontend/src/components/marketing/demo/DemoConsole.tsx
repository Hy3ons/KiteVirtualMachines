import { SerialConsoleFrame } from '../../SerialConsoleFrame';
import type { DemoSnapshot } from './demoStory';
import { DEMO_VM, demoIgnore } from './demoData';

// DemoConsole shares the live serial-console frame and renders deterministic text without a socket.
export function DemoConsole({ snapshot }: { readonly snapshot: DemoSnapshot }) {
  return <div className="serial-workspace demo-console">
    <SerialConsoleFrame status="connected" message="Connected. Serial console output will appear below." reconnectDisabled={false} onReconnect={demoIgnore}>
      <pre className="demo-terminal-output">{snapshot.terminal}<span className="demo-terminal-caret" /></pre>
    </SerialConsoleFrame>
    <div className="serial-support"><section className="machine-surface"><h2>연결 정보</h2><dl className="machine-properties"><div><dt>SSH 사용자</dt><dd>{DEMO_VM.sshId}</dd></div><div><dt>도메인</dt><dd>{DEMO_VM.domain}</dd></div><div><dt>이미지</dt><dd>{DEMO_VM.image}</dd></div><div><dt>전원</dt><dd>On</dd></div></dl></section><section className="machine-surface"><h2>시작하기 전에</h2><p>시리얼 콘솔은 VM의 ttyS0에 연결됩니다. VM을 만들 때 지정한 SSH ID와 초기 비밀번호로 로그인하세요.</p><p className="machine-section-caption">2 vCPU · 4Gi 메모리 · 25Gi 디스크</p></section></div>
  </div>;
}
