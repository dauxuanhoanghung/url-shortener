import type { SSEUrlDeletedEvent } from '../types'

type UrlDeletedHandler = (event: SSEUrlDeletedEvent) => void

let es: EventSource | null = null
const handlers: UrlDeletedHandler[] = []

export const sseService = {
  connect(token: string) {
    if (es) return
    es = new EventSource(`/api/v1/events?token=${encodeURIComponent(token)}`)

    es.addEventListener('url_deleted', (e: MessageEvent) => {
      try {
        const payload = JSON.parse(e.data) as SSEUrlDeletedEvent
        handlers.forEach((h) => h(payload))
      } catch {
        // malformed event — ignore
      }
    })

    es.onerror = () => {
      // EventSource auto-reconnects; nothing to do here.
    }
  },

  disconnect() {
    es?.close()
    es = null
  },

  onUrlDeleted(handler: UrlDeletedHandler): () => void {
    handlers.push(handler)
    return () => {
      const idx = handlers.indexOf(handler)
      if (idx !== -1) handlers.splice(idx, 1)
    }
  },
}
