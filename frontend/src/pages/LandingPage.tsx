import { Navigate, Link } from 'react-router'
import { motion } from 'framer-motion'
import { t } from '../i18n/i18n'
import { useAuth } from '../stores/auth'
import { Spinner } from '../components/States'
import { IconCalendar, IconCart, IconCards } from '../components/Icons'

const APP_NAME = import.meta.env.VITE_APP_NAME ?? 'MealSwipe'

const STEPS = [
  { icon: <IconCards className="size-7" />, title: 'landing.step1Title', text: 'landing.step1Text' },
  { icon: <IconCalendar className="size-7" />, title: 'landing.step2Title', text: 'landing.step2Text' },
  { icon: <IconCart className="size-7" />, title: 'landing.step3Title', text: 'landing.step3Text' },
] as const

export function LandingPage() {
  const { status } = useAuth()

  if (status === 'authed') return <Navigate to="/swipe" replace />
  if (status === 'boot') {
    return (
      <div className="flex min-h-dvh items-center justify-center">
        <Spinner className="size-10" />
      </div>
    )
  }

  return (
    <div className="min-h-dvh bg-gradient-to-b from-brand-50 via-cream to-cream">
      <header className="mx-auto flex max-w-5xl items-center justify-between px-5 py-5">
        <span className="flex items-center gap-2.5">
          <span className="flex size-10 items-center justify-center rounded-2xl bg-brand-600 text-xl text-white shadow-card">
            🍽
          </span>
          <span className="text-2xl font-extrabold tracking-tight text-stone-900">{APP_NAME}</span>
        </span>
        <nav className="flex items-center gap-2">
          <Link
            to="/login"
            className="rounded-2xl px-4 py-2.5 text-sm font-semibold text-stone-600 transition hover:bg-white/70 hover:text-stone-900"
          >
            {t('auth.login')}
          </Link>
          <Link
            to="/register"
            className="rounded-2xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white shadow-card transition hover:bg-brand-700 active:scale-95"
          >
            {t('auth.register')}
          </Link>
        </nav>
      </header>

      <main className="mx-auto max-w-5xl px-5">
        {/* Hero */}
        <section className="flex flex-col items-center gap-10 py-12 text-center lg:py-20">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
          >
            <h1 className="mx-auto max-w-2xl text-4xl font-extrabold leading-tight tracking-tight text-stone-900 sm:text-5xl">
              {t('landing.heroTitle')}
            </h1>
            <p className="mx-auto mt-5 max-w-xl text-lg leading-relaxed text-stone-600">
              {t('landing.heroSubtitle', { app: APP_NAME })}
            </p>
            <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
              <Link
                to="/register"
                className="rounded-2xl bg-brand-600 px-7 py-3.5 text-base font-bold text-white shadow-soft transition hover:bg-brand-700 active:scale-95"
              >
                {t('landing.ctaStart')}
              </Link>
              <Link
                to="/login"
                className="rounded-2xl bg-white px-7 py-3.5 text-base font-bold text-stone-700 shadow-card transition hover:bg-brand-50 active:scale-95"
              >
                {t('landing.ctaLogin')}
              </Link>
            </div>
          </motion.div>

          {/* Decorative stacked cards */}
          <motion.div
            aria-hidden
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: 0.25, duration: 0.5 }}
            className="relative h-40 w-full max-w-sm"
          >
            <div className="absolute left-1/2 top-0 h-36 w-64 -translate-x-1/2 -rotate-6 rounded-3xl bg-gradient-to-br from-brand-300 to-brand-500 shadow-soft" />
            <div className="absolute left-1/2 top-2 h-36 w-64 -translate-x-1/2 rotate-3 rounded-3xl bg-gradient-to-br from-amber-200 to-amber-400 shadow-soft" />
            <div className="absolute left-1/2 top-4 flex h-36 w-64 -translate-x-1/2 items-center justify-center rounded-3xl bg-white text-5xl shadow-soft">
              🍲
            </div>
          </motion.div>
        </section>

        {/* How it works */}
        <section className="pb-20">
          <h2 className="text-center text-2xl font-bold text-stone-900">{t('landing.howTitle')}</h2>
          <div className="mt-8 grid gap-5 sm:grid-cols-3">
            {STEPS.map((step, i) => (
              <motion.article
                key={step.title}
                initial={{ opacity: 0, y: 16 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: i * 0.12, duration: 0.4 }}
                className="rounded-3xl bg-white p-6 shadow-card"
              >
                <div className="flex size-14 items-center justify-center rounded-2xl bg-brand-100 text-brand-700">
                  {step.icon}
                </div>
                <h3 className="mt-4 text-lg font-bold text-stone-900">{t(step.title)}</h3>
                <p className="mt-2 text-sm leading-relaxed text-stone-600">{t(step.text)}</p>
              </motion.article>
            ))}
          </div>
        </section>
      </main>

      <footer className="border-t border-stone-200/60 py-6 text-center text-sm text-stone-400">
        {t('landing.footer')}
      </footer>
    </div>
  )
}
