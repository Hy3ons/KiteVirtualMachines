import { describe, expect, it } from 'vitest';
import { demoScrollProgress, demoScrollTop } from '../src/components/marketing/demo/demoScroll';

describe('bounded demo scrolling without a browser', () => {
  it('brings a lower field into view while keeping room above it', () => {
    expect(demoScrollTop({ top: 600, height: 40, viewport: 520, content: 1100 }, .3)).toBe(456);
  });
  it('clamps a submit button near the bottom to the available scroll range', () => {
    expect(demoScrollTop({ top: 1000, height: 40, viewport: 520, content: 1100 }, .3)).toBe(580);
  });
  it('never overscrolls an upper target or a page shorter than the viewport', () => {
    expect(demoScrollTop({ top: 40, height: 40, viewport: 520, content: 1100 }, .3)).toBe(0);
    expect(demoScrollTop({ top: 400, height: 40, viewport: 520, content: 460 }, .3)).toBe(0);
  });
  it('replaying and seeking derive the same scroll progress without accumulated deltas', () => {
    expect(demoScrollProgress(8, 8.5, .4)).toBe(0);
    expect(demoScrollProgress(9, 8.5, .4)).toBe(1);
    expect(demoScrollProgress(8.7, 8.5, .4)).toBeCloseTo(.5);
    expect(demoScrollProgress(8, 8.5, .4)).toBe(0);
  });
});
