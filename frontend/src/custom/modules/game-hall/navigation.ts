import { h } from 'vue'
import { FeatureFlags, makeSidebarFlag } from '@/utils/featureFlags'
import type { CustomNavigationItem } from '../../registry'

const GameHallIcon = {
  render: () => h(
    'svg',
    { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
    [
      h('rect', { x: '3', y: '7', width: '18', height: '11', rx: '2', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
      h('path', { d: 'M7 12h4m-2-2v4m6-2h.01', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
    ],
  ),
}

export const gameHallNavigation: readonly CustomNavigationItem[] = [
  {
    path: '/game-hall',
    label: 'Game Hall',
    labelKey: 'gameHall.title',
    icon: GameHallIcon,
    hideInSimpleMode: true,
    featureFlag: makeSidebarFlag(FeatureFlags.gameHall),
  },
]
