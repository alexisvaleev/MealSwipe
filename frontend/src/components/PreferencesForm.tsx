import { useEffect, useMemo, useRef, useState } from 'react'
import { t } from '../i18n/i18n'
import { allergenLabel, dietLabel, mealTypeLabel } from '../lib/labels'
import { useIngredients } from '../lib/queries'
import {
  ALLERGENS,
  DIET_TYPES,
  MEAL_TYPES,
  type Ingredient,
  type Preferences,
} from '../lib/types'
import { IconMinus, IconPlus, IconSearch, IconX } from './Icons'

interface SectionProps {
  prefs: Preferences
  patch: (p: Partial<Preferences>) => void
}

function Stepper({
  value,
  min,
  max,
  onChange,
  ariaLabel,
}: {
  value: number
  min: number
  max: number
  onChange: (v: number) => void
  ariaLabel: string
}) {
  return (
    <div className="inline-flex items-center gap-3 rounded-2xl bg-white p-1.5 shadow-chip">
      <button
        type="button"
        aria-label={`${ariaLabel} −`}
        onClick={() => onChange(Math.max(min, value - 1))}
        className="flex size-10 items-center justify-center rounded-xl bg-cream-dark text-stone-600 transition hover:bg-brand-100 hover:text-brand-700 active:scale-90 disabled:opacity-40"
        disabled={value <= min}
      >
        <IconMinus className="size-5" />
      </button>
      <span className="w-10 text-center text-2xl font-bold tabular-nums text-stone-900">
        {value}
      </span>
      <button
        type="button"
        aria-label={`${ariaLabel} +`}
        onClick={() => onChange(Math.min(max, value + 1))}
        className="flex size-10 items-center justify-center rounded-xl bg-cream-dark text-stone-600 transition hover:bg-brand-100 hover:text-brand-700 active:scale-90 disabled:opacity-40"
        disabled={value >= max}
      >
        <IconPlus className="size-5" />
      </button>
    </div>
  )
}

function Chip({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: string
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={`rounded-2xl px-4 py-2.5 text-sm font-medium transition active:scale-95 ${
        active
          ? 'bg-brand-600 text-white shadow-chip'
          : 'bg-white text-stone-600 shadow-chip hover:bg-brand-50'
      }`}
    >
      {children}
    </button>
  )
}

// ---------------- Sections (shared by onboarding steps and settings) ----------

export function HouseholdSection({ prefs, patch }: SectionProps) {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-base font-semibold text-stone-900">{t('prefs.household')}</h3>
        <p className="mt-0.5 text-sm text-stone-500">{t('prefs.householdHint')}</p>
        <div className="mt-3">
          <Stepper
            value={prefs.household_size}
            min={1}
            max={10}
            onChange={(v) => patch({ household_size: v })}
            ariaLabel={t('prefs.household')}
          />
        </div>
      </div>
      <div>
        <h3 className="text-base font-semibold text-stone-900">{t('prefs.mealsPerDay')}</h3>
        <div className="mt-3">
          <Stepper
            value={prefs.meals_per_day}
            min={1}
            max={4}
            onChange={(v) => patch({ meals_per_day: v })}
            ariaLabel={t('prefs.mealsPerDay')}
          />
        </div>
      </div>
      <div>
        <h3 className="text-base font-semibold text-stone-900">{t('prefs.mealTypes')}</h3>
        <div className="mt-3 flex flex-wrap gap-2" role="group" aria-label={t('prefs.mealTypes')}>
          {MEAL_TYPES.map((mt) => {
            const active = prefs.meal_types.includes(mt)
            return (
              <Chip
                key={mt}
                active={active}
                onClick={() =>
                  patch({
                    meal_types: active
                      ? prefs.meal_types.filter((m) => m !== mt)
                      : [...prefs.meal_types, mt],
                  })
                }
              >
                {mealTypeLabel(mt)}
              </Chip>
            )
          })}
        </div>
        {prefs.meal_types.length === 0 && (
          <p role="alert" className="mt-2 text-sm text-rose-600">
            {t('prefs.mealTypesRequired')}
          </p>
        )}
      </div>
    </div>
  )
}

