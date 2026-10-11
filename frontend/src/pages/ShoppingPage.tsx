import { useMemo, useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import { useQueryClient } from '@tanstack/react-query'
import { t, type TranslationKey } from '../i18n/i18n'
import { formatQuantity, mondayOf } from '../lib/format'
import {
  qk,
  useAddItem,
  useAddPantryItem,
  useDeleteItem,
  useMenu,
  usePatchItem,
  useRegenerateList,
  useShoppingList,
} from '../lib/queries'
import {
  INGREDIENT_CATEGORIES,
  type IngredientCategory,
  type ShoppingList,
  type ShoppingListItem,
} from '../lib/types'
import {
  EmptyState,
  ErrorState,
  Skeleton,
  Spinner,
} from '../components/States'
import { useToast } from '../components/Toast'
import {
  IconCart,
  IconCheck,
  IconPlus,
  IconTrash,
} from '../components/Icons'

const CATEGORY_KEY = (c: string): TranslationKey =>
  (c in RU_CATEGORIES
    ? `shopping.category.${c}`
    : 'shopping.category.other') as TranslationKey

const RU_CATEGORIES: Record<string, true> = Object.fromEntries(
  INGREDIENT_CATEGORIES.map((c) => [c, true]),
)

const UNIT_OPTIONS = ['', 'g', 'kg', 'ml', 'l', 'pcs', 'tbsp', 'tsp', 'cup']

function unitLabel(u: string): string {
  return u ? t(`unit.${u}` as TranslationKey) : '—'
}

export function ShoppingPage() {
  const { toast } = useToast()
  const week = mondayOf(new Date())
  const { data: menu, isLoading: menuLoading, isError: menuError, error: menuErr, refetch } = useMenu(week)
  const { data: list, isLoading: listLoading, isError: listError, error: listErr, refetch: refetchList } =
    useShoppingList(menu?.id)
  const regenerate = useRegenerateList(list?.id)

  const groups = useMemo(() => {
    const byCat = new Map<string, ShoppingListItem[]>()
    const items = [...(list?.items ?? [])].sort(
      (a, b) => a.position - b.position || a.name.localeCompare(b.name, 'ru'),
    )
    for (const item of items) {
      const cat = RU_CATEGORIES[item.category] ? item.category : 'other'
      const arr = byCat.get(cat) ?? []
      arr.push(item)
      byCat.set(cat, arr)
    }
    // Keep a stable, store-like category order.
    return INGREDIENT_CATEGORIES.filter((c) => byCat.has(c)).map(
      (c) => [c, byCat.get(c)!] as const,
    )
  }, [list])

  const total = list?.items.length ?? 0
  const done = list?.items.filter((i) => i.checked).length ?? 0
  const pct = total > 0 ? Math.round((done / total) * 100) : 0

  if (menuLoading) return <ListSkeleton />
  if (menuError) return <ErrorState error={menuErr} onRetry={() => void refetch()} />

  if (!menu) {
    return (
      <EmptyState
        icon={<IconCart className="size-8" />}
        title={t('shopping.noMenu')}
        text={t('shopping.noMenuText')}
        action={
          <Link
            to="/menu"
            className="rounded-2xl bg-brand-600 px-5 py-2.5 text-sm font-semibold text-white shadow-chip transition hover:bg-brand-700"
          >
            {t('shopping.toMenu')}
          </Link>
        }
      />
    )
  }

  return (
    <div className="mx-auto max-w-2xl">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold text-stone-900">{t('shopping.title')}</h1>
        <button
          type="button"
          onClick={() =>
            regenerate.mutate(undefined, {
              onSuccess: () => toast(t('shopping.regenerated'), 'success'),
              onError: () => toast(t('error.generic'), 'error'),
            })
          }
          disabled={regenerate.isPending || !list}
          className="flex items-center gap-2 rounded-2xl bg-white px-4 py-2.5 text-sm font-semibold text-brand-700 shadow-chip transition hover:bg-brand-50 active:scale-95 disabled:opacity-60"
        >
          {regenerate.isPending ? (
            <Spinner className="size-4 border-2" />
          ) : (
            <IconCart className="size-4" />
          )}
          {regenerate.isPending ? t('shopping.regenerating') : t('shopping.regenerate')}
        </button>
      </div>

      {/* Progress */}
      <div className="mb-6">
        <div className="mb-1.5 flex items-center justify-between text-xs font-medium text-stone-500">
          <span>{t('shopping.progress', { done, total })}</span>
          <span>{pct}%</span>
        </div>
        <div
          role="progressbar"
          aria-valuenow={done}
          aria-valuemin={0}
          aria-valuemax={total}
          className="h-2.5 overflow-hidden rounded-full bg-stone-200"
        >
          <div
            className={`h-full rounded-full transition-all duration-300 ${
              pct === 100 ? 'bg-emerald-500' : 'bg-brand-500'
            }`}
            style={{ width: `${pct}%` }}
          />
        </div>
        {total > 0 && done === total && (
          <p className="mt-2 text-sm font-medium text-emerald-600">{t('shopping.allDone')}</p>
        )}
      </div>

      {listLoading && <ListSkeleton />}
      {listError && <ErrorState error={listErr} onRetry={() => void refetchList()} />}

      {list && list.items.length === 0 && (
        <EmptyState
          icon={<IconCart className="size-8" />}
          title={t('shopping.empty')}
          text={t('shopping.emptyText')}
        />
      )}

      {list &&
        groups.map(([cat, items]) => (
          <section key={cat} className="mb-5" aria-label={t(CATEGORY_KEY(cat))}>
            <h2 className="mb-2 px-1 text-sm font-bold uppercase tracking-wide text-stone-500">
              {t(CATEGORY_KEY(cat))}
              <span className="ml-1.5 font-normal text-stone-400">{items.length}</span>
            </h2>
            <ul className="divide-y divide-stone-100 overflow-hidden rounded-2xl bg-white shadow-card">
              {items.map((item) => (
                <ShoppingRow key={item.id} item={item} listId={list.id} />
              ))}
            </ul>
          </section>
        ))}

      {list && <AddItemRow listId={list.id} />}
    </div>
  )
}

function ListSkeleton() {
  return (
    <div className="mx-auto max-w-2xl space-y-4">
      <Skeleton className="h-8 w-40" />
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-64 w-full rounded-3xl" />
    </div>
  )
}

// ---------------- Row ----------------

function ShoppingRow({ item, listId }: { item: ShoppingListItem; listId: string }) {
  const { toast } = useToast()
  const qc = useQueryClient()
  const patch = usePatchItem(listId)
  const del = useDeleteItem(listId)
  const pantry = useAddPantryItem()

  const [qty, setQty] = useState(item.quantity != null ? String(item.quantity) : '')

  const toggleChecked = () => {
    const next = !item.checked
    // Optimistic update for snappy checkbox feel.
    qc.setQueryData<ShoppingList>(qk.shoppingList(listId), (old) =>
      old
        ? {
            ...old,
            items: old.items.map((i) => (i.id === item.id ? { ...i, checked: next } : i)),
          }
        : old,
    )
    patch.mutate(
      { itemId: item.id, patch: { checked: next } },
      { onError: () => toast(t('error.saveFailed'), 'error') },
    )
  }

  const commitQty = () => {
    const trimmed = qty.trim()
    if (trimmed === '') return
    const n = Number(trimmed.replace(',', '.'))
    if (!Number.isFinite(n) || n <= 0 || n === item.quantity) return
    patch.mutate(
      { itemId: item.id, patch: { quantity: n } },
      { onError: () => toast(t('error.saveFailed'), 'error') },
    )
  }

  const remove = () =>
    del.mutate(item.id, {
      onSuccess: () => toast(t('shopping.itemDeleted'), 'info'),
      onError: () => toast(t('error.saveFailed'), 'error'),
    })

  const haveAtHome = () => {
    if (!item.ingredient_id) return
    del.mutate(item.id, {
      onSuccess: () => toast(t('shopping.haveAtHomeToast', { name: item.name }), 'success'),
      onError: () => toast(t('error.saveFailed'), 'error'),
    })
    pantry.mutate(item.ingredient_id)
  }

  const busy = patch.isPending || del.isPending

  return (
    <li className={`flex items-center gap-3 px-4 py-3 ${busy ? 'opacity-60' : ''}`}>
      <button
        type="button"
        role="checkbox"
        aria-checked={item.checked}
        aria-label={item.name}
        onClick={toggleChecked}
        className={`flex size-6 shrink-0 items-center justify-center rounded-full border-2 transition active:scale-90 ${
          item.checked
            ? 'border-emerald-500 bg-emerald-500 text-white'
            : 'border-stone-300 bg-white hover:border-brand-500'
        }`}
      >
        {item.checked && <IconCheck className="size-4" />}
      </button>

      <div className="min-w-0 flex-1">
        <p
          className={`truncate text-sm font-medium ${
            item.checked ? 'text-stone-400 line-through' : 'text-stone-800'
          }`}
        >
          {item.name}
          {item.is_manual && (
            <span className="ml-1.5 text-[10px] font-normal text-stone-400">
              ({t('shopping.manual').toLowerCase()})
            </span>
          )}
        </p>
      </div>

      <div className="flex shrink-0 items-center gap-1.5 text-sm text-stone-500">
        <input
          type="text"
          inputMode="decimal"
          value={qty}
          aria-label={`${t('shopping.qty')}: ${item.name}`}
          onChange={(e) => setQty(e.target.value)}
          onBlur={commitQty}
          onKeyDown={(e) => e.key === 'Enter' && (e.target as HTMLInputElement).blur()}
          className="w-12 rounded-lg bg-cream-dark px-2 py-1 text-right text-sm tabular-nums focus:outline-none focus:ring-2 focus:ring-brand-500 sm:w-14"
        />
        <span className="w-10 text-xs sm:w-12">
          {item.unit ? unitLabel(item.unit) : formatQuantity(null, '')}
        </span>
      </div>

      {item.ingredient_id && (
        <button
          type="button"
          onClick={haveAtHome}
          title={t('shopping.haveAtHome')}
          aria-label={`${t('shopping.haveAtHome')}: ${item.name}`}
          className="flex shrink-0 items-center gap-1 rounded-full bg-emerald-50 px-2.5 py-1 text-[11px] font-semibold text-emerald-700 transition hover:bg-emerald-100"
        >
          <IconCheck className="size-3.5 sm:hidden" />
          <span className="hidden sm:inline">{t('shopping.haveAtHome')}</span>
        </button>
      )}

      <button
        type="button"
        onClick={remove}
        aria-label={`${t('common.delete')}: ${item.name}`}
        className="flex size-8 shrink-0 items-center justify-center rounded-full text-stone-300 transition hover:bg-rose-50 hover:text-rose-600"
      >
        <IconTrash className="size-4" />
      </button>
    </li>
  )
}

// ---------------- Manual add ----------------

function AddItemRow({ listId }: { listId: string }) {
  const { toast } = useToast()
  const add = useAddItem(listId)
  const [name, setName] = useState('')
  const [qty, setQty] = useState('')
  const [unit, setUnit] = useState('')
  const [category, setCategory] = useState<IngredientCategory>('other')

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) {
      toast(t('shopping.nameRequired'), 'error')
      return
    }
    const qtyNum = qty.trim() === '' ? undefined : Number(qty.trim().replace(',', '.'))
    if (qtyNum !== undefined && (!Number.isFinite(qtyNum) || qtyNum <= 0)) {
      toast(t('shopping.nameRequired'), 'error')
      return
    }
    add.mutate(
      {
        name: trimmed,
        quantity: qtyNum,
        unit: unit || undefined,
        category,
      },
      {
        onSuccess: () => {
          setName('')
          setQty('')
          toast(t('shopping.itemAdded'), 'success')
        },
        onError: () => toast(t('error.saveFailed'), 'error'),
      },
    )
  }

  return (
    <form
      onSubmit={submit}
      className="mt-6 flex flex-wrap items-center gap-2 rounded-3xl bg-white p-4 shadow-card"
    >
      <IconPlus className="size-5 shrink-0 text-brand-600" />
      <input
        type="text"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder={t('shopping.addPlaceholder')}
        aria-label={t('shopping.addPlaceholder')}
        className="min-w-0 flex-1 rounded-xl bg-cream-dark px-3 py-2.5 text-sm placeholder:text-stone-400 focus:outline-none focus:ring-2 focus:ring-brand-500"
      />
      <input
        type="text"
        inputMode="decimal"
        value={qty}
        onChange={(e) => setQty(e.target.value)}
        placeholder={t('shopping.qty')}
        aria-label={t('shopping.qty')}
        className="w-20 rounded-xl bg-cream-dark px-3 py-2.5 text-sm placeholder:text-stone-400 focus:outline-none focus:ring-2 focus:ring-brand-500"
      />
      <select
        value={unit}
        onChange={(e) => setUnit(e.target.value)}
        aria-label={t('shopping.unit')}
        className="rounded-xl bg-cream-dark px-2.5 py-2.5 text-sm text-stone-600 focus:outline-none focus:ring-2 focus:ring-brand-500"
      >
        {UNIT_OPTIONS.map((u) => (
          <option key={u} value={u}>
            {unitLabel(u)}
          </option>
        ))}
      </select>
      <select
        value={category}
        onChange={(e) => setCategory(e.target.value as IngredientCategory)}
        aria-label={t('shopping.category.other')}
        className="rounded-xl bg-cream-dark px-2.5 py-2.5 text-sm text-stone-600 focus:outline-none focus:ring-2 focus:ring-brand-500"
      >
        {INGREDIENT_CATEGORIES.map((c) => (
          <option key={c} value={c}>
            {t(CATEGORY_KEY(c))}
          </option>
        ))}
      </select>
      <button
        type="submit"
        disabled={add.isPending}
        className="flex items-center gap-2 rounded-2xl bg-brand-600 px-5 py-2.5 text-sm font-bold text-white shadow-chip transition hover:bg-brand-700 active:scale-95 disabled:opacity-60"
      >
        {add.isPending ? <Spinner className="size-4 border-2 border-white/40 border-t-white" /> : null}
        {t('shopping.addItem')}
      </button>
    </form>
  )
}
