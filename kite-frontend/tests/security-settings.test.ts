import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import type { PopconfirmProps } from 'antd';
import { beforeEach, expect, it, vi } from 'vitest';
import { SecuritySettings } from '../src/screens/admin/SecuritySettings';
import { useSecuritySettings } from '../src/screens/admin/useSecuritySettings';
import type { RuntimeSecretRotation } from '../src/api/types';

const harness = vi.hoisted(() => {
  const confirmations: PopconfirmProps[] = [];
  return { confirmations, rotate: vi.fn<(payload: RuntimeSecretRotation) => Promise<void>>() };
});

vi.mock('antd', async importOriginal => {
  const actual = await importOriginal<typeof import('antd')>();
  return { ...actual, Popconfirm: (props: PopconfirmProps) => {
    harness.confirmations.push(props);
    return props.children;
  } };
});

// RenderSettings uses the real form hook while replacing only the secret mutation callback.
function RenderSettings() {
  const settings = useSecuritySettings(null);
  return createElement(SecuritySettings, { settings: { ...settings, handleRotateRuntimeSecrets: harness.rotate } });
}

beforeEach(() => {
  harness.confirmations.length = 0;
  harness.rotate.mockReset().mockResolvedValue(undefined);
});

it('warns about legacy password hashes and waits for confirmation before rotating Salt', async () => {
  const html = renderToStaticMarkup(createElement(RenderSettings));
  expect(html).toContain('구형 계정의 로그인');
  expect(html).toContain('bcrypt');
  expect(harness.rotate).not.toHaveBeenCalled();
  expect(harness.confirmations).toHaveLength(2);
  const confirmation = harness.confirmations[0];
  if (!confirmation?.onConfirm) throw new TypeError('Salt rotation requires a confirmation callback');
  await confirmation.onConfirm();
  expect(harness.rotate).toHaveBeenCalledExactlyOnceWith({ rotatePasswordSalt: true });
});

it('requires confirmation before rotating both runtime secrets', async () => {
  renderToStaticMarkup(createElement(RenderSettings));
  expect(harness.rotate).not.toHaveBeenCalled();
  const confirmation = harness.confirmations[1];
  if (!confirmation?.onConfirm) throw new TypeError('Combined rotation requires a confirmation callback');
  await confirmation.onConfirm();
  expect(harness.rotate).toHaveBeenCalledExactlyOnceWith({ rotateJWTSecret: true, rotatePasswordSalt: true });
});
