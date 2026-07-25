import { h } from 'vue'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import type { CustomNavigationItem } from '../../registry'

const CheckinIcon = {
  render: () => h(
    'svg',
    { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
    [
      h('path', { d: 'M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 012.25-2.25h13.5A2.25 2.25 0 0121 7.5v11.25m-18 0A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75m-18 0v-7.5A2.25 2.25 0 015.25 9h13.5A2.25 2.25 0 0121 11.25v7.5', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
      h('path', { d: 'm8.25 15 2.25 2.25 5.25-5.25', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
    ],
  ),
}

export const activityNavigation: readonly CustomNavigationItem[] = [
  {
    path: '/checkin',
    label: 'Check-in',
    labelKey: 'nav.checkin',
    icon: CheckinIcon,
    hideInSimpleMode: true,
    slot: 'after-affiliate',
    featureFlag: () => (
      isFeatureFlagEnabled(FeatureFlags.checkin)
      || isFeatureFlagEnabled(FeatureFlags.checkinLuck)
    ),
  },
]
