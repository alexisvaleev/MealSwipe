import type { ReactNode } from 'react'
import { t } from '../i18n/i18n'
import { ApiError } from '../lib/api'

export function Spinner({ className = '' }: { className?: string }) {
  return (
    <span
      role="status"
      aria-label={t('common.loading')}
      className={`inline-block size-6 animate-spin rounded-full border-[3px] border-brand-200 border-t-brand-600 ${className}`}
    />
  )
}

export function EmptyState({
  icon,
  title,
  text,
  action,
}: {
  icon?: ReactNode
  title: string
  text?: string
  action?: ReactNode
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 px-6 py-16 text-center">
      {icon && (
        <div className="flex size-16 items-center justify-center rounded-3xl bg-brand-100 text-brand-600">
          {icon}
        </div>
      )}
      <h2 className="text-lg font-semibold text-stone-800">{title}</h2>
      {text && <p className="max-w-sm text-sm text-stone-500">{text}</p>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  )
}

export function ErrorState({
  error,
  onRetry,
  title,
}: {
  error?: unknown
  onRetry?: () => void
  title?: string
}) {
  const isNetwork = error instanceof ApiError && error.code === 'network_error'
  const message =
    error instanceof ApiError && error.status !== 0 && error.code !== 'network_error'
      ? error.message
      : isNetwork || !(error instanceof ApiError)
        ? t('common.networkError')
        : t('error.loadFailed')
  return (
    <div className="flex flex-col items-center justify-center gap-3 px-6 py-16 text-center">
      <div className="flex size-16 items-center justify-center rounded-3xl bg-rose-100 text-2xl text-rose-500">
        !
      </div>
      <h2 className="text-lg font-semibold text-stone-800">
        {title ?? t('error.loadFailed')}
      </h2>
      <p className="max-w-sm text-sm text-stone-500">{message}</p>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="mt-2 rounded-2xl bg-brand-600 px-5 py-2.5 text-sm font-semibold text-white shadow-chip transition hover:bg-brand-700 active:scale-95"
        >
          {t('common.retry')}
        </button>
      )}
    </div>
  )
}

export function Skeleton({ className = '' }: { className?: string }) {
  return <div className={`animate-pulse rounded-2xl bg-stone-200/80 ${className}`} />
}

/** Card-grid skeleton used by favorites/menu pages. */
export function CardGridSkeleton({ count = 6 }: { count?: number }) {
  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
      {Array.from({ length: count }, (_, i) => (
        <div key={i} className="overflow-hidden rounded-3xl bg-white shadow-card">
          <Skeleton className="aspect-[4/3] rounded-none" />
          <div className="space-y-2 p-3">
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-3 w-1/2" />
          </div>
        </div>
      ))}
    </div>
  )
}
