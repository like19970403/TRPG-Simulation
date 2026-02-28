import { createBrowserRouter, Navigate } from 'react-router'
import { AuthGuard } from './components/auth-guard'
import { GuestGuard } from './components/guest-guard'
import { AuthLayout } from './layouts/auth-layout'
import { AppLayout } from './layouts/app-layout'
import { LoginPage } from './pages/login-page'
import { RegisterPage } from './pages/register-page'
import { DashboardPage } from './pages/dashboard-page'
import { NotFoundPage } from './pages/not-found-page'
import { ScenarioListPage } from './pages/scenario-list-page'
import { ScenarioDetailPage } from './pages/scenario-detail-page'
import { ScenarioEditPage } from './pages/scenario-edit-page'
import { ROUTES } from './lib/constants'

export const router = createBrowserRouter([
  {
    path: ROUTES.HOME,
    element: <Navigate to={ROUTES.DASHBOARD} replace />,
  },
  {
    element: <GuestGuard />,
    children: [
      {
        element: <AuthLayout />,
        children: [
          { path: ROUTES.LOGIN, element: <LoginPage /> },
          { path: ROUTES.REGISTER, element: <RegisterPage /> },
        ],
      },
    ],
  },
  {
    element: <AuthGuard />,
    children: [
      {
        element: <AppLayout />,
        children: [
          { path: ROUTES.DASHBOARD, element: <DashboardPage /> },
          { path: ROUTES.SCENARIOS, element: <ScenarioListPage /> },
          { path: ROUTES.SCENARIO_NEW, element: <ScenarioEditPage /> },
          { path: ROUTES.SCENARIO_DETAIL, element: <ScenarioDetailPage /> },
          { path: ROUTES.SCENARIO_EDIT, element: <ScenarioEditPage /> },
        ],
      },
    ],
  },
  {
    path: '*',
    element: <NotFoundPage />,
  },
])
