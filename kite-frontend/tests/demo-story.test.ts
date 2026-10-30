import { describe, expect, it } from 'vitest';
import { DEMO_CHAPTERS, DEMO_DURATION, demoSnapshotAt } from '../src/components/marketing/demo/demoStory';

describe('product demonstration clock (no browser or API)', () => {
  it('starts with an empty local login and no VM', () => {
    expect(demoSnapshotAt(0)).toMatchObject({ scene: 'login', loginEmail: '', loginPassword: '', hasVm: false, menuOpen: false });
  });
  it('types account information before showing the dashboard', () => {
    expect(demoSnapshotAt(1.7).loginEmail.length).toBeGreaterThan(0);
    expect(demoSnapshotAt(1.7).loginEmail).not.toBe('builder@kite.local');
    expect(demoSnapshotAt(5.4)).toMatchObject({ scene: 'dashboard', loginEmail: 'builder@kite.local', hasVm: false });
  });
  it('populates the same creation fields before provisioning', () => {
    expect(demoSnapshotAt(15.5)).toMatchObject({ scene: 'create', name: 'demo-vm', domainPrefix: 'demo-vm', sshId: 'builder', busy: true, hasVm: false });
    expect(demoSnapshotAt(16.5)).toMatchObject({ scene: 'dashboard', hasVm: true, phase: 'Creating', busy: false });
    expect(demoSnapshotAt(18)).toMatchObject({ scene: 'dashboard', hasVm: true, phase: 'Running' });
  });
  it('opens the row menu before entering the serial console', () => {
    expect(demoSnapshotAt(19.5)).toMatchObject({ scene: 'dashboard', menuOpen: true });
    expect(demoSnapshotAt(21)).toMatchObject({ scene: 'console', menuOpen: false });
    expect(demoSnapshotAt(21).terminal).toContain('demo-vm login:');
  });
  it('demonstrates OS login followed by commands and output', () => {
    const terminal = demoSnapshotAt(DEMO_DURATION).terminal;
    expect(terminal).toContain('login: builder\nPassword:');
    expect(terminal).toContain('whoami\nbuilder');
    expect(terminal).toContain('uname -s\nLinux');
    expect(terminal).toContain('echo "Hello from Kite"\nHello from Kite');
    expect(terminal).not.toContain('KiteDemo2026!');
  });
  it('types directory and Git commands with matching working-directory prompts and output', () => {
    expect(demoSnapshotAt(28.6).terminal).toMatch(/\$ mkdir k$/);
    expect(demoSnapshotAt(29).terminal).not.toContain('git init');
    const terminal = demoSnapshotAt(41).terminal;
    expect(terminal).toContain('ls\nkite-demo');
    expect(terminal).toContain('cd kite-demo\nbuilder@demo-vm:~/kite-demo$ git init -b main');
    expect(terminal).toContain('Initialized empty Git repository in /home/builder/kite-demo/.git/');
    expect(terminal).toContain('touch README.md\nbuilder@demo-vm:~/kite-demo$ ls\nREADME.md');
    expect(terminal).toContain('git status --short\n?? README.md');
  });
  it('returns to the dashboard and confirms deletion before removing the VM', () => {
    expect(demoSnapshotAt(43)).toMatchObject({ scene: 'delete', hasVm: true, phase: 'Running', menuOpen: false, deleteConfirm: false });
    expect(demoSnapshotAt(45)).toMatchObject({ menuOpen: true, deleteConfirm: false });
    expect(demoSnapshotAt(46)).toMatchObject({ menuOpen: false, deleteConfirm: true, hasVm: true });
    expect(demoSnapshotAt(47.2)).toMatchObject({ deleteConfirm: true, busy: true, deleting: false });
    expect(demoSnapshotAt(48)).toMatchObject({ deleteConfirm: false, deleting: true, hasVm: true, phase: 'Terminating' });
    expect(demoSnapshotAt(49)).toMatchObject({ hasVm: false, deleting: false });
  });
  it('ends on the landing page with no VM and resets the next cycle to an empty login', () => {
    expect(demoSnapshotAt(51)).toMatchObject({ scene: 'landing', hasVm: false, menuOpen: false, deleteConfirm: false });
    for (let cycle = 0; cycle < 3; cycle++) {
      demoSnapshotAt(DEMO_DURATION);
      expect(demoSnapshotAt(0)).toMatchObject({ scene: 'login', hasVm: false, loginEmail: '', loginPassword: '', deleting: false, deleteConfirm: false });
    }
  });
  it('chapter jumps and replay derive a complete state without accumulated mutations', () => {
    for (const chapter of DEMO_CHAPTERS) expect(demoSnapshotAt(chapter.time).scene).toBe(chapter.id);
    const initial = demoSnapshotAt(0);
    demoSnapshotAt(DEMO_DURATION);
    expect(demoSnapshotAt(0)).toEqual(initial);
  });
});
