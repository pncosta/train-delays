import { serviceColor } from '../lib/services'

interface ServiceBadgeProps {
  code: string
}

export function ServiceBadge({ code }: ServiceBadgeProps) {
  return (
    <span className="service-badge upper" style={{ background: serviceColor(code) }}>
      {code}
    </span>
  )
}
