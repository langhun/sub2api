export default {
  checkin: {
    title: 'Daily Check-in', checked: 'Checked in today', todayReward: 'Earned ${amount} today', streakDays: '{days}-day streak',
    rangeHint: 'Reward range ${min} - ${max}', luckTitle: 'Lucky Check-in', luckButton: 'Confirm lucky check-in',
    luckSuccess: '{multiplier}x multiplier, earned ${amount}', luckLoss: '{multiplier}x multiplier, lost ${amount}', luckEven: '1.00x multiplier, no gain or loss',
    betAmount: 'Bet amount', betAmountPlaceholder: 'Enter an amount', multiplierRange: 'Multiplier range ${min}x - ${max}x',
    checkinType: 'Check-in type', normalCheckin: 'Normal check-in', luckCheckin: 'Lucky check-in',
    blindboxCommon: 'Common', blindboxRare: 'Rare', blindboxEpic: 'Epic', blindboxLegendary: 'Legendary',
    blindboxBalanceReward: 'Balance +${value}', blindboxConcurrencyReward: 'Concurrency +{value}', blindboxSubscriptionReward: '{days}-day subscription',
    blindboxInvitationReward: 'Invitation code x1', blindboxInviteCode: 'Your invitation code', blindboxHistory: 'Blind-box prizes', blindboxHistoryDesc: 'Prizes earned from check-in blind boxes', blindboxDays: ' days',
    page: {
      description: 'Daily rewards, blind-box prizes, and account overview', balance: 'Balance', streak: 'Check-in streak', days: ' days', concurrency: 'Concurrency', blindboxCount: 'Blind boxes',
      noBlindbox: 'No blind-box prizes yet', todayResult: "Today's check-in", todayReward: 'Reward', todayMultiplier: 'Multiplier', todayBlindbox: 'Blind box',
      todayNormal: 'Normal check-in', todayLuck: 'Lucky check-in', todayNoResult: 'Not checked in today', rarityBreakdown: 'Rarity breakdown', blindboxInfo: 'Blind-box rules',
      triggerType: 'Trigger', triggerTotal: 'Total check-ins', triggerStreak: 'Check-in streak', triggerInterval: 'Interval',
      blindboxNextIn: '{days} days until the next blind box', blindboxNextCycle: 'Triggers every {interval} consecutive days', calendarTitle: 'Check-in calendar'
    }
  },
  leaderboard: {
    title: 'Leaderboard', subtitle: 'See the most active users', empty: 'No data', streakDays: '{days} days',
    balanceSubtitle: '{count} total check-ins', consumptionSubtitle: '{count} requests', checkinSubtitle: '{total} check-ins · Latest {date} · Earned ${reward}',
    tabs: { balance: 'Balance', consumption: 'Consumption', checkin: 'Check-in' }, periods: { daily: 'Daily', weekly: 'Weekly', monthly: 'Monthly' }
  },
  transfer: { title: 'Balance Transfer', totalSent: 'Total sent', totalReceived: 'Total received', totalFee: 'Fees paid', receiverId: 'Recipient user ID', amount: 'Amount', feePreview: 'Fee', total: 'Total debit', memo: 'Message', success: 'Transfer completed', submit: 'Confirm transfer', failed: 'Transfer failed' },
  redpacket: { title: 'Red Packets', create: 'Send', claim: 'Claim', myPackets: 'My red packets' }
}
