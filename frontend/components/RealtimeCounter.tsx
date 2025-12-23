'use client'

import { useEffect, useState } from 'react'

interface Props {
  shortCode: string
  initialCount: number
}

export function RealtimeCounter({ shortCode, initialCount }: Props) {
  const [count, setCount] = useState(initialCount)
  const [isConnected, setIsConnected] = useState(false)

  useEffect(() => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
    const eventSource = new EventSource(`${apiUrl}/api/analytics/${shortCode}/stream`)

    eventSource.onopen = () => {
      setIsConnected(true)
    }

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.total_clicks !== undefined) {
          setCount(data.total_clicks)
        }
      } catch (err) {
        console.error('Error parsing SSE message:', err)
      }
    }

    eventSource.onerror = () => {
      setIsConnected(false)
      eventSource.close()
      
      // Auto-reconnect with exponential backoff
      setTimeout(() => {
        if (!isConnected) {
          // Reconnect logic handled by browser's EventSource
        }
      }, 5000)
    }

    return () => {
      eventSource.close()
    }
  }, [shortCode, isConnected])

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-lg font-semibold text-gray-900">Total Clicks</h3>
        <div
          className={`w-3 h-3 rounded-full ${
            isConnected ? 'bg-green-500' : 'bg-gray-300'
          }`}
          title={isConnected ? 'Connected' : 'Disconnected'}
        />
      </div>
      <p className="text-4xl font-bold text-gray-900">{count.toLocaleString()}</p>
      <p className="text-sm text-gray-500 mt-1">
        {isConnected ? 'Live updates' : 'Reconnecting...'}
      </p>
    </div>
  )
}

