import { useNavigate, useParams } from 'react-router'
import { t } from '../i18n/i18n'
import { RecipeDetailLoader } from '../components/RecipeDetail'
import { IconChevronLeft } from '../components/Icons'

/** Deep-linkable recipe route /recipes/:id — same detail view as the modal. */
export function RecipePage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  if (!id) return null

  return (
    <div className="mx-auto max-w-2xl">
      <button
        type="button"
        onClick={() => navigate(-1)}
        className="mb-4 inline-flex items-center gap-1 rounded-2xl bg-white px-3 py-2 text-sm font-semibold text-stone-600 shadow-chip transition hover:text-brand-700"
      >
        <IconChevronLeft className="size-4" />
        {t('common.back')}
      </button>
      <div className="overflow-hidden rounded-3xl bg-white shadow-card">
        <RecipeDetailLoader recipeId={id} />
      </div>
    </div>
  )
}
