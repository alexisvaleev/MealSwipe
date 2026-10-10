import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { SwipeCard } from './SwipeCard'
import type { RecipeSummary } from '../lib/types'

const recipe: RecipeSummary = {
  id: 'r1',
  title: 'Овсянка с бананом',
  description: 'Быстрый завтрак',
  image_url: null,
  category: 'Каша',
  meal_types: ['breakfast'],
  prep_minutes: 5,
  cook_minutes: 10,
  servings: 1,
  cost_per_serving: 80,
  kcal_per_serving: 380,
  ingredients_count: 4,
}

describe('SwipeCard', () => {
  it('renders title, category and meta chips', () => {
    render(
      <SwipeCard recipe={recipe} onSwipe={() => {}} onOpenDetails={() => {}} />,
    )
    expect(screen.getByRole('group', { name: 'Овсянка с бананом' })).toBeInTheDocument()
    expect(screen.getByText('Овсянка с бананом')).toBeInTheDocument()
    expect(screen.getByText('Каша')).toBeInTheDocument()
    expect(screen.getByText('15 мин')).toBeInTheDocument()
    expect(screen.getByText(/380 ккал/)).toBeInTheDocument()
    expect(screen.getByText(/80 ₽\/порц\./)).toBeInTheDocument()
    expect(screen.getByText('4 ингр.')).toBeInTheDocument()
  })

  it('shows the gradient placeholder when image_url is null', () => {
    const { container } = render(
      <SwipeCard recipe={recipe} onSwipe={() => {}} onOpenDetails={() => {}} />,
    )
    expect(screen.getByTestId('recipe-image-fallback')).toBeInTheDocument()
    // No real <img> element is rendered.
    expect(container.querySelector('img')).toBeNull()
  })

  it('falls back to the placeholder when the image fails to load', () => {
    const { container } = render(
      <SwipeCard
        recipe={{ ...recipe, image_url: 'https://example.com/broken.jpg' }}
        onSwipe={() => {}}
        onOpenDetails={() => {}}
      />,
    )
    const img = container.querySelector('img')
    expect(img).not.toBeNull()
    expect(img).toHaveAttribute('loading', 'lazy')
    fireEvent.error(img!)
    expect(screen.getByTestId('recipe-image-fallback')).toBeInTheDocument()
  })

  it('fires onOpenDetails when the photo area is clicked', () => {
    const onOpenDetails = vi.fn()
    render(
      <SwipeCard recipe={recipe} onSwipe={() => {}} onOpenDetails={onOpenDetails} />,
    )
    fireEvent.click(screen.getByRole('button', { name: /Овсянка с бананом/ }))
    expect(onOpenDetails).toHaveBeenCalledOnce()
  })
})
