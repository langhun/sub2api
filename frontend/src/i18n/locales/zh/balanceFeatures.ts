export default {
  checkin: {
    title: '每日签到', checked: '今日已签到', todayReward: '今日获得 ${amount}', streakDays: '连续 {days} 天',
    rangeHint: '奖励范围 ${min} - ${max}', luckTitle: '幸运签到', luckButton: '确认幸运签到',
    luckSuccess: '倍率 ${multiplier}x，获得 ${amount}', luckLoss: '倍率 ${multiplier}x，失去 ${amount}', luckEven: '倍率 1.00x，不赚不赔',
    betAmount: '下注金额', betAmountPlaceholder: '输入下注金额', multiplierRange: '倍率范围 ${min}x - ${max}x',
    checkinType: '签到类型', normalCheckin: '普通签到', luckCheckin: '幸运签到',
    blindboxCommon: '普通', blindboxRare: '稀有', blindboxEpic: '史诗', blindboxLegendary: '传说',
    blindboxBalanceReward: '余额 +${value}', blindboxConcurrencyReward: '并发 +{value}', blindboxSubscriptionReward: '{days} 天订阅体验',
    blindboxInvitationReward: '邀请码 x1', blindboxInviteCode: '您的邀请码', blindboxHistory: '盲盒奖品', blindboxHistoryDesc: '签到盲盒获得的历史奖品', blindboxDays: '天',
    page: {
      description: '每日签到奖励、盲盒奖品和账户概览', balance: '余额', streak: '连续签到', days: '天', concurrency: '并发', blindboxCount: '盲盒',
      noBlindbox: '还没有盲盒奖品，坚持签到吧', todayResult: '今日签到', todayReward: '奖励', todayMultiplier: '倍率', todayBlindbox: '盲盒',
      todayNormal: '普通签到', todayLuck: '幸运签到', todayNoResult: '今日尚未签到', rarityBreakdown: '稀有度分布', blindboxInfo: '盲盒规则',
      triggerType: '触发方式', triggerTotal: '累计签到', triggerStreak: '连续签到', triggerInterval: '触发间隔',
      blindboxNextIn: '距离下次盲盒还需 {days} 天', blindboxNextCycle: '每 {interval} 天连续签到触发一次盲盒', calendarTitle: '签到日历'
    }
  },
  leaderboard: {
    title: '排行榜', subtitle: '查看站内活跃排行', empty: '暂无数据', streakDays: '{days} 天',
    balanceSubtitle: '累计签到 {count} 次', consumptionSubtitle: '{count} 次请求', checkinSubtitle: '累计 {total} 次 · 最近 {date} · 获得 ${reward}',
    tabs: { balance: '余额排行', consumption: '消耗排行', checkin: '签到排行' }, periods: { daily: '日榜', weekly: '周榜', monthly: '月榜' }
  },
  transfer: { title: '余额转账', totalSent: '累计转出', totalReceived: '累计转入', totalFee: '手续费', receiverId: '接收方用户 ID', amount: '金额', feePreview: '手续费', total: '合计扣款', memo: '留言', success: '转账成功', submit: '确认转账', failed: '转账失败' },
  redpacket: { title: '红包中心', create: '发红包', claim: '领红包', myPackets: '我的红包' }
}
