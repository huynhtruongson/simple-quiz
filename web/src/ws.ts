import type { IncomingWsMessage, OutgoingWsMessage } from './types'

export interface WsClientHandlers {
  onMessage: (message: OutgoingWsMessage) => void
  onClose: () => void
  onError: () => void
}

export class QuizWsClient {
  private socket: WebSocket

  constructor(roomID: string, handlers: WsClientHandlers) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsURL = `${protocol}//${window.location.host}/ws/rooms/${encodeURIComponent(roomID)}`
    this.socket = new WebSocket(wsURL)

    this.socket.addEventListener('message', (event) => {
      try {
        const message = JSON.parse(event.data) as OutgoingWsMessage
        handlers.onMessage(message)
      } catch {
        // Ignore malformed frames to keep UI resilient.
      }
    })

    this.socket.addEventListener('close', () => handlers.onClose())
    this.socket.addEventListener('error', () => handlers.onError())
  }

  onOpen(callback: () => void): void {
    this.socket.addEventListener('open', callback, { once: true })
  }

  send(message: IncomingWsMessage): void {
    if (this.socket.readyState !== WebSocket.OPEN) {
      return
    }
    this.socket.send(JSON.stringify(message))
  }

  close(): void {
    if (this.socket.readyState === WebSocket.OPEN || this.socket.readyState === WebSocket.CONNECTING) {
      this.socket.close()
    }
  }
}
