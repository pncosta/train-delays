import type { LeaderboardEntry } from '../api/stats'
import { delaySeverity, fmtPt, fmtPtInt, serviceName, stationName } from '../lib/services'
import { ServiceBadge } from './ServiceBadge'

interface LeaderboardProps {
  entries: LeaderboardEntry[]
}

export function Leaderboard({ entries }: LeaderboardProps) {
  const ranked = entries.slice(0, 5)

  return (
    <section className="panel">
      <div className="panel__head">
        <div>
          <h2 className="panel__title">Pior atraso médio</h2>
          <p className="panel__subtitle">por comboio, últimos 7 dias</p>
        </div>
      </div>
      <div className="lb">
        {ranked.map((entry, i) => {
          const severity = delaySeverity(entry.value)
          return (
            <div
              className="lb-row"
              key={`${entry.train_number}-${entry.service_type}-${entry.origin_station}-${entry.destination_station}`}
            >
              <span className="lb-row__rank">#{i + 1}</span>
              <span className="lb-row__name">
                {stationName(entry.origin_station_name)} &rarr; {stationName(entry.destination_station_name)}
                <small className="upper">
                  <ServiceBadge code={entry.service_type} /> {serviceName(entry.service_type)} · comboio{' '}
                  {entry.train_number} · {fmtPtInt(entry.count)} viagens
                </small>
              </span>
              <span className="lb-row__chip upper" style={{ color: severity.color, background: severity.background }}>
                {severity.label}
              </span>
              <span className="lb-row__delay">+{fmtPt(entry.value)} min</span>
            </div>
          )
        })}
      </div>
    </section>
  )
}
