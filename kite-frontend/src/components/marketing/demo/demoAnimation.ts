import { gsap } from 'gsap';
import { DEMO_DURATION, DEMO_PLAYBACK_RATE } from './demoStory';

type DemoMotion = { readonly update: (time: number) => void; readonly reset: () => void };
type DemoAnimationTargets = {
  readonly pointer: gsap.TweenTarget;
  readonly glyph: gsap.TweenTarget;
  readonly ring: gsap.TweenTarget;
  readonly progress: gsap.TweenTarget;
  readonly point: (selector: string) => { readonly x: number; readonly y: number };
  readonly scroll: DemoMotion;
  readonly camera: DemoMotion;
};

// createDemoAnimation owns the story clock and pointer motion over explicit GSAP targets.
// point measures the current scene; onTick publishes cycle seconds to the React controller.
// DOM discovery, camera and scroll adapters are provided by createDemoTimeline.
export function createDemoAnimation({ pointer, glyph, ring, progress, point, scroll, camera }: DemoAnimationTargets, onTick: (time: number) => void) {
  const timeline = gsap.timeline({ paused: true, repeat: -1, repeatRefresh: true,
    onUpdate: () => { onTick(timeline.time()); scroll.update(timeline.time()); camera.update(timeline.time()); },
    onRepeat: () => { scroll.reset(); camera.reset(); onTick(0); },
  }).timeScale(DEMO_PLAYBACK_RATE);

  const context = gsap.context(() => {
    gsap.set(pointer, { x: 80, y: 180, autoAlpha: 1 });
    gsap.set(glyph, { transformOrigin: '0px 0px', rotation: 0, scale: 1 });
    timeline.set(pointer, { x: 80, y: 180, autoAlpha: 1 }, 0);
    timeline.set(glyph, { rotation: 0, scale: 1 }, 0);
    timeline.set(ring, { autoAlpha: 0 }, 0);
    timeline.fromTo(progress, { scaleX: 0 }, { scaleX: 1, duration: DEMO_DURATION, ease: 'none' }, 0);
    // move measures at tween start, after React has committed the scene or popup.
    const move = (selector: string, at: number, duration = .6) => {
      timeline.to(pointer, { x: () => point(selector).x, y: () => point(selector).y, duration, ease: 'power3.inOut' }, at);
      timeline.to(glyph, { rotation: () => point(selector).x > Number(gsap.getProperty(pointer, 'x')) ? -6 : 6, duration: .18, ease: 'power2.out' }, at);
      timeline.to(glyph, { rotation: 0, duration: .2, ease: 'power2.out' }, at + duration - .2);
    };
    // click is visual feedback only: it never dispatches a DOM event or submits a real form.
    const click = (at: number) => {
      timeline.to(glyph, { scale: .9, duration: .1, yoyo: true, repeat: 1, ease: 'power2.out' }, at);
      timeline.fromTo(ring, { scale: .5, autoAlpha: .6 }, { scale: 1.7, autoAlpha: 0, duration: .32, ease: 'power2.out', immediateRender: false }, at);
    };
    move('#demo_login_username', .4); click(1.02);
    move('#demo_login_password', 2.4, .45); click(2.87);
    move('.demo-login .marketing-submit', 3.85); click(4.5);
    move('.demo-dashboard .workspace-list-controls .ant-btn-primary', 5.8); click(6.45);
    move('#demo_vm_create_name', 7.2, .25); click(7.48);
    move('#demo_vm_create_domainPrefix', 9, .45); click(9.48);
    move('#demo_vm_create_sshId', 11, .45); click(11.48);
    move('#demo_vm_create_sshPassword', 12.6, .45); click(13.08);
    move('.workspace-vm-modal .form-actions .ant-btn-primary', 14.55); click(15.2);
    move('button[aria-label="demo-vm 작업 메뉴"]', 18.2); click(18.85);
    move('.workspace-vm-actions-menu .ant-dropdown-menu-item:nth-child(2)', 19.45, .5); click(20.05);
    move('.demo-terminal-output', 21.4, .3); click(21.75);
    move('.console-sidebar .sidebar-link[href="/dashboard"]', 41.8, .5); click(42.35);
    move('button[aria-label="demo-vm 작업 메뉴"]', 43.8, .5); click(44.4);
    move('.workspace-vm-actions-menu .ant-dropdown-menu-item-danger', 45.05, .4); click(45.5);
    move('.demo-delete-modal .ant-modal-footer .ant-btn-dangerous', 46.5, .3); click(46.85);
    move('.console-brand', 49.7, .45); click(50.2);
    move('.demo-landing .marketing-button--primary', 52.8, .5); click(53.45);
    timeline.to(pointer, { autoAlpha: 0, duration: .2 }, 53.72);
  });

  return {
    timeline,
    // seek recalculates DOM geometry; the caller renders the corresponding snapshot first.
    seek(time: number) {
      scroll.update(time);
      camera.update(time);
      timeline.pause().invalidate().time(time, false);
      if (time === 0) gsap.set(pointer, { x: 80, y: 180 });
      gsap.set(pointer, { autoAlpha: time < 53.72 ? 1 : 0 });
      gsap.set(glyph, { rotation: 0, scale: 1 });
      gsap.set(ring, { autoAlpha: 0 });
    },
    // dispose kills the parent timeline as well as all context-owned child tweens.
    dispose() { timeline.kill(); context.revert(); scroll.reset(); camera.reset(); },
  };
}

export type DemoPlayback = ReturnType<typeof createDemoAnimation>;
