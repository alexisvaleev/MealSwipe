import { t, plural } from '../i18n/i18n'
import type { TranslationKey } from '../i18n/i18n'
import type { RecipeIngredient, RecipeSummary } from './types'

/** Units the backend may emit (SPEC §7-8). Unknown units are passed through. */
const UNIT_KEYS: Record<string, TranslationKey> = {
  g: 'unit.g',
  kg: 'unit.kg',
  ml: 'unit.ml',
  l: 'unit.l',
  pcs: 'unit.pcs',
  tbsp: 'unit.tbsp',
  tsp: 'unit.tsp',
  cup: 'unit.cup',
  pinch: 'unit.pinch',
  to_taste: 'unit.to_taste',
  slice: 'unit.slice',
  clove: 'unit.clove',
}

/** Trim a number for display: 1.5 → "1,5", 2.0 → "2" (RU decimal comma). */
export function formatNumber(n: number): string {
  const rounded = Math.round(n * 100) / 100
  return rounded.toLocaleString('ru-RU', { maximumFractionDigits: 2 })
}

/**
 * Format a quantity + unit for display. Applies SPEC §7 display rules:
 * ≥1000 g → kg, ≥1000 ml → l (one decimal).
 */
export function formatQuantity(
  quantity: number | null | undefined,
  unit: string | null | undefined,
): string {
  if (unit === 'to_taste' || unit === 'pinch' && !quantity) {
    return t(UNIT_KEYS[unit] ?? 'unit.pcs')
  }
  if (quantity === null || quantity === undefined || quantity === 0) {
    return unit ? t(UNIT_KEYS[unit] ?? 'unit.pcs') : ''
  }
  let qty = quantity
  let u = unit ?? ''
  if (u === 'g' && qty >= 1000) {
    qty = qty / 1000
    u = 'kg'
  } else if (u === 'ml' && qty >= 1000) {
    qty = qty / 1000
    u = 'l'
  }
  const label = UNIT_KEYS[u] ? t(UNIT_KEYS[u]) : u
  return label ? `${formatNumber(qty)} ${label}` : formatNumber(qty)
}

/** Scale a recipe-ingredient quantity by servings selector / base servings. */
export function scaleQuantity(
  quantity: number | null | undefined,
  baseServings: number,
  targetServings: number,
): number | null {
  if (quantity === null || quantity === undefined) return null
  if (!baseServings || baseServings <= 0) return quantity
  return (quantity * targetServings) / baseServings
}

/** Format an ingredient line: "200 г муки" (note appended by caller if needed). */
export function formatIngredient(
  ingredient: RecipeIngredient,
  baseServings: number,
  targetServings: number,
): string {
  const scaled = scaleQuantity(ingredient.quantity, baseServings, targetServings)
  return formatQuantity(scaled, ingredient.unit)
}

/** Number of ingredients shown on a recipe card, tolerating shape variants. */
export function ingredientCount(recipe: RecipeSummary): number | null {
  if (typeof recipe.ingredients_count === 'number') return recipe.ingredients_count
  if (typeof recipe.ingredient_count === 'number') return recipe.ingredient_count
  if (Array.isArray(recipe.ingredients)) return recipe.ingredients.length
  return null
}

export function totalMinutes(recipe: RecipeSummary): number | null {
  const prep = recipe.prep_minutes ?? 0
  const cook = recipe.cook_minutes ?? 0
  const total = prep + cook
  return total > 0 ? total : null
}

export function servingsLabel(n: number): string {
  return plural(n, {
    one: t('common.servings_one'),
    few: t('common.servings_few'),
    many: t('common.servings_many'),
  })
}

// ---- Week helpers (menu weeks start on Monday) ----

function pad(n: number): string {
  return String(n).padStart(2, '0')
}

export function toISODate(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** Monday of the week containing `date` (ISO YYYY-MM-DD). */
export function mondayOf(date: Date): string {
  const d = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  const day = (d.getDay() + 6) % 7 // 0 = Monday
  d.setDate(d.getDate() - day)
  return toISODate(d)
}

export function addDays(iso: string, days: number): string {
  const d = new Date(`${iso}T00:00:00`)
  d.setDate(d.getDate() + days)
  return toISODate(d)
}

export function addWeeks(iso: string, weeks: number): string {
  return addDays(iso, weeks * 7)
}

export function formatDateShort(iso: string): string {
  return new Date(`${iso}T00:00:00`).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'long',
  })
}
