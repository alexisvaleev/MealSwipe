# MealSwipe — frontend

Mobile-first swipe UI for the MealSwipe self-hosted meal planner.
Russian UI; all strings live in `src/i18n/ru.ts`.

## Stack

Vite 8 · React 19 · TypeScript (strict) · Tailwind CSS 4 (`@tailwindcss/vite`) ·
framer-motion · react-router · TanStack Query 5 · zustand · vitest + Testing Library.

## Scripts

| Command | Purpose |
|---|---|
| `npm run dev` | Dev server (proxies `/api` → `http://localhost:8080`) |
| `npm run build` | `tsc -b` + production bundle to `dist/` |
| `npm run preview` | Serve the production build |
| `npm test` | `vitest run` |
| `npm run lint` | eslint (flat config) |
| `npm run typecheck` | `tsc --noEmit` for app + vite config |

## Env

`VITE_APP_NAME` — branding (default `MealSwipe`).

## API

All calls go through `src/lib/api.ts` → `/api/v1` with session cookies
(`credentials: 'include'`) and `X-CSRF-Token` on mutations. See SPEC.md §5.
