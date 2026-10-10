import { useState, type FormEvent, type ReactNode } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router'
import { t } from '../i18n/i18n'
import { ApiError } from '../lib/api'
import { useAuth } from '../stores/auth'
import { useToast } from '../components/Toast'
import { Spinner } from '../components/States'

const APP_NAME = import.meta.env.VITE_APP_NAME ?? 'MealSwipe'

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

function AuthLayout({ children, title, subtitle }: { children: ReactNode; title: string; subtitle: string }) {
  return (
    <div className="flex min-h-dvh flex-col items-center justify-center bg-gradient-to-b from-brand-50 to-cream px-4 py-10">
      <Link to="/" className="mb-6 flex items-center gap-2.5">
        <span className="flex size-11 items-center justify-center rounded-2xl bg-brand-600 text-xl text-white shadow-card">
          🍽
        </span>
        <span className="text-2xl font-extrabold tracking-tight text-stone-900">{APP_NAME}</span>
      </Link>
      <div className="w-full max-w-sm rounded-3xl bg-white p-7 shadow-soft">
        <h1 className="text-2xl font-bold text-stone-900">{title}</h1>
        <p className="mt-1 text-sm text-stone-500">{subtitle}</p>
        <div className="mt-6">{children}</div>
      </div>
    </div>
  )
}

const inputCls =
  'w-full rounded-2xl border-0 bg-cream-dark px-4 py-3 text-sm text-stone-900 placeholder:text-stone-400 focus:outline-none focus:ring-2 focus:ring-brand-500'
const labelCls = 'mb-1.5 block text-sm font-medium text-stone-700'
const errCls = 'mt-1 text-xs text-rose-600'

function mapAuthError(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.code === 'invalid_credentials' || err.status === 401) return t('auth.invalidCredentials')
    if (err.code === 'email_taken' || err.status === 409) return t('auth.emailTaken')
    if (err.code === 'network_error') return t('common.networkError')
    return err.message
  }
  return t('error.generic')
}

