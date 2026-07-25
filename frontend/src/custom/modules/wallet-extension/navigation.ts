import { h } from 'vue'
import { FeatureFlags, makeSidebarFlag } from '@/utils/featureFlags'
import type { CustomNavigationItem } from '../../registry'

const DirectTransferIcon = {
  render: () => h(
    'svg',
    { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
    [
      h('rect', { x: '3', y: '5', width: '18', height: '14', rx: '2', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
      h('path', { d: 'M7 10h.01M7 14h3m7-4h.01', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
    ],
  ),
}

export const walletExtensionNavigation: readonly CustomNavigationItem[] = [
  {
    path: '/transfer',
    label: 'Balance Transfer',
    labelKey: 'nav.transfer',
    icon: DirectTransferIcon,
    hideInSimpleMode: true,
    slot: 'after-affiliate',
    featureFlag: makeSidebarFlag(FeatureFlags.transfer),
  },
]
