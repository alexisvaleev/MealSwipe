import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

// vitest globals are off → register RTL cleanup explicitly.
afterEach(() => cleanup())

// jsdom lacks matchMedia / ResizeObserver — framer-motion touches them.
if (typeof window !== 'undefined') {
  if (!window.matchMedia) {
    window.matchMedia = ((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    })) as unknown as typeof window.matchMedia
  }
  if (!('ResizeObserver' in window)) {
    class RO {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
    ;(window as unknown as { ResizeObserver: typeof RO }).ResizeObserver = RO
  }
  if (!('IntersectionObserver' in window)) {
    class IO {
      readonly root = null
      readonly rootMargin = ''
      readonly thresholds = []
      observe() {}
      unobserve() {}
      disconnect() {}
      takeRecords() {
        return []
      }
    }
    ;(window as unknown as { IntersectionObserver: typeof IO }).IntersectionObserver = IO
    ;(globalThis as unknown as { IntersectionObserver: typeof IO }).IntersectionObserver = IO
  }
}
