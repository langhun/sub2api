export default {
  settings: {
    tabs: { balanceFeatures: 'Balance Features' },
    balanceFeatures: {
      checkinTitle: 'Check-in Settings', checkinDescription: 'Configure normal and lucky check-in rewards', normalCheckin: 'Enable normal check-in', luckCheckin: 'Enable lucky check-in',
      minReward: 'Minimum reward', maxReward: 'Maximum reward', minMultiplier: 'Minimum multiplier', maxMultiplier: 'Maximum multiplier',
      transferTitle: 'Transfers and Red Packets', transferDescription: 'Configure balance transfer limits and red-packet rules', transferEnabled: 'Enable balance transfers', feeRate: 'Fee rate',
      minAmount: 'Minimum amount', maxAmount: 'Maximum amount', dailyLimit: 'Daily amount limit', dailyCount: 'Daily transfer count', vipExempt: 'No fees for VIP users',
      redpacketEnabled: 'Enable red packets', redpacketMaxCount: 'Maximum packet count', redpacketExpireHours: 'Expiration (hours)'
    },
    checkin: {
      blindboxTitle: 'Check-in Blind Box', blindboxDescription: 'Configure triggers and maintain the prize pool', blindboxTriggerType: 'Trigger type', blindboxTriggerStreak: 'Consecutive check-ins',
      blindboxTriggerTotal: 'Total check-ins', blindboxInterval: 'Trigger interval', blindboxIntervalHint: 'Open a blind box after the configured number of check-ins'
    }
  },
  blindbox: {
    createItem: 'Add prize', editItem: 'Edit prize', totalItems: 'Total prizes', enabledItems: 'Enabled', totalDraws: 'Total draws', empty: 'No prizes',
    colName: 'Name', colRarity: 'Rarity', colRewardType: 'Reward type', colReward: 'Reward', colWeight: 'Weight', colStatus: 'Status', colActions: 'Actions',
    namePlaceholder: 'Prize name', rewardBalance: 'Balance', rewardConcurrency: 'Concurrency', rewardSubscription: 'Subscription', rewardInvitation: 'Invitation code', minValue: 'Minimum value', maxValue: 'Maximum value',
    concurrencyValue: 'Concurrency value', subscriptionGroup: 'Subscription group', selectGroup: 'Select a group', subscriptionDays: 'Subscription days', weightHint: 'Higher weights are drawn more often', days: 'days', confirmDelete: 'Delete this prize?',
    deliveryTitle: 'Reward delivery operations', deliveryDescription: 'Inspect blind-box delivery state and handle jobs that exhausted automatic retries.', deliveryStatusFilter: 'Delivery status', deliveryAllStatuses: 'All statuses',
    deliveryStatus: { pending: 'Pending', delivering: 'Delivering', delivered: 'Delivered', failed: 'Failed', compensated: 'Compensated' },
    deliveryEmpty: 'No deliveries match the current filter', deliveryColSource: 'Source', deliveryColUser: 'User', deliveryColReward: 'Reward', deliveryColStatus: 'Delivery status', deliveryColAttempts: 'Attempts', deliveryColTime: 'Updated',
    deliveryRetry: 'Retry delivery', deliveryRetryTitle: 'Retry this delivery?', deliveryRetryMessage: 'The job will return to the pending queue for the background worker to process again.', deliveryRetryFailed: 'Could not retry the delivery',
    deliveryCompensate: 'Mark compensated', deliveryCompensateTitle: 'Record manual compensation', deliveryCompensateMessage: 'Use this only after compensation was completed elsewhere. The system will not grant the reward again.', deliveryCompensateReason: 'Compensation note', deliveryConfirmCompensate: 'Confirm compensation', deliveryCompensateFailed: 'Could not record compensation',
    deliveryLoadFailed: 'Could not load reward deliveries', deliveryTotal: '{total} deliveries'
  },
  codeFormat: {
    title: 'Code and phrase formats', description: 'Configure newly generated redemption, invitation, and packet codes. Existing codes remain valid.', balance: 'Balance codes', concurrency: 'Concurrency codes', subscription: 'Subscription codes', invitation: 'Invitation codes', redpacket: 'Packet phrases', prefix: 'Prefix', characterSet: 'Characters', separator: 'Separator', custom: 'Custom', customSeparator: 'Custom character', combinationSpace: 'Available combinations: {count}', groupLength: 'Group size', groupCount: 'Groups', length: 'Example length {length}', none: 'None', uppercase: 'Uppercase', numeric: 'Numeric', alphanumeric: 'Alphanumeric', hex: 'Hexadecimal', invalidCharset: 'Select a valid character set', invalidGroupLength: 'Group size must be an integer from 1 to 32', invalidGroupCount: 'Groups must be an integer from 1 to 16', invalidSeparator: 'Separator must be one printable ASCII character', invalidPrefix: 'Prefix must be printable ASCII and cannot contain the separator'
  }
}
