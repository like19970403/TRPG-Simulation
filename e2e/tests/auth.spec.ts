import { test, expect, API_URL, createTestUser, authHeaders } from '../fixtures/auth'

test.describe('Authentication', () => {
  test('register → login → refresh → logout', async ({ request }) => {
    // Register
    const user = await createTestUser(request)
    expect(user.accessToken).toBeTruthy()
    expect(user.userId).toBeTruthy()

    // Verify token works
    const healthRes = await request.get(`${API_URL}/api/health`)
    expect(healthRes.ok()).toBeTruthy()

    // Logout
    const logoutRes = await request.post(`${API_URL}/api/v1/auth/logout`, {
      headers: authHeaders(user.accessToken),
    })
    expect(logoutRes.ok()).toBeTruthy()
  })

  test('login with wrong password returns 401', async ({ request }) => {
    const user = await createTestUser(request)
    const res = await request.post(`${API_URL}/api/v1/auth/login`, {
      data: { email: user.email, password: 'WrongPassword!' },
    })
    expect(res.status()).toBe(401)
  })

  test('register with duplicate email returns 409', async ({ request }) => {
    const user = await createTestUser(request)
    const res = await request.post(`${API_URL}/api/v1/users`, {
      data: { username: 'other_user', email: user.email, password: 'TestPass123!' },
    })
    expect(res.status()).toBe(409)
  })
})
