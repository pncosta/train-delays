export interface ServiceStats {
  service_type: string
  total_trips: number
  avg_delay: number
  on_time_count: number
  delayed_count: number
  cancelled_count: number
}

export interface DashboardResponse {
  breakdown: ServiceStats[]
}

export async function getSummary(): Promise<DashboardResponse> {
  const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/stats/summary`)
  if (!response.ok) {
    throw new Error(`Server error: ${response.status}`)
  }
  const data = await response.json()
  if (!data) {
    throw new Error('Server returned an empty summary')
  }
  return data
}
