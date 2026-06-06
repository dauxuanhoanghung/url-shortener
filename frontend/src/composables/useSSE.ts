import { onUnmounted } from 'vue'

interface SSEOptions {
  /** Query-param token for EventSource auth (EventSource can't set headers). */
  token: string
  url?: string
}

/**
 * Generic SSE composable. Manages one EventSource per call site and cleans up
 * all listeners on component unmount.
 *
 * Usage:
 *   const sse = useSSE({ token })
 *   sse.on('url_deleted', (data) => { ... })
 *   sse.on('metadata_updated', (data) => { ... })
 */
export function useSSE(options: SSEOptions) {
  const endpoint = options.url ?? '/api/v1/events'
  const src = new EventSource(`${endpoint}?token=${encodeURIComponent(options.token)}`)

  const cleanups: Array<() => void> = []

  function on<T = unknown>(eventType: string, handler: (data: T) => void): () => void {
    const listener = (e: MessageEvent) => {
      try {
        handler(JSON.parse(e.data) as T)
      } catch {
        // malformed payload — ignore
      }
    }
    src.addEventListener(eventType, listener)
    const off = () => src.removeEventListener(eventType, listener)
    cleanups.push(off)
    return off
  }

  function close() {
    cleanups.forEach((off) => off())
    cleanups.length = 0
    src.close()
  }

  onUnmounted(close)

  return { on, close }
}
