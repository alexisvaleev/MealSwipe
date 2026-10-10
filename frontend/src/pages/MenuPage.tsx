import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import { t } from '../i18n/i18n'
import { ApiError } from '../lib/api'
import { addWeeks, formatDateShort, mondayOf } from '../lib/format'
import { dayFullLabel, dayLabel, mealTypeLabel } from '../lib/labels'
import {
  useAddMenuItem,
  useGenerateMenu,
  useMenu,
  usePreferences,
  useRecipes,
  useRemoveMenuItem,
  useSwapItem,
  useUpdateMenuItem,
} from '../lib/queries'
import {
  MEAL_TYPES,
  type MealType,
  type MenuItem,
  type UnfilledSlot,
} from '../lib/types'
import { Modal } from '../components/Modal'
import { RecipeModal } from '../components/RecipeModal'
import { RecipeImage } from '../components/RecipeImage'
import {
  CardGridSkeleton,
  EmptyState,
  ErrorState,
  Spinner,
} from '../components/States'
import { useToast } from '../components/Toast'
import {
  IconCalendar,
  IconCart,
  IconChevronLeft,
  IconChevronRight,
  IconLock,
  IconMinus,
  IconPlus,
  IconSearch,
  IconSwap,
  IconUnlock,
  IconX,
} from '../components/Icons'

const unfilledKey = (day: number, mt: MealType) => `${day}:${mt}`

