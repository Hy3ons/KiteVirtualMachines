import { DEMO_WIDTH } from './demoStory';
import { createDemoScroll } from './demoScroll';
import { createDemoCamera } from './demoCamera';
import { createDemoAnimation } from './demoAnimation';
export type { DemoPlayback } from './demoAnimation';

class MissingDemoMarkupError extends Error { readonly name = 'MissingDemoMarkupError'; }

// createDemoTimeline animates a virtual pointer over the real demo DOM and advances the story clock.
// root scopes all targets and onTick publishes local cycle seconds, including every infinite repeat.
export function createDemoTimeline(root: HTMLElement, onTick: (time: number) => void) {
  const canvas = root.querySelector<HTMLElement>('[data-demo-canvas]');
  const layer = root.querySelector<HTMLElement>('[data-demo-camera]');
  const pointer = root.querySelector<HTMLElement>('[data-demo-pointer]');
  const ring = root.querySelector<HTMLElement>('[data-demo-click]');
  const glyph = pointer?.querySelector('svg');
  const progress = root.querySelector<HTMLElement>('[data-demo-progress]');
  if (!canvas || !layer || !pointer || !ring || !glyph || !progress) throw new MissingDemoMarkupError('Product demo canvas is incomplete');
  const scroll = createDemoScroll(root);
  const camera = createDemoCamera(root, layer);

  // point measures a rendered target in unscaled canvas coordinates, including inline overlays.
  // Missing targets during a chapter jump use a quiet resting position until their scene is mounted.
  const point = (selector: string) => {
    const target = root.querySelector<HTMLElement>(selector);
    if (!target) return { x: 80, y: 180 };
    const bounds = canvas.getBoundingClientRect();
    const rect = target.getBoundingClientRect();
    const scale = bounds.width / DEMO_WIDTH;
    return { x: (rect.left - bounds.left + rect.width / 2) / scale, y: (rect.top - bounds.top + rect.height / 2) / scale };
  };

  return createDemoAnimation({ pointer, glyph, ring, progress, point, scroll, camera }, onTick);
}
