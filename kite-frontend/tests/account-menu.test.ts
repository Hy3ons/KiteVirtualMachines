import { createRequire } from 'node:module';
import { createElement, forwardRef, isValidElement } from 'react';
import type { ReactNode } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { PopoverProps } from 'antd';
import type { TooltipProps, TooltipRef } from '@rc-component/tooltip/lib/Tooltip';
import { AccountMenu } from '../src/components/AccountMenu';

const harness = vi.hoisted(() => {
  const changes: boolean[] = [];
  const popup: { requestOpen: ((visible: boolean) => void) | null; content: ReactNode } = { requestOpen: null, content: null };
  return { changes, popup, accessLevel: 3, usage: vi.fn() };
});

vi.mock('../src/store/useAuthStore', () => ({ useAuthStore: () => ({ username: 'review-user', email: 'review@example.test', accessLevel: harness.accessLevel }) }));
vi.mock('../src/hooks/useLogout', () => ({ useLogout: () => vi.fn() }));
vi.mock('../src/components/AccountVmUsage', () => ({ AccountVmUsage: () => { harness.usage(); return null; } }));
vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>();
  return { ...actual, Popover: (props: PopoverProps) => {
    harness.popup.content = typeof props.content === 'function' ? props.content() : props.content;
    return createElement(actual.Popover, { ...props, onOpenChange: (next) => { harness.changes.push(next); props.onOpenChange?.(next); } });
  } };
});

// Ant Design uses this CommonJS module internally; replacing only its DOM renderer
// preserves the real Popover -> Tooltip visibility gate without a browser engine.
const require = createRequire(import.meta.url);
const tooltip: typeof import('@rc-component/tooltip') = require('@rc-component/tooltip');
const originalTooltip = tooltip.default;

beforeEach(() => {
  harness.changes.length = 0;
  harness.popup.content = null;
  harness.popup.requestOpen = null;
  harness.accessLevel = 3;
  harness.usage.mockClear();
  tooltip.default = forwardRef<TooltipRef, TooltipProps>((props, ref) => {
    // The browser-free renderer has no DOM instance to expose through the ref.
    void ref;
    harness.popup.requestOpen = props.onVisibleChange ?? null;
    return null;
  });
});
afterEach(() => { tooltip.default = originalTooltip; vi.restoreAllMocks(); });

describe('account popup without a browser', () => {
  it('notifies the account menu to open on the first visibility request', () => {
    // Given the actual menu in its initial closed state.
    renderToStaticMarkup(createElement(AccountMenu));
    // When the trigger requests opening through the real Ant Design Tooltip gate.
    expect(harness.popup.requestOpen).toBeTypeOf('function');
    harness.popup.requestOpen?.(true);
    // Then the gate must deliver the request instead of suppressing an empty overlay.
    expect(harness.changes).toEqual([true]);
  });

  it('provides identity and navigation while deferring VM queries until opened', () => {
    // Given a closed administrator menu, including a possible pre-render of its content.
    renderToStaticMarkup(createElement(AccountMenu));
    const content = harness.popup.content;
    if (!isValidElement(content)) throw new TypeError('Account popup must have an initial content shell');
    // When its shell is rendered.
    const html = renderToStaticMarkup(content);
    // Then identity/navigation are ready but no VM usage component has mounted.
    expect(html).toContain('review-user');
    expect(html).toContain('review@example.test');
    expect(html).toContain('href="/dashboard"');
    expect(html).toContain('href="/admin/settings"');
    expect(harness.usage).not.toHaveBeenCalled();
  });

  it('omits administrator settings for a regular account', () => {
    // Given an ordinary user.
    harness.accessLevel = 1;
    renderToStaticMarkup(createElement(AccountMenu));
    const content = harness.popup.content;
    if (!isValidElement(content)) throw new TypeError('Account popup must have an initial content shell');
    // When the menu content renders.
    const html = renderToStaticMarkup(content);
    // Then dashboard navigation remains and unauthorized settings are absent.
    expect(html).toContain('href="/dashboard"');
    expect(html).not.toContain('href="/admin/settings"');
  });

  it('keeps the inert landing demonstration from opening or fetching account data', () => {
    // Given the deliberately inert miniature preview.
    renderToStaticMarkup(createElement(AccountMenu, { preview: true }));
    // When its visibility gate receives a request directly.
    harness.popup.requestOpen?.(true);
    // Then no popup content, state notification or VM query is exposed.
    expect(harness.popup.content).toBeNull();
    expect(harness.changes).toEqual([]);
    expect(harness.usage).not.toHaveBeenCalled();
  });
});
