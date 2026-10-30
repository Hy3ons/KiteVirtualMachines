import { apiClient } from './http';
import { createEndpoints } from './endpoints';
import type { ConsoleTicketResponse } from './types';

const apiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL || '/api/v1';

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
  return createEndpoints(apiClient).vmApi.createConsoleTicket(vmName, options);
};
