import { forwardRef, useEffect } from 'react'
import { animate, motion, useMotionValue, useTransform, type PanInfo } from 'framer-motion'
import { t } from '../i18n/i18n'
import { formatNumber, ingredientCount, totalMinutes } from '../lib/format'
import type { RecipeSummary, SwipeDirection } from '../lib/types'
import { RecipeImage } from './RecipeImage'
import { IconClock, IconFlame, IconWallet } from './Icons'

const SWIPE_THRESHOLD = 120
const VELOCITY_THRESHOLD = 500

interface Props {
  recipe: RecipeSummary
  onSwipe: (direction: SwipeDirection) => void
  onOpenDetails: () => void
  /** Cards underneath render smaller/deeper — they don't drag. */
  interactive?: boolean
  /** When set, the card animates off-screen in this direction (-1 left, +1 right). */
  exitDirection?: -1 | 1
}

/**
 * One swipeable recipe card (SPEC §9): drag horizontally past ~120px or with
 * enough velocity to like (right) / dislike (left). Rotation + like/dislike
 * stamps are driven by the drag position.
 */
export const SwipeCard = forwardRef<HTMLDivElement, Props>(function SwipeCard(
  { recipe, onSwipe, onOpenDetails, interactive = true, exitDirection },
  ref,
) {
  const x = useMotionValue(0)
  const rotate = useTransform(x, [-300, 0, 300], [-14, 0, 14])
  const opacity = useTransform(x, [-500, -250, 0, 250, 500], [0, 1, 1, 1, 0])
  const likeOpacity = useTransform(x, [30, SWIPE_THRESHOLD], [0, 1])
  const dislikeOpacity = useTransform(x, [-SWIPE_THRESHOLD, -30], [1, 0])

  // Button/keyboard swipes: fly off-screen in the given direction.
  useEffect(() => {
    if (exitDirection) {
      void animate(x, exitDirection * 640, { duration: 0.24, ease: 'easeIn' })
    }
  }, [exitDirection, x])

  const handleDragEnd = (_: unknown, info: PanInfo) => {
    const { offset, velocity } = info
    if (offset.x > SWIPE_THRESHOLD || velocity.x > VELOCITY_THRESHOLD) {
      onSwipe('like')
    } else if (offset.x < -SWIPE_THRESHOLD || velocity.x < -VELOCITY_THRESHOLD) {
      onSwipe('dislike')
    }
  }

  const minutes = totalMinutes(recipe)
  const count = ingredientCount(recipe)

  return (
    <motion.div
      ref={ref}
      data-testid="swipe-card"
      drag={interactive && !exitDirection ? 'x' : false}
      dragConstraints={{ left: 0, right: 0 }}
      dragElastic={0.9}
      onDragEnd={handleDragEnd}
      style={interactive ? { x, rotate, opacity } : undefined}
      whileDrag={{ scale: 1.02, cursor: 'grabbing' }}
      className="absolute inset-0 cursor-grab touch-manipulation select-none"
      role="group"
      aria-label={recipe.title}
    >
      <div className="relative flex h-full flex-col overflow-hidden rounded-3xl bg-white shadow-soft">
        {/* Like / dislike stamps */}
        {interactive && (
          <>
            <motion.span
              aria-hidden
              style={{ opacity: likeOpacity }}
              className="absolute left-4 top-6 z-20 -rotate-12 rounded-2xl border-4 border-emerald-500 px-3 py-1 text-2xl font-extrabold tracking-wide text-emerald-500"
            >
              {t('swipe.liked')}
            </motion.span>
            <motion.span
              aria-hidden
              style={{ opacity: dislikeOpacity }}
              className="absolute right-4 top-6 z-20 rotate-12 rounded-2xl border-4 border-rose-500 px-3 py-1 text-2xl font-extrabold tracking-wide text-rose-500"
            >
              {t('swipe.disliked')}
            </motion.span>
          </>
        )}

        <button
          type="button"
          onClick={onOpenDetails}
          aria-label={`${t('swipe.details')}: ${recipe.title}`}
          className="relative block h-[58%] w-full overflow-hidden text-left"
          tabIndex={interactive ? 0 : -1}
        >
          <RecipeImage
            src={recipe.image_url}
            alt={recipe.title}
            className="size-full"
          />
          <div
            aria-hidden
            className="pointer-events-none absolute inset-x-0 bottom-0 h-24 bg-gradient-to-t from-black/60 to-transparent"
          />
          {recipe.category && (
            <span className="absolute left-3 top-3 rounded-full bg-white/90 px-3 py-1 text-xs font-semibold text-stone-700 shadow-chip backdrop-blur">
              {recipe.category}
            </span>
          )}
        </button>

        <div className="flex min-h-0 flex-1 flex-col justify-between gap-2 p-4">
          <div>
            <h2 className="line-clamp-2 text-xl font-bold leading-snug text-stone-900">
              {recipe.title}
            </h2>
            {recipe.description && (
              <p className="mt-1 line-clamp-2 text-sm text-stone-500">
                {recipe.description}
              </p>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-sm text-stone-600">
            {minutes !== null && (
              <span className="inline-flex items-center gap-1.5">
                <IconClock className="size-4 text-brand-600" />
                {t('swipe.cookTime', { minutes })}
              </span>
            )}
            {recipe.kcal_per_serving != null && (
              <span className="inline-flex items-center gap-1.5">
                <IconFlame className="size-4 text-brand-600" />
                {t('swipe.kcalPerServing', { kcal: formatNumber(recipe.kcal_per_serving) })}
              </span>
            )}
            {recipe.cost_per_serving != null && (
              <span className="inline-flex items-center gap-1.5">
                <IconWallet className="size-4 text-brand-600" />
                {t('swipe.costPerServing', { cost: formatNumber(recipe.cost_per_serving) })}
              </span>
            )}
            {count !== null && (
              <span className="text-stone-500">
                {t('common.ingredientsCount', { count })}
              </span>
            )}
          </div>
        </div>
      </div>
    </motion.div>
  )
})