/** Searchable picker for excluded ingredients (GET /ingredients?q=). */
function ExcludedPicker({ prefs, patch }: SectionProps) {
  const [q, setQ] = useState('')
  const [debounced, setDebounced] = useState('')
  const [open, setOpen] = useState(false)
  const boxRef = useRef<HTMLDivElement>(null)
  // Names for ids selected during this session (fetched lists may be paginated).
  const [pickedNames, setPickedNames] = useState<Map<number, string>>(new Map())

  useEffect(() => {
    const id = window.setTimeout(() => setDebounced(q.trim()), 250)
    return () => window.clearTimeout(id)
  }, [q])

  const { data: results } = useIngredients(debounced)
  // Also load an unfiltered page once so saved ids resolve to names.
  const { data: initial } = useIngredients('')

  const names = useMemo(() => {
    const map = new Map(pickedNames)
    for (const ing of initial ?? []) map.set(ing.id, ing.name)
    for (const ing of results ?? []) map.set(ing.id, ing.name)
    return map
  }, [pickedNames, initial, results])

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (boxRef.current && !boxRef.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [])

  const options = useMemo(
    () => (results ?? []).filter((i: Ingredient) => !prefs.excluded_ingredient_ids.includes(i.id)),
    [results, prefs.excluded_ingredient_ids],
  )

  const add = (ing: Ingredient) => {
    patch({ excluded_ingredient_ids: [...prefs.excluded_ingredient_ids, ing.id] })
    setPickedNames((prev) => new Map(prev).set(ing.id, ing.name))
    setQ('')
  }

  const remove = (id: number) =>
    patch({ excluded_ingredient_ids: prefs.excluded_ingredient_ids.filter((x) => x !== id) })

  return (
    <div>
      <h3 className="text-base font-semibold text-stone-900">{t('prefs.excluded')}</h3>
      <p className="mt-0.5 text-sm text-stone-500">{t('prefs.excludedHint')}</p>

      {prefs.excluded_ingredient_ids.length > 0 && (
        <ul className="mt-3 flex flex-wrap gap-2">
          {prefs.excluded_ingredient_ids.map((id) => (
            <li
              key={id}
              className="inline-flex items-center gap-1.5 rounded-full bg-stone-800 py-1.5 pl-3.5 pr-2 text-sm text-white"
            >
              {names.get(id) ?? id}
              <button
                type="button"
                aria-label={`${t('common.delete')}: ${names.get(id) ?? id}`}
                onClick={() => remove(id)}
                className="flex size-5 items-center justify-center rounded-full bg-white/20 hover:bg-white/30"
              >
                <IconX className="size-3" />
              </button>
            </li>
          ))}
        </ul>
      )}

      <div ref={boxRef} className="relative mt-3">
        <IconSearch className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-stone-400" />
        <input
          type="search"
          value={q}
          onChange={(e) => {
            setQ(e.target.value)
            setOpen(true)
          }}
          onFocus={() => setOpen(true)}
          placeholder={t('prefs.excludedPlaceholder')}
          className="w-full rounded-2xl border-0 bg-white py-3 pl-10 pr-4 text-sm shadow-chip placeholder:text-stone-400 focus:outline-none focus:ring-2 focus:ring-brand-500"
        />
        {open && debounced.length > 0 && (
          <ul className="absolute inset-x-0 top-full z-20 mt-1.5 max-h-56 overflow-y-auto rounded-2xl bg-white py-1.5 shadow-soft">
            {options.map((ing) => (
              <li key={ing.id}>
                <button
                  type="button"
                  onClick={() => add(ing)}
                  className="w-full px-4 py-2.5 text-left text-sm text-stone-700 hover:bg-brand-50"
                >
                  {ing.name}
                </button>
              </li>
            ))}
            {options.length === 0 && (
              <li className="px-4 py-2.5 text-sm text-stone-400">{t('prefs.excludedEmpty')}</li>
            )}
          </ul>
        )}
      </div>
    </div>
  )
}

