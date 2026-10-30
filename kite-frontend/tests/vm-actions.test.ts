import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import type { ButtonProps, DropdownProps, MenuProps, TabsProps } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { VmRowActions } from '../src/screens/dashboard/VmRowActions';
import { VmDetailTabs } from '../src/screens/machine/VmDetailTabs';
import { vmSchema } from '../src/api/contracts';
import { vm as fixtureVm } from './fixtures/api';
import type { VmPhase } from '../src/api/types';
import { canControlVm } from '../src/domain/access';

const harness = vi.hoisted(() => {
  const state: { menu: MenuProps | null; copyDisabled: boolean | null; readonly copy: () => Promise<void> } = {
    menu: null, copyDisabled: null, copy: vi.fn<() => Promise<void>>(),
  };
  return state;
});

// Only the Ant Design menu/tab presentation seam is replaced. Product components
// still choose their real action callbacks and availability without a DOM engine.
vi.mock('antd', async importOriginal => {
  const actual = await importOriginal<typeof import('antd')>();
  return { ...actual,
    Dropdown: (props: DropdownProps) => { harness.menu = props.menu ?? null; return null; },
    Tabs: (props: TabsProps) => props.items?.find(item => item.key === 'ssh')?.children ?? null,
    Button: (props: ButtonProps) => {
      if (props.onClick === harness.copy) harness.copyDisabled = props.disabled ?? false;
      return createElement(actual.Button, props);
    },
  };
});

beforeEach(() => { harness.menu = null; harness.copyDisabled = null; });

// disabledAction requires a real menu action so missing actions cannot pass as enabled.
function disabledAction(key: string): boolean {
  const item = harness.menu?.items?.find(candidate => candidate && 'key' in candidate && candidate.key === key);
  if (!item || !('disabled' in item)) throw new TypeError('Expected an action with explicit availability');
  return item.disabled ?? false;
}

describe('SSH preparation and live console actions', () => {
  it.each(['Running', 'Stopped'] satisfies VmPhase[])('enables assigned VM lifecycle actions for Level 0 while %s', phase => {
    const vm = vmSchema.parse({ ...fixtureVm, phase, delete: false });
    renderToStaticMarkup(createElement(VmRowActions, { vm, canMutate: canControlVm(0), onConnect: vi.fn(), onOpenConsole: vi.fn(), onStart: vi.fn(), onStop: vi.fn(), onDelete: vi.fn() }));
    expect(disabledAction('power')).toBe(false);
    expect(disabledAction('delete')).toBe(false);
  });
  it.each([
    { phase: 'Stopped', deleting: false, sshDisabled: false, consoleDisabled: true },
    { phase: 'Running', deleting: false, sshDisabled: false, consoleDisabled: false },
    { phase: 'Stopped', deleting: true, sshDisabled: true, consoleDisabled: true },
    { phase: 'Running', deleting: true, sshDisabled: true, consoleDisabled: true },
  ] satisfies readonly { readonly phase: VmPhase; readonly deleting: boolean; readonly sshDisabled: boolean; readonly consoleDisabled: boolean }[])('keeps SSH instructions distinct from a live console for $phase / deleting=$deleting', scenario => {
    // Given a VM and a read-only account; instructions do not mutate that VM.
    const vm = vmSchema.parse({ ...fixtureVm, phase: scenario.phase, delete: scenario.deleting });
    // When the actual row component builds its menu.
    renderToStaticMarkup(createElement(VmRowActions, { vm, canMutate: false, onConnect: vi.fn(), onOpenConsole: vi.fn(), onStart: vi.fn(), onStop: vi.fn(), onDelete: vi.fn() }));
    // Then preparation works before startup while a live console requires Running.
    expect([disabledAction('ssh'), disabledAction('console')]).toEqual([scenario.sshDisabled, scenario.consoleDisabled]);
  });

  it.each(['Stopped', 'Running', 'Deleting'] satisfies readonly VmPhase[])('allows copying a ready command independently of VM phase %s', phase => {
    // Given authorized metadata and a ready gateway command, including during shutdown.
    const vm = vmSchema.parse({ ...fixtureVm, phase, delete: phase === 'Deleting' });
    // When the real SSH details tab renders.
    renderToStaticMarkup(createElement(VmDetailTabs, { vm, secure: false, canMutate: false, sshState: { ready: true, command: 'ssh ubuntu@example.test', configPort: '22' }, onCopy: harness.copy, onDelete: vi.fn() }));
    // Then copying instructions remains available; it does not initiate a connection.
    expect(harness.copyDisabled).toBe(false);
  });
});