export function MenuPage() {
  const { toast } = useToast()
  const [week, setWeek] = useState(() => mondayOf(new Date()))
  const thisMonday = mondayOf(new Date())
  const { data: prefs } = usePreferences()
  const { data: menu, isLoading, isError, error, refetch } = useMenu(week)
  const generate = useGenerateMenu()

  const mealTypes = useMemo<MealType[]>(() => {
    const configured = prefs?.meal_types ?? []
    const ordered = MEAL_TYPES.filter((mt) => configured.includes(mt))
    if (ordered.length > 0) return ordered
    // Fall back to whatever the menu actually contains.
    const inMenu = new Set<MealType>()
    for (const item of menu?.items ?? []) inMenu.add(item.meal_type)
    return inMenu.size > 0 ? MEAL_TYPES.filter((mt) => inMenu.has(mt)) : ['breakfast', 'lunch', 'dinner']
  }, [prefs, menu])

  // Unfilled slots reported by the last generate call (keyed to its week).
  const [unfilledResult, setUnfilledResult] = useState<{ week: string; map: Map<string, UnfilledSlot> } | null>(null)
  const unfilled = useMemo(
    () => (unfilledResult?.week === week ? unfilledResult.map : new Map<string, UnfilledSlot>()),
    [unfilledResult, week],
  )
  const [confirmRegen, setConfirmRegen] = useState(false)
  const [detailsId, setDetailsId] = useState<string | null>(null)
  const [picker, setPicker] = useState<{ day: number; mealType: MealType } | null>(null)

  const runGenerate = (weekStart: string) => {
    generate.mutate(weekStart, {
      onSuccess: (res) => {
        const map = new Map<string, UnfilledSlot>()
        for (const u of res.unfilled ?? []) map.set(unfilledKey(u.day_index, u.meal_type), u)
        setUnfilledResult({ week: res.menu.week_start, map })
        const n = map.size
        toast(n ? t('menu.generatedUnfilled', { count: n }) : t('menu.generated'), 'success')
      },
      onError: () => toast(t('error.generic'), 'error'),
    })
    setConfirmRegen(false)
  }

  return (
    <div>
      <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold text-stone-900">{t('menu.title')}</h1>
          <p className="mt-0.5 text-sm text-stone-500">
            {t('menu.weekOf', { date: formatDateShort(week) })}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            aria-label={t('menu.prevWeek')}
            onClick={() => setWeek((w) => addWeeks(w, -1))}
            className="flex size-10 items-center justify-center rounded-2xl bg-white text-stone-500 shadow-chip transition hover:text-brand-700 active:scale-95"
          >
            <IconChevronLeft className="size-5" />
          </button>
          {week !== thisMonday && (
            <button
              type="button"
              onClick={() => setWeek(thisMonday)}
              className="rounded-2xl bg-white px-3 py-2 text-xs font-semibold text-brand-700 shadow-chip hover:bg-brand-50"
            >
              {t('menu.thisWeek')}
            </button>
          )}
          <button
            type="button"
            aria-label={t('menu.nextWeek')}
            onClick={() => setWeek((w) => addWeeks(w, 1))}
            className="flex size-10 items-center justify-center rounded-2xl bg-white text-stone-500 shadow-chip transition hover:text-brand-700 active:scale-95"
          >
            <IconChevronRight className="size-5" />
          </button>
          {menu && (
            <button
              type="button"
              onClick={() => setConfirmRegen(true)}
              className="rounded-2xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white shadow-card transition hover:bg-brand-700 active:scale-95"
            >
              {t('menu.regenerate')}
            </button>
          )}
        </div>
      </div>

      {isLoading && <CardGridSkeleton count={7} />}
      {isError && <ErrorState error={error} onRetry={() => void refetch()} />}

      {!isLoading && !isError && menu === null && (
        <EmptyState
          icon={<IconCalendar className="size-8" />}
          title={t('menu.empty')}
          text={t('menu.emptyText')}
          action={
            <button
              type="button"
              onClick={() => runGenerate(week)}
              disabled={generate.isPending}
              className="flex items-center gap-2 rounded-2xl bg-brand-600 px-6 py-3 text-sm font-bold text-white shadow-card transition hover:bg-brand-700 active:scale-95 disabled:opacity-60"
            >
              {generate.isPending && <Spinner className="size-4 border-2 border-white/40 border-t-white" />}
              {generate.isPending ? t('menu.generating') : t('menu.generate')}
            </button>
          }
        />
      )}

      {menu && (
        <>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-7 lg:gap-2.5">
            {Array.from({ length: 7 }, (_, day) => (
              <DayColumn
                key={day}
                day={day}
                week={week}
                mealTypes={mealTypes}
                items={menu.items.filter((i) => i.day_index === day)}
                unfilled={unfilled}
                menuId={menu.id}
                onOpenRecipe={setDetailsId}
                onPick={(mt) => setPicker({ day, mealType: mt })}
              />
            ))}
          </div>

          <div className="mt-6 flex justify-center">
            <Link
              to="/shopping"
              className="flex items-center gap-2 rounded-2xl bg-stone-800 px-6 py-3 text-sm font-bold text-white shadow-card transition hover:bg-stone-700 active:scale-95"
            >
              <IconCart className="size-5" />
              {t('menu.shoppingCta')}
            </Link>
          </div>
        </>
      )}

      {/* Regenerate confirm */}
      <Modal open={confirmRegen} onClose={() => setConfirmRegen(false)} ariaLabel={t('menu.regenerate')}>
        <div className="p-6 sm:p-8">
          <h2 className="text-xl font-bold text-stone-900">{t('menu.regenerateConfirmTitle')}</h2>
          <p className="mt-2 text-sm leading-relaxed text-stone-600">
            {t('menu.regenerateConfirmText')}
          </p>
          <div className="mt-6 flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setConfirmRegen(false)}
              className="rounded-2xl px-4 py-2.5 text-sm font-semibold text-stone-500 transition hover:bg-cream-dark"
            >
              {t('common.cancel')}
            </button>
            <button
              type="button"
              onClick={() => runGenerate(week)}
              disabled={generate.isPending}
              className="flex items-center gap-2 rounded-2xl bg-brand-600 px-5 py-2.5 text-sm font-semibold text-white shadow-chip transition hover:bg-brand-700 disabled:opacity-60"
            >
              {generate.isPending && <Spinner className="size-4 border-2 border-white/40 border-t-white" />}
              {t('menu.regenerateConfirmOk')}
            </button>
          </div>
        </div>
      </Modal>

      {/* Recipe picker for empty slots (mounted only when open) */}
      {picker && (
        <RecipePickerModal
          menuId={menu?.id}
          picker={picker}
          onClose={() => setPicker(null)}
        />
      )}

      <RecipeModal recipeId={detailsId} onClose={() => setDetailsId(null)} />
    </div>
  )
}

