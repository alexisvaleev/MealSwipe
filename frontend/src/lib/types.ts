/** Domain types matching SPEC §4–§5. Field names mirror the JSON contract. */

export type MealType = 'breakfast' | 'lunch' | 'dinner' | 'snack'
export const MEAL_TYPES: readonly MealType[] = ['breakfast', 'lunch', 'dinner', 'snack']

export type DietType = 'any' | 'vegetarian' | 'vegan' | 'pescatarian'
export const DIET_TYPES: readonly DietType[] = ['any', 'vegetarian', 'vegan', 'pescatarian']

export type Allergen =
  | 'milk'
  | 'eggs'
  | 'gluten'
  | 'nuts'
  | 'peanuts'
  | 'fish'
  | 'shellfish'
  | 'soy'
  | 'sesame'
export const ALLERGENS: readonly Allergen[] = [
  'milk',
  'eggs',
  'gluten',
  'nuts',
  'peanuts',
  'fish',
  'shellfish',
  'soy',
  'sesame',
]

export type IngredientCategory =
  | 'produce'
  | 'dairy'
  | 'meat'
  | 'fish'
  | 'bakery'
  | 'grains'
  | 'frozen'
  | 'canned'
  | 'spices'
  | 'oils'
  | 'other'
export const INGREDIENT_CATEGORIES: readonly IngredientCategory[] = [
  'produce',
  'dairy',
  'meat',
  'fish',
  'bakery',
  'grains',
  'frozen',
  'canned',
  'spices',
  'oils',
  'other',
]

export interface User {
  id: string
  username: string
  display_name: string
  has_preferences: boolean
}

/** A shared kitchen: members share favorites/menus/shopping/pantry;
 *  preferences and swipes stay personal. */
export interface HouseholdMember {
  id: string
  display_name: string
  joined_at: string
}

export interface Household {
  id: string
  name: string
  invite_code: string
  members: HouseholdMember[]
}

export interface Preferences {
  household_size: number
  meals_per_day: number
  meal_types: MealType[]
  weekly_budget?: number | null
  max_cook_minutes?: number | null
  diet_type: DietType
  allergens: Allergen[]
  excluded_ingredient_ids: number[]
  goal?: string | null
}

/** Client-side defaults matching the backend's unset-preferences shape. */
export const DEFAULT_PREFERENCES: Preferences = {
  household_size: 2,
  meals_per_day: 3,
  meal_types: ['breakfast', 'lunch', 'dinner'],
  weekly_budget: null,
  max_cook_minutes: null,
  diet_type: 'any',
  allergens: [],
  excluded_ingredient_ids: [],
  goal: '',
}

export type SwipeDirection = 'like' | 'dislike'

export interface Ingredient {
  id: number
  name: string
  category: IngredientCategory
  unit_family: 'mass' | 'volume' | 'count'
  allergen_tags: Allergen[]
}

/** Ingredient line inside a full recipe. */
export interface RecipeIngredient {
  ingredient_id?: number | null
  name: string
  quantity: number | null
  unit: string | null
  note?: string | null
}

/** Card-level recipe shape returned by /discover, /recipes, /favorites, embedded in menu items. */
export interface RecipeSummary {
  id: string
  slug?: string
  title: string
  description?: string | null
  image_url?: string | null
  image_credit?: string | null
  category?: string | null
  tags?: string[]
  meal_types: MealType[]
  diet_flags?: string[]
  prep_minutes?: number | null
  cook_minutes?: number | null
  servings: number
  cost_per_serving?: number | null
  kcal_per_serving?: number | null
  nutrition_estimate?: boolean
  /** Present on some endpoints; both spellings tolerated. */
  ingredients_count?: number
  ingredient_count?: number
  ingredients?: RecipeIngredient[]
  favorited_at?: string
  favorited?: boolean
  /** Some endpoints may report favorite membership; treated as optional. */

}

export interface Recipe extends RecipeSummary {
  instructions: string[]
  ingredients: RecipeIngredient[]
}

export interface MenuItem {
  id: string
  day_index: number // 0 = Monday
  meal_type: MealType
  servings: number
  locked: boolean
  recipe: RecipeSummary
}

export interface Menu {
  id: string
  week_start: string // Monday, YYYY-MM-DD
  items: MenuItem[]
}

export interface UnfilledSlot {
  day_index: number
  meal_type: MealType
  reason?: string
}

export interface ShoppingListItem {
  id: string
  ingredient_id?: number | null
  name: string
  quantity?: number | null
  unit?: string | null
  category: IngredientCategory | string
  checked: boolean
  is_manual: boolean
  position: number
}

export interface ShoppingList {
  id: string
  menu_id?: string | null
  items: ShoppingListItem[]
}
