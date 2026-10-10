import { create } from 'zustand'
import { api, ApiError, setCsrfToken, setUnauthorizedHandler } from '../lib/api'
import type { User } from '../lib/types'

export type AuthStatus = 'boot' | 'authed' | 'guest' | 'error'

interface AuthState {
  status: AuthStatus
  user: User | null
  /** Probe the session cookie; called once on app mount. */
  init: () => Promise<void>
  login: (email: string, password: string) => Promise<User>
  register: (email: string, password: string, displayName: string) => Promise<User>
  logout: () => Promise<void>
  setUser: (user: User | null) => void
}

interface AuthResponse {
  user: User
  csrf: string
}

let initInFlight: Promise<void> | null = null

export const useAuth = create<AuthState>((set) => ({
  status: 'boot',
  user: null,

  init: () => {
    // Dedupe concurrent callers (App mount + AppShell mount + retries).
    initInFlight ??= (async () => {
      try {
        const res = await api<AuthResponse>('/auth/me', { skipUnauthorized: true })
        setCsrfToken(res.csrf)
        set({ status: 'authed', user: res.user })
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) {
          set({ status: 'guest', user: null })
        } else {
          // Backend unreachable — distinguish from "not logged in" so we can retry.
          set({ status: 'error', user: null })
        }
      } finally {
        initInFlight = null
      }
    })()
    return initInFlight
  },

  login: async (email, password) => {
    const res = await api<AuthResponse>('/auth/login', {
      method: 'POST',
      body: { email, password },
      skipUnauthorized: true,
    })
    setCsrfToken(res.csrf)
    set({ status: 'authed', user: res.user })
    return res.user
  },

  register: async (email, password, displayName) => {
    const res = await api<AuthResponse>('/auth/register', {
      method: 'POST',
      body: { email, password, display_name: displayName },
      skipUnauthorized: true,
    })
    setCsrfToken(res.csrf)
    set({ status: 'authed', user: res.user })
    return res.user
  },

  logout: async () => {
    try {
      await api('/auth/logout', { method: 'POST' })
    } finally {
      setCsrfToken(null)
      set({ status: 'guest', user: null })
    }
  },

  setUser: (user) => set({ user, status: user ? 'authed' : 'guest' }),
}))

// Wire the 401 hook: any authed request that returns 401 clears the session.
// Route guards redirect to /login once status flips to 'guest'.
setUnauthorizedHandler(() => {
  const { status } = useAuth.getState()
  setCsrfToken(null)
  if (status === 'authed' || status === 'boot') {
    useAuth.setState({ status: 'guest', user: null })
  }
})
