import { DEMO_HEIGHT, DEMO_WIDTH } from './demoStory';
import { demoScrollProgress } from './demoScroll';

export type DemoCameraPose = { readonly x: number; readonly y: number; readonly scale: number };
type CameraCue = { readonly at: number; readonly duration: number; readonly zoom: number; readonly target?: string };
const overview: DemoCameraPose = { x: 0, y: 0, scale: 1 };
const cues: readonly CameraCue[] = [
  { at: .05, duration: .3, zoom: 1.75, target: '.marketing-auth-panel' },
  { at: 4.72, duration: .45, zoom: 1 },
  { at: 5.32, duration: .43, zoom: 1.65, target: '.workspace-list-controls .ant-btn-primary' },
  { at: 6.5, duration: .28, zoom: 1 },
  { at: 6.85, duration: .3, zoom: 2, target: '#demo_vm_create_name' },
  { at: 8.55, duration: .35, zoom: 2, target: '#demo_vm_create_domainPrefix' },
  { at: 10.55, duration: .35, zoom: 2, target: '#demo_vm_create_sshId' },
  { at: 12.32, duration: .23, zoom: 2, target: '#demo_vm_create_sshPassword' },
  { at: 14.05, duration: .4, zoom: 1.9, target: '.workspace-vm-modal .form-actions .ant-btn-primary' },
  { at: 15.4, duration: .45, zoom: 1 },
  { at: 16.25, duration: .55, zoom: 1.75, target: 'button[aria-label="demo-vm 작업 메뉴"]' },
  { at: 19.05, duration: .3, zoom: 1.85, target: '.workspace-vm-actions-menu' },
  { at: 20.27, duration: .43, zoom: 1 },
  { at: 20.95, duration: .4, zoom: 1.75, target: '.demo-console .console-terminal-frame' },
  { at: 41.2, duration: .5, zoom: 1 },
  { at: 43, duration: .6, zoom: 1.75, target: 'button[aria-label="demo-vm 작업 메뉴"]' },
  { at: 44.6, duration: .35, zoom: 1.85, target: '.workspace-vm-actions-menu' },
  { at: 45.55, duration: .27, zoom: 1 },
  { at: 46, duration: .35, zoom: 1.7, target: '.demo-delete-modal .ant-modal-container' },
  { at: 47.05, duration: .4, zoom: 1 },
  { at: 48.45, duration: .5, zoom: 1.4, target: '.workspace-resource-panel' },
  { at: 49.3, duration: .3, zoom: 1 },
  { at: 51.2, duration: .4, zoom: 1.55, target: '.demo-landing .marketing-button--primary' },
  { at: 53.68, duration: .26, zoom: 1 },
];

// demoCameraFrame centers a world-space point in the visible browser crop.
// zoom sets magnification; visibleHeight excludes the part clipped by the hero.
export function demoCameraFrame(point: { readonly x: number; readonly y: number }, zoom: number, visibleHeight: number): DemoCameraPose {
  return {
    x: Math.max(DEMO_WIDTH * (1 - zoom), Math.min(0, DEMO_WIDTH / 2 - point.x * zoom)),
    y: Math.max(visibleHeight - DEMO_HEIGHT * zoom, Math.min(0, visibleHeight / 2 - point.y * zoom)),
    scale: zoom,
  };
}

// demoCameraAt resolves framing from story time without retaining previous transitions.
// frame supplies current DOM measurements; missing targets resolve to the overview.
export function demoCameraAt(time: number, frame: (selector: string, zoom: number) => DemoCameraPose | null): DemoCameraPose {
  let pose = overview;
  for (const cue of cues) {
    if (time < cue.at) break;
    const destination = cue.target ? frame(cue.target, cue.zoom) ?? overview : overview;
    const progress = demoScrollProgress(time, cue.at, cue.duration);
    pose = { x: pose.x + (destination.x - pose.x) * progress, y: pose.y + (destination.y - pose.y) * progress, scale: pose.scale + (destination.scale - pose.scale) * progress };
  }
  return pose;
}

// createDemoCamera pans only layer, leaving browser chrome and the cursor fixed.
// root owns the demo bounds and lookup scope; DOM coordinates are inverted through the current camera.
export function createDemoCamera(root: HTMLElement, layer: HTMLElement) {
  const motion = window.matchMedia('(prefers-reduced-motion: reduce)');
  let pose = overview;
  const render = (next: DemoCameraPose) => {
    pose = next;
    layer.style.transform = `translate3d(${next.x}px, ${next.y}px, 0) scale(${next.scale})`;
  };
  return {
    update(time: number) {
      const canvas = layer.parentElement;
      if (!canvas || motion.matches) { render(overview); return; }
      const bounds = canvas.getBoundingClientRect();
      const scale = bounds.width / DEMO_WIDTH;
      if (scale <= 0) return;
      const hero = root.closest('.marketing-hero');
      const visibleHeight = hero ? Math.max(1, Math.min(DEMO_HEIGHT, (hero.getBoundingClientRect().bottom - bounds.top) / scale)) : DEMO_HEIGHT;
      const next = demoCameraAt(time, (selector, zoom) => {
        const target = layer.querySelector<HTMLElement>(selector);
        if (!target) return null;
        const rect = target.getBoundingClientRect();
        return demoCameraFrame({ x: ((rect.left + rect.width / 2 - bounds.left) / scale - pose.x) / pose.scale, y: ((rect.top + rect.height / 2 - bounds.top) / scale - pose.y) / pose.scale }, zoom, visibleHeight);
      });
      render(next);
    },
    // reset restores the overview when the GSAP lifecycle is disposed.
    reset() { render(overview); },
  };
}
