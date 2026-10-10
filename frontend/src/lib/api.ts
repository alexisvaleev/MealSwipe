/**
 * API client for the MealSwipe backend — `/api/v1`, JSON, session cookie auth.
 *
 * - `credentials: 'include'` sends the HttpOnly session cookie.
 * - CSRF token is kept in memory (from /auth/login|register|me responses) and
 *   sent as `X-CSRF-Token` on every mutating request.
 * - Error envelope `{error:{code,message}}` is parsed into a typed ApiError.
 * - 401 responses clear auth state via the registered handler and redirect
 *   to /login.
 */

export const API_BASE = '/api/v1'

export class ApiError extends Error {
  readonly code: string
  readonly status: number

  constructor(code: string, message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

// ---- CSRF token (memory only; refreshed from auth responses) ----

let csrfToken: string | null = null

export function setCsrfToken(token: string | null): void {
  csrfToken = token
}

export function getCsrfToken(): string | null {
  return csrfToken
}

// ---- 401 handling ----

type UnauthorizedHandler = () => void

let onUnauthorized: UnauthorizedHandler = () => {
  // Fallback: hard redirect (guarded so we don't loop on public pages).
  if (
    typeof window !== 'undefined' &&
    !/^\/(login|register)(\/|$)/.test(window.location.pathname)
  ) {
    window.location.assign('/login')
  }
}

/** Called once by the auth store to wire up "clear auth state" on 401. */
export function setUnauthorizedHandler(handler: UnauthorizedHandler): void {
  onUnauthorized = handler
}

// ---- fetch wrapper ----

const MUTATING = new Set(['POST', 'PUT', 'PATCH', 'DELETE'])

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  query?: Record<string, string | number | boolean | undefined | null>
  /** Skip the 401 handler (used by /auth/me on boot). */
  skipUnauthorized?: boolean
}

function buildUrl(path: string, query?: RequestOptions['query']): string {
  const url = `${API_BASE}${path}`
  if (!query) return url
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === '') continue
    params.set(key, String(value))
  }
  const qs = params.toString()
  return qs ? `${url}?${qs}` : url
}

interface ErrorEnvelope {
  error?: { code?: string; message?: string }
}

export async function api<T = unknown>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { method = 'GET', body, query, skipUnauthorized } = options

  const headers: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (MUTATING.has(method) && csrfToken) {
    headers['X-CSRF-Token'] = csrfToken
  }

  let res: Response
  try {
    res = await fetch(buildUrl(path, query), {
      method,
      headers,
      credentials: 'include',
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError('network_error', 'Network request failed', 0)
  }

  if (res.status === 401 && !skipUnauthorized) {
    onUnauthorized()
  }

  if (res.status === 204) {
    return undefined as T
  }

  const text = await res.text()
  let data: unknown = undefined
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      // Non-JSON body — fall through to status handling.
    }
  }

  if (!res.ok) {
    const envelope = data as ErrorEnvelope | undefined
    const code = envelope?.error?.code ?? `http_${res.status}`
    const message = envelope?.error?.message ?? res.statusText ?? 'Request failed'
    throw new ApiError(code, message, res.status)
  }

  return data as T
}
