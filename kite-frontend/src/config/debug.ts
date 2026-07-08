export const DEBUG_DIRECT_ROUTES_ENABLED =
  import.meta.env.MODE === 'debug' && import.meta.env.VITE_DEBUG_DIRECT_ROUTES === 'true';

export const DEBUG_ADMIN_SESSION = {
  accessLevel: 3,
  username: 'admin',
  namespace: 'system',
} as const;

export const seedDebugAuthSession = (storage: Storage): boolean => {
  if (!DEBUG_DIRECT_ROUTES_ENABLED || storage.getItem('kite_authenticated')) {
    return false;
  }

  storage.setItem('kite_authenticated', 'true');
  storage.setItem('kite_access_level', DEBUG_ADMIN_SESSION.accessLevel.toString());
  storage.setItem('kite_username', DEBUG_ADMIN_SESSION.username);
  storage.setItem('kite_namespace', DEBUG_ADMIN_SESSION.namespace);
  storage.removeItem('kite_profile_image');
  return true;
};
