'use client';

import { createContext, useCallback, useContext, useMemo, useState } from 'react';
import type { ReactNode } from 'react';

type ConsoleNavigationState = {
  readonly collapsed: boolean;
  readonly toggleCollapsed: () => void;
};

const ConsoleNavigationContext = createContext<ConsoleNavigationState | null>(null);

// ConsoleNavigationProvider keeps the sidebar preference in the persistent root layout.
// children can change routes without resetting this UI-only state or sharing account data.
export function ConsoleNavigationProvider({ children }: { readonly children: ReactNode }) {
  const [collapsed, setCollapsed] = useState(false);
  const toggleCollapsed = useCallback(() => setCollapsed((value) => !value), []);
  const value = useMemo(() => ({ collapsed, toggleCollapsed }), [collapsed, toggleCollapsed]);
  return <ConsoleNavigationContext.Provider value={value}>{children}</ConsoleNavigationContext.Provider>;
}

class MissingConsoleNavigationProviderError extends Error { readonly name = 'MissingConsoleNavigationProviderError'; }

// useConsoleNavigation reads the shared rail preference across workspace and admin screens.
export function useConsoleNavigation() {
  const state = useContext(ConsoleNavigationContext);
  if (!state) throw new MissingConsoleNavigationProviderError('ConsoleNavigationProvider is required');
  return state;
}
