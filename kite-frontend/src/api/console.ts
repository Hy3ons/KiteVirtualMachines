import { apiClient } from './axios';
import type { ConsoleTicketResponse } from './types';

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || '/api/v1';

type ConsoleTicketRequestOptions = {
  readonly signal?: AbortSignal;
};

export const buildConsoleWebSocketUrl = (vmName: string, ticket: string): string => {
  const base = new URL(apiBaseUrl, window.location.origin);
  base.protocol = base.protocol === 'https:' ? 'wss:' : 'ws:';
  base.pathname = `${base.pathname.replace(/\/$/, '')}/vms/${encodeURIComponent(vmName)}/console`;
  base.search = '';
  base.searchParams.set('ticket', ticket);
  return base.toString();
};

export const createConsoleTicket = async (
  vmName: string,
  options: ConsoleTicketRequestOptions = {},
): Promise<ConsoleTicketResponse> => {
  const { data } = await apiClient.post<ConsoleTicketResponse>(
    `/vms/${vmName}/console-ticket`,
    undefined,
    { signal: options.signal },
  );
  return data;
};
