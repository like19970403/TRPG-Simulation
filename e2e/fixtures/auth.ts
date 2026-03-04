import { test as base, expect, type APIRequestContext } from '@playwright/test'

const API_URL = process.env.E2E_API_URL ?? 'http://localhost:8080'

let userCounter = 0

interface TestUser {
  username: string
  email: string
  password: string
  accessToken: string
  userId: string
}

/** Register a new unique test user via the API. */
async function createTestUser(request: APIRequestContext): Promise<TestUser> {
  const n = ++userCounter
  const ts = Date.now()
  const username = `e2euser_${ts}_${n}`
  const email = `${username}@test.local`
  const password = 'TestPass123!'

  const regRes = await request.post(`${API_URL}/api/v1/users`, {
    data: { username, email, password },
  })
  expect(regRes.ok()).toBeTruthy()
  const regBody = await regRes.json()

  const loginRes = await request.post(`${API_URL}/api/v1/auth/login`, {
    data: { email, password },
  })
  expect(loginRes.ok()).toBeTruthy()
  const loginBody = await loginRes.json()

  return {
    username,
    email,
    password,
    accessToken: loginBody.accessToken,
    userId: regBody.id,
  }
}

/** Authenticated API helper. */
function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}` }
}

/** Extended test fixture with a pre-authenticated GM user. */
export const test = base.extend<{ gmUser: TestUser }>({
  gmUser: async ({ request }, use) => {
    const user = await createTestUser(request)
    await use(user)
  },
})

export { expect, createTestUser, authHeaders, API_URL, type TestUser }
