export const walletExtensionAdminLocaleMessages = {
  en: {
    settings: {
      balanceFeatures: {
        transferTitle: 'Transfers and Red Packets', transferDescription: 'Configure balance transfer limits and red-packet rules', transferEnabled: 'Enable balance transfers', feeRate: 'Fee rate',
        minAmount: 'Minimum amount', maxAmount: 'Maximum amount', dailyLimit: 'Daily amount limit', dailyCount: 'Daily transfer count', vipExempt: 'No fees for VIP users',
        redpacketEnabled: 'Enable red packets', redpacketMaxCount: 'Maximum packet count', redpacketExpireHours: 'Expiration (hours)',
      },
    },
  },
  zh: {
    settings: {
      balanceFeatures: {
        transferTitle: '转账与红包', transferDescription: '配置用户余额流转限制与红包规则', transferEnabled: '启用余额转账', feeRate: '手续费率',
        minAmount: '单笔最小金额', maxAmount: '单笔最大金额', dailyLimit: '每日金额上限', dailyCount: '每日次数上限', vipExempt: 'VIP 免手续费',
        redpacketEnabled: '启用红包', redpacketMaxCount: '红包最大份数', redpacketExpireHours: '红包有效期（小时）',
      },
    },
  },
} as const
