import { useState } from 'react'
import { useNavigate } from 'react-router'
import { AnimatePresence, motion } from 'framer-motion'
import { t } from '../i18n/i18n'
import { usePreferences, useSavePreferences } from '../lib/queries'
import { useAuth } from '../stores/auth'
import { useToast } from '../components/Toast'
import { ErrorState, Spinner } from '../components/States'
import {
  DietSection,
  ExtrasSection,
  HouseholdSection,
} from '../components/PreferencesForm'
import { DEFAULT_PREFERENCES, type Preferences } from '../lib/types'

const STEPS = ['prefs.stepFood', 'prefs.stepDiet', 'prefs.stepExtras'] as const

export function OnboardingPage() {
  const prefsQuery = usePreferences()

  if (prefsQuery.isLoading) {
    return (
      <div className="flex min-h-[60dvh] items-center justify-center">
        <Spinner className="size-10" />
      </div>
    )
  }
  if (prefsQuery.isError) {
    return <ErrorState error={prefsQuery.error} onRetry={() => void prefsQuery.refetch()} />
  }
  // Mount the wizard only once preferences are known so the draft starts
  // from server values (defaults when unset) — no sync effect needed.
  return <Wizard initial={{ ...DEFAULT_PREFERENCES, ...prefsQuery.data }} />
}

function Wizard({ initial }: { initial: Preferences }) {
  const navigate = useNavigate()
  const { toast } = useToast()
  const { user, setUser } = useAuth()
  const save = useSavePreferences()

  const [step, setStep] = useState(0)
  const [draft, setDraft] = useState<Preferences>(initial)
  const [error, setError] = useState('')

  const patch = (p: Partial<Preferences>) => setDraft((d) => ({ ...d, ...p }))

  const finish = () => {
    if (draft.meal_types.length === 0) {
      setError(t('prefs.mealTypesRequired'))
      setStep(0)
      return
    }
    save.mutate({ ...draft, goal: draft.goal?.trim() || null }, {
      onSuccess: () => {
        if (user) setUser({ ...user, has_preferences: true })
        toast(t('prefs.saved'), 'success')
        navigate('/swipe', { replace: true })
      },
      onError: () => toast(t('prefs.saveError'), 'error'),
    })
  }

  const next = () => {
    if (step === 0 && draft.meal_types.length === 0) {
      setError(t('prefs.mealTypesRequired'))
      return
    }
    setError('')
    if (step < STEPS.length - 1) setStep(step + 1)
    else finish()
  }

  return (
    <div className="mx-auto max-w-lg">
      <div className="rounded-3xl bg-white p-6 shadow-card sm:p-8">
        <div className="mb-6">
          <p className="text-xs font-semibold uppercase tracking-wider text-brand-600">
            {t('prefs.step', { current: step + 1, total: STEPS.length })}
          </p>
          <h1 className="mt-1 text-2xl font-bold text-stone-900">{t('prefs.onboardingTitle')}</h1>
          <p className="mt-1 text-sm text-stone-500">{t('prefs.onboardingSubtitle')}</p>

          {/* Progress */}
          <div className="mt-4 flex gap-1.5" aria-hidden>
            {STEPS.map((s, i) => (
              <span
                key={s}
                className={`h-1.5 flex-1 rounded-full transition-colors ${
                  i <= step ? 'bg-brand-600' : 'bg-stone-200'
                }`}
              />
            ))}
          </div>
          <h2 className="mt-4 text-lg font-bold text-stone-800">{t(STEPS[step])}</h2>
        </div>

        <AnimatePresence mode="wait">
          <motion.div
            key={step}
            initial={{ opacity: 0, x: 24 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -24 }}
            transition={{ duration: 0.2 }}
          >
            {step === 0 && <HouseholdSection prefs={draft} patch={patch} />}
            {step === 1 && <DietSection prefs={draft} patch={patch} />}
            {step === 2 && <ExtrasSection prefs={draft} patch={patch} />}
          </motion.div>
        </AnimatePresence>

        {error && (
          <p role="alert" className="mt-4 text-sm text-rose-600">
            {error}
          </p>
        )}

        <div className="mt-8 flex items-center justify-between">
          <div>
            {step > 0 ? (
              <button
                type="button"
                onClick={() => setStep(step - 1)}
                className="rounded-2xl px-4 py-2.5 text-sm font-semibold text-stone-500 transition hover:bg-cream-dark"
              >
                {t('prefs.backStep')}
              </button>
            ) : (
              <button
                type="button"
                onClick={() => navigate('/swipe')}
                className="rounded-2xl px-4 py-2.5 text-sm font-semibold text-stone-400 transition hover:bg-cream-dark hover:text-stone-600"
              >
                {t('prefs.skip')}
              </button>
            )}
          </div>
          <button
            type="button"
            onClick={next}
            disabled={save.isPending}
            className="flex items-center gap-2 rounded-2xl bg-brand-600 px-6 py-3 text-sm font-bold text-white shadow-card transition hover:bg-brand-700 active:scale-95 disabled:opacity-60"
          >
            {save.isPending && <Spinner className="size-4 border-2 border-white/40 border-t-white" />}
            {step < STEPS.length - 1 ? t('prefs.next') : t('prefs.finish')}
          </button>
        </div>

        {step > 0 && (
          <div className="mt-3 text-center">
            <button
              type="button"
              onClick={() => navigate('/swipe')}
              className="text-sm text-stone-400 underline-offset-2 hover:text-stone-600 hover:underline"
            >
              {t('prefs.skip')}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
