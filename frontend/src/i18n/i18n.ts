import { ru, type TranslationKey } from './ru'

export type { TranslationKey }

type Params = Record<string, string | number>

/**
 * Translate a key into the current (Russian) locale.
 * Interpolates {placeholder} tokens from `params`.
 * Unknown keys return the key itself so missing strings are visible in dev.
 */
export function t(key: TranslationKey, params?: Params): string {
  const template: string = ru[key]
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (match, name: string) =>
    name in params ? String(params[name]) : match,
  )
}

/** Russian plural rules: 1 → one, 2-4 → few, else many. */
export function plural(
  n: number,
  forms: { one: string; few: string; many: string },
): string {
  const abs = Math.abs(n) % 100
  const last = abs % 10
  if (abs > 10 && abs < 20) return forms.many
  if (last > 1 && last < 5) return forms.few
  if (last === 1) return forms.one
  return forms.many
}
