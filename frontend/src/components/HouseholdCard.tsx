import { useState, type FormEvent } from 'react'
import { plural, t } from '../i18n/i18n'
import { formatDateShort } from '../lib/format'
import {
  useHousehold,
  useJoinHousehold,
  useLeaveHousehold,
  useRenameHousehold,
  useRotateInvite,
} from '../lib/queries'
import { useAuth } from '../stores/auth'
import { useToast } from './Toast'
import { ErrorState, Skeleton, Spinner } from './States'
import { Modal } from './Modal'
import { IconCheck, IconCopy, IconUsers } from './Icons'

const inputCls =
  'w-full rounded-2xl border-0 bg-cream-dark px-4 py-3 text-sm text-stone-900 placeholder:text-stone-400 focus:outline-none focus:ring-2 focus:ring-brand-500'
const btnCls =
  'inline-flex items-center justify-center gap-2 rounded-2xl px-4 py-2.5 text-sm font-semibold transition active:scale-95 disabled:opacity-60'

/**
 * Settings section for the shared kitchen: members, invite code,
 * join-by-code and leave. All mutations invalidate the shared caches.
 */
export function HouseholdCard() {
  const { toast } = useToast()
  const user = useAuth((s) => s.user)
  const { data: h, isLoading, isError, error, refetch } = useHousehold()
  const join = useJoinHousehold()
  const leave = useLeaveHousehold()
  const rename = useRenameHousehold()
  const rotate = useRotateInvite()

  const [code, setCode] = useState('')
  const [name, setName] = useState<string | null>(null)
  const [confirmLeave, setConfirmLeave] = useState(false)
  const [copied, setCopied] = useState(false)

  if (isLoading) {
    return (
      <div className="mt-3 space-y-3">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-12 w-2/3" />
      </div>
    )
  }
  if (isError || !h) {
    return <ErrorState error={error} onRetry={() => void refetch()} />
  }

  const nameDraft = name ?? h.name
  const solo = h.members.length <= 1

  const copyCode = async () => {
    try {
      await navigator.clipboard.writeText(h.invite_code)
      setCopied(true)
      toast(t('household.copied'), 'success')
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      toast(t('error.generic'), 'error')
    }
  }

  const submitJoin = (e: FormEvent) => {
    e.preventDefault()
    const c = code.trim().toUpperCase()
    if (!c) return
    join.mutate(c, {
      onSuccess: (res) => {
        toast(t('household.joined', { name: res.household.name }), 'success')
        setCode('')
      },
      onError: () => toast(t('household.joinError'), 'error'),
    })
  }

  const submitRename = () => {
    const n = nameDraft.trim()
    if (!n || n === h.name) return
    rename.mutate(n, {
      onSuccess: () => {
        setName(null)
        toast(t('household.renamed'), 'success')
      },
      onError: () => toast(t('error.saveFailed'), 'error'),
    })
  }

  const submitLeave = () => {
    leave.mutate(undefined, {
      onSuccess: () => {
        setConfirmLeave(false)
        toast(t('household.left'), 'info')
      },
      onError: () => toast(t('error.generic'), 'error'),
    })
  }

  return (
    <div className="mt-3 space-y-5">
      <p className="text-sm leading-relaxed text-stone-500">{t('household.subtitle')}</p>

      {/* Name + members */}
      <div>
        <label htmlFor="household-name" className="mb-1.5 block text-sm font-medium text-stone-700">
          {t('household.nameLabel')}
        </label>
        <div className="flex gap-2">
          <input
            id="household-name"
            value={nameDraft}
            onChange={(e) => setName(e.target.value)}
            maxLength={60}
            className={inputCls}
          />
          <button
            type="button"
            onClick={submitRename}
            disabled={rename.isPending || !nameDraft.trim() || nameDraft.trim() === h.name}
            className={`${btnCls} shrink-0 bg-cream-dark text-stone-600 hover:bg-stone-200`}
          >
            {t('common.save')}
          </button>
        </div>
      </div>

      <div>
        <h3 className="flex items-center gap-2 text-sm font-medium text-stone-700">
          <IconUsers className="size-4 text-brand-600" />
          {t('household.members')} ·{' '}
          {plural(h.members.length, {
            one: t('household.membersCount_one', { count: h.members.length }),
            few: t('household.membersCount_few', { count: h.members.length }),
            many: t('household.membersCount_many', { count: h.members.length }),
          })}
        </h3>
        <ul className="mt-2 space-y-1.5">
          {h.members.map((m) => (
            <li
              key={m.id}
              className="flex items-center justify-between rounded-2xl bg-cream-dark px-4 py-2.5 text-sm"
            >
              <span className="font-medium text-stone-800">
                {m.display_name}
                {m.id === user?.id && (
                  <span className="ml-2 text-xs font-normal text-stone-400">
                    {t('household.you')}
                  </span>
                )}
              </span>
              <span className="text-xs text-stone-400">{formatDateShort(m.joined_at.slice(0, 10))}</span>
            </li>
          ))}
        </ul>
      </div>

      {/* Invite code */}
      <div>
        <h3 className="text-sm font-medium text-stone-700">{t('household.inviteCode')}</h3>
        <div className="mt-2 flex items-center gap-2">
          <code className="flex-1 rounded-2xl bg-stone-800 px-4 py-3 text-center font-mono text-lg font-bold tracking-[0.2em] text-amber-200">
            {h.invite_code}
          </code>
          <button
            type="button"
            onClick={() => void copyCode()}
            aria-label={t('household.copy')}
            className={`${btnCls} bg-brand-600 text-white hover:bg-brand-700`}
          >
            {copied ? <IconCheck className="size-4" /> : <IconCopy className="size-4" />}
            {t('household.copy')}
          </button>
          <button
            type="button"
            onClick={() =>
              rotate.mutate(undefined, {
                onSuccess: () => toast(t('household.rotated'), 'info'),
                onError: () => toast(t('error.generic'), 'error'),
              })
            }
            disabled={rotate.isPending}
            className={`${btnCls} bg-cream-dark text-stone-600 hover:bg-stone-200`}
          >
            {t('household.rotate')}
          </button>
        </div>
        <p className="mt-1.5 text-xs text-stone-400">{t('household.inviteHint')}</p>
      </div>

      {/* Join another kitchen / leave */}
      <div>
        <h3 className="text-sm font-medium text-stone-700">{t('household.joinTitle')}</h3>
        <form onSubmit={submitJoin} className="mt-2 flex gap-2">
          <input
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            placeholder={t('household.joinPlaceholder')}
            maxLength={16}
            className={`${inputCls} font-mono uppercase tracking-widest`}
          />
          <button
            type="submit"
            disabled={join.isPending || code.trim().length < 4}
            className={`${btnCls} shrink-0 bg-brand-600 text-white hover:bg-brand-700`}
          >
            {join.isPending && <Spinner className="size-4 border-2 border-white/40 border-t-white" />}
            {join.isPending ? t('household.joining') : t('household.join')}
          </button>
        </form>
      </div>

      {!solo && (
        <button
          type="button"
          onClick={() => setConfirmLeave(true)}
          className={`${btnCls} w-full bg-rose-50 text-rose-700 hover:bg-rose-100`}
        >
          {t('household.leave')}
        </button>
      )}

      <Modal open={confirmLeave} onClose={() => setConfirmLeave(false)} ariaLabel={t('household.leaveConfirmTitle')}>
        <div className="p-6 sm:p-8">
          <h2 className="text-xl font-bold text-stone-900">{t('household.leaveConfirmTitle')}</h2>
          <p className="mt-2 text-sm leading-relaxed text-stone-600">{t('household.leaveConfirmText')}</p>
          <div className="mt-6 flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setConfirmLeave(false)}
              className={`${btnCls} bg-cream-dark text-stone-600 hover:bg-stone-200`}
            >
              {t('common.cancel')}
            </button>
            <button
              type="button"
              onClick={submitLeave}
              disabled={leave.isPending}
              className={`${btnCls} bg-rose-600 text-white hover:bg-rose-700`}
            >
              {leave.isPending && <Spinner className="size-4 border-2 border-white/40 border-t-white" />}
              {t('household.leave')}
            </button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
