import { describe, expect, it, vi } from 'vitest';
import { createDemoAnimation } from '../src/components/marketing/demo/demoAnimation';
import { DEMO_DURATION, DEMO_PLAYBACK_RATE, demoSnapshotAt } from '../src/components/marketing/demo/demoStory';


// timelineFixture supplies plain GSAP targets and a measured point without a browser.
// The real animation engine must wrap the story clock and remeasure targets each cycle.
function timelineFixture() {
  let targetX = 400;
  const glyph = { rotation: 0, scale: 1, transformOrigin: '' };
  const pointer = { x: 0, y: 0, autoAlpha: 1, querySelector: () => glyph };
  const ring = { scale: 1, autoAlpha: 0 };
  const progress = { scaleX: 0 };
  const tick = vi.fn<(time: number) => void>();
  return { playback: createDemoAnimation({ pointer, glyph, ring, progress, point: () => ({ x: targetX + 50, y: 220 }), scroll: { update: vi.fn(), reset: vi.fn() }, camera: { update: vi.fn(), reset: vi.fn() } }, tick), tick, pointer, progress, moveTarget: (x: number) => { targetX = x; } };
}

// lastTick requires a real animation callback before checking the current scene.
function lastTick(call: readonly [number] | undefined): number {
  if (!call) throw new TypeError('The animation did not publish its current time');
  return call[0];
}

describe('real GSAP repetition without a browser', () => {
  it('loops from landing back to login, restoring the pointer on consecutive cycles', () => {
    const { playback, tick, pointer, progress } = timelineFixture();
    try {
      expect(playback.timeline.repeat()).toBe(-1);
      expect(playback.timeline.timeScale()).toBe(DEMO_PLAYBACK_RATE);
      for (let cycle = 0; cycle < 3; cycle++) {
        playback.timeline.totalTime(cycle * DEMO_DURATION + 53.98, false);
        expect(demoSnapshotAt(lastTick(tick.mock.lastCall)).scene).toBe('landing');
        expect(pointer.autoAlpha).toBe(0);
        playback.timeline.totalTime((cycle + 1) * DEMO_DURATION + .1, false);
        expect(playback.timeline.time()).toBeCloseTo(.1);
        expect(demoSnapshotAt(lastTick(tick.mock.lastCall))).toMatchObject({ scene: 'login', hasVm: false, loginEmail: '' });
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
