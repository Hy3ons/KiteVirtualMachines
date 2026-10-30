type ScrollTargetMetrics = {
  readonly top: number;
  readonly height: number;
  readonly viewport: number;
  readonly content: number;
};

// demoScrollTop places a measured target at alignment within its local scroll viewport.
// metrics use unscaled content pixels; the result is clamped to the actual scroll range.
export function demoScrollTop(metrics: ScrollTargetMetrics, alignment: number): number {
  const desired = metrics.top - (metrics.viewport - metrics.height) * alignment;
  return Math.max(0, Math.min(desired, metrics.content - metrics.viewport));
}

// demoScrollProgress derives a reversible cubic scroll ease from GSAP story seconds.
// start and duration identify one cue; seeking before/after it clamps to its endpoints.
export function demoScrollProgress(time: number, start: number, duration: number): number {
  const progress = Math.max(0, Math.min(1, (time - start) / duration));
  return progress < .5 ? 4 * progress ** 3 : 1 - (-2 * progress + 2) ** 3 / 2;
}

type ScrollCue = { readonly at: number; readonly duration: number; readonly target: string; readonly alignment: number };
type ScrollTrack = { readonly enter: number; readonly leave: number; readonly area: string; readonly cues: readonly ScrollCue[] };
const tracks: readonly ScrollTrack[] = [
  { enter: 5.3, leave: 6.8, area: '.console-main', cues: [
    { at: 5.35, duration: .4, target: '.workspace-list-controls .ant-btn-primary', alignment: .35 },
  ] },
  { enter: 6.8, leave: 16, area: '.workspace-vm-modal .ant-modal-body', cues: [
    { at: 6.85, duration: .1, target: '#demo_vm_create_name', alignment: .3 },
    { at: 8.5, duration: .4, target: '#demo_vm_create_domainPrefix', alignment: .3 },
    { at: 10.5, duration: .4, target: '#demo_vm_create_sshId', alignment: .3 },
    { at: 12.32, duration: .23, target: '#demo_vm_create_sshPassword', alignment: .3 },
    { at: 14.05, duration: .4, target: '.form-actions .ant-btn-primary', alignment: .65 },
  ] },
  { enter: 16, leave: 20.8, area: '.console-main', cues: [
    { at: 16.1, duration: .6, target: 'button[aria-label="demo-vm 작업 메뉴"]', alignment: .35 },
  ] },
  { enter: 20.8, leave: 42.8, area: '.console-main', cues: [
    { at: 20.85, duration: .2, target: '.console-panel', alignment: .05 },
  ] },
  { enter: 42.8, leave: 50.8, area: '.console-main', cues: [
    { at: 42.85, duration: .55, target: 'button[aria-label="demo-vm 작업 메뉴"]', alignment: .35 },
    { at: 48.35, duration: .5, target: '.workspace-resource-panel', alignment: .2 },
  ] },
];

// createDemoScroll scopes native scrolling to the embedded application and modal body.
// update resolves each cue from current geometry/time; it never scrolls the host document.
export function createDemoScroll(root: HTMLElement) {
  return {
    update(time: number) {
      for (const track of tracks) {
        if (time < track.enter || time >= track.leave) continue;
        const area = root.querySelector<HTMLElement>(track.area);
        if (!area) continue;
        const bounds = area.getBoundingClientRect();
        const scale = bounds.width / area.offsetWidth;
        if (!Number.isFinite(scale) || scale <= 0) continue;
        let position = 0;
        for (const cue of track.cues) {
          if (time < cue.at) break;
          const target = area.querySelector<HTMLElement>(cue.target);
          if (!target) continue;
          const rect = target.getBoundingClientRect();
          const destination = demoScrollTop({ top: (rect.top - bounds.top) / scale + area.scrollTop, height: rect.height / scale, viewport: area.clientHeight, content: area.scrollHeight }, cue.alignment);
          position += (destination - position) * demoScrollProgress(time, cue.at, cue.duration);
        }
        area.scrollTop = position;
      }
      const terminal = root.querySelector<HTMLElement>('.demo-console .console-terminal-frame');
      if (terminal) terminal.scrollTop = Math.max(0, terminal.scrollHeight - terminal.clientHeight);
    },
    // reset restores local scroll state during timeline disposal without touching other pages.
    reset() {
      root.querySelectorAll<HTMLElement>('.console-main, .ant-modal-body, .console-terminal-frame').forEach((area) => { area.scrollTop = 0; });
    },
  };
}
