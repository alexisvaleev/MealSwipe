import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  api,
  ApiError,
  getCsrfToken,
  setCsrfToken,
  setUnauthorizedHandler,
} from './api'

const jsonResponse = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })

describe('api client', () => {
  beforeEach(() => {
    setCsrfToken(null)
    setUnauthorizedHandler(() => {})
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('sends X-CSRF-Token on mutating requests when a token is known', async () => {
    setCsrfToken('token-123')
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ ok: true }))

    await api('/menus/1/items', { method: 'POST', body: { day_index: 0 } })

    const [url, init] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe('/api/v1/menus/1/items')
    const headers = init?.headers as Record<string, string>
    expect(headers['X-CSRF-Token']).toBe('token-123')
    expect(headers['Content-Type']).toBe('application/json')
    expect(init?.credentials).toBe('include')
    expect(init?.body).toBe(JSON.stringify({ day_index: 0 }))
  })

  it('does not send CSRF header on GET requests', async () => {
    setCsrfToken('token-123')
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ recipes: [] }))

    await api('/discover', { query: { limit: 15 } })

    const [url, init] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe('/api/v1/discover?limit=15')
    const headers = init?.headers as Record<string, string>
    expect(headers['X-CSRF-Token']).toBeUndefined()
  })

  it('parses the error envelope {error:{code,message}} into ApiError', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      jsonResponse({ error: { code: 'invalid_credentials', message: 'Неверная почта' } }, 401),
    )

    const err = (await api('/auth/login', { method: 'POST', skipUnauthorized: true }).catch(
      (e) => e,
    )) as ApiError
    expect(err).toBeInstanceOf(ApiError)
    expect(err.code).toBe('invalid_credentials')
    expect(err.message).toBe('Неверная почта')
    expect(err.status).toBe(401)
  })

  it('falls back to http_<status> code when the envelope is missing', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(new Response('oops', { status: 500 }))

    const err = (await api('/health').catch((e) => e)) as ApiError
    expect(err).toBeInstanceOf(ApiError)
    expect(err.code).toBe('http_500')
    expect(err.status).toBe(500)
  })

  it('invokes the unauthorized handler on 401', async () => {
    const handler = vi.fn()
    setUnauthorizedHandler(handler)
    vi.mocked(fetch).mockResolvedValueOnce(
      jsonResponse({ error: { code: 'unauthorized', message: 'no' } }, 401),
    )

    await expect(api('/favorites')).rejects.toBeInstanceOf(ApiError)
    expect(handler).toHaveBeenCalledOnce()
  })

  it('skips the unauthorized handler when skipUnauthorized is set', async () => {
    const handler = vi.fn()
    setUnauthorizedHandler(handler)
    vi.mocked(fetch).mockResolvedValueOnce(
      jsonResponse({ error: { code: 'unauthorized', message: 'no' } }, 401),
    )

    await expect(api('/auth/me', { skipUnauthorized: true })).rejects.toBeInstanceOf(ApiError)
    expect(handler).not.toHaveBeenCalled()
  })

  it('maps network failures to ApiError with code network_error', async () => {
    vi.mocked(fetch).mockRejectedValueOnce(new TypeError('Failed to fetch'))

    const err = (await api('/discover').catch((e) => e)) as ApiError
    expect(err).toBeInstanceOf(ApiError)
    expect(err.code).toBe('network_error')
    expect(err.status).toBe(0)
  })

  it('returns undefined for 204 No Content', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(new Response(null, { status: 204 }))
    await expect(api('/auth/logout', { method: 'POST' })).resolves.toBeUndefined()
  })

  it('skips empty query params', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ ingredients: [] }))
    await api('/ingredients', { query: { q: '', category: undefined, page: 2 } })
    expect(vi.mocked(fetch).mock.calls[0][0]).toBe('/api/v1/ingredients?page=2')
  })

  it('exposes getCsrfToken', () => {
    expect(getCsrfToken()).toBeNull()
    setCsrfToken('abc')
    expect(getCsrfToken()).toBe('abc')
  })
})