// ---------------- Day column ----------------

function DayColumn({
  day,
  week,
  mealTypes,
  items,
  unfilled,
  menuId,
  onOpenRecipe,
  onPick,
}: {
  day: number
  week: string
  mealTypes: MealType[]
  items: MenuItem[]
  unfilled: Map<string, UnfilledSlot>
  menuId: string
  onOpenRecipe: (id: string) => void
  onPick: (mealType: MealType) => void
}) {
  const date = new Date(`${week}T00:00:00`)
  date.setDate(date.getDate() + day)
  const dateLabel = date.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })

  return (
    <section
      aria-label={dayFullLabel(day)}
      className="rounded-3xl bg-white/70 p-3 shadow-card lg:bg-transparent lg:p-1.5 lg:shadow-none"
    >
      <h2 className="mb-2.5 flex items-baseline justify-between px-1">
        <span className="text-sm font-bold text-stone-800">{dayLabel(day)}</span>
        <span className="text-xs text-stone-400">{dateLabel}</span>
      </h2>
      <div className="space-y-2.5">
        {mealTypes.map((mt) => {
          const item = items.find((i) => i.meal_type === mt)
          return item ? (
            <FilledSlot
              key={mt}
              item={item}
              menuId={menuId}
              onOpenRecipe={onOpenRecipe}
            />
          ) : (
            <EmptySlot
              key={mt}
              mealType={mt}
              unfilled={unfilled.get(unfilledKey(day, mt))}
              onPick={() => onPick(mt)}
            />
          )
        })}
      </div>
    </section>
  )
}

// ---------------- Slots ----------------

function SlotMealTypeBadge({ mt }: { mt: MealType }) {
  return (
    <span className="inline-block rounded-full bg-brand-100 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-brand-800">
      {mealTypeLabel(mt)}
    </span>
  )
}

