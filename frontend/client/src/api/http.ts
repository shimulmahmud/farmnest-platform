/**
 * Tiny fetch wrapper for the backend API.
 * - Base URL: VITE_API_URL (e.g. https://api.example.com/api/v1) or "/api/v1" (Vite dev proxy).
 * - Unwraps the { success, data, meta } envelope and throws ApiError on failures.
 * - Sends the access token, and refreshes it once on a 401.
 */
const BASE = (import.meta.env.VITE_API_URL as string | undefined)?.replace(/\/$/, '') || '/api/v1';

const ACCESS_KEY = 'farmnest-access-token';
const REFRESH_KEY = 'farmnest-refresh-token';

export class ApiError extends Error {
  status: number;
  code: string;
  constructor(message: string, status: number, code = 'ERROR') {
    super(message);
    this.status = status;
    this.code = code;
  }
}

export const tokens = {
  get access() {
    return localStorage.getItem(ACCESS_KEY);
  },
  get refresh() {
    return localStorage.getItem(REFRESH_KEY);
  },
  set(access: string, refresh: string) {
    localStorage.setItem(ACCESS_KEY, access);
    localStorage.setItem(REFRESH_KEY, refresh);
  },
  clear() {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
  },
};

export interface Paginated<T> {
  items: T[];
  page: number;
  perPage: number;
  total: number;
  totalPages: number;
}

type Query = Record<string, string | number | boolean | undefined>;

interface Envelope<T> {
  success: boolean;
  data?: T;
  meta?: { page?: number; per_page?: number; total?: number; total_pages?: number };
  error?: { code?: string; message?: string };
}

function buildUrl(path: string, query?: Query): string {
  const qs = query
    ? Object.entries(query)
        .filter(([, v]) => v !== undefined && v !== '')
        .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
        .join('&')
    : '';
  return `${BASE}${path}${qs ? `?${qs}` : ''}`;
}

let refreshing: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  const refresh = tokens.refresh;
  if (!refresh) return false;
  if (!refreshing) {
    refreshing = (async () => {
      try {
        const res = await fetch(buildUrl('/auth/refresh'), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refresh }),
        });
        const body = (await res.json()) as Envelope<{ access_token: string; refresh_token: string }>;
        if (!res.ok || !body.success || !body.data) return false;
        tokens.set(body.data.access_token, body.data.refresh_token);
        return true;
      } catch {
        return false;
      } finally {
        refreshing = null;
      }
    })();
  }
  return refreshing;
}

async function raw<T>(
  method: string,
  path: string,
  opts: { query?: Query; body?: unknown; auth?: boolean } = {},
  retried = false,
): Promise<Envelope<T>> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (opts.body !== undefined) headers['Content-Type'] = 'application/json';
  if (opts.auth !== false && tokens.access) headers.Authorization = `Bearer ${tokens.access}`;

  let res: Response;
  try {
    res = await fetch(buildUrl(path, opts.query), {
      method,
      headers,
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    });
  } catch {
    throw new ApiError('Cannot reach the server. Please check your connection and try again.', 0, 'NETWORK');
  }

  if (res.status === 401 && !retried && opts.auth !== false && tokens.refresh) {
    if (await tryRefresh()) return raw<T>(method, path, opts, true);
    tokens.clear();
    window.dispatchEvent(new Event('farmnest:signed-out'));
  }

  let body: Envelope<T>;
  try {
    body = (await res.json()) as Envelope<T>;
  } catch {
    throw new ApiError(`Unexpected server response (${res.status})`, res.status);
  }
  if (!res.ok || body.success === false) {
    throw new ApiError(body.error?.message || `Request failed (${res.status})`, res.status, body.error?.code);
  }
  return body;
}

export async function get<T>(path: string, query?: Query, auth = false): Promise<T> {
  return (await raw<T>('GET', path, { query, auth })).data as T;
}

export async function post<T>(path: string, body?: unknown, auth = true): Promise<T> {
  return (await raw<T>('POST', path, { body: body ?? {}, auth })).data as T;
}

export async function getPage<T>(path: string, query?: Query, auth = false): Promise<Paginated<T>> {
  const env = await raw<T[]>('GET', path, { query, auth });
  return {
    items: env.data ?? [],
    page: env.meta?.page ?? 1,
    perPage: env.meta?.per_page ?? (env.data?.length ?? 0),
    total: env.meta?.total ?? (env.data?.length ?? 0),
    totalPages: env.meta?.total_pages ?? 1,
  };
}
