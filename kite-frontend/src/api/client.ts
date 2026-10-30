import ky from 'ky';
import type { Options } from 'ky';
import { z } from 'zod';

const errorBody = z.object({ message: z.string() });

export class ApiError extends Error {
  readonly name = 'ApiError';
  readonly status: number;

  // status identifies the failed response; message is safe backend error text.
  // Browser screens and SSR use this error without exposing authentication cookies.
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

// ApiContractError identifies a successful response that was not valid JSON.
export class ApiContractError extends Error {
  readonly name = 'ApiContractError';
  constructor(readonly method: string, readonly endpoint: string) {
    super(`Invalid JSON response for ${method} ${endpoint}`);
  }
}

type ClientOptions = {
  readonly cookie?: string;
  readonly timeout?: number;
  readonly onUnauthorized?: () => void;
};

// createApiClient uses baseUrl for /api/v1 and options for request-scoped SSR
// cookies, timeout and session expiry handling. It never retries or caches requests.
export function createApiClient(baseUrl: string, options: ClientOptions = {}) {
  const transport = ky.create({
    credentials: 'include', cache: 'no-store', timeout: options.timeout ?? 10_000,
    retry: 0, throwHttpErrors: false,
    headers: options.cookie ? { cookie: options.cookie } : {},
  });

  // request preserves the method, payload and cancellation supplied by API adapters.
  // It returns decoded JSON using the existing data response contract.
  async function request(method: string, endpoint: string, settings: Options = {}): Promise<{ readonly data: unknown }> {
    const response = await transport(`${baseUrl.replace(/\/$/, '')}${endpoint}`, { ...settings, method });
    if (!response.ok) {
      const raw: unknown = await response.json().catch((error: unknown) => {
        if (error instanceof SyntaxError) return null;
        throw error;
      });
      const parsed = errorBody.safeParse(raw);
      if (response.status === 401 && endpoint !== '/auth/login') options.onUnauthorized?.();
      throw new ApiError(response.status, parsed.success ? parsed.data.message : `API request failed (${response.status})`);
    }
    try {
      return { data: await response.json<unknown>() };
    } catch (error) {
      if (error instanceof SyntaxError) throw new ApiContractError(method, endpoint);
      throw error;
    }
  }

  return {
    get: (endpoint: string) => request('GET', endpoint),
    post: (endpoint: string, json?: unknown, settings: Options = {}) => request('POST', endpoint, { ...settings, ...(json === undefined ? {} : { json }) }),
    patch: (endpoint: string, json?: unknown) => request('PATCH', endpoint, { json }),
    delete: (endpoint: string) => request('DELETE', endpoint),
  };
}

export type ApiClient = ReturnType<typeof createApiClient>;
