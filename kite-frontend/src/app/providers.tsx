'use client';

import { App, ConfigProvider } from 'antd';
import type { ReactNode } from 'react';
import { kiteTheme } from '../theme';
import { ConsoleNavigationProvider } from '../store/useConsoleNavigation';

// Providers supplies the shared theme, notification portals and persistent console navigation.
// children contains server-rendered pages and interactive client components.
export function Providers({ children }: { readonly children: ReactNode }) {
  return <ConfigProvider theme={kiteTheme}><App><ConsoleNavigationProvider>{children}</ConsoleNavigationProvider></App></ConfigProvider>;
}