function FilledSlot({
  item,
  menuId,
  onOpenRecipe,
}: {
  item: MenuItem
  menuId: string
  onOpenRecipe: (id: string) => void
}) {
  const { toast } = useToast()
  const update = useUpdateMenuItem(menuId)
  const remove = useRemoveMenuItem(menuId)
  const swap = useSwapItem(menuId)
  const busy = update.isPending || remove.isPending || swap.isPending

  const setServings = (servings: number) => {
    if (servings < 1 || servings > 20) return
    update.mutate(
      { itemId: item.id, patch: { servings } },
      { onError: () => toast(t('error.saveFailed'), 'error') },
    )
  }

  const toggleLock = () =>
    update.mutate(
      { itemId: item.id, patch: { locked: !item.locked } },
      { onError: () => toast(t('error.saveFailed'), 'error') },
    )

  const doSwap = () =>
    swap.mutate(item.id, {
      onError: (err) => {
        if (err instanceof ApiError && (err.status === 404 || err.code === 'no_candidates')) {
          toast(t('menu.noSwapCandidates'))
        } else {
          toast(t('menu.swapError'), 'error')
        }
      },
    })

  const doRemove = () =>
    remove.mutate(item.id, {
      onSuccess: () => toast(t('menu.itemRemoved'), 'info'),
      onError: () => toast(t('error.saveFailed'), 'error'),
    })

  return (
    <article
      className={`overflow-hidden rounded-2xl bg-white shadow-card transition ${busy ? 'opacity-70' : ''}`}
    >
      <button
        type="button"
        onClick={() => onOpenRecipe(item.recipe.id)}
        aria-label={`${t('swipe.details')}: ${item.recipe.title}`}
        className="relative block w-full text-left"
      >
        <RecipeImage
          src={item.recipe.image_url}
          alt={item.recipe.title}
          className="h-20 w-full"
          fallbackClassName="text-2xl"
        />
        {item.locked && (
          <span className="absolute left-1.5 top-1.5 rounded-full bg-stone-800/80 p-1 text-white backdrop-blur">
            <IconLock className="size-3" />
          </span>
        )}
      </button>
      <div className="p-2">
        <SlotMealTypeBadge mt={item.meal_type} />
        <button
          type="button"
          onClick={() => onOpenRecipe(item.recipe.id)}
          className="mt-1 line-clamp-2 w-full text-left text-xs font-semibold leading-snug text-stone-900 hover:text-brand-700"
        >
          {item.recipe.title}
        </button>

        {/* Servings stepper */}
        <div className="mt-1.5 flex items-center gap-1 text-[11px] text-stone-500">
          <span className="sr-only">{t('menu.servings')}</span>
          <button
            type="button"
            aria-label={`${t('menu.servings')} −`}
            onClick={() => setServings(item.servings - 1)}
            disabled={item.servings <= 1 || busy}
            className="flex size-6 items-center justify-center rounded-full bg-cream-dark text-stone-500 transition hover:text-brand-700 disabled:opacity-40"
          >
            <IconMinus className="size-3.5" />
          </button>
          <span className="w-4 text-center font-bold tabular-nums text-stone-800">
            {item.servings}
          </span>
          <button
            type="button"
            aria-label={`${t('menu.servings')} +`}
            onClick={() => setServings(item.servings + 1)}
            disabled={item.servings >= 20 || busy}
            className="flex size-6 items-center justify-center rounded-full bg-cream-dark text-stone-500 transition hover:text-brand-700 disabled:opacity-40"
          >
            <IconPlus className="size-3.5" />
          </button>
          <span className="ml-auto" />
          <IconBtn
            label={item.locked ? t('menu.unlock') : t('menu.lock')}
            onClick={toggleLock}
            disabled={busy}
            active={item.locked}
          >
            {item.locked ? <IconLock className="size-3.5" /> : <IconUnlock className="size-3.5" />}
          </IconBtn>
          <IconBtn label={t('menu.swap')} onClick={doSwap} disabled={busy || item.locked}>
            <IconSwap className="size-3.5" />
          </IconBtn>
          <IconBtn label={t('menu.remove')} onClick={doRemove} disabled={busy} danger>
            <IconX className="size-3.5" />
          </IconBtn>
        </div>
      </div>
    </article>
  )
}

function IconBtn({
  label,
  onClick,
  disabled,
  active,
  danger,
  children,
}: {
  label: string
  onClick: () => void
  disabled?: boolean
  active?: boolean
  danger?: boolean
  children: ReactNode
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      onClick={onClick}
      disabled={disabled}
      className={`flex size-6 items-center justify-center rounded-full transition active:scale-90 disabled:opacity-40 ${
        active
          ? 'bg-stone-800 text-white'
          : danger
            ? 'text-stone-400 hover:bg-rose-50 hover:text-rose-600'
            : 'text-stone-400 hover:bg-brand-50 hover:text-brand-700'
      }`}
    >
      {children}
    </button>
  )
}

function EmptySlot({
  mealType,
  unfilled,
  onPick,
}: {
  mealType: MealType
  unfilled?: UnfilledSlot
  onPick: () => void
}) {
  return (
    <div className="rounded-2xl border-2 border-dashed border-stone-300 bg-white/50 p-3 text-center">
      <SlotMealTypeBadge mt={mealType} />
      <p className="mt-1.5 text-[11px] leading-snug text-stone-400">
        {unfilled ? t('menu.unfilled') : t('menu.addDish')}
      </p>
      <button
        type="button"
        onClick={onPick}
        aria-label={t('menu.addDish')}
        className="mt-2 inline-flex items-center gap-1 rounded-full bg-white px-3 py-1.5 text-[11px] font-semibold text-brand-700 shadow-chip transition hover:bg-brand-50 active:scale-95"
      >
        <IconPlus className="size-3.5" />
        {t('menu.addDish')}
      </button>
    </div>
  )
}

