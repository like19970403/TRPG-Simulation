import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { GameWebSocket } from './websocket'

// Mock WebSocket
class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  readyState = MockWebSocket.CONNECTING
  url: string
  onopen: ((ev: Event) => void) | null = null
  onclose: ((ev: CloseEvent) => void) | null = null
  onmessage: ((ev: MessageEvent) => void) | null = null
  onerror: ((ev: Event) => void) | null = null
  send = vi.fn()
  close = vi.fn()

  constructor(url: string) {
    this.url = url
    // Simulate async open
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN
      this.onopen?.(new Event('open'))
    }, 0)
  }

  simulateMessage(data: string) {
    this.onmessage?.(new MessageEvent('message', { data }))
  }

  simulateClose(code = 1006) {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.({ code } as CloseEvent)
  }
}

let instances: MockWebSocket[] = []

beforeEach(() => {
  instances = []
  // Create a proper constructor class to replace global WebSocket
  const OrigMock = MockWebSocket
  class TrackedWebSocket extends OrigMock {
    constructor(url: string) {
      super(url)
      instances.push(this)
    }
  }
  Object.defineProperty(TrackedWebSocket, 'OPEN', { value: 1 })
  Object.defineProperty(TrackedWebSocket, 'CONNECTING', { value: 0 })
  Object.defineProperty(TrackedWebSocket, 'CLOSING', { value: 2 })
  Object.defineProperty(TrackedWebSocket, 'CLOSED', { value: 3 })
  vi.stubGlobal('WebSocket', TrackedWebSocket)
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('GameWebSocket', () => {
  it('constructs correct WebSocket URL with token and last_event_seq', () => {
    const ws = new GameWebSocket(
      'session-123',
      () => 'jwt-token-abc',
      () => 42,
    )
    ws.connect()

    expect(instances).toHaveLength(1)
    const url = instances[0].url
    expect(url).toContain('/api/v1/sessions/session-123/ws')
    expect(url).toContain('token=jwt-token-abc')
    expect(url).toContain('last_event_seq=42')
  })

  it('calls onMessage with parsed envelope on incoming data', async () => {
    const ws = new GameWebSocket(
      'session-123',
      () => 'token',
      () => 0,
    )
    const onMessage = vi.fn()
    ws.onMessage = onMessage
    ws.connect()

    // Wait for open
    await vi.advanceTimersByTimeAsync(0)

    const envelope = {
      type: 'state_sync',
      session_id: 'session-123',
      sender_id: '',
      payload: { status: 'active' },
      timestamp: 1000,
    }
    instances[0].simulateMessage(JSON.stringify(envelope))

    expect(onMessage).toHaveBeenCalledTimes(1)
    expect(onMessage).toHaveBeenCalledWith(envelope)
  })

  it('schedules reconnect on unexpected close but not on manual disconnect', async () => {
    const ws = new GameWebSocket(
      'session-123',
      () => 'token',
      () => 0,
    )
    ws.connect()
    await vi.advanceTimersByTimeAsync(0)

    expect(instances).toHaveLength(1)

    // Simulate unexpected close
    instances[0].simulateClose(1006)

    // Advance timer past reconnect delay (1s initial)
    await vi.advanceTimersByTimeAsync(1000)

    // Should have created a second WebSocket (reconnect)
    expect(instances).toHaveLength(2)

    // Now manually disconnect
    ws.disconnect()

    // Simulate close on the second connection
    instances[1].simulateClose(1000)

    // Advance timer — should NOT reconnect
    await vi.advanceTimersByTimeAsync(5000)
    expect(instances).toHaveLength(2) // No new connection
  })
})
