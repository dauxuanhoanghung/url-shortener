import type { SSEUrlDeletedEvent, SSEMetadataUpdatedEvent } from '../types'

type UrlDeletedHandler = (event: SSEUrlDeletedEvent) => void
type MetadataUpdatedHandler = (event: SSEMetadataUpdatedEvent) => void

let es: EventSource | null = null
const deletedHandlers: UrlDeletedHandler[] = []
const metadataHandlers: MetadataUpdatedHandler[] = []

export const sseService = {
  connect(token: string) {
    if (es) return
    es = new EventSource(`/api/v1/events?token=${encodeURIComponent(token)}`)

    es.addEventListener('url_deleted', (e: MessageEvent) => {
      try {
        const payload = JSON.parse(e.data) as SSEUrlDeletedEvent
        deletedHandlers.forEach((h) => h(payload))
      } catch {
        // malformed event — ignore
      }
    })

    es.addEventListener('metadata_updated', (e: MessageEvent) => {
      try {
        const payload = JSON.parse(e.data) as SSEMetadataUpdatedEvent
        metadataHandlers.forEach((h) => h(payload))
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
    deletedHandlers.push(handler)
    return () => {
      const idx = deletedHandlers.indexOf(handler)
      if (idx !== -1) deletedHandlers.splice(idx, 1)
    }
  },

  onMetadataUpdated(handler: MetadataUpdatedHandler): () => void {
    metadataHandlers.push(handler)
    return () => {
      const idx = metadataHandlers.indexOf(handler)
      if (idx !== -1) metadataHandlers.splice(idx, 1)
    }
  },
}
