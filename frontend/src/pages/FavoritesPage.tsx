import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router'
import { t, type TranslationKey } from '../i18n/i18n'
import { formatNumber, totalMinutes } from '../lib/format'
import { useFavorite, useFavorites } from '../lib/queries'
import type { RecipeSummary } from '../lib/types'
import { CardGridSkeleton, EmptyState, ErrorState } from '../components/States'
import { RecipeImage } from '../components/RecipeImage'
import { RecipeModal } from '../components/RecipeModal'
import { useToast } from '../components/Toast'
import {
  IconCalendar,
  IconClock,
  IconSearch,
  IconStarFilled,
} from '../components/Icons'

type SortKey = 'recent' | 'title' | 'time'

const SORT_OPTIONS: { value: SortKey; labelKey: TranslationKey }[] = [
  { value: 'recent', labelKey: 'favorites.sortRecent' },
  { value: 'title', labelKey: 'favorites.sortTitle' },
  { value: 'time', labelKey: 'favorites.sortTime' },
]

export function FavoritesPage() {
  const { toast } = useToast()
  const [search, setSearch] = useState('')
  const [debouncedQ, setDebouncedQ] = useState('')
  const [category, setCategory] = useState('')
  const [sort, setSort] = useState<SortKey>('recent')
  const [detailsId, setDetailsId] = useState<string | null>(null)

  useEffect(() => {
    const id = window.setTimeout(() => setDebouncedQ(search.trim()), 300)
    return () => window.clearTimeout(id)
  }, [search])

  const { data, isLoading, isError, error, refetch } = useFavorites({
    q: debouncedQ || undefined,
    category: category || undefined,
  })
  const favorite = useFavorite()

  const categories = useMemo(() => {
    const set = new Set<string>()
    for (const r of data ?? []) if (r.category) set.add(r.category)
    return [...set].sort((a, b) => a.localeCompare(b, 'ru'))
  }, [data])

  const recipes = useMemo(() => {
    const list = [...(data ?? [])]
    switch (sort) {
      case 'title':
        list.sort((a, b) => a.title.localeCompare(b.title, 'ru'))
        break
      case 'time':
        list.sort((a, b) => (totalMinutes(a) ?? 9999) - (totalMinutes(b) ?? 9999))
        break
      default:
        list.sort((a, b) => (b.favorited_at ?? '').localeCompare(a.favorited_at ?? ''))
    }
    return list
  }, [data, sort])

  const unfavorite = (recipe: RecipeSummary) => {
    favorite.mutate(
      { recipeId: recipe.id, favorite: false },
      {
        onSuccess: () => toast(t('recipe.unfavoritedToast'), 'info'),
        onError: () => toast(t('error.saveFailed'), 'error'),
      },
    )
  }

  return (
    <div>
      <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold text-stone-900">
          {t('favorites.title')}
          {data && data.length > 0 && (
            <span className="ml-2 text-base font-normal text-stone-400">
              {t('favorites.count', { count: recipes.length })}
            </span>
          )}
        </h1>
        <label className="flex items-center gap-2 text-sm text-stone-500">
          <select
            value={sort}
            onChange={(e) => setSort(e.target.value as SortKey)}
            className="rounded-xl border-0 bg-white px-3 py-2 text-sm font-medium text-stone-700 shadow-chip focus:outline-none focus:ring-2 focus:ring-brand-500"
          >
            {SORT_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {t(o.labelKey)}
              </option>
            ))}
          </select>
        </label>
      </div>

      {/* Search + category chips */}
      <div className="mb-4 space-y-3">
        <div className="relative">
          <IconSearch className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-stone-400" />
          <input
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t('favorites.searchPlaceholder')}
            aria-label={t('common.search')}
            className="w-full rounded-2xl border-0 bg-white py-3 pl-10 pr-4 text-sm shadow-chip placeholder:text-stone-400 focus:outline-none focus:ring-2 focus:ring-brand-500"
          />
        </div>
        {categories.length > 0 && (
          <div className="flex flex-wrap gap-2">
            <FilterChip active={category === ''} onClick={() => setCategory('')}>
              {t('common.all')}
            </FilterChip>
            {categories.map((c) => (
              <FilterChip key={c} active={category === c} onClick={() => setCategory(c)}>
                {c}
              </FilterChip>
            ))}
          </div>
        )}
      </div>

      {isLoading && <CardGridSkeleton />}

      {isError && <ErrorState error={error} onRetry={() => void refetch()} />}

      {!isLoading && !isError && recipes.length === 0 && (
        <EmptyState
          icon={<IconStarFilled className="size-8" />}
          title={t('favorites.empty')}
          text={t('favorites.emptyText')}
          action={
            <Link
              to="/swipe"
              className="rounded-2xl bg-brand-600 px-5 py-2.5 text-sm font-semibold text-white shadow-chip transition hover:bg-brand-700"
            >
              {t('favorites.toSwipe')}
            </Link>
          }
        />
      )}

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
        {recipes.map((recipe) => {
          const minutes = totalMinutes(recipe)
          return (
            <article
              key={recipe.id}
              className="group overflow-hidden rounded-3xl bg-white shadow-card transition hover:shadow-soft"
            >
              <div className="relative">
                <button
                  type="button"
                  onClick={() => setDetailsId(recipe.id)}
                  className="block w-full text-left"
                  aria-label={`${t('swipe.details')}: ${recipe.title}`}
                >
                  <RecipeImage
                    src={recipe.image_url}
                    alt={recipe.title}
                    className="aspect-[4/3] w-full"
                  />
                </button>
                <button
                  type="button"
                  aria-label={`${t('favorites.removeTitle')}: ${recipe.title}`}
                  title={t('favorites.removeTitle')}
                  onClick={() => unfavorite(recipe)}
                  className="absolute right-2.5 top-2.5 flex size-9 items-center justify-center rounded-full bg-white/90 text-amber-500 shadow-chip backdrop-blur transition hover:scale-110"
                >
                  <IconStarFilled className="size-5" />
                </button>
              </div>
              <div className="p-3">
                <h2 className="line-clamp-2 text-sm font-semibold leading-snug text-stone-900">
                  {recipe.title}
                </h2>
                <div className="mt-1.5 flex items-center justify-between text-xs text-stone-500">
                  {minutes !== null ? (
                    <span className="inline-flex items-center gap-1">
                      <IconClock className="size-3.5 text-brand-600" />
                      {t('swipe.cookTime', { minutes })}
                    </span>
                  ) : recipe.kcal_per_serving != null ? (
                    <span>
                      {t('common.approx')}
                      {formatNumber(recipe.kcal_per_serving)} {t('common.kcal')}
                    </span>
                  ) : (
                    <span />
                  )}
                  <button
                    type="button"
                    aria-label={`${t('recipe.toMenu')}: ${recipe.title}`}
                    title={t('recipe.toMenu')}
                    onClick={() => setDetailsId(recipe.id)}
                    className="flex size-8 items-center justify-center rounded-full text-stone-400 transition hover:bg-brand-50 hover:text-brand-700"
                  >
                    <IconCalendar className="size-4" />
                  </button>
                </div>
              </div>
            </article>
          )
        })}
      </div>

      <RecipeModal recipeId={detailsId} onClose={() => setDetailsId(null)} />
    </div>
  )
}

function FilterChip({
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
      className={`rounded-full px-3.5 py-1.5 text-xs font-semibold transition active:scale-95 ${
        active
          ? 'bg-stone-800 text-white'
          : 'bg-white text-stone-500 shadow-chip hover:bg-brand-50 hover:text-brand-700'
      }`}
    >
      {children}
    </button>
  )
}

