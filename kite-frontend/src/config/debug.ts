export const USE_MOCK_API = process.env.NEXT_PUBLIC_KITE_MODE === 'debug' && process.env.NEXT_PUBLIC_USE_MOCK === 'true';
export const DEBUG_DIRECT_ROUTES_ENABLED = USE_MOCK_API && process.env.NEXT_PUBLIC_DEBUG_DIRECT_ROUTES === 'true';
export const DEBUG_ADMIN_SESSION = { accessLevel: 3, username: 'admin', email: 'admin@kite.local', namespace: 'system' } as const;
