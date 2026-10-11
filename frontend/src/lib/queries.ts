/**
 * TanStack Query hooks — one per API resource (SPEC §5).
 */
import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseQueryOptions,
} from '@tanstack/react-query'
import { api, ApiError } from './api'
import type {
  Household,
  Ingredient,
  MealType,
  Menu,
  MenuItem,
  Preferences,
  Recipe,
  RecipeSummary,
  ShoppingList,
  ShoppingListItem,
  SwipeDirection,
  UnfilledSlot,
} from './types'

export const qk = {
  discover: ['discover'] as const,
  favorites: (q: string, category: string) => ['favorites', { q, category }] as const,
  favoritesAll: ['favorites'] as const,
  recipe: (id: string) => ['recipe', id] as const,
  recipes: (mealType?: MealType, q?: string) => ['recipes', { mealType, q }] as const,
  menu: (week: string) => ['menu', week] as const,
  shoppingList: (menuId: string) => ['shopping-list', menuId] as const,
  preferences: ['preferences'] as const,
  ingredients: (q: string) => ['ingredients', q] as const,
  household: ['household'] as const,
}

// ---------- Household (shared kitchen) ----------

export function useHousehold() {
  return useQuery({
    queryKey: qk.household,
    queryFn: async () => {
      const res = await api<{ household: Household }>('/household')
      return res.household
    },
  })
}

/** Join/leave swaps the whole shared scope — drop every household-derived cache. */
function invalidateShared(qc: ReturnType<typeof useQueryClient>) {
  void qc.invalidateQueries({ queryKey: qk.household })
  void qc.invalidateQueries({ queryKey: ['menu'] })
  void qc.invalidateQueries({ queryKey: qk.favoritesAll })
  void qc.invalidateQueries({ queryKey: ['shopping-list'] })
  void qc.invalidateQueries({ queryKey: qk.discover })
  void qc.invalidateQueries({ queryKey: ['recipes'] })
  void qc.invalidateQueries({ queryKey: ['recipe'] })
}

export function useJoinHousehold() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (code: string) =>
      api<{ household: Household }>('/household/join', {
        method: 'POST',
        body: { code },
      }),
    onSuccess: (res) => {
      qc.setQueryData(qk.household, res.household)
      invalidateShared(qc)
    },
  })
}

export function useLeaveHousehold() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () =>
      api<{ household: Household }>('/household/leave', { method: 'POST' }),
    onSuccess: (res) => {
      qc.setQueryData(qk.household, res.household)
      invalidateShared(qc)
    },
  })
}

export function useRenameHousehold() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) =>
      api<{ household: Household }>('/household', {
        method: 'PATCH',
        body: { name },
      }),
    onSuccess: (res) => qc.setQueryData(qk.household, res.household),
  })
}

export function useRotateInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () =>
      api<{ household: Household }>('/household/invite/rotate', { method: 'POST' }),
    onSuccess: (res) => qc.setQueryData(qk.household, res.household),
  })
}

// ---------- Discover / swipes ----------

export function useDiscover(limit = 15) {
  return useQuery({
    queryKey: qk.discover,
    queryFn: async () => {
      const res = await api<{ recipes: RecipeSummary[] }>('/discover', {
        query: { limit },
      })
      return res.recipes ?? []
    },
    staleTime: 60_000,
  })
}

export function useSwipe() {
  return useMutation({
    mutationFn: ({ recipeId, direction }: { recipeId: string; direction: SwipeDirection }) =>
      api(`/recipes/${recipeId}/swipe`, { method: 'PUT', body: { direction } }),
  })
}

export function useUndo() {
  return useMutation({
    mutationFn: () => api<{ undone: boolean }>('/swipes/undo', { method: 'POST' }),
  })
}

// ---------- Favorites ----------

export function useFavorites(params: { q?: string; category?: string }) {
  return useQuery({
    queryKey: qk.favorites(params.q ?? '', params.category ?? ''),
    queryFn: async () => {
      const res = await api<{ recipes: RecipeSummary[] }>('/favorites', {
        query: { q: params.q, category: params.category },
      })
      return res.recipes ?? []
    },
  })
}

export function useFavorite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ recipeId, favorite }: { recipeId: string; favorite: boolean }) =>
      api(`/recipes/${recipeId}/favorite`, {
        method: favorite ? 'PUT' : 'DELETE',
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.favoritesAll })
    },
  })
}

// ---------- Recipes ----------

export function useRecipes(params: { meal_type?: MealType; q?: string; page_size?: number }) {
  return useQuery({
    queryKey: [...qk.recipes(params.meal_type, params.q), params.page_size] as const,
    queryFn: async () => {
      const res = await api<{ recipes: RecipeSummary[]; total?: number }>('/recipes', {
        query: { meal_type: params.meal_type, q: params.q, page_size: params.page_size },
      })
      return res.recipes ?? []
    },
  })
}

