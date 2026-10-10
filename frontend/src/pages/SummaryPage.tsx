import { useEffect, useState } from 'react'
import { getSummary, getWorstAverageDelays, type LeaderboardEntry, type ServiceStats } from '../api/stats'
import { onTimePercent } from '../lib/services'
import { Ticker } from '../components/Ticker'
import { Masthead } from '../components/Masthead'
import { StatTiles } from '../components/StatTiles'
import { ServiceBreakdown } from '../components/ServiceBreakdown'
import { Leaderboard } from '../components/Leaderboard'
import { Footer } from '../components/Footer'

export function SummaryPage() {
  const [breakdown, setBreakdown] = useState<ServiceStats[] | null>(null)
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    Promise.all([getSummary(), getWorstAverageDelays()])
      .then(([summary, worstAverage]) => {
        setBreakdown(summary.breakdown)
        setLeaderboard(worstAverage)
      })
      .catch((err: Error) => setError(err.message))
  }, [])

  if (error) {
    return <p className="wrap">Failed to load summary: {error}</p>
  }

  if (!breakdown || !leaderboard) {
    return <p className="wrap">Loading summary...</p>
  }

  const total = breakdown.find((row) => row.service_type === 'TOTAL_SYSTEM')
  const services = breakdown.filter((row) => row.service_type !== 'TOTAL_SYSTEM')
  const pontualidade = total ? onTimePercent(total.on_time_count, total.total_trips) : 0
  const worstService = services.reduce<ServiceStats | null>(
    (worst, row) => (!worst || row.avg_delay > worst.avg_delay ? row : worst),
    null,
  )

  return (
    <>
      <Ticker
        totalTrips={total?.total_trips ?? 0}
        onTimePercent={pontualidade}
        worstService={worstService ? { code: worstService.service_type, avgDelay: worstService.avg_delay } : null}
      />
      <div className="wrap">
        <Masthead />
        {total && <StatTiles total={total} />}
        <div className="grid-2">
          <ServiceBreakdown services={services} />
          <Leaderboard entries={leaderboard} />
        </div>
        <Footer />
      </div>
    </>
  )
}
