import { useState } from 'react'
import { useNavigate } from 'react-router'
import { t } from '../i18n/i18n'
import { usePreferences, useSavePreferences } from '../lib/queries'
import { useAuth } from '../stores/auth'
import { useToast } from '../components/Toast'
import { ErrorState, Skeleton, Spinner } from '../components/States'
import {
  DietSection,
  ExtrasSection,
  HouseholdSection,
} from '../components/PreferencesForm'
import { DEFAULT_PREFERENCES, type Preferences } from '../lib/types'
import { IconLogout } from '../components/Icons'

export function SettingsPage() {
  const navigate = useNavigate()
  const { user, logout } = useAuth()
  const prefsQuery = usePreferences()

  const doLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h1 className="text-2xl font-bold text-stone-900">{t('settings.title')}</h1>

      {/* Account */}
      <section className="rounded-3xl bg-white p-6 shadow-card" aria-label={t('settings.account')}>
        <h2 className="text-base font-bold text-stone-900">{t('settings.account')}</h2>
        <dl className="mt-3 space-y-2 text-sm">
          <div className="flex justify-between gap-4">
            <dt className="text-stone-500">{t('auth.displayName')}</dt>
            <dd className="font-medium text-stone-800">{user?.display_name}</dd>
          </div>
          <div className="flex justify-between gap-4">
            <dt className="text-stone-500">{t('settings.email')}</dt>
            <dd className="truncate font-medium text-stone-800">{user?.email}</dd>
          </div>
        </dl>
        <button
          type="button"
          onClick={() => void doLogout()}
          className="mt-4 inline-flex items-center gap-2 rounded-2xl bg-cream-dark px-4 py-2.5 text-sm font-semibold text-stone-600 transition hover:bg-stone-200 hover:text-stone-800 active:scale-95"
        >
          <IconLogout className="size-4" />
          {t('auth.logout')}
        </button>
      </section>

      {/* Preferences — same form as onboarding */}
      <section className="rounded-3xl bg-white p-6 shadow-card sm:p-8" aria-label={t('settings.preferences')}>
        <h2 className="text-base font-bold text-stone-900">{t('settings.preferences')}</h2>

        {prefsQuery.isLoading && (
          <div className="mt-5 space-y-4">
            <Skeleton className="h-10 w-56" />
            <Skeleton className="h-24 w-full" />
            <Skeleton className="h-24 w-full" />
          </div>
        )}
        {prefsQuery.isError && (
          <ErrorState error={prefsQuery.error} onRetry={() => void prefsQuery.refetch()} />
        )}
        {prefsQuery.data && (
          <SettingsForm initial={{ ...DEFAULT_PREFERENCES, ...prefsQuery.data }} />
        )}
      </section>
    </div>
  )
}

/** Mounted once preferences are loaded so the draft starts from server state. */
function SettingsForm({ initial }: { initial: Preferences }) {
  const { toast } = useToast()
  const { user, setUser } = useAuth()
  const save = useSavePreferences()
  const [draft, setDraft] = useState<Preferences>(initial)
  const [dirty, setDirty] = useState(false)

  const patch = (p: Partial<Preferences>) => {
    setDraft((d) => ({ ...d, ...p }))
    setDirty(true)
  }

  const submit = () => {
    if (draft.meal_types.length === 0) {
      toast(t('prefs.mealTypesRequired'), 'error')
      return
    }
    save.mutate({ ...draft, goal: draft.goal?.trim() || null }, {
      onSuccess: () => {
        if (user) setUser({ ...user, has_preferences: true })
        setDirty(false)
        toast(t('prefs.saved'), 'success')
      },
      onError: () => toast(t('prefs.saveError'), 'error'),
    })
  }

  return (
    <div className="mt-6 space-y-10">
      <HouseholdSection prefs={draft} patch={patch} />
      <hr className="border-stone-100" />
      <DietSection prefs={draft} patch={patch} />
      <hr className="border-stone-100" />
      <ExtrasSection prefs={draft} patch={patch} />

      <div className="sticky bottom-24 flex justify-end lg:bottom-8">
        <button
          type="button"
          onClick={submit}
          disabled={save.isPending || !dirty}
          className="flex items-center gap-2 rounded-2xl bg-brand-600 px-7 py-3 text-sm font-bold text-white shadow-soft transition hover:bg-brand-700 active:scale-95 disabled:opacity-50"
        >
          {save.isPending && (
            <Spinner className="size-4 border-2 border-white/40 border-t-white" />
          )}
          {save.isPending ? t('common.saving') : t('common.save')}
        </button>
      </div>
    </div>
  )
}
