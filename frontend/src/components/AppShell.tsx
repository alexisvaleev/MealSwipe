import { useEffect, useState, type ReactNode } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router'
import { AnimatePresence, motion } from 'framer-motion'
import { t, type TranslationKey } from '../i18n/i18n'
import { useAuth } from '../stores/auth'
import { Spinner } from './States'
import { IconCalendar, IconCards, IconCart, IconGear, IconHeart, IconThumbUp, IconX } from './Icons'

const APP_NAME = import.meta.env.VITE_APP_NAME ?? 'MealSwipe'

interface NavItem {
  to: string
  key: TranslationKey
  icon: ReactNode
}

const NAV_ITEMS: NavItem[] = [
  { to: '/swipe', key: 'nav.swipe', icon: <IconCards className="size-6" /> },
  { to: '/likes', key: 'nav.likes', icon: <IconThumbUp className="size-6" /> },
  { to: '/favorites', key: 'nav.favorites', icon: <IconHeart className="size-6" /> },
  { to: '/menu', key: 'nav.menu', icon: <IconCalendar className="size-6" /> },
  { to: '/shopping', key: 'nav.shopping', icon: <IconCart className="size-6" /> },
  { to: '/settings', key: 'nav.settings', icon: <IconGear className="size-6" /> },
]

function NavItemLink({ item, compact }: { item: NavItem; compact?: boolean }) {
  return (
    <NavLink
      to={item.to}
      className={({ isActive }) =>
        compact
          ? `flex flex-1 flex-col items-center gap-0.5 py-2 text-[11px] font-medium transition ${
              isActive ? 'text-brand-700' : 'text-stone-400 hover:text-stone-600'
            }`
          : `flex items-center gap-3 rounded-2xl px-4 py-3 text-sm font-semibold transition ${
              isActive
                ? 'bg-brand-100 text-brand-800'
                : 'text-stone-500 hover:bg-cream-dark hover:text-stone-800'
            }`
      }
    >
      {item.icon}
      <span>{t(item.key)}</span>
    </NavLink>
  )
}

/** Dismissible nudge shown when preferences haven't been filled in yet. */
function PreferencesNudge() {
  const user = useAuth((s) => s.user)
  const [dismissed, setDismissed] = useState(false)
  const location = useLocation()
  if (!user || user.has_preferences || dismissed) return null
  if (location.pathname === '/onboarding') return null
  return (
    <div className="mx-4 mt-3 flex items-center gap-3 rounded-2xl bg-brand-100 px-4 py-3 text-sm text-brand-900 lg:mx-6">
      <span className="flex-1">{t('settings.nudge')}</span>
      <NavLink
        to="/onboarding"
        className="shrink-0 rounded-xl bg-brand-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-brand-700"
      >
        {t('prefs.next')}
      </NavLink>
      <button
        type="button"
        aria-label={t('common.close')}
        onClick={() => setDismissed(true)}
        className="shrink-0 text-brand-700 hover:text-brand-900"
      >
        <IconX className="size-4" />
      </button>
    </div>
  )
}

/**
 * Guarded app layout: requires auth (redirects /login), renders the shell —
 * left sidebar on ≥lg, bottom tab bar on mobile.
 */
export function AppShell() {
  const { status, init } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    if (status === 'boot') void init()
  }, [status, init])

  useEffect(() => {
    if (status === 'guest') {
      navigate('/login', { replace: true, state: { from: location.pathname } })
    }
  }, [status, navigate, location.pathname])

  if (status === 'boot' || status === 'guest') {
    return (
      <div className="flex min-h-dvh items-center justify-center">
        <Spinner className="size-10" />
      </div>
    )
  }

  if (status === 'error') {
    return (
      <div className="flex min-h-dvh flex-col items-center justify-center gap-4 px-6 text-center">
        <p className="max-w-sm text-sm text-stone-500">{t('common.networkError')}</p>
        <button
          type="button"
          onClick={() => void init()}
          className="rounded-2xl bg-brand-600 px-5 py-2.5 text-sm font-semibold text-white shadow-chip hover:bg-brand-700"
        >
          {t('common.retry')}
        </button>
      </div>
    )
  }

  return (
    <div className="flex min-h-dvh">
      {/* Sidebar — desktop */}
      <aside className="sticky top-0 hidden h-dvh w-64 shrink-0 flex-col border-r border-stone-200/70 bg-white/60 px-4 py-6 backdrop-blur lg:flex">
        <NavLink to="/swipe" className="mb-8 flex items-center gap-2.5 px-2">
          <span className="flex size-9 items-center justify-center rounded-2xl bg-brand-600 text-lg text-white shadow-chip">
            🍽
          </span>
          <span className="text-xl font-extrabold tracking-tight text-stone-900">
            {APP_NAME}
          </span>
        </NavLink>
        <nav className="flex flex-col gap-1" aria-label={t('nav.main')}>
          {NAV_ITEMS.map((item) => (
            <NavItemLink key={item.to} item={item} />
          ))}
        </nav>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        {/* Mobile top bar */}
        <header className="sticky top-0 z-30 flex items-center justify-between border-b border-stone-200/60 bg-cream/80 px-4 py-3 backdrop-blur lg:hidden">
          <NavLink to="/swipe" className="flex items-center gap-2">
            <span className="flex size-8 items-center justify-center rounded-xl bg-brand-600 text-base text-white shadow-chip">
              🍽
            </span>
            <span className="text-lg font-extrabold tracking-tight text-stone-900">
              {APP_NAME}
            </span>
          </NavLink>
        </header>

        <PreferencesNudge />

        <main className="mx-auto w-full max-w-5xl flex-1 px-4 pb-28 pt-4 lg:px-8 lg:pb-10 lg:pt-6">
          <AnimatePresence mode="wait">
            <motion.div
              key={location.pathname}
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.18 }}
              className="h-full"
            >
              <Outlet />
            </motion.div>
          </AnimatePresence>
        </main>
      </div>

      {/* Bottom tab bar — mobile */}
      <nav
        aria-label={t('nav.main')}
        className="fixed inset-x-0 bottom-0 z-40 flex border-t border-stone-200/70 bg-white/90 pb-[env(safe-area-inset-bottom)] backdrop-blur lg:hidden"
      >
        {NAV_ITEMS.map((item) => (
          <NavItemLink key={item.to} item={item} compact />
        ))}
      </nav>
    </div>
  )
}
