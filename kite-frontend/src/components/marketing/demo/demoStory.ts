export const DEMO_WIDTH = 1920;
export const DEMO_HEIGHT = 920;
export const DEMO_DURATION = 54;
export const DEMO_PLAYBACK_RATE = 1.4;
export const DEMO_CHAPTERS = [
  { id: 'login', label: '로그인', time: 0 },
  { id: 'dashboard', label: '대시보드', time: 5.4 },
  { id: 'create', label: 'VM 생성', time: 6.9 },
  { id: 'console', label: '콘솔', time: 24.2 },
  { id: 'delete', label: 'VM 삭제', time: 43 },
  { id: 'landing', label: '홈으로', time: 50.9 },
] as const;

export type DemoScene = typeof DEMO_CHAPTERS[number]['id'];
export type DemoSnapshot = ReturnType<typeof demoSnapshotAt>;

// typedText derives visible characters from timeline time, so seeking and replay never retain stale input.
function typedText(time: number, start: number, text: string, duration: number): string {
  return text.slice(0, Math.max(0, Math.min(text.length, Math.floor((time - start) / duration * text.length))));
}

const commands = [
  { start: 25, duration: .6, end: 26, text: 'whoami', output: 'builder' },
  { start: 27, duration: .3, end: 27.6, text: 'ls', output: '' },
  { start: 28.2, duration: .8, end: 29.2, text: 'mkdir kite-demo', output: '' },
  { start: 29.6, duration: .25, end: 30.1, text: 'ls', output: 'kite-demo' },
  { start: 30.6, duration: .6, end: 31.4, text: 'cd kite-demo', output: '', directory: '~/kite-demo' },
  { start: 32, duration: .9, end: 33.2, text: 'git init -b main', output: 'Initialized empty Git repository in /home/builder/kite-demo/.git/' },
  { start: 33.8, duration: .8, end: 34.8, text: 'touch README.md', output: '' },
  { start: 35.4, duration: .25, end: 35.9, text: 'ls', output: 'README.md' },
  { start: 36.5, duration: .9, end: 37.7, text: 'git status --short', output: '?? README.md' },
  { start: 38.2, duration: .6, end: 39, text: 'uname -s', output: 'Linux' },
  { start: 39.5, duration: 1.1, end: 40.8, text: 'echo "Hello from Kite"', output: 'Hello from Kite' },
] as const;

// terminalAt types inert example commands and appends output only after each command completes.
// time is the shared story clock; directory changes affect subsequent prompts without a real shell.
function terminalAt(time: number): string {
  let terminal = 'Ubuntu 22.04 LTS demo-vm ttyS0\n\ndemo-vm login: ' + typedText(time, 21.9, 'builder', .6);
  if (time >= 22.65) terminal += '\nPassword: ';
  if (time < 24) return terminal;
  let directory = '~';
  terminal += '\n\nWelcome to Ubuntu 22.04 LTS\n\nbuilder@demo-vm:~$ ';
  for (const command of commands) {
    if (time < command.start) break;
    terminal += typedText(time, command.start, command.text, command.duration);
    if (time < command.end) break;
    if ('directory' in command) directory = command.directory;
    terminal += '\n' + (command.output ? command.output + '\n' : '') + `builder@demo-vm:${directory}$ `;
  }
  return terminal;
}

// demoSnapshotAt is the sole deterministic story clock for the inert product demonstration.
// time is GSAP timeline seconds; the result contains sample display data, never backend operations.
export function demoSnapshotAt(time: number) {
  const scene: DemoScene = time < 5.3 ? 'login' : time < 6.8 ? 'dashboard' : time < 16 ? 'create' : time < 20.8 ? 'dashboard' : time < 42.8 ? 'console' : time < 50.8 ? 'delete' : 'landing';
  const loginEmail = typedText(time, 1.1, 'builder@kite.local', 1.25);
  const loginPassword = typedText(time, 2.95, 'KiteDemo2026!', .75);
  const name = typedText(time, 7.6, 'demo-vm', .9);
  const domainPrefix = typedText(time, 9.6, 'demo-vm', .9);
  const sshId = typedText(time, 11.6, 'builder', .7);
  const sshPassword = typedText(time, 13.2, 'KiteDemo2026!', .75);
  const deleting = time >= 47.5 && time < 48.3;
  const phase: 'Creating' | 'Running' | 'Terminating' = time < 17.8 ? 'Creating' : deleting ? 'Terminating' : 'Running';
  const hasVm = time >= 16 && time < 48.3;
  const menuOpen = (time >= 19 && time < 20.8) || (time >= 44.55 && time < 45.85);
  const deleteConfirm = time >= 45.85 && time < 47.5;
  const busy = (time >= 4.65 && time < 5.3) || (time >= 15.4 && time < 16) || (time >= 47 && time < 47.5);
  const terminal = terminalAt(time);
  const key = [scene, loginEmail, loginPassword, name, domainPrefix, sshId, sshPassword, phase, hasVm, menuOpen, deleteConfirm, deleting, busy, terminal].join('|');
  return { key, scene, loginEmail, loginPassword, name, domainPrefix, sshId, sshPassword, phase, hasVm, menuOpen, deleteConfirm, deleting, busy, terminal };
}
