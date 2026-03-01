export const API = {
  REGISTER: '/api/v1/users',
  LOGIN: '/api/v1/auth/login',
  REFRESH: '/api/v1/auth/refresh',
  LOGOUT: '/api/v1/auth/logout',
  SCENARIOS: '/api/v1/scenarios',
  SESSIONS: '/api/v1/sessions',
} as const

export const ROUTES = {
  HOME: '/',
  LOGIN: '/login',
  REGISTER: '/register',
  DASHBOARD: '/dashboard',
  SCENARIOS: '/scenarios',
  SCENARIO_NEW: '/scenarios/new',
  SCENARIO_DETAIL: '/scenarios/:id',
  SCENARIO_EDIT: '/scenarios/:id/edit',
  GM_CONSOLE: '/sessions/:id/gm',
  PLAYER_GAME: '/sessions/:id/play',
} as const
