import { describe, expect, it, vi } from 'vitest';
import { createDemoTimeline } from '../src/components/marketing/demo/demoTimeline';
import { DEMO_DURATION, DEMO_PLAYBACK_RATE, DEMO_WIDTH, demoSnapshotAt } from '../src/components/marketing/demo/demoStory';

vi.mock('../src/components/marketing/demo/demoScroll', () => ({ createDemoScroll: () => ({ update: vi.fn(), reset: vi.fn() }) }));
vi.mock('../src/components/marketing/demo/demoCamera', () => ({ createDemoCamera: () => ({ update: vi.fn(), reset: vi.fn() }) }));

// timelineFixture supplies plain-object GSAP targets and geometry without a browser or DOM engine.
// Only scrolling/camera adapters are stubbed; the real GSAP timeline must wrap and refresh targets.
function timelineFixture() {
  let targetX = 400;
  const glyph = { rotation: 0, scale: 1, transformOrigin: '' };
  const pointer = { x: 0, y: 0, autoAlpha: 1, querySelector: () => glyph };
  const ring = { scale: 1, autoAlpha: 0 };
  const progress = { scaleX: 0 };
  const canvas = { getBoundingClientRect: () => ({ left: 0, top: 0, width: DEMO_WIDTH }) };
  const target = { getBoundingClientRect: () => ({ left: targetX, top: 200, width: 100, height: 40 }) };
  const markup: Record<string, object> = { '[data-demo-canvas]': canvas, '[data-demo-camera]': {}, '[data-demo-pointer]': pointer, '[data-demo-click]': ring, '[data-demo-progress]': progress };
  const root = { querySelector: (selector: string) => markup[selector] ?? target, querySelectorAll: () => [] } as unknown as HTMLElement;
  const tick = vi.fn<(time: number) => void>();
  return { playback: createDemoTimeline(root, tick), tick, pointer, progress, moveTarget: (x: number) => { targetX = x; } };
}

describe('real GSAP repetition without a browser', () => {
  it('loops from landing back to login, restoring the pointer on consecutive cycles', () => {
    const { playback, tick, pointer, progress } = timelineFixture();
    try {
      expect(playback.timeline.repeat()).toBe(-1);
      expect(playback.timeline.timeScale()).toBe(DEMO_PLAYBACK_RATE);
      for (let cycle = 0; cycle < 3; cycle++) {
        playback.timeline.totalTime(cycle * DEMO_DURATION + 53.98, false);
        expect(demoSnapshotAt(tick.mock.lastCall![0]).scene).toBe('landing');
        expect(pointer.autoAlpha).toBe(0);
        playback.timeline.totalTime((cycle + 1) * DEMO_DURATION + .1, false);
        expect(playback.timeline.time()).toBeCloseTo(.1);
        expect(demoSnapshotAt(tick.mock.lastCall![0])).toMatchObject({ scene: 'login', hasVm: false, loginEmail: '' });
        expect(pointer.autoAlpha).toBe(1);
        expect(progress.scaleX).toBeLessThan(.01);
      }
    } finally { playback.dispose(); }
  });
  it('remeasures pointer targets on the next loop and preserves pause/chapter seeking', () => {
    const { playback, pointer, moveTarget } = timelineFixture();
    try {
      playback.timeline.totalTime(1.05, false);
      expect(pointer.x).toBeCloseTo(450);
      moveTarget(700);
      playback.timeline.totalTime(DEMO_DURATION + 1.05, false);
      expect(pointer.x).toBeCloseTo(750);
      playback.seek(46.5);
      expect(playback.timeline.paused()).toBe(true);
      expect(playback.timeline.time()).toBeCloseTo(46.5);
      expect(demoSnapshotAt(playback.timeline.time()).deleteConfirm).toBe(true);
      playback.seek(0);
      expect(pointer).toMatchObject({ x: 80, y: 180, autoAlpha: 1 });
    } finally { playback.dispose(); }
  });
});
