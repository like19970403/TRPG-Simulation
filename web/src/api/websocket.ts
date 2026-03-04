import type { WsEnvelope, WsAction } from './types'

export type WsEventHandler = (envelope: WsEnvelope) => void

/**
 * Framework-agnostic WebSocket connection manager for game sessions.
 * Handles connection, auto-reconnect with exponential backoff, and message dispatch.
 */
export class GameWebSocket {
  private ws: WebSocket | null = null
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectAttempts = 0
  private readonly maxReconnectDelay = 30_000
  private readonly maxReconnectAttempts = 10
  private manualClose = false

  onMessage: WsEventHandler | null = null
  onOpen: (() => void) | null = null
  onClose: ((reason: string) => void) | null = null
  onError: ((error: Event) => void) | null = null
  onReconnectExhausted: (() => void) | null = null

  constructor(
    private readonly sessionId: string,
    private readonly getToken: () => string | null,
    private readonly getLastSeq: () => number,
    private readonly refreshToken?: () => Promise<string | null>,
  ) {}

  async connect(): Promise<void> {
    this.manualClose = false
    this.clearReconnectTimer()

    let token = this.getToken()

    // On reconnect attempts, try to refresh the token first
    if (this.reconnectAttempts > 0 && this.refreshToken) {
      const freshToken = await this.refreshToken()
      if (freshToken) {
        token = freshToken
      }
    }

    if (!token) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const seq = this.getLastSeq()
    const url = `${protocol}//${host}/api/v1/sessions/${this.sessionId}/ws?token=${token}&last_event_seq=${seq}`

    this.ws = new WebSocket(url)

    this.ws.onopen = () => {
      this.reconnectAttempts = 0
      this.onOpen?.()
    }

    this.ws.onmessage = (event: MessageEvent) => {
      try {
        const envelope: WsEnvelope = JSON.parse(event.data as string)
        this.onMessage?.(envelope)
      } catch {
        // Ignore malformed messages
      }
    }

    this.ws.onclose = (event: CloseEvent) => {
      this.onClose?.(event.reason || `code ${event.code}`)
      if (!this.manualClose) {
        this.scheduleReconnect()
      }
    }

    this.ws.onerror = (event: Event) => {
      this.onError?.(event)
    }
  }

  disconnect(): void {
    this.manualClose = true
    this.clearReconnectTimer()
    if (this.ws) {
      this.ws.close(1000, 'client disconnect')
      this.ws = null
    }
  }

  send(action: WsAction): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(action))
    }
  }

  get connected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.onReconnectExhausted?.()
      return
    }
    const delay = Math.min(
      1000 * Math.pow(2, this.reconnectAttempts),
      this.maxReconnectDelay,
    )
    this.reconnectAttempts++
    this.reconnectTimer = setTimeout(() => {
      this.connect()
    }, delay)
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }
}
