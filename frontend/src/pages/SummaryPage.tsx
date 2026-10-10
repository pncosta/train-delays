import { useEffect, useState } from 'react'
import { getSummary, type ServiceStats } from '../api/stats'

export function SummaryPage() {
  const [breakdown, setBreakdown] = useState<ServiceStats[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getSummary()
      .then((data) => setBreakdown(data.breakdown))
      .catch((err: Error) => setError(err.message))
  }, [])

  if (error) {
    return <p>Failed to load summary: {error}</p>
  }

  if (!breakdown) {
    return <p>Loading summary...</p>
  }

  return (
    <table>
      <thead>
        <tr>
          <th>Service</th>
          <th>Trips</th>
          <th>Avg delay (min)</th>
          <th>On time</th>
          <th>Delayed</th>
          <th>Cancelled</th>
        </tr>
      </thead>
      <tbody>
        {breakdown.map((row) => (
          <tr key={row.service_type}>
            <td>{row.service_type}</td>
            <td>{row.total_trips}</td>
            <td>{row.avg_delay.toFixed(1)}</td>
            <td>{row.on_time_count}</td>
            <td>{row.delayed_count}</td>
            <td>{row.cancelled_count}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
