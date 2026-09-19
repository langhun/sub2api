import { apiClient } from '@/api/client'

export interface ActiveProbeConfig { enabled: boolean }

const path = '/custom/admin/channel-monitor-v2-active-probes/config'

export async function getActiveProbeConfig(): Promise<ActiveProbeConfig> {
  const { data } = await apiClient.get<ActiveProbeConfig>(path)
  return data
}

export async function updateActiveProbeConfig(config: ActiveProbeConfig): Promise<ActiveProbeConfig> {
  const { data } = await apiClient.put<ActiveProbeConfig>(path, config)
  return data
}
