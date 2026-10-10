import { t } from '../i18n/i18n'
import { Modal } from './Modal'
import { RecipeDetailLoader } from './RecipeDetail'

interface Props {
  recipeId: string | null
  onClose: () => void
}

/** Recipe detail shown as a modal over the current screen. */
export function RecipeModal({ recipeId, onClose }: Props) {
  return (
    <Modal open={recipeId !== null} onClose={onClose} wide ariaLabel={t('swipe.details')}>
      {recipeId && <RecipeDetailLoader recipeId={recipeId} />}
    </Modal>
  )
}
