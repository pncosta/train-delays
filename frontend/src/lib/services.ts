// `service_type` is an unvalidated free-text column fed straight from CP's API, not
// a guaranteed enum — this covers the codes we've observed so far, with a fallback
// (see serviceName/serviceColor below) for anything new.
export const SERVICE_META: Record<string, { name: string; color: string }> = {
  AP: { name: 'Alfa Pendular', color: 'var(--svc-ap)' },
  E: { name: 'Especial', color: 'var(--svc-e)' },
  IC: { name: 'Intercidades', color: 'var(--svc-ic)' },
  IN: { name: 'Internacional', color: 'var(--svc-in)' },
  IR: { name: 'Inter-Regional', color: 'var(--svc-ir)' },
  R: { name: 'Regional', color: 'var(--svc-r)' },
  U: { name: 'Urbano', color: 'var(--svc-u)' },
}

export function serviceName(code: string): string {
  return SERVICE_META[code]?.name ?? code
}

export function serviceColor(code: string): string {
  return SERVICE_META[code]?.color ?? 'var(--ink-faint)'
}

export function fmtPt(value: number, decimals = 1): string {
  return value.toFixed(decimals).replace('.', ',')
}

export function fmtPtInt(value: number): string {
  return value.toLocaleString('pt-PT')
}

export function onTimePercent(onTimeCount: number, totalTrips: number): number {
  return totalTrips > 0 ? (onTimeCount / totalTrips) * 100 : 0
}

const MISSING_STATION_NAME = '<em falta>'

export function stationName(name: string): string {
  return name || MISSING_STATION_NAME
}

export type Severity = { color: string; background: string; label: string }

export function delaySeverity(avgDelay: number): Severity {
  if (avgDelay >= 15) {
    return { color: 'var(--status-bad)', background: 'rgba(255,71,87,0.15)', label: 'crítico' }
  }
  if (avgDelay >= 8) {
    return { color: 'var(--status-warn)', background: 'rgba(255,197,61,0.15)', label: 'atenção' }
  }
  return { color: 'var(--status-good)', background: 'rgba(57,230,122,0.15)', label: 'ok' }
}
