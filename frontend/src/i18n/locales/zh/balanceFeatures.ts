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
    title: '排行榜', subtitle: '按统一统计口径查看站内活跃排行', empty: '暂无榜单数据', streakDays: '{days} 天', rank: '第 {rank} 名', daysValue: '{value} 天', loadFailed: '排行榜加载失败，请重试', privacyNotice: '榜单昵称已按站点隐私规则处理，数据仅用于站内活动展示。',
    balanceSubtitle: '累计签到 {count} 次', consumptionSubtitle: '{count} 次请求', checkinSubtitle: '累计 {total} 次 · 最近 {date} · 获得 ${reward}', transferSubtitle: '成功转账 {count} 次',
    filterLabel: '榜单筛选', periodFilterLabel: '统计周期', distributionTitle: '{board}分布', distributionHint: '基于当前页上榜数据计算金额与占比', pageTotal: '当前页总值', listedUsers: '用户数', currentPageUsers: '当前页 {count} 人', chartHint: '悬停圆环切片可查看用户、金额和占比', rankingTitle: '排行明细', currentPage: '第 {page} 页', others: '其他用户', noBoards: '暂无可用榜单', noBoardsHint: '管理员尚未开放任何排行榜，请稍后再试。',
    distributionNames: { balance: '余额', consumption: '消费', checkin: '签到', transfer: '转账' }, distributionHints: { balance: '查看余额排行用户的金额占比', consumption: '查看当前周期所有消费用户的金额占比', checkin: '查看签到排行用户的连续天数占比', transfer: '查看当前周期所有转账用户的金额占比' }, summaryHeaders: { balance: '金额', consumption: '总金额', checkin: '总天数', transfer: '总金额' },
    columns: { user: '用户', share: '占比' }, activityHeaders: { balance: '签到', consumption: '请求', checkin: '累计签到', transfer: '转账' }, valueHeaders: { balance: '余额', consumption: '消费金额', checkin: '连续天数', transfer: '转出金额' },
    tabs: { balance: '余额排行', consumption: '消耗排行', checkin: '签到排行', transfer: '转账排行' }, periods: { daily: '日榜', weekly: '周榜', monthly: '月榜' }
  },
  transfer: {
    title: '余额转账', description: '核对接收方与费用后完成站内余额转账', currentBalance: '当前余额', totalSent: '累计转出', totalReceived: '累计转入', totalFee: '手续费', newTransfer: '发起转账', irreversible: '转账到账后不可由用户撤销，请仔细核对接收方。', receiverId: '接收方', receiverPlaceholder: '输入邮箱、用户名或用户 ID', receiverResolved: '已选择 {recipient}', receiverSearchFailed: '未找到可转账的接收方', recipient: '接收方', amount: '金额', feePreview: '手续费', calculateFee: '计算手续费', total: '合计扣款', available: '可用：{amount}', dailyRemaining: '今日剩余额度 {amount} · 剩余 {count} 次', memo: '留言', memoPlaceholder: '可选留言', continue: '核验并继续', success: '转账成功', successWithId: '转账成功，流水号 #{id}', submit: '确认转账', failed: '转账失败，请稍后重试', validationFailed: '无法核验接收方或费用，请检查输入', confirmTitle: '确认余额转账', confirmMessage: '请再次核对本次转账，确认后将立即到账。', confirmRecipient: '接收方：{recipient}', confirmAmount: '到账 {amount}，手续费 {fee}，合计扣款 {total}', history: '转账明细', historyHint: '按转出与转入查看资金记录', historyFailed: '转账记录加载失败', emptyHistory: '暂无转账记录', sent: '我转出的', received: '我收到的', toUser: '转给用户 #{id}', fromUser: '来自用户 #{id}', transferTypes: { direct: '余额转账', redpacket: '红包领取', batch: '批量发放' }, status: { completed: '已完成', frozen: '已标记异常', revoked: '已撤销' }
  },
  redpacket: {
    title: '红包中心', description: '发红包，领红包，分享快乐', create: '发红包', createHint: '发送余额红包给好友', claim: '领红包', claimHint: '输入口令领取红包', myPackets: '我的红包', sent: '我发出的', received: '我领取的', sentHistoryHint: '查看红包剩余份数、金额和有效期', totalAmount: '总金额', count: '红包份数', type: '分配方式', equal: '等额红包', equalHint: '每份金额相同', random: '拼手气红包', randomHint: '由服务端随机分配每份金额', memo: '祝福语', memoPlaceholder: '恭喜发财', perPacket: '预计每份', confirmTitle: '确认发红包', confirmMessage: '将发出 {count} 份、共 {amount} 的红包，确认继续？', createdTitle: '红包已创建', shareHint: '分享以下红包口令，对方即可领取。', createFailed: '红包创建失败，请检查金额和份数', claimTitle: '输入红包口令', code: '红包口令', codePlaceholder: '输入红包口令', claimSuccess: '领取成功', claimFailed: '领取失败，请核对口令或红包状态', historyFailed: '红包记录加载失败', detailFailed: '红包详情加载失败', claimRecords: '领取记录', empty: '暂无红包记录', remaining: '剩余 {remaining}/{total} 份', detailTitle: '红包详情', remainingAmount: '剩余金额', expiresAt: '过期时间', progress: '领取进度', validUntil: '有效期至 {date}', status: { active: '领取中', expired: '已过期', exhausted: '已领完' }
  }
  ,gameHall: {
    title: '娱乐大厅', description: '使用独立 DG 钱包参与站内娱乐玩法', mainBalance: '主余额', dgBalance: 'DG 娱乐余额', jackpot: '公共奖池',
    exchangeTitle: '余额兑换', exchangeHint: '主余额与 DG 按 1:1 兑换，每次操作均产生可审计流水。', toDG: '主余额转 DG', toMain: 'DG 转回主余额', amount: '兑换金额', rate: '兑换比例', afterExchange: '兑换后 主余额 / DG', exchangeAction: '确认兑换', exchangeSuccess: '兑换成功', loadFailed: '娱乐大厅加载失败，请重试', exchangeFailed: '兑换失败，请检查余额或功能状态', playFailed: '本局未能完成结算，请稍后重试',
    safetyTitle: '资金与玩法说明', safetyWallet: 'DG 钱包与主余额隔离，游戏不会直接扣除主余额。', safetySettlement: '游戏结果由服务端生成并原子结算。', riskNotice: '娱乐玩法存在损失 DG 的可能，请合理控制投入。',
    slotsTitle: '三轴老虎机', betRange: '单局投注 {min} - {max} DG', betAmount: '投注金额', serverResult: '点击开始后，页面只展示服务端已结算结果。', playAction: '开始一局', noGames: '当前没有开放的游戏', win: '本局获胜', loss: '本局未中奖', push: '本局持平', payoutSummary: '派彩 {payout} DG · 倍率 {multiplier}x',
    confirmExchangeTitle: '确认余额兑换', confirmExchangeMessage: '确认按 1:1 将 {amount} 执行“{direction}”？', transactions: '钱包流水', rounds: '游戏记录', historyFailed: '历史记录加载失败', emptyHistory: '暂无历史记录', betValue: '投注 {value} DG', transactionTypes: { balance_to_dg: '主余额转入', dg_to_balance: '转回主余额', bet: '游戏投注', payout: '游戏派彩' },
    symbols: { star: '星耀', seven: '七号', diamond: '钻石', cherry: '樱桃', bell: '铃铛', lemon: '柠檬', bar: 'BAR' }
  },
  activityErrors: { featureDisabled: '该功能当前未开放', insufficientBalance: '可用余额不足', limitExceeded: '已超过当前额度或次数限制', transferDailyLimit: '已超过今日转账额度', transferDailyCount: '今日转账次数已用完', transferSelf: '不能向自己转账', transferAmountInvalid: '转账金额不符合当前限制', receiverNotFound: '未找到可转账的接收方', alreadyClaimed: '您已经领取过该红包', redpacketExpired: '红包已过期', redpacketExhausted: '红包已领完', redpacketNotFound: '红包口令不存在', cannotClaimOwn: '不能领取自己发出的红包', duplicateRequest: '请求正在处理或已提交，请勿重复操作' }
}