export function DietSection({ prefs, patch }: SectionProps) {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-base font-semibold text-stone-900">{t('prefs.diet')}</h3>
        <div className="mt-3 flex flex-wrap gap-2">
          {DIET_TYPES.map((d) => (
            <Chip key={d} active={prefs.diet_type === d} onClick={() => patch({ diet_type: d })}>
              {dietLabel(d)}
            </Chip>
          ))}
        </div>
      </div>

      <div>
        <h3 className="text-base font-semibold text-stone-900">{t('prefs.allergens')}</h3>
        <div className="mt-3 flex flex-wrap gap-2" role="group" aria-label={t('prefs.allergens')}>
          {ALLERGENS.map((a) => {
            const active = prefs.allergens.includes(a)
            return (
              <Chip
                key={a}
                active={active}
                onClick={() =>
                  patch({
                    allergens: active
                      ? prefs.allergens.filter((x) => x !== a)
                      : [...prefs.allergens, a],
                  })
                }
              >
                {allergenLabel(a)}
              </Chip>
            )
          })}
        </div>
      </div>

      <ExcludedPicker prefs={prefs} patch={patch} />
    </div>
  )
}

export function ExtrasSection({ prefs, patch }: SectionProps) {
  return (
    <div className="space-y-6">
      <div>
        <label htmlFor="pf-budget" className="text-base font-semibold text-stone-900">
          {t('prefs.budget')} <span className="font-normal text-stone-400">({t('common.optional')})</span>
        </label>
        <p className="mt-0.5 text-sm text-stone-500">{t('prefs.budgetHint')}</p>
        <input
          id="pf-budget"
          type="number"
          min={0}
          step={100}
          inputMode="numeric"
          value={prefs.weekly_budget ?? ''}
          onChange={(e) =>
            patch({ weekly_budget: e.target.value === '' ? null : Number(e.target.value) })
          }
          className="mt-3 w-full max-w-xs rounded-2xl border-0 bg-white px-4 py-3 text-sm shadow-chip focus:outline-none focus:ring-2 focus:ring-brand-500"
        />
      </div>

      <div>
        <label htmlFor="pf-maxtime" className="text-base font-semibold text-stone-900">
          {t('prefs.maxTime')}{' '}
          <span className="font-normal text-stone-400">({t('common.optional')})</span>
        </label>
        <p className="mt-0.5 text-sm text-stone-500">{t('prefs.maxTimeHint')}</p>
        <input
          id="pf-maxtime"
          type="number"
          min={5}
          step={5}
          inputMode="numeric"
          value={prefs.max_cook_minutes ?? ''}
          onChange={(e) =>
            patch({ max_cook_minutes: e.target.value === '' ? null : Number(e.target.value) })
          }
          className="mt-3 w-full max-w-xs rounded-2xl border-0 bg-white px-4 py-3 text-sm shadow-chip focus:outline-none focus:ring-2 focus:ring-brand-500"
        />
      </div>

      <div>
        <label htmlFor="pf-goal" className="text-base font-semibold text-stone-900">
          {t('prefs.goal')}{' '}
          <span className="font-normal text-stone-400">({t('common.optional')})</span>
        </label>
        <p className="mt-0.5 text-sm text-stone-500">{t('prefs.goalHint')}</p>
        <input
          id="pf-goal"
          type="text"
          maxLength={140}
          value={prefs.goal ?? ''}
          onChange={(e) => patch({ goal: e.target.value })}
          placeholder={t('prefs.goalPlaceholder')}
          className="mt-3 w-full rounded-2xl border-0 bg-white px-4 py-3 text-sm shadow-chip placeholder:text-stone-400 focus:outline-none focus:ring-2 focus:ring-brand-500"
        />
      </div>
    </div>
  )
}
