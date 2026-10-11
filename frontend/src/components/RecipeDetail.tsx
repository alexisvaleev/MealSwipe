import { useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router'
import { t } from '../i18n/i18n'
import { formatIngredient, formatNumber, servingsLabel, totalMinutes } from '../lib/format'
import { mondayOf } from '../lib/format'
import { dayLabel, mealTypeLabel } from '../lib/labels'
import {
  useAddMenuItem,
  useFavorite,
  useMenu,
  useRecipe,
} from '../lib/queries'
import { MEAL_TYPES, type MealType, type Recipe } from '../lib/types'
import { useToast } from './Toast'
import { ErrorState, Skeleton, Spinner } from './States'
import { RecipeImage } from './RecipeImage'
import { IconCalendar, IconCheck, IconClock, IconFlame, IconMinus, IconPlus, IconStar, IconStarFilled, IconWallet } from './Icons'
import { useAuth } from '../stores/auth'

function MetaChip({ icon, children }: { icon: ReactNode; children: ReactNode }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-cream-dark px-3 py-1.5 text-sm font-medium text-stone-700">
      <span className="text-brand-600">{icon}</span>
      {children}
    </span>
  )
}

/** "Add to menu" panel inside the recipe detail — picks day + meal type. */
function AddToMenuPanel({ recipe }: { recipe: Recipe }) {
  const { toast } = useToast()
  const week = mondayOf(new Date())
  const { data: menu } = useMenu(week)
  const addItem = useAddMenuItem(menu?.id ?? '')
  const [day, setDay] = useState(0)
  const [mealType, setMealType] = useState<MealType>(recipe.meal_types[0] ?? 'dinner')
  const [open, setOpen] = useState(false)
  const navigate = useNavigate()

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="inline-flex items-center gap-2 rounded-2xl border-2 border-brand-600 px-4 py-2.5 text-sm font-semibold text-brand-700 transition hover:bg-brand-50 active:scale-95"
      >
        <IconCalendar className="size-4" />
        {t('recipe.toMenu')}
      </button>
    )
  }

  if (!menu) {
    return (
      <div className="rounded-2xl bg-cream-dark p-4 text-sm text-stone-600">
        <p>{t('shopping.noMenuText')}</p>
        <button
          type="button"
          onClick={() => navigate('/menu')}
          className="mt-2 font-semibold text-brand-700 underline underline-offset-2"
        >
          {t('shopping.toMenu')}
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-3 rounded-2xl bg-cream-dark p-4">
      <div className="flex flex-wrap gap-1.5">
        {Array.from({ length: 7 }, (_, i) => (
          <button
            key={i}
            type="button"
            onClick={() => setDay(i)}
            aria-pressed={day === i}
            className={`rounded-xl px-3 py-1.5 text-sm font-medium transition ${
              day === i
                ? 'bg-brand-600 text-white shadow-chip'
                : 'bg-white text-stone-600 hover:bg-brand-100'
            }`}
          >
            {dayLabel(i)}
          </button>
        ))}
      </div>
      <div className="flex flex-wrap gap-1.5">
        {MEAL_TYPES.filter((mt) => recipe.meal_types.includes(mt)).map((mt) => (
          <button
            key={mt}
            type="button"
            onClick={() => setMealType(mt)}
            aria-pressed={mealType === mt}
            className={`rounded-xl px-3 py-1.5 text-sm font-medium transition ${
              mealType === mt
                ? 'bg-brand-600 text-white shadow-chip'
                : 'bg-white text-stone-600 hover:bg-brand-100'
            }`}
          >
            {mealTypeLabel(mt)}
          </button>
        ))}
      </div>
      <button
        type="button"
        disabled={addItem.isPending}
        onClick={() =>
          addItem.mutate(
            {
              day_index: day,
              meal_type: mealType,
              recipe_id: recipe.id,
              servings: recipe.servings || 2,
            },
            {
              onSuccess: () => {
                toast(t('recipe.addedToMenu'), 'success')
                setOpen(false)
              },
              onError: () => toast(t('error.saveFailed'), 'error'),
            },
          )
        }
        className="inline-flex items-center gap-2 rounded-2xl bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white shadow-chip transition hover:bg-brand-700 active:scale-95 disabled:opacity-60"
      >
        {addItem.isPending ? <Spinner className="size-4 border-2" /> : <IconCheck className="size-4" />}
        {t('common.add')}
      </button>
    </div>
  )
}

