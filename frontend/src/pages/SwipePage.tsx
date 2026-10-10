import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router'
import { AnimatePresence, motion } from 'framer-motion'
import { t } from '../i18n/i18n'
import { ApiError } from '../lib/api'
import { useDiscover, useFavorite, useSwipe, useUndo } from '../lib/queries'
import type { RecipeSummary, SwipeDirection } from '../lib/types'
import { SwipeCard } from '../components/SwipeCard'
import { RecipeModal } from '../components/RecipeModal'
import { ErrorState, Skeleton } from '../components/States'
import { useToast } from '../components/Toast'
import { IconHeart, IconInfo, IconStar, IconUndo, IconX } from '../components/Icons'

const BATCH_SIZE = 15
const FLY_OUT_MS = 230

export function SwipePage() {
  const { toast } = useToast()
  const navigate = useNavigate()
  const {
    data: discoverData,
    isLoading: discoverLoading,
    isError: discoverError,
    error: discoverErr,
    isFetching: discoverFetching,
    refetch: discoverRefetch,
  } = useDiscover(BATCH_SIZE)
  const swipe = useSwipe()
  const undo = useUndo()
  const favorite = useFavorite()

  const [deck, setDeck] = useState<RecipeSummary[]>([])
  const [exiting, setExiting] = useState<{ id: string; dir: -1 | 1 } | null>(null)
  const [history, setHistory] = useState<RecipeSummary[]>([])
  const [detailsId, setDetailsId] = useState<string | null>(null)
  const seenIds = useRef(new Set<string>())
  // Server exhausted its unseen recipes → stop prefetching until "restart".
  const exhaustedRef = useRef(false)

  // Append newly fetched recipes to the deck (dedupe by id).
  useEffect(() => {
    if (!discoverData) return
    const fresh = discoverData.filter((r) => !seenIds.current.has(r.id))
    fresh.forEach((r) => seenIds.current.add(r.id))
    if (fresh.length > 0) {
      setDeck((prev) => [...prev, ...fresh])
    } else {
      exhaustedRef.current = true
    }
  }, [discoverData])

  // Prefetch the next batch when fewer than 3 cards remain.
  useEffect(() => {
    if (deck.length > 0 && deck.length < 3 && !discoverFetching && !exhaustedRef.current) {
      void discoverRefetch()
    }
  }, [deck.length, discoverFetching, discoverRefetch])

  const top = deck[0]

  const doSwipe = useCallback(
    (direction: SwipeDirection) => {
      if (!top || exiting) return
      const dir = direction === 'like' ? 1 : -1
      setExiting({ id: top.id, dir })
      window.setTimeout(() => {
        setDeck((d) => d.slice(1))
        setExiting(null)
        setHistory((h) => [...h, top])
      }, FLY_OUT_MS)
      swipe.mutate(
        { recipeId: top.id, direction },
        { onError: () => toast(t('swipe.swipeError'), 'error') },
      )
    },
    [top, exiting, swipe, toast],
  )

  const doFavorite = useCallback(() => {
    if (!top) return
    favorite.mutate(
      { recipeId: top.id, favorite: true },
      {
        onSuccess: () => toast(t('swipe.favorited'), 'success'),
        onError: () => toast(t('error.saveFailed'), 'error'),
      },
    )
  }, [top, favorite, toast])

  const doUndo = useCallback(() => {
    undo.mutate(undefined, {
      onSuccess: () => {
        setHistory((h) => {
          const last = h[h.length - 1]
          if (last) {
            setDeck((d) => (d.some((r) => r.id === last.id) ? d : [last, ...d]))
          }
          return h.slice(0, -1)
        })
        toast(t('swipe.undone'), 'success')
      },
      onError: (err) => {
        if (err instanceof ApiError && err.status === 404) toast(t('swipe.nothingToUndo'))
        else toast(t('error.generic'), 'error')
      },
    })
  }, [undo, toast])

  const openDetails = useCallback(() => {
    if (top) setDetailsId(top.id)
  }, [top])

  // Keyboard controls: ← dislike, → like, ↑ favorite, u undo, i details.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (detailsId) return
      const target = e.target as HTMLElement | null
      if (target && /^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName)) return
      switch (e.key) {
        case 'ArrowLeft':
          e.preventDefault()
          doSwipe('dislike')
          break
        case 'ArrowRight':
          e.preventDefault()
          doSwipe('like')
          break
        case 'ArrowUp':
          e.preventDefault()
          doFavorite()
          break
        case 'u':
        case 'U':
          doUndo()
          break
        case 'i':
        case 'I':
          openDetails()
          break
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [doSwipe, doFavorite, doUndo, openDetails, detailsId])

  const restart = () => {
    exhaustedRef.current = false
    void discoverRefetch()
  }

  // ---------- render ----------

  if (discoverLoading && deck.length === 0) {
    return (
      <div className="mx-auto flex max-w-sm flex-col items-center pt-6">
        <Skeleton className="h-[min(62dvh,520px)] w-full rounded-3xl" />
        <div className="mt-5 flex gap-4">
          {Array.from({ length: 5 }, (_, i) => (
            <Skeleton key={i} className="size-12 rounded-full" />
          ))}
        </div>
      </div>
    )
  }

  if (discoverError && deck.length === 0) {
    return <ErrorState error={discoverErr} onRetry={() => void discoverRefetch()} />
  }

  const deckEmpty = deck.length === 0

  return (
    <div className="mx-auto flex max-w-sm flex-col items-center">
      <p className="mb-4 hidden text-center text-xs text-stone-400 sm:block">{t('swipe.hint')}</p>

      <div className="relative h-[min(62dvh,540px)] w-full">
        <AnimatePresence>
          {deck.slice(0, 3).map((recipe, i) => (
            <motion.div
              key={recipe.id}
              className="absolute inset-0"
              style={{ zIndex: 10 - i }}
              initial={{ scale: 0.94, y: i * 12 }}
              animate={{ scale: 1 - i * 0.04, y: i * 12 }}
              transition={{ type: 'spring', stiffness: 320, damping: 28 }}
            >
              <SwipeCard
                recipe={recipe}
                interactive={i === 0 && !exiting}
                exitDirection={exiting?.id === recipe.id ? exiting.dir : undefined}
                onSwipe={doSwipe}
                onOpenDetails={openDetails}
              />
            </motion.div>
          ))}
        </AnimatePresence>

        {deckEmpty && !discoverFetching && (
          <div className="flex h-full flex-col items-center justify-center gap-4 rounded-3xl border-2 border-dashed border-stone-300 bg-white/60 px-6 text-center">
            <span className="text-4xl">🍽</span>
            <h2 className="text-xl font-bold text-stone-800">{t('swipe.endTitle')}</h2>
            <p className="max-w-xs text-sm text-stone-500">{t('swipe.endText')}</p>
            <div className="flex flex-wrap justify-center gap-2.5">
              <button
                type="button"
                onClick={() => navigate('/settings')}
                className="rounded-2xl border-2 border-brand-600 px-5 py-2.5 text-sm font-semibold text-brand-700 transition hover:bg-brand-50"
              >
                {t('swipe.changeFilters')}
              </button>
              <button
                type="button"
                onClick={restart}
                className="rounded-2xl bg-brand-600 px-5 py-2.5 text-sm font-semibold text-white shadow-chip transition hover:bg-brand-700 active:scale-95"
              >
                {t('swipe.restart')}
              </button>
            </div>
          </div>
        )}

        {deckEmpty && discoverFetching && (
          <Skeleton className="h-full w-full rounded-3xl" />
        )}
      </div>

      {/* Action buttons */}
      <div className="mt-5 flex items-center gap-3">
        <ActionButton
          label={t('swipe.undo')}
          onClick={doUndo}
          disabled={history.length === 0 || undo.isPending}
          className="size-11 bg-white text-stone-500 hover:text-brand-700"
        >
          <IconUndo className="size-5" />
        </ActionButton>
        <ActionButton
          label={t('swipe.dislike')}
          onClick={() => doSwipe('dislike')}
          disabled={!top || Boolean(exiting)}
          className="size-14 bg-white text-rose-500 hover:bg-rose-50"
        >
          <IconX className="size-7" />
        </ActionButton>
        <ActionButton
          label={t('swipe.favorite')}
          onClick={doFavorite}
          disabled={!top || favorite.isPending}
          className="size-12 bg-white text-amber-500 hover:bg-amber-50"
        >
          <IconStar className="size-6" />
        </ActionButton>
        <ActionButton
          label={t('swipe.like')}
          onClick={() => doSwipe('like')}
          disabled={!top || Boolean(exiting)}
          className="size-14 bg-brand-600 text-white shadow-soft hover:bg-brand-700"
        >
          <IconHeart className="size-7" />
        </ActionButton>
        <ActionButton
          label={t('swipe.details')}
          onClick={openDetails}
          disabled={!top}
          className="size-11 bg-white text-stone-500 hover:text-brand-700"
        >
          <IconInfo className="size-5" />
        </ActionButton>
      </div>

      <RecipeModal recipeId={detailsId} onClose={() => setDetailsId(null)} />
    </div>
  )
}

function ActionButton({
  label,
  onClick,
  disabled,
  className,
  children,
}: {
  label: string
  onClick: () => void
  disabled?: boolean
  className: string
  children: ReactNode
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      onClick={onClick}
      disabled={disabled}
      className={`flex items-center justify-center rounded-full shadow-card transition hover:scale-105 active:scale-95 disabled:opacity-40 disabled:hover:scale-100 ${className}`}
    >
      {children}
    </button>
  )
}
