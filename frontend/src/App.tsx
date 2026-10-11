import { useEffect } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router'
import { useAuth } from './stores/auth'
import { AppShell } from './components/AppShell'
import { ToastProvider } from './components/Toast'
import { LandingPage } from './pages/LandingPage'
import { LoginPage, RegisterPage } from './pages/AuthPages'
import { OnboardingPage } from './pages/OnboardingPage'
import { SwipePage } from './pages/SwipePage'
import { FavoritesPage } from './pages/FavoritesPage'
import { LikesPage } from './pages/LikesPage'
import { MenuPage } from './pages/MenuPage'
import { ShoppingPage } from './pages/ShoppingPage'
import { SettingsPage } from './pages/SettingsPage'
import { RecipePage } from './pages/RecipePage'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 15_000, refetchOnWindowFocus: false },
  },
})

export default function App() {
  const init = useAuth((s) => s.init)

  // Probe the session cookie once on mount (also refreshes the CSRF token).
  useEffect(() => {
    void init()
  }, [init])

  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <BrowserRouter>
        <Routes>
          <Route path="/" element={<LandingPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />

          {/* Guarded app routes */}
          <Route element={<AppShell />}>
            <Route path="/onboarding" element={<OnboardingPage />} />
            <Route path="/swipe" element={<SwipePage />} />
            <Route path="/likes" element={<LikesPage />} />
            <Route path="/favorites" element={<FavoritesPage />} />
            <Route path="/menu" element={<MenuPage />} />
            <Route path="/shopping" element={<ShoppingPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/recipes/:id" element={<RecipePage />} />
          </Route>

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
        </BrowserRouter>
      </ToastProvider>
    </QueryClientProvider>
  )
}
