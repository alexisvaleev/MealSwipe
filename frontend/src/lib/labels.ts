/** Russian labels for enum-like API values, shared across pages. */
import { t, type TranslationKey } from '../i18n/i18n'
import {
  INGREDIENT_CATEGORIES,
  type Allergen,
  type DietType,
  type MealType,
} from './types'

const MEAL_TYPE_KEY: Record<MealType, TranslationKey> = {
  breakfast: 'prefs.mealType.breakfast',
  lunch: 'prefs.mealType.lunch',
  dinner: 'prefs.mealType.dinner',
  snack: 'prefs.mealType.snack',
}

const ALLERGEN_KEY: Record<Allergen, TranslationKey> = {
  milk: 'prefs.allergen.milk',
  eggs: 'prefs.allergen.eggs',
  gluten: 'prefs.allergen.gluten',
  nuts: 'prefs.allergen.nuts',
  peanuts: 'prefs.allergen.peanuts',
  fish: 'prefs.allergen.fish',
  shellfish: 'prefs.allergen.shellfish',
  soy: 'prefs.allergen.soy',
  sesame: 'prefs.allergen.sesame',
}

const DIET_KEY: Record<DietType, TranslationKey> = {
  any: 'prefs.diet.any',
  vegetarian: 'prefs.diet.vegetarian',
  vegan: 'prefs.diet.vegan',
  pescatarian: 'prefs.diet.pescatarian',
}

export function mealTypeLabel(mt: MealType): string {
  return t(MEAL_TYPE_KEY[mt])
}

export function allergenLabel(a: Allergen): string {
  return t(ALLERGEN_KEY[a])
}

export function dietLabel(d: DietType): string {
  return t(DIET_KEY[d])
}

const KNOWN_CATEGORIES = new Set<string>(INGREDIENT_CATEGORIES)

/** Shopping/ingredient category label; unknown values fall back to "other". */
export function categoryLabel(category: string): string {
  const key: TranslationKey = KNOWN_CATEGORIES.has(category)
    ? (`shopping.category.${category}` as TranslationKey)
    : 'shopping.category.other'
  return t(key)
}

export function dayLabel(dayIndex: number): string {
  return t(`menu.day.${dayIndex}` as TranslationKey)
}

export function dayFullLabel(dayIndex: number): string {
  return t(`menu.dayFull.${dayIndex}` as TranslationKey)
}
