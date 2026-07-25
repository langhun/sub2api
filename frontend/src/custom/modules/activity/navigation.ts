import { h } from 'vue'
import { isFeatureFlagEnabled, makeSidebarFlag } from '@/utils/featureFlags'
import type { CustomNavigationItem } from '../../registry'
import { activityFeatureFlags } from './settings'
import { walletExtensionFeatureFlags } from '../wallet-extension/settings'

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

const RedPacketIcon = {
  render: () => h(
    'svg',
    { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
    [
      h('path', { d: 'M20.25 12.75v5.625a2.25 2.25 0 0 1-2.25 2.25H6a2.25 2.25 0 0 1-2.25-2.25V12.75m16.5 0V8.625A2.25 2.25 0 0 0 18 6.375h-1.5a4.5 4.5 0 0 0-9 0H6a2.25 2.25 0 0 0-2.25 2.25v4.125m16.5 0H3.75m8.25-6.375v14.25m0-14.25H9.75', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
    ],
  ),
}

const LeaderboardIcon = {
  render: () => h(
    'svg',
    { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
    [
      h('path', { d: 'M3.75 19.5h16.5M6.75 16.5v-4.125a.375.375 0 0 1 .375-.375h2.25a.375.375 0 0 1 .375.375V16.5m0 0V8.625a.375.375 0 0 1 .375-.375h2.25a.375.375 0 0 1 .375.375V16.5m0 0v-10.5a.375.375 0 0 1 .375-.375h2.25a.375.375 0 0 1 .375.375V16.5', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
    ],
  ),
}

function isLeaderboardEnabled() {
  return isFeatureFlagEnabled(activityFeatureFlags.leaderboard) && (
    isFeatureFlagEnabled(activityFeatureFlags.leaderboardBalance)
    || isFeatureFlagEnabled(activityFeatureFlags.leaderboardConsumption)
    || isFeatureFlagEnabled(activityFeatureFlags.leaderboardCheckin)
    || (isFeatureFlagEnabled(activityFeatureFlags.leaderboardTransfer) && isFeatureFlagEnabled(walletExtensionFeatureFlags.transfer))
  )
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
      isFeatureFlagEnabled(activityFeatureFlags.checkin)
      || isFeatureFlagEnabled(activityFeatureFlags.checkinLuck)
    ),
  },
  {
    path: '/redpacket',
    label: 'Red Packets',
    labelKey: 'nav.redpacket',
    icon: RedPacketIcon,
    hideInSimpleMode: true,
    slot: 'after-transfer',
    featureFlag: makeSidebarFlag(walletExtensionFeatureFlags.redpacket),
  },
  {
    path: '/leaderboard',
    label: 'Leaderboard',
    labelKey: 'nav.leaderboard',
    icon: LeaderboardIcon,
    hideInSimpleMode: true,
    slot: 'after-transfer',
    featureFlag: isLeaderboardEnabled,
  },
]
