import { Suspense, lazy, type ReactNode } from 'react'
import { BrowserRouter, Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { AppShell } from './components/layout/AppShell'
import { Skeleton } from './components/ui/Feedback'
import { AnalysisProvider } from './lib/analysis'
import { AuthProvider, useAuth } from './lib/auth'
import { LoginPage } from './pages/LoginPage'

const DashboardPage = lazy(() => import('./pages/DashboardPage').then((m) => ({ default: m.DashboardPage })))
const MetersPage = lazy(() => import('./pages/MetersPage').then((m) => ({ default: m.MetersPage })))
const MeterDetailPage = lazy(() => import('./pages/MeterDetailPage').then((m) => ({ default: m.MeterDetailPage })))
const AnomaliesPage = lazy(() => import('./pages/AnomaliesPage').then((m) => ({ default: m.AnomaliesPage })))
const InvestigationPage = lazy(() => import('./pages/InvestigationPage').then((m) => ({ default: m.InvestigationPage })))
const NotFoundPage = lazy(() => import('./pages/NotFoundPage').then((m) => ({ default: m.NotFoundPage })))

function RequireAuth({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const location = useLocation()
  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
  return <AnalysisProvider>{children}</AnalysisProvider>
}

function page(element: ReactNode) {
  return <Suspense fallback={<Skeleton className="h-96" />}>{element}</Suspense>
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            element={
              <RequireAuth>
                <AppShell />
              </RequireAuth>
            }
          >
            <Route index element={page(<DashboardPage />)} />
            <Route path="meters" element={page(<MetersPage />)} />
            <Route path="meters/:meterId" element={page(<MeterDetailPage />)} />
            <Route path="anomalies" element={page(<AnomaliesPage />)} />
            <Route path="anomalies/:id" element={page(<InvestigationPage />)} />
            <Route path="*" element={page(<NotFoundPage />)} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
