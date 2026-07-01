import { expect, test } from '@playwright/test';

const directRoutes = [
  { path: '/', text: 'admin님 환영합니다' },
  { path: '/signup', text: 'Create an Account' },
  { path: '/dashboard', text: 'My Virtual Machines' },
  { path: '/dashboard/kite-machine/dev-vm-1', text: 'dev-vm-1' },
  { path: '/dashboard/kite-machine/dev-vm-1/console', text: 'Serial console은 VM 내부 OS의 ttyS0' },
  { path: '/dashboard/kite-machine/arbitrary-debug-vm', text: 'arbitrary-debug-vm' },
  { path: '/admin/dashboard', text: 'Users' },
  { path: '/admin/settings', text: '시스템 전역 설정' },
] as const;

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    window.localStorage.clear();
    window.sessionStorage.clear();
  });
});

for (const route of directRoutes) {
  test(`renders ${route.path} directly in debug mode`, async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', (message) => {
      if (message.type() === 'error') {
        consoleErrors.push(message.text());
      }
    });

    await page.goto(route.path);

    await expect(page.getByText(route.text).first()).toBeVisible();
    expect(new URL(page.url()).pathname).toBe(route.path);
    expect(consoleErrors).toEqual([]);
  });
}

test('creates and opens a VM with the stateful debug API', async ({ page }) => {
  await page.goto('/dashboard');

  await page.getByRole('button', { name: 'Create VM' }).click();
  await expect(page.getByText('VM 생성 시 한 번 설정됩니다.')).toBeVisible();
  await page.getByPlaceholder('my-ubuntu-vm').fill('pw-debug-vm');
  await page.getByPlaceholder('my-web').fill('pw');
  await page.getByRole('textbox', { name: 'SSH User ID' }).fill('playwright');
  await page.getByPlaceholder('Strong password').fill('debug-password');
  await page.getByRole('button', { name: 'Create', exact: true }).click();

  await expect(page.getByRole('link', { name: 'pw-debug-vm' })).toBeVisible();
  await page.getByRole('link', { name: 'pw-debug-vm' }).click();

  await expect(page.getByRole('heading', { name: 'pw-debug-vm' })).toBeVisible();
  await expect(page.getByText('Running').first()).toBeVisible();
});

test('explains gateway SSH and console OS password paths', async ({ page }) => {
  await page.goto('/dashboard');

  await page.getByRole('button', { name: 'Connect' }).first().click();
  await expect(page.getByText('SSH 명령은 VM에 직접 비밀번호 로그인하지 않습니다.')).toBeVisible();
  await expect(page.getByText('Kite 관리 키로 해당 VM에 연결합니다.')).toBeVisible();

  await page.goto('/dashboard/kite-machine/dev-vm-1/console');
  await expect(page.getByText('VM 생성 시 입력한 SSH ID와 초기 비밀번호로 로그인')).toBeVisible();
  await expect(page.getByText('Kite에서 변경하거나 복구할 수 없습니다.')).toBeVisible();
});
