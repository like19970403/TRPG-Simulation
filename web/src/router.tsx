import { createBrowserRouter, Navigate } from 'react-router'
import { AuthGuard } from './components/auth-guard'
import { GuestGuard } from './components/guest-guard'
import { GmGuard } from './components/gm/gm-guard'
import { AuthLayout } from './layouts/auth-layout'
import { AppLayout } from './layouts/app-layout'
import { LoginPage } from './pages/login-page'
import { RegisterPage } from './pages/register-page'
import { DashboardPage } from './pages/dashboard-page'
import { NotFoundPage } from './pages/not-found-page'
import { ErrorPage } from './pages/error-page'
import { ScenarioListPage } from './pages/scenario-list-page'
import { ScenarioDetailPage } from './pages/scenario-detail-page'
import { ScenarioEditPage } from './pages/scenario-edit-page'
import { GmConsolePage } from './pages/gm-console-page'
import { PlayerGuard } from './components/player/player-guard'
import { PlayerGamePage } from './pages/player-game-page'
import { SessionListPage } from './pages/session-list-page'
import { SessionLobbyPage } from './pages/session-lobby-page'
import { CharacterListPage } from './pages/character-list-page'
import { SessionReplayPage } from './pages/session-replay-page'
import { ROUTES } from './lib/constants'

export const router = createBrowserRouter([
  {
    path: ROUTES.HOME,
    element: <Navigate to={ROUTES.DASHBOARD} replace />,
    errorElement: <ErrorPage />,
  },
  {
    element: <GuestGuard />,
    errorElement: <ErrorPage />,
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
    errorElement: <ErrorPage />,
    children: [
      {
        element: <AppLayout />,
        children: [
          { path: ROUTES.DASHBOARD, element: <DashboardPage /> },
          { path: ROUTES.SCENARIOS, element: <ScenarioListPage /> },
          { path: ROUTES.SCENARIO_NEW, element: <ScenarioEditPage /> },
          { path: ROUTES.SCENARIO_DETAIL, element: <ScenarioDetailPage /> },
          { path: ROUTES.SCENARIO_EDIT, element: <ScenarioEditPage /> },
          { path: ROUTES.SESSIONS, element: <SessionListPage /> },
          { path: ROUTES.SESSION_LOBBY, element: <SessionLobbyPage /> },
          { path: ROUTES.CHARACTERS, element: <CharacterListPage /> },
        ],
      },
      {
        element: <GmGuard />,
        children: [
          { path: ROUTES.GM_CONSOLE, element: <GmConsolePage /> },
        ],
      },
      {
        element: <PlayerGuard />,
        children: [
          { path: ROUTES.PLAYER_GAME, element: <PlayerGamePage /> },
        ],
      },
      { path: ROUTES.SESSION_REPLAY, element: <SessionReplayPage /> },
    ],
  },
  {
    path: '*',
    element: <NotFoundPage />,
  },
])
