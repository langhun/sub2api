import ActiveProbeSettingsExtension from './ActiveProbeSettingsExtension.vue'
import ActiveProbeSourceBadge from './ActiveProbeSourceBadge.vue'

export const channelMonitorV2SettingsExtensions = [
  { id: 'channel-monitor-v2-active-probes', component: ActiveProbeSettingsExtension },
] as const

export const channelMonitorV2SourceExtensions = [
  { id: 'channel-monitor-v2-active-probes', component: ActiveProbeSourceBadge },
] as const
