import { describe, expect, it } from 'vitest'
import { ru } from './ru'
import { plural, t } from './i18n'

describe('i18n t()', () => {
  it('returns the Russian string for a known key', () => {
    expect(t('nav.swipe')).toBe('Подбор')
    expect(t('auth.login')).toBe('Войти')
  })

  it('interpolates {placeholders}', () => {
    expect(t('auth.welcomeBack', { name: 'Маша' })).toBe('С возвращением, Маша!')
    expect(t('shopping.progress', { done: 3, total: 10 })).toBe('Куплено 3 из 10')
  })

  it('leaves unknown placeholders untouched', () => {
    expect(t('auth.welcomeBack', { other: 'x' })).toBe('С возвращением, {name}!')
  })

  it('every dictionary value is a non-empty string', () => {
    for (const [key, value] of Object.entries(ru)) {
      expect(typeof value, key).toBe('string')
      expect(value.length, key).toBeGreaterThan(0)
    }
  })
})

describe('plural() — Russian plural rules', () => {
  const forms = { one: 'порция', few: 'порции', many: 'порций' }
  it.each([
    [1, 'порция'],
    [2, 'порции'],
    [3, 'порции'],
    [4, 'порции'],
    [5, 'порций'],
    [11, 'порций'],
    [21, 'порция'],
    [25, 'порций'],
    [101, 'порция'],
  ])('%i → %s', (n, expected) => {
    expect(plural(n, forms)).toBe(expected)
  })
})
