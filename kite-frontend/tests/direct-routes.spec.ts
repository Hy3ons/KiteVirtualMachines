import { expect, test } from '@playwright/test';

const directRoutes = [
  { path: '/', text: 'admin님 환영합니다' },
  { path: '/signup', text: 'Create an Account' },
  { path: '/dashboard', text: 'My Virtual Machines' },
  { path: '/dashboard/kite-machine/dev-vm-1', text: 'dev-vm-1' },
  { path: '/dashboard/kite-machine/arbitrary-debug-vm', text: 'arbitrary-debug-vm' },
  { path: '/admin/dashboard', text: '유저 권한 관리' },
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
