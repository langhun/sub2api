import { h } from 'vue'
import type { CustomNavigationItem } from '../../registry'

const AccountDrainIcon = {
  render: () => h(
    'svg',
    { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
    [
      h('path', { d: 'M4 19V5m0 14h16M8 16l3-4 3 2 4-6', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
      h('path', { d: 'M15 8h3v3', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
    ],
  ),
}

export const accountDrainNavigation: readonly CustomNavigationItem[] = [
  {
    path: '/admin/account-drain',
    label: '账号定向消耗',
    icon: AccountDrainIcon,
    hideInSimpleMode: true,
    section: 'admin',
  },
]
