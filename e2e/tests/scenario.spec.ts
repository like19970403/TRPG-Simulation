import { test, expect, API_URL, authHeaders } from '../fixtures/auth'

const sampleContent = {
  title: 'E2E Test Scenario',
  start_scene: 'start',
  scenes: [
    {
      id: 'start',
      name: 'Start Scene',
      content: 'Welcome to the test scenario.',
      transitions: [{ target: 'end', trigger: 'gm_decision', label: 'End' }],
    },
    { id: 'end', name: 'End Scene', content: 'The end.' },
  ],
  items: [{ id: 'key', name: 'Key', type: 'item', description: 'A test key' }],
  npcs: [],
  variables: [],
}

test.describe('Scenario CRUD', () => {
  test('create → get → update → publish → archive', async ({ request, gmUser }) => {
    const headers = authHeaders(gmUser.accessToken)

    // Create
    const createRes = await request.post(`${API_URL}/api/v1/scenarios`, {
      headers,
      data: { title: 'E2E Scenario', description: 'Test', content: sampleContent },
    })
    expect(createRes.status()).toBe(201)
    const created = await createRes.json()
    expect(created.id).toBeTruthy()
    expect(created.status).toBe('draft')

    const id = created.id

    // Get
    const getRes = await request.get(`${API_URL}/api/v1/scenarios/${id}`, { headers })
    expect(getRes.ok()).toBeTruthy()
    const fetched = await getRes.json()
    expect(fetched.title).toBe('E2E Scenario')

    // Update
    const updateRes = await request.put(`${API_URL}/api/v1/scenarios/${id}`, {
      headers,
      data: { title: 'Updated Scenario', description: 'Updated', content: sampleContent },
    })
    expect(updateRes.ok()).toBeTruthy()
    const updated = await updateRes.json()
    expect(updated.title).toBe('Updated Scenario')

    // Publish
    const pubRes = await request.post(`${API_URL}/api/v1/scenarios/${id}/publish`, { headers })
    expect(pubRes.ok()).toBeTruthy()
    const published = await pubRes.json()
    expect(published.status).toBe('published')

    // Archive
    const archRes = await request.post(`${API_URL}/api/v1/scenarios/${id}/archive`, { headers })
    expect(archRes.ok()).toBeTruthy()
    const archived = await archRes.json()
    expect(archived.status).toBe('archived')
  })

  test('publish blocked when scenario has validation errors', async ({ request, gmUser }) => {
    const headers = authHeaders(gmUser.accessToken)

    // Create scenario with invalid transition target
    const badContent = {
      ...sampleContent,
      scenes: [
        {
          id: 'start',
          name: 'Start',
          content: 'Test',
          transitions: [{ target: 'nonexistent', trigger: 'gm_decision' }],
        },
      ],
    }

    const createRes = await request.post(`${API_URL}/api/v1/scenarios`, {
      headers,
      data: { title: 'Bad Scenario', description: '', content: badContent },
    })
    expect(createRes.status()).toBe(201)
    const created = await createRes.json()
    // Should have validation warnings
    expect(created.validationWarnings).toBeDefined()

    // Publish should be blocked
    const pubRes = await request.post(`${API_URL}/api/v1/scenarios/${created.id}/publish`, {
      headers,
    })
    expect(pubRes.status()).toBe(400)
    const err = await pubRes.json()
    expect(err.error).toBe('VALIDATION_ERROR')
  })
})
