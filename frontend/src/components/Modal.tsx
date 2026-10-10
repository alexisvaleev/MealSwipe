import { useEffect, useRef, type ReactNode } from 'react'
import { motion } from 'framer-motion'
import { t } from '../i18n/i18n'
import { IconX } from './Icons'

interface Props {
  open: boolean
  onClose: () => void
  children: ReactNode
  /** Accessible label for the dialog. */
  ariaLabel?: string
  /** Max width utility class. */
  wide?: boolean
}

/** Accessible modal: backdrop click + Esc close, scroll-lock, bottom-sheet on mobile. */
export function Modal({ open, onClose, children, ariaLabel, wide }: Props) {
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    panelRef.current?.focus()
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = prev
    }
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center sm:p-6">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="absolute inset-0 bg-stone-900/50 backdrop-blur-sm"
        onClick={onClose}
        aria-hidden
      />
      <motion.div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label={ariaLabel}
        tabIndex={-1}
        initial={{ opacity: 0, y: 40, scale: 0.98 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ type: 'spring', stiffness: 380, damping: 34 }}
        className={`relative z-10 max-h-[92dvh] w-full overflow-y-auto rounded-t-3xl bg-white shadow-soft outline-none sm:rounded-3xl ${
          wide ? 'sm:max-w-3xl' : 'sm:max-w-lg'
        }`}
      >
        <button
          type="button"
          onClick={onClose}
          aria-label={t('common.close')}
          className="absolute right-3 top-3 z-20 flex size-10 items-center justify-center rounded-full bg-white/90 text-stone-600 shadow-chip backdrop-blur transition hover:bg-white hover:text-stone-900"
        >
          <IconX className="size-5" />
        </button>
        {children}
      </motion.div>
    </div>
  )
}
