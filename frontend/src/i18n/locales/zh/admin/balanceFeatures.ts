export default {
  settings: {
    tabs: { balanceFeatures: '余额功能' },
    balanceFeatures: {
      checkinTitle: '签到设置', checkinDescription: '配置普通签到和幸运签到奖励', normalCheckin: '启用普通签到', luckCheckin: '启用幸运签到',
      minReward: '最小奖励', maxReward: '最大奖励', minMultiplier: '最小倍率', maxMultiplier: '最大倍率',
      transferTitle: '转账与红包', transferDescription: '配置用户余额流转限制与红包规则', transferEnabled: '启用余额转账', feeRate: '手续费率',
      minAmount: '单笔最小金额', maxAmount: '单笔最大金额', dailyLimit: '每日金额上限', dailyCount: '每日次数上限', vipExempt: 'VIP 免手续费',
      redpacketEnabled: '启用红包', redpacketMaxCount: '红包最大份数', redpacketExpireHours: '红包有效期（小时）'
    },
    checkin: {
      blindboxTitle: '签到盲盒', blindboxDescription: '配置触发规则并维护盲盒奖池', blindboxTriggerType: '触发方式', blindboxTriggerStreak: '连续签到',
      blindboxTriggerTotal: '累计签到', blindboxInterval: '触发间隔', blindboxIntervalHint: '达到指定签到天数时触发盲盒'
    }
  },
  blindbox: {
    createItem: '新增奖品', editItem: '编辑奖品', totalItems: '奖品总数', enabledItems: '已启用', totalDraws: '累计抽取', empty: '暂无奖品',
    colName: '名称', colRarity: '稀有度', colRewardType: '奖励类型', colReward: '奖励', colWeight: '权重', colStatus: '状态', colActions: '操作',
    namePlaceholder: '奖品名称', rewardBalance: '余额', rewardConcurrency: '并发', rewardSubscription: '订阅', rewardInvitation: '邀请码', minValue: '最小值', maxValue: '最大值',
    concurrencyValue: '并发值', subscriptionGroup: '订阅分组', selectGroup: '选择分组', subscriptionDays: '订阅天数', weightHint: '权重越高越容易抽中', days: '天', confirmDelete: '确定删除此奖品吗？'
  }
}
