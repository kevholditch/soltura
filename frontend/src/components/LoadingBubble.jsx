import { useState, useEffect } from 'react'

export default function LoadingBubble() {
  const [seconds, setSeconds] = useState(0)

  useEffect(() => {
    const interval = setInterval(() => setSeconds(s => s + 1), 1000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div className="agent-bubble loading-bubble">
      <span className="loading-dots">
        <span>.</span><span>.</span><span>.</span>
      </span>
      <span className="thinking-timer">{seconds}s</span>
    </div>
  )
}