export function useRecipe(id: string | undefined) {
  return useQuery({
    queryKey: qk.recipe(id ?? ''),
    queryFn: async () => {
      const res = await api<{ recipe: Recipe }>(`/recipes/${id}`)
      return res.recipe
    },
    enabled: Boolean(id),
  })
}

// ---------- Menu ----------

export function useMenu(week: string, options?: { enabled?: boolean }) {
  return useQuery<Menu | null>({
    queryKey: qk.menu(week),
    queryFn: async () => {
      try {
        const res = await api<{ menu: Menu }>('/menus/current', { query: { week } })
        return res.menu ?? null
      } catch (err) {
        if (err instanceof ApiError && err.status === 404) return null
        throw err
      }
    },
    enabled: options?.enabled ?? true,
  })
}

export function useGenerateMenu() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (weekStart: string) => {
      const res = await api<{ menu: Menu; unfilled?: UnfilledSlot[] }>('/menus/generate', {
        method: 'POST',
        body: { week_start: weekStart },
      })
      return res
    },
    onSuccess: (res) => {
      qc.setQueryData(qk.menu(res.menu.week_start), res.menu)
      void qc.invalidateQueries({ queryKey: ['menu'] })
    },
  })
}

type MenuItemPatch = {
  recipe_id?: string
  servings?: number
  day_index?: number
  meal_type?: MealType
  locked?: boolean
}

export function useUpdateMenuItem(menuId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ itemId, patch }: { itemId: string; patch: MenuItemPatch }) =>
      api<{ item: MenuItem }>(`/menus/${menuId}/items/${itemId}`, {
        method: 'PATCH',
        body: patch,
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['menu'] }),
  })
}

export function useAddMenuItem(menuId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: {
      day_index: number
      meal_type: MealType
      recipe_id: string
      servings: number
    }) => api<{ item: MenuItem }>(`/menus/${menuId}/items`, { method: 'POST', body }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['menu'] }),
  })
}

export function useRemoveMenuItem(menuId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (itemId: string) =>
      api(`/menus/${menuId}/items/${itemId}`, { method: 'DELETE' }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['menu'] }),
  })
}

export function useSwapItem(menuId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (itemId: string) =>
      api<{ item: MenuItem }>(`/menus/${menuId}/items/${itemId}/swap`, { method: 'POST' }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['menu'] }),
  })
}

// ---------- Shopping list ----------

export function useShoppingList(menuId: string | undefined) {
  return useQuery({
    queryKey: qk.shoppingList(menuId ?? ''),
    queryFn: async () => {
      const res = await api<{ list: ShoppingList }>(`/menus/${menuId}/shopping-list`)
      return res.list
    },
    enabled: Boolean(menuId),
  })
}

export function useRegenerateList(listId: string | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      const res = await api<{ list: ShoppingList }>(
        `/shopping-lists/${listId}/regenerate`,
        { method: 'POST' },
      )
      return res.list
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['shopping-list'] }),
  })
}

export function usePatchItem(listId: string | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      itemId,
      patch,
    }: {
      itemId: string
      patch: { checked?: boolean; quantity?: number | null; name?: string }
    }) =>
      api<{ item: ShoppingListItem }>(`/shopping-lists/${listId}/items/${itemId}`, {
        method: 'PATCH',
        body: patch,
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['shopping-list'] }),
  })
}

export function useAddItem(listId: string | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { name: string; quantity?: number; unit?: string; category?: string }) =>
      api<{ item: ShoppingListItem }>(`/shopping-lists/${listId}/items`, {
        method: 'POST',
        body,
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['shopping-list'] }),
  })
}

export function useDeleteItem(listId: string | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (itemId: string) =>
      api(`/shopping-lists/${listId}/items/${itemId}`, { method: 'DELETE' }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['shopping-list'] }),
  })
}

/** PUT /pantry/{ingredient_id} — mark as "already have at home" (backend extension per spec). */
export function useAddPantryItem() {
  return useMutation({
    mutationFn: (ingredientId: string | number) =>
      api(`/pantry/${ingredientId}`, { method: 'PUT' }),
  })
}

// ---------- Preferences & ingredients ----------

export function usePreferences(options?: Partial<UseQueryOptions<Preferences>>) {
  return useQuery({
    queryKey: qk.preferences,
    queryFn: async () => {
      const res = await api<{ preferences: Preferences }>('/preferences')
      return res.preferences
    },
    ...options,
  })
}

export function useSavePreferences() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (prefs: Preferences) => {
      const res = await api<{ preferences: Preferences }>('/preferences', {
        method: 'PUT',
        body: prefs,
      })
      return res.preferences
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.preferences }),
  })
}

export function useIngredients(q: string, enabled = true) {
  return useQuery({
    queryKey: qk.ingredients(q),
    queryFn: async () => {
      const res = await api<{ ingredients: Ingredient[] }>('/ingredients', {
        query: { q },
      })
      return res.ingredients ?? []
    },
    enabled,
    staleTime: 60_000,
  })
}
