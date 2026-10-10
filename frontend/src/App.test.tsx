import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import App from './App'

/** Mounts the real router+shell; /auth/me is stubbed to 401 → guest. */
describe('App smoke test', () => {
  it('renders the landing page for guests with RU copy', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: { code: 'unauthorized', message: 'x' } }), {
          status: 401,
        }),
      ),
    )
    render(<App />)
    expect(
      await screen.findByRole('heading', { name: /Что приготовить на этой неделе/ }),
    ).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Начать подбирать' })).toBeInTheDocument()
    vi.unstubAllGlobals()
  })
})
