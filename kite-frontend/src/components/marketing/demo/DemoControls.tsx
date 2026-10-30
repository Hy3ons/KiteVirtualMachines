import { PauseOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons';
import { DEMO_CHAPTERS } from './demoStory';
import type { useDemoPlayback } from './useDemoPlayback';

// DemoControls is the only interactive area; the simulated app itself stays inert.
// playback exposes user intent; chapter jumps are available only in explicit debug mode.
export function DemoControls({ playback }: { readonly playback: ReturnType<typeof useDemoPlayback> }) {
  const { playing, ready, reduced, failed, snapshot } = playback;
  const showChapters = process.env.NEXT_PUBLIC_KITE_MODE === 'debug';
  return <figcaption className="demo-caption">
    <div className="demo-controls">
      {showChapters && <nav className="demo-chapters" aria-label="제품 시연 단계">{DEMO_CHAPTERS.map((chapter, index) => <button key={chapter.id} type="button" aria-current={snapshot.scene === chapter.id ? 'step' : undefined} onClick={() => playback.chapter(chapter.id)} disabled={!ready}><span>0{index + 1}</span>{chapter.label}</button>)}</nav>}
      <div className="demo-playback-controls"><button type="button" onClick={playback.toggle} disabled={!ready || reduced} aria-label={playing ? '제품 시연 일시정지' : '제품 시연 재생'}>{playing ? <PauseOutlined /> : <PlayCircleOutlined />}<span>{playing ? '일시정지' : '재생'}</span></button><button type="button" onClick={playback.replay} disabled={!ready || reduced}><ReloadOutlined /><span>다시 보기</span></button></div>
    </div>
    {(failed || reduced) && <p className="demo-disclosure">{failed ? '시연을 불러오지 못했습니다. 제품 화면 예시를 표시합니다.' : showChapters ? '모션 감소 설정 · 단계 버튼으로 살펴보세요.' : '모션 감소 설정에 따라 정지 화면을 표시합니다.'}</p>}
  </figcaption>;
}
