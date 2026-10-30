'use client';

import { useEffect, useRef, useState } from 'react';
import type { RefObject } from 'react';
import { flushSync } from 'react-dom';
import { DEMO_CHAPTERS, DEMO_DURATION, DEMO_WIDTH, demoSnapshotAt } from './demoStory';
import type { DemoScene } from './demoStory';
import type { DemoPlayback } from './demoTimeline';

// useDemoPlayback owns deferred GSAP loading, lifecycle cleanup and user playback intent.
// root is the bounded demo figure; only local snapshots and pointer styles are changed.
export function useDemoPlayback(root: RefObject<HTMLElement | null>) {
  const [snapshot, setSnapshot] = useState(() => demoSnapshotAt(0));
  const [scale, setScale] = useState(900 / DEMO_WIDTH);
  const [status, setStatus] = useState({ ready: false, playing: false, reduced: false, failed: false });
  const playback = useRef<DemoPlayback | null>(null);
  const key = useRef(snapshot.key);
  const intent = useRef(true);
  const visible = useRef(false);
  const reduced = useRef(false);

  // publish skips React updates while only the pointer moves; characters/stages update discretely.
  const publish = (time: number) => {
    const next = demoSnapshotAt(time);
    if (key.current !== next.key) { key.current = next.key; setSnapshot(next); }
  };
  const sync = () => {
    const playing = Boolean(playback.current && intent.current && visible.current && !document.hidden && !reduced.current);
    if (playing) playback.current?.timeline.resume(); else playback.current?.timeline.pause();
    setStatus((value) => value.playing === playing ? value : { ...value, playing });
  };

  useEffect(() => {
    const element = root.current;
    if (!element) return;
    let active = true;
    let loading = false;
    const motion = window.matchMedia('(prefers-reduced-motion: reduce)');
    reduced.current = motion.matches;
    // load only imports the animation engine after the demonstration enters the viewport.
    const load = async () => {
      if (loading || playback.current) return;
      loading = true;
      try {
        const { createDemoTimeline } = await import('./demoTimeline');
        if (!active) return;
        playback.current = createDemoTimeline(element, publish);
        setStatus({ ready: true, playing: false, reduced: reduced.current, failed: false });
        if (reduced.current) {
          intent.current = false;
          flushSync(() => publish(24.2));
          playback.current.seek(24.2);
        }
        sync();
      } catch (error: unknown) {
        if (!(error instanceof Error)) throw error;
        if (active) setStatus({ ready: false, playing: false, reduced: reduced.current, failed: true });
      }
    };
    const resize = new ResizeObserver(([entry]) => {
      if (entry && entry.contentRect.width > 0) setScale(entry.contentRect.width / DEMO_WIDTH);
    });
    const viewport = element.querySelector('[data-demo-viewport]');
    if (viewport) resize.observe(viewport);
    const observer = new IntersectionObserver(([entry]) => {
      visible.current = Boolean(entry?.isIntersecting && entry.intersectionRatio >= .15);
      if (visible.current) void load();
      sync();
    }, { threshold: [0, .15] });
    observer.observe(element);
    const onMotionChange = () => {
      reduced.current = motion.matches;
      setStatus((value) => ({ ...value, reduced: motion.matches }));
      if (motion.matches) {
        intent.current = false;
        const current = playback.current;
        if (current) current.seek(current.timeline.time());
      }
      sync();
    };
    document.addEventListener('visibilitychange', sync);
    motion.addEventListener('change', onMotionChange);
    return () => {
      active = false;
      resize.disconnect(); observer.disconnect();
      document.removeEventListener('visibilitychange', sync);
      motion.removeEventListener('change', onMotionChange);
      playback.current?.dispose(); playback.current = null;
    };
  // One effect owns each observer/timeline; callbacks read mutable playback intent, not snapshots.
  }, [root]);

  // jump renders first so seeking measures the correct form, menu or console DOM.
  const jumpTo = (time: number) => {
    intent.current = false;
    flushSync(() => publish(time));
    playback.current?.seek(time);
    sync();
  };
  const toggle = () => {
    if (reduced.current) return;
    const timeline = playback.current?.timeline;
    if (timeline && timeline.time() >= DEMO_DURATION) {
      flushSync(() => publish(0)); playback.current?.seek(0);
    }
    intent.current = !status.playing;
    sync();
  };
  const replay = () => {
    jumpTo(0); intent.current = !reduced.current; sync();
  };
  const chapter = (scene: DemoScene) => {
    const chapterTime = DEMO_CHAPTERS.find((item) => item.id === scene)?.time;
    if (chapterTime !== undefined) jumpTo(chapterTime);
  };
  return { snapshot, scale, ...status, toggle, replay, chapter };
}
