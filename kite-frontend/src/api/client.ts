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
  async function request<T>(method: string, endpoint: string, settings: Options = {}): Promise<{ readonly data: T }> {
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
    return { data: await response.json<T>() };
  }

  return {
    get: <T>(endpoint: string) => request<T>('GET', endpoint),
    post: <T>(endpoint: string, json?: unknown, settings: Options = {}) => request<T>('POST', endpoint, { ...settings, ...(json === undefined ? {} : { json }) }),
    patch: <T>(endpoint: string, json?: unknown) => request<T>('PATCH', endpoint, { json }),
    delete: <T>(endpoint: string) => request<T>('DELETE', endpoint),
  };
}

export type ApiClient = ReturnType<typeof createApiClient>;