// ---------------- Recipe picker modal ----------------

function RecipePickerModal({
  menuId,
  picker,
  onClose,
}: {
  menuId: string | undefined
  picker: { day: number; mealType: MealType } | null
  onClose: () => void
}) {
  const { toast } = useToast()
  const [q, setQ] = useState('')
  const [debounced, setDebounced] = useState('')
  const { data: recipes, isLoading } = useRecipes(
    { meal_type: picker?.mealType, q: debounced || undefined, page_size: 50 },
  )
  const addItem = useAddMenuItem(menuId ?? '')

  useEffect(() => {
    const id = window.setTimeout(() => setDebounced(q.trim()), 250)
    return () => window.clearTimeout(id)
  }, [q])

  if (!picker) return null

  const add = (recipeId: string, servings: number) => {
    if (!menuId) return
    addItem.mutate(
      {
        day_index: picker.day,
        meal_type: picker.mealType,
        recipe_id: recipeId,
        servings: servings || 2,
      },
      {
        onSuccess: () => {
          toast(t('recipe.addedToMenu'), 'success')
          onClose()
        },
        onError: () => toast(t('error.saveFailed'), 'error'),
      },
    )
  }

  return (
    <Modal open onClose={onClose} ariaLabel={t('menu.pickRecipe')}>
      <div className="p-5 sm:p-6">
        <h2 className="text-lg font-bold text-stone-900">{t('menu.pickRecipe')}</h2>
        <p className="text-sm text-stone-500">
          {t('menu.pickRecipeFor', {
            meal: mealTypeLabel(picker.mealType).toLowerCase(),
            day: dayFullLabel(picker.day).toLowerCase(),
          })}
        </p>
        <div className="relative mt-4">
          <IconSearch className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-stone-400" />
          <input
            type="search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={t('favorites.searchPlaceholder')}
            aria-label={t('common.search')}
            className="w-full rounded-2xl border-0 bg-cream-dark py-3 pl-10 pr-4 text-sm placeholder:text-stone-400 focus:outline-none focus:ring-2 focus:ring-brand-500"
          />
        </div>
        <ul className="mt-4 max-h-[55dvh] space-y-2 overflow-y-auto">
          {isLoading &&
            Array.from({ length: 4 }, (_, i) => (
              <li key={i} className="h-16 animate-pulse rounded-2xl bg-stone-100" />
            ))}
          {!isLoading && (recipes ?? []).length === 0 && (
            <li className="py-6 text-center text-sm text-stone-400">{t('menu.unfilled')}</li>
          )}
          {(recipes ?? []).map((r) => (
            <li key={r.id}>
              <button
                type="button"
                onClick={() => add(r.id, r.servings)}
                disabled={addItem.isPending}
                className="flex w-full items-center gap-3 rounded-2xl bg-white p-2 text-left shadow-chip transition hover:bg-brand-50 disabled:opacity-60"
              >
                <RecipeImage
                  src={r.image_url}
                  alt={r.title}
                  className="size-12 shrink-0 rounded-xl"
                  fallbackClassName="text-lg"
                />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-semibold text-stone-800">
                    {r.title}
                  </span>
                  {r.category && (
                    <span className="block truncate text-xs text-stone-400">{r.category}</span>
                  )}
                </span>
                <IconPlus className="mr-2 size-4 shrink-0 text-brand-600" />
              </button>
            </li>
          ))}
        </ul>
      </div>
    </Modal>
  )
}
