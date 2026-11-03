import { useEffect, useRef, useState } from 'react'
import type { ServiceStats } from '../api/stats'
import { fmtPt, fmtPtInt, serviceName } from '../lib/services'
import { ServiceBadge } from './ServiceBadge'

interface ServiceBreakdownProps {
  services: ServiceStats[]
}

export function ServiceBreakdown({ services }: ServiceBreakdownProps) {
  const sorted = services.slice().sort((a, b) => b.total_trips - a.total_trips)
  const max = Math.max(...sorted.map((s) => s.total_trips), 1)

  const [hoverKey, setHoverKey] = useState<string | null>(null)
  const [pinnedKey, setPinnedKey] = useState<string | null>(null)
  const activeKey = pinnedKey ?? hoverKey
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function dismissOnOutsideClick(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setPinnedKey(null)
      }
    }
    function dismissOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setPinnedKey(null)
        setHoverKey(null)
      }
    }
    document.addEventListener('mousedown', dismissOnOutsideClick)
    document.addEventListener('keydown', dismissOnEscape)
    return () => {
      document.removeEventListener('mousedown', dismissOnOutsideClick)
      document.removeEventListener('keydown', dismissOnEscape)
    }
  }, [])

  return (
    <section className="panel">
      <div className="panel__head">
        <div>
          <h2 className="panel__title">Viagens por serviço</h2>
          <p className="panel__subtitle">últimos 7 dias</p>
        </div>
      </div>
      <div className="bars" ref={containerRef}>
        {sorted.map((s) => {
          const onTime = s.on_time_count
          // Cancelled trips are folded into "delayed" for this two-state split —
          // the data to break them out as a third segment is there if ever wanted.
          const rest = s.total_trips - s.on_time_count
          const onTimeKey = `${s.service_type}-ontime`
          const restKey = `${s.service_type}-rest`

          return (
            <div className="bar-row" key={s.service_type}>
              <span className="bar-row__label">
                {serviceName(s.service_type)}
                <ServiceBadge code={s.service_type} />
              </span>
              <div className="bar-row__track">
                <div className="bar-row__fill" style={{ width: `${(s.total_trips / max) * 100}%` }}>
                  {onTime > 0 && (
                    <BarSegment
                      segmentKey={onTimeKey}
                      color="var(--status-good)"
                      flexGrow={onTime}
                      tooltip={`Pontual — ${fmtPtInt(onTime)} viagens`}
                      isActive={activeKey === onTimeKey}
                      onHover={setHoverKey}
                      onTogglePin={setPinnedKey}
                    />
                  )}
                  {rest > 0 && (
                    <BarSegment
                      segmentKey={restKey}
                      color="var(--status-bad)"
                      flexGrow={rest}
                      tooltip={`Atrasado ou cancelado — ${fmtPtInt(rest)} viagens`}
                      isActive={activeKey === restKey}
                      onHover={setHoverKey}
                      onTogglePin={setPinnedKey}
                    />
                  )}
                </div>
              </div>
              <span className="bar-row__value">
                <span className="bar-row__value-total">{fmtPtInt(s.total_trips)}</span>
                <span className="bar-row__value-delay">{fmtPt(s.avg_delay)} min</span>
              </span>
            </div>
          )
        })}
      </div>
    </section>
  )
}

interface BarSegmentProps {
  segmentKey: string
  color: string
  flexGrow: number
  tooltip: string
  isActive: boolean
  onHover: (key: string | null) => void
  onTogglePin: (update: (prev: string | null) => string | null) => void
}

function BarSegment({ segmentKey, color, flexGrow, tooltip, isActive, onHover, onTogglePin }: BarSegmentProps) {
  const togglePin = () => onTogglePin((prev) => (prev === segmentKey ? null : segmentKey))

  return (
    <div
      className="bar-row__segment"
      role="button"
      tabIndex={0}
      aria-label={tooltip}
      style={{ flexGrow, background: color }}
      onMouseEnter={() => onHover(segmentKey)}
      onMouseLeave={() => onHover(null)}
      onFocus={() => onHover(segmentKey)}
      onBlur={() => onHover(null)}
      onClick={togglePin}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault()
          togglePin()
        }
      }}
    >
      {isActive && <span className="bar-row__tooltip">{tooltip}</span>}
    </div>
  )
}
