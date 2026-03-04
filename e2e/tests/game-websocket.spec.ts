import { test, expect, API_URL, createTestUser, authHeaders } from '../fixtures/auth'
import WebSocket from 'ws'

const sampleContent = {
  title: 'WS Test Scenario',
  start_scene: 'room1',
  scenes: [
    {
      id: 'room1',
      name: 'Room 1',
      content: 'A dark room.',
      transitions: [{ target: 'room2', trigger: 'gm_decision', label: 'Go to Room 2' }],
    },
    { id: 'room2', name: 'Room 2', content: 'A bright room.' },
  ],
  items: [{ id: 'key', name: 'Key', type: 'item', description: 'A rusty key' }],
  npcs: [],
  variables: [{ name: 'found_key', type: 'bool', default: false }],
}

/** Helper: open a WebSocket connection and wait for state_sync. */
function connectWS(
  sessionId: string,
  token: string,
): Promise<{ ws: WebSocket; stateSync: unknown }> {
  return new Promise((resolve, reject) => {
    const wsUrl = `ws://localhost:8080/api/v1/sessions/${sessionId}/ws?token=${token}`
    const ws = new WebSocket(wsUrl)
    const timeout = setTimeout(() => {
      ws.close()
      reject(new Error('WS connect timeout'))
    }, 10_000)

    ws.on('message', (data) => {
      const msg = JSON.parse(data.toString())
      if (msg.type === 'state_sync') {
        clearTimeout(timeout)
        resolve({ ws, stateSync: msg.payload })
      }
    })
    ws.on('error', (err) => {
      clearTimeout(timeout)
      reject(err)
    })
  })
}

/** Helper: wait for a specific event type on a WS connection. */
function waitForEvent(ws: WebSocket, eventType: string, timeoutMs = 5000): Promise<unknown> {
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => reject(new Error(`Timeout waiting for ${eventType}`)), timeoutMs)
    const handler = (data: WebSocket.Data) => {
      const msg = JSON.parse(data.toString())
      if (msg.type === eventType) {
        clearTimeout(timeout)
        ws.removeListener('message', handler)
        resolve(msg.payload)
      }
    }
    ws.on('message', handler)
  })
}

test.describe('Game WebSocket', () => {
  test('connect → state_sync → advance_scene → dice_roll', async ({ request, gmUser }) => {
    const gmHeaders = authHeaders(gmUser.accessToken)

    // Setup
    const scRes = await request.post(`${API_URL}/api/v1/scenarios`, {
      headers: gmHeaders,
      data: { title: 'WS Test', description: '', content: sampleContent },
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

    // Start session
    await request.post(`${API_URL}/api/v1/sessions/${session.id}/start`, {
      headers: gmHeaders,
    })

    // Connect GM via WebSocket
    const { ws: gmWs, stateSync } = await connectWS(session.id, gmUser.accessToken)
    expect(stateSync).toBeTruthy()

    try {
      // Advance scene
      const sceneChangedPromise = waitForEvent(gmWs, 'scene_changed')
      gmWs.send(JSON.stringify({ type: 'advance_scene', payload: { scene_id: 'room2' } }))
      const scenePayload = (await sceneChangedPromise) as { scene_id: string }
      expect(scenePayload.scene_id).toBe('room2')

      // Dice roll
      const dicePromise = waitForEvent(gmWs, 'dice_rolled')
      gmWs.send(JSON.stringify({ type: 'dice_roll', payload: { formula: '2d6', purpose: 'E2E test' } }))
      const dicePayload = (await dicePromise) as { total: number; results: number[] }
      expect(dicePayload.total).toBeGreaterThanOrEqual(2)
      expect(dicePayload.total).toBeLessThanOrEqual(12)
    } finally {
      gmWs.close()
    }
  })
})
