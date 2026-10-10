import { useState } from 'react'
import { IconBowl } from './Icons'

interface Props {
  src?: string | null
  alt: string
  className?: string
  /** Extra classes on the fallback container (defaults to a warm gradient). */
  fallbackClassName?: string
}

/**
 * Recipe photo with graceful degradation: null URL or failed load shows a
 * warm gradient placeholder with a bowl icon (image_url may be null per SPEC).
 */
export function RecipeImage({ src, alt, className = '', fallbackClassName = '' }: Props) {
  const [failed, setFailed] = useState(false)

  if (!src || failed) {
    return (
      <div
        role="img"
        aria-label={alt}
        data-testid="recipe-image-fallback"
        className={`flex items-center justify-center bg-gradient-to-br from-brand-200 via-brand-300 to-brand-500 text-5xl text-white/90 ${fallbackClassName} ${className}`}
      >
        <IconBowl className="size-1/3 max-w-16 drop-shadow" strokeWidth={1.5} />
      </div>
    )
  }

  return (
    <img
      src={src}
      alt={alt}
      loading="lazy"
      onError={() => setFailed(true)}
      className={`object-cover ${className}`}
    />
  )
}
