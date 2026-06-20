const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export const buildConsoleWebSocketUrl = (vmName: string, ticket: string): string => {
  const base = new URL(apiBaseUrl, window.location.origin);
  base.protocol = base.protocol === 'https:' ? 'wss:' : 'ws:';
  base.pathname = `${base.pathname.replace(/\/$/, '')}/vms/${encodeURIComponent(vmName)}/console`;
  base.search = '';
  base.searchParams.set('ticket', ticket);
  return base.toString();
};
