import { describe, expect, it } from 'vitest'
import {
  addWeeks,
  formatQuantity,
  ingredientCount,
  mondayOf,
  scaleQuantity,
  totalMinutes,
} from './format'
import type { RecipeSummary } from './types'

describe('scaleQuantity', () => {
  it('scales proportionally to servings', () => {
    expect(scaleQuantity(200, 4, 2)).toBe(100)
    expect(scaleQuantity(50, 1, 4)).toBe(200)
  })
  it('returns quantity unchanged for invalid base', () => {
    expect(scaleQuantity(50, 0, 4)).toBe(50)
  })
  it('passes null through', () => {
    expect(scaleQuantity(null, 2, 4)).toBeNull()
    expect(scaleQuantity(undefined, 2, 4)).toBeNull()
  })
})

describe('formatQuantity', () => {
  it('formats grams under 1000 as г', () => {
    expect(formatQuantity(250, 'g')).toBe('250 г')
  })
  it('converts ≥1000 g to kg per SPEC §7', () => {
    expect(formatQuantity(1500, 'g')).toBe('1,5 кг')
    expect(formatQuantity(2000, 'ml')).toBe('2 л')
  })
  it('handles pcs and special units', () => {
    expect(formatQuantity(2, 'pcs')).toBe('2 шт.')
    expect(formatQuantity(null, 'to_taste')).toBe('по вкусу')
    expect(formatQuantity(0, 'g')).toBe('г')
  })
  it('passes unknown units through', () => {
    expect(formatQuantity(3, 'bottle')).toBe('3 bottle')
  })
})

describe('week helpers', () => {
  it('mondayOf returns the Monday of the current week', () => {
    // 2026-10-11 is a Sunday → Monday is 2026-10-05.
    expect(mondayOf(new Date(2026, 9, 11))).toBe('2026-10-05')
    // A Wednesday.
    expect(mondayOf(new Date(2026, 9, 7))).toBe('2026-10-05')
    // Already Monday.
    expect(mondayOf(new Date(2026, 9, 5))).toBe('2026-10-05')
  })
  it('addWeeks shifts by whole weeks', () => {
    expect(addWeeks('2026-10-05', 1)).toBe('2026-10-12')
    expect(addWeeks('2026-10-05', -1)).toBe('2026-09-28')
  })
})

describe('recipe card helpers', () => {
  const base: RecipeSummary = { id: '1', title: 'T', meal_types: ['dinner'], servings: 2 }
  it('ingredientCount tolerates both count spellings and arrays', () => {
    expect(ingredientCount({ ...base, ingredients_count: 7 })).toBe(7)
    expect(ingredientCount({ ...base, ingredient_count: 5 })).toBe(5)
    expect(
      ingredientCount({
        ...base,
        ingredients: [
          { name: 'a', quantity: 1, unit: 'pcs' },
          { name: 'b', quantity: 2, unit: 'pcs' },
        ],
      }),
    ).toBe(2)
    expect(ingredientCount(base)).toBeNull()
  })
  it('totalMinutes sums prep+cook, null when absent', () => {
    expect(totalMinutes({ ...base, prep_minutes: 10, cook_minutes: 20 })).toBe(30)
    expect(totalMinutes(base)).toBeNull()
  })
})