export function LoginPage() {
  const { status, login } = useAuth()
  const { toast } = useToast()
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [errors, setErrors] = useState<{ email?: string; password?: string }>({})
  const [serverError, setServerError] = useState('')
  const [busy, setBusy] = useState(false)

  if (status === 'authed') return <Navigate to={from ?? '/swipe'} replace />

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const next: typeof errors = {}
    if (!email.trim()) next.email = t('auth.emailRequired')
    else if (!EMAIL_RE.test(email.trim())) next.email = t('auth.emailInvalid')
    if (!password) next.password = t('auth.passwordRequired')
    setErrors(next)
    if (Object.keys(next).length > 0) return

    setBusy(true)
    setServerError('')
    try {
      const user = await login(email.trim(), password)
      toast(t('auth.welcomeBack', { name: user.display_name }), 'success')
      navigate(user.has_preferences ? (from ?? '/swipe') : '/onboarding', { replace: true })
    } catch (err) {
      setServerError(mapAuthError(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthLayout title={t('auth.loginTitle')} subtitle={t('auth.loginSubtitle')}>
      <form onSubmit={submit} noValidate className="space-y-4">
        <div>
          <label htmlFor="login-email" className={labelCls}>{t('auth.email')}</label>
          <input
            id="login-email"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            aria-invalid={Boolean(errors.email)}
            className={inputCls}
          />
          {errors.email && <p role="alert" className={errCls}>{errors.email}</p>}
        </div>
        <div>
          <label htmlFor="login-password" className={labelCls}>{t('auth.password')}</label>
          <input
            id="login-password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            aria-invalid={Boolean(errors.password)}
            className={inputCls}
          />
          {errors.password && <p role="alert" className={errCls}>{errors.password}</p>}
        </div>

        {serverError && (
          <p role="alert" className="rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">
            {serverError}
          </p>
        )}

        <button
          type="submit"
          disabled={busy}
          className="flex w-full items-center justify-center gap-2 rounded-2xl bg-brand-600 py-3.5 text-base font-bold text-white shadow-card transition hover:bg-brand-700 active:scale-[0.98] disabled:opacity-60"
        >
          {busy && <Spinner className="size-4 border-2 border-white/40 border-t-white" />}
          {busy ? t('auth.loggingIn') : t('auth.login')}
        </button>

        <p className="pt-2 text-center text-sm text-stone-500">
          {t('auth.noAccount')}{' '}
          <Link to="/register" className="font-semibold text-brand-700 hover:underline">
            {t('auth.createOne')}
          </Link>
        </p>
      </form>
    </AuthLayout>
  )
}

export function RegisterPage() {
  const { status, register } = useAuth()
  const { toast } = useToast()
  const navigate = useNavigate()

  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [errors, setErrors] = useState<{ name?: string; email?: string; password?: string }>({})
  const [serverError, setServerError] = useState('')
  const [busy, setBusy] = useState(false)

  if (status === 'authed') return <Navigate to="/onboarding" replace />

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const next: typeof errors = {}
    if (!name.trim()) next.name = t('auth.nameRequired')
    if (!email.trim()) next.email = t('auth.emailRequired')
    else if (!EMAIL_RE.test(email.trim())) next.email = t('auth.emailInvalid')
    if (!password) next.password = t('auth.passwordRequired')
    else if (password.length < 8) next.password = t('auth.passwordTooShort')
    setErrors(next)
    if (Object.keys(next).length > 0) return

    setBusy(true)
    setServerError('')
    try {
      await register(email.trim(), password, name.trim())
      toast(t('auth.accountCreated'), 'success')
      navigate('/onboarding', { replace: true })
    } catch (err) {
      setServerError(mapAuthError(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthLayout title={t('auth.registerTitle')} subtitle={t('auth.registerSubtitle')}>
      <form onSubmit={submit} noValidate className="space-y-4">
        <div>
          <label htmlFor="reg-name" className={labelCls}>{t('auth.displayName')}</label>
          <input
            id="reg-name"
            type="text"
            autoComplete="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            aria-invalid={Boolean(errors.name)}
            className={inputCls}
          />
          {errors.name && <p role="alert" className={errCls}>{errors.name}</p>}
        </div>
        <div>
          <label htmlFor="reg-email" className={labelCls}>{t('auth.email')}</label>
          <input
            id="reg-email"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            aria-invalid={Boolean(errors.email)}
            className={inputCls}
          />
          {errors.email && <p role="alert" className={errCls}>{errors.email}</p>}
        </div>
        <div>
          <label htmlFor="reg-password" className={labelCls}>
            {t('auth.password')}
            <span className="ml-1.5 font-normal text-stone-400">({t('auth.passwordHint')})</span>
          </label>
          <input
            id="reg-password"
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            aria-invalid={Boolean(errors.password)}
            className={inputCls}
          />
          {errors.password && <p role="alert" className={errCls}>{errors.password}</p>}
        </div>

        {serverError && (
          <p role="alert" className="rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">
            {serverError}
          </p>
        )}

        <button
          type="submit"
          disabled={busy}
          className="flex w-full items-center justify-center gap-2 rounded-2xl bg-brand-600 py-3.5 text-base font-bold text-white shadow-card transition hover:bg-brand-700 active:scale-[0.98] disabled:opacity-60"
        >
          {busy && <Spinner className="size-4 border-2 border-white/40 border-t-white" />}
          {busy ? t('auth.registering') : t('auth.register')}
        </button>

        <p className="pt-2 text-center text-sm text-stone-500">
          {t('auth.haveAccount')}{' '}
          <Link to="/login" className="font-semibold text-brand-700 hover:underline">
            {t('auth.signIn')}
          </Link>
        </p>
      </form>
    </AuthLayout>
  )
}
