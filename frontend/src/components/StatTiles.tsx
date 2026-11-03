import type { ServiceStats } from '../api/stats'
import { fmtPt, fmtPtInt, onTimePercent } from '../lib/services'

interface StatTilesProps {
  total: ServiceStats
}

export function StatTiles({ total }: StatTilesProps) {
  const pontualidade = onTimePercent(total.on_time_count, total.total_trips)

  return (
    <section className="stats">
      <div className="stat">
        <span className="stat__label upper">viagens · 7 dias</span>
        <span className="stat__value">{fmtPtInt(total.total_trips)}</span>
        <span className="stat__sub">todos os serviços</span>
      </div>
      <div className="stat">
        <span className="stat__label upper">pontualidade</span>
        <span className="stat__value good">{fmtPt(pontualidade)}%</span>
        <span className="stat__sub">chegou com menos de 3 min de atraso</span>
      </div>
      <div className="stat">
        <span className="stat__label upper">atraso médio</span>
        <span className="stat__value accent">
          {fmtPt(total.avg_delay)} <small style={{ fontSize: 16 }}>min</small>
        </span>
        <span className="stat__sub">por viagem, sistema inteiro</span>
      </div>
      <div className="stat">
        <span className="stat__label upper">suprimidos</span>
        <span className="stat__value">{fmtPtInt(total.cancelled_count)}</span>
        <span className="stat__sub">comboios cancelados</span>
      </div>
    </section>
  )
}
