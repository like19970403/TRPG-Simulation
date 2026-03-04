import { test, expect, API_URL, createTestUser, authHeaders } from '../fixtures/auth'

const sampleContent = {
  title: 'Session Test Scenario',
  start_scene: 'start',
  scenes: [
    {
      id: 'start',
      name: 'Start',
      content: 'Begin.',
      transitions: [{ target: 'end', trigger: 'gm_decision', label: 'End' }],
    },
    { id: 'end', name: 'End', content: 'Done.' },
  ],
  items: [],
  npcs: [],
  variables: [],
}

test.describe('Session Lifecycle', () => {
  test('create session → join → start → pause → resume → end', async ({ request, gmUser }) => {
    const gmHeaders = authHeaders(gmUser.accessToken)

    // Create and publish scenario
    const scRes = await request.post(`${API_URL}/api/v1/scenarios`, {
      headers: gmHeaders,
      data: { title: 'Lifecycle Test', description: '', content: sampleContent },
    })
    const scenario = await scRes.json()
    await request.post(`${API_URL}/api/v1/scenarios/${scenario.id}/publish`, {
      headers: gmHeaders,
    })

    // Create session
    const sessRes = await request.post(`${API_URL}/api/v1/sessions`, {
      headers: gmHeaders,
      data: { scenarioId: scenario.id },
    })
    expect(sessRes.status()).toBe(201)
    const session = await sessRes.json()
    expect(session.status).toBe('lobby')
    expect(session.inviteCode).toBeTruthy()

    // Player joins
    const player = await createTestUser(request)
    const playerHeaders = authHeaders(player.accessToken)
    const joinRes = await request.post(`${API_URL}/api/v1/sessions/join`, {
      headers: playerHeaders,
      data: { inviteCode: session.inviteCode },
    })
    expect(joinRes.ok()).toBeTruthy()

    // List players
    const playersRes = await request.get(`${API_URL}/api/v1/sessions/${session.id}/players`, {
      headers: gmHeaders,
    })
    expect(playersRes.ok()).toBeTruthy()
    const { players } = await playersRes.json()
    expect(players.length).toBe(1)
    expect(players[0].userId).toBe(player.userId)

    // Start session
    const startRes = await request.post(`${API_URL}/api/v1/sessions/${session.id}/start`, {
      headers: gmHeaders,
    })
    expect(startRes.ok()).toBeTruthy()
    const started = await startRes.json()
    expect(started.status).toBe('active')

    // Pause
    const pauseRes = await request.post(`${API_URL}/api/v1/sessions/${session.id}/pause`, {
      headers: gmHeaders,
    })
    expect(pauseRes.ok()).toBeTruthy()
    expect((await pauseRes.json()).status).toBe('paused')

    // Resume
    const resumeRes = await request.post(`${API_URL}/api/v1/sessions/${session.id}/resume`, {
      headers: gmHeaders,
    })
    expect(resumeRes.ok()).toBeTruthy()
    expect((await resumeRes.json()).status).toBe('active')

    // End
    const endRes = await request.post(`${API_URL}/api/v1/sessions/${session.id}/end`, {
      headers: gmHeaders,
    })
    expect(endRes.ok()).toBeTruthy()
    expect((await endRes.json()).status).toBe('completed')
  })

  test('remove player from session', async ({ request, gmUser }) => {
    const gmHeaders = authHeaders(gmUser.accessToken)

    // Setup scenario + session
    const scRes = await request.post(`${API_URL}/api/v1/scenarios`, {
      headers: gmHeaders,
      data: { title: 'Remove Test', description: '', content: sampleContent },
    })
    const scenario = await scRes.json()
    await request.post(`${API_URL}/api/v1/scenarios/${scenario.id}/publish`, {
      headers: gmHeaders,
    })
    const sessRes = await request.post(`${API_URL}/api/v1/sessions`, {
      headers: gmHeaders,
      data: { scenarioId: scenario.id },
    })
    const session = await sessRes.json()

    // Player joins
    const player = await createTestUser(request)
    await request.post(`${API_URL}/api/v1/sessions/join`, {
      headers: authHeaders(player.accessToken),
      data: { inviteCode: session.inviteCode },
    })

    // GM removes player
    const removeRes = await request.delete(
      `${API_URL}/api/v1/sessions/${session.id}/players/${player.userId}`,
      { headers: gmHeaders },
    )
    expect(removeRes.status()).toBe(204)

    // Verify player list empty
    const playersRes = await request.get(`${API_URL}/api/v1/sessions/${session.id}/players`, {
      headers: gmHeaders,
    })
    const { players } = await playersRes.json()
    expect(players.length).toBe(0)
  })

  test('delete session', async ({ request, gmUser }) => {
    const gmHeaders = authHeaders(gmUser.accessToken)

    const scRes = await request.post(`${API_URL}/api/v1/scenarios`, {
      headers: gmHeaders,
      data: { title: 'Delete Test', description: '', content: sampleContent },
    })
    const scenario = await scRes.json()
    await request.post(`${API_URL}/api/v1/scenarios/${scenario.id}/publish`, {
      headers: gmHeaders,
    })
    const sessRes = await request.post(`${API_URL}/api/v1/sessions`, {
      headers: gmHeaders,
      data: { scenarioId: scenario.id },
    })
    const session = await sessRes.json()

    const delRes = await request.delete(`${API_URL}/api/v1/sessions/${session.id}`, {
      headers: gmHeaders,
    })
    expect(delRes.status()).toBe(204)
  })
})
