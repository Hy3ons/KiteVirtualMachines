import { describe, expect, it } from 'vitest';
import { demoCameraAt, demoCameraFrame } from '../src/components/marketing/demo/demoCamera';
import { DEMO_DURATION, DEMO_PLAYBACK_RATE } from '../src/components/marketing/demo/demoStory';

const frame = (_selector: string, zoom: number) => demoCameraFrame({ x: 960, y: 380 }, zoom, 760);

describe('demo camera framing without a browser', () => {
  it('centers a target inside the visible crop rather than the hidden browser bottom', () => {
    expect(demoCameraFrame({ x: 960, y: 380 }, 2, 760)).toEqual({ x: -960, y: -380, scale: 2 });
  });
  it('clamps corner framing to keep the page covering the visible viewport', () => {
    expect(demoCameraFrame({ x: 0, y: 0 }, 2, 760)).toEqual({ x: 0, y: 0, scale: 2 });
    expect(demoCameraFrame({ x: 1920, y: 920 }, 2, 760)).toEqual({ x: -1920, y: -1080, scale: 2 });
  });
  it('zooms into login, creation fields, actions and the console', () => {
    expect(demoCameraAt(2, frame).scale).toBe(1.75);
    expect(demoCameraAt(9.5, frame).scale).toBe(2);
    expect(demoCameraAt(19.5, frame).scale).toBe(1.85);
    expect(demoCameraAt(25, frame).scale).toBe(1.75);
    expect(demoCameraAt(46.5, frame).scale).toBe(1.7);
    expect(demoCameraAt(52, frame).scale).toBe(1.55);
  });
  it('returns to overview before each scene change and after the last command', () => {
    for (const time of [0, 5.2, 6.79, 15.9, 20.75, 41.75, 45.83, 47.46, 49.65, 53.96]) expect(demoCameraAt(time, frame)).toEqual({ x: 0, y: 0, scale: 1 });
  });
  it('replay and a missing scene never retain another scene zoom', () => {
    demoCameraAt(25, frame);
    expect(demoCameraAt(0, frame)).toEqual({ x: 0, y: 0, scale: 1 });
    expect(demoCameraAt(25, () => null)).toEqual({ x: 0, y: 0, scale: 1 });
  });
  it('completes commands, deletion and landing in about 39 elapsed seconds per loop', () => {
    expect(DEMO_DURATION / DEMO_PLAYBACK_RATE).toBeGreaterThan(38);
    expect(DEMO_DURATION / DEMO_PLAYBACK_RATE).toBeLessThan(40);
  });
});