export function RecipeDetailBody({ recipe }: { recipe: Recipe }) {
  const { toast } = useToast()
  const user = useAuth((s) => s.user)
  const favorite = useFavorite()
  const [servings, setServings] = useState(recipe.servings || 2)
  // Favorite state: server hint (favorited / favorited_at) with local override.
  const [favOverride, setFavOverride] = useState<boolean | null>(null)
  const isFav = favOverride ?? recipe.favorited ?? recipe.favorited_at != null

  const base = recipe.servings || 1
  const minutes = totalMinutes(recipe)

  const toggleFavorite = () => {
    const next = !isFav
    setFavOverride(next)
    favorite.mutate(
      { recipeId: recipe.id, favorite: next },
      {
        onSuccess: () =>
          toast(next ? t('recipe.favoritedToast') : t('recipe.unfavoritedToast'), 'success'),
        onError: () => {
          setFavOverride(!next)
          toast(t('error.saveFailed'), 'error')
        },
      },
    )
  }

  return (
    <div className="pb-6">
      <div className="relative h-60 w-full sm:h-72">
        <RecipeImage src={recipe.image_url} alt={recipe.title} className="size-full" />
      </div>

      <div className="space-y-6 px-5 pt-4 sm:px-7">
        <div>
          {recipe.category && (
            <span className="mb-1.5 inline-block rounded-full bg-brand-100 px-3 py-1 text-xs font-semibold text-brand-800">
              {recipe.category}
            </span>
          )}
          <h2 className="text-2xl font-bold leading-tight text-stone-900">{recipe.title}</h2>
          {recipe.description && (
            <p className="mt-2 text-sm leading-relaxed text-stone-600">{recipe.description}</p>
          )}
        </div>

        <div className="flex flex-wrap gap-2">
          {minutes !== null && (
            <MetaChip icon={<IconClock className="size-4" />}>
              {t('swipe.cookTime', { minutes })}
            </MetaChip>
          )}
          {recipe.kcal_per_serving != null && (
            <MetaChip icon={<IconFlame className="size-4" />}>
              {t('common.approx')} {formatNumber(recipe.kcal_per_serving)} {t('common.kcal')}{' '}
              {t('common.perServing')}
            </MetaChip>
          )}
          {recipe.cost_per_serving != null && (
            <MetaChip icon={<IconWallet className="size-4" />}>
              {t('common.approx')} {formatNumber(recipe.cost_per_serving)} {t('common.rubles')}{' '}
              {t('common.perServing')}
            </MetaChip>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <button
            type="button"
            onClick={toggleFavorite}
            aria-pressed={isFav}
            className={`inline-flex items-center gap-2 rounded-2xl px-4 py-2.5 text-sm font-semibold shadow-chip transition active:scale-95 ${
              isFav
                ? 'bg-amber-400 text-amber-950 hover:bg-amber-300'
                : 'bg-brand-600 text-white hover:bg-brand-700'
            }`}
          >
            {isFav ? <IconStarFilled className="size-4" /> : <IconStar className="size-4" />}
            {isFav ? t('recipe.favoriteRemove') : t('recipe.favoriteAdd')}
          </button>
          {user && <AddToMenuPanel recipe={recipe} />}
        </div>

        <section>
          <div className="mb-3 flex items-center justify-between">
            <h3 className="text-lg font-bold text-stone-900">{t('recipe.ingredients')}</h3>
            <div className="flex items-center gap-2 text-sm text-stone-600">
              <span>{t('recipe.servings')}</span>
              <div className="flex items-center gap-1 rounded-full bg-cream-dark p-1">
                <button
                  type="button"
                  aria-label="−"
                  onClick={() => setServings((s) => Math.max(1, s - 1))}
                  className="flex size-7 items-center justify-center rounded-full bg-white text-stone-600 shadow-chip transition hover:text-brand-700 active:scale-90"
                >
                  <IconMinus className="size-4" />
                </button>
                <span className="w-8 text-center font-semibold tabular-nums">{servings}</span>
                <button
                  type="button"
                  aria-label="+"
                  onClick={() => setServings((s) => Math.min(20, s + 1))}
                  className="flex size-7 items-center justify-center rounded-full bg-white text-stone-600 shadow-chip transition hover:text-brand-700 active:scale-90"
                >
                  <IconPlus className="size-4" />
                </button>
              </div>
            </div>
          </div>
          <ul className="divide-y divide-stone-100 overflow-hidden rounded-2xl border border-stone-100 bg-white">
            {recipe.ingredients.map((ing, i) => {
              const qty = formatIngredient(ing, base, servings)
              return (
                <li key={`${ing.ingredient_id ?? ing.name}-${i}`} className="flex items-baseline justify-between gap-3 px-4 py-2.5 text-sm">
                  <span className="text-stone-800">{ing.name}</span>
                  <span className="shrink-0 text-right font-medium text-stone-500">
                    {qty}
                    {ing.note ? ` · ${ing.note}` : ''}
                  </span>
                </li>
              )
            })}
            {recipe.ingredients.length === 0 && (
              <li className="px-4 py-3 text-sm text-stone-400">{t('common.emptyGeneric')}</li>
            )}
          </ul>
          <p className="mt-1.5 text-xs text-stone-400">
            {servings} {servingsLabel(servings)}
          </p>
        </section>

        <section>
          <h3 className="mb-3 text-lg font-bold text-stone-900">{t('recipe.instructions')}</h3>
          <ol className="space-y-3">
            {recipe.instructions.map((step, i) => (
              <li key={i} className="flex gap-3">
                <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-brand-600 text-xs font-bold text-white">
                  {i + 1}
                </span>
                <p className="pt-0.5 text-sm leading-relaxed text-stone-700">{step}</p>
              </li>
            ))}
          </ol>
        </section>
      </div>
    </div>
  )
}

export function RecipeDetailLoader({ recipeId }: { recipeId: string }) {
  const { data: recipe, isLoading, isError, refetch } = useRecipe(recipeId)
  if (isLoading) {
    return (
      <div className="space-y-4 p-6">
        <Skeleton className="h-52 w-full" />
        <Skeleton className="h-6 w-2/3" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-5/6" />
        <Skeleton className="h-32 w-full" />
      </div>
    )
  }
  if (isError || !recipe) {
    return (
      <ErrorState
        title={t('recipe.loadingError')}
        error={isError ? undefined : new Error('empty')}
        onRetry={() => void refetch()}
      />
    )
  }
  // key resets servings selector / favorite override when navigating between recipes.
  return <RecipeDetailBody key={recipe.id} recipe={recipe} />
}
