import { useEffect, useRef, useCallback, useState } from 'react'

export function useWebSocket(url, options = {}) {
  const [connected, setConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState(null)
  const [error, setError] = useState(null)
  const wsRef = useRef(null)
  const reconnectTimeoutRef = useRef(null)
  const reconnectAttemptsRef = useRef(0)
  const mountedRef = useRef(true)

  const maxReconnectAttempts = options.maxReconnectAttempts || 10
  const reconnectInterval = options.reconnectInterval || 3000

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return

    try {
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        if (!mountedRef.current) return
        setConnected(true)
        setError(null)
        reconnectAttemptsRef.current = 0
        options.onOpen?.()
      }

      ws.onmessage = (event) => {
        if (!mountedRef.current) return
        try {
          const data = JSON.parse(event.data)
          setLastMessage(data)
          options.onMessage?.(data)
        } catch {
          setLastMessage(event.data)
          options.onMessage?.(event.data)
        }
      }

      ws.onerror = (err) => {
        if (!mountedRef.current) return
        setError('WebSocket error')
        options.onError?.(err)
      }

      ws.onclose = (event) => {
        if (!mountedRef.current) return
        setConnected(false)
        options.onClose?.(event)

        if (reconnectAttemptsRef.current < maxReconnectAttempts) {
          const delay = Math.min(
            reconnectInterval * Math.pow(1.5, reconnectAttemptsRef.current),
            30000
          )
          reconnectAttemptsRef.current++

          reconnectTimeoutRef.current = setTimeout(() => {
            connect()
          }, delay)
        }
      }
    } catch (err) {
      if (mountedRef.current) {
        setError(err.message)
      }
    }
  }, [url, maxReconnectAttempts, reconnectInterval])

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    if (wsRef.current) {
      wsRef.current.close()
    }
  }, [])

  const send = useCallback((data) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data))
    }
  }, [])

  useEffect(() => {
    mountedRef.current = true
    connect()
    return () => {
      mountedRef.current = false
      disconnect()
    }
  }, [connect, disconnect])

  return { connected, lastMessage, error, send, disconnect, connect }
}
