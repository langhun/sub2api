export const gameHallLocaleMessages = {
  en: {
    gameHall: {
      title: 'Game Hall', description: 'Use an isolated DG wallet for on-site games', mainBalance: 'Main balance', dgBalance: 'DG balance', jackpot: 'Jackpot',
      exchangeTitle: 'Balance exchange', exchangeHint: 'Main balance and DG exchange at 1:1 with an auditable transaction record.', toDG: 'Main to DG', toMain: 'DG to main', amount: 'Amount', rate: 'Rate', afterExchange: 'After exchange: main / DG', exchangeAction: 'Confirm exchange', exchangeSuccess: 'Exchange completed', exchangeRange: 'Per exchange {min} - {max}', exchangeDailyRemaining: '{remaining} of {limit} remaining today across both directions', exchangeDailyUnlimited: 'No daily limit across both directions', unlimited: 'Unlimited', loadFailed: 'Could not load the game hall. Try again.', exchangeFailed: 'Exchange failed. Check your balance and feature availability.', playFailed: 'The round could not be settled. Try again later.',
      safetyTitle: 'Funds and game rules', safetyWallet: 'The DG wallet is isolated from your main balance.', safetySettlement: 'Results are generated and settled atomically by the server.', riskNotice: 'Games can lose DG. Set a reasonable spending limit.',
      slotsTitle: 'Three-reel slots', betRange: 'Bet {min} - {max} DG per round', betAmount: 'Bet amount', serverResult: 'The page only presents the result after server settlement.', playAction: 'Play round', noGames: 'No games are currently available', win: 'Round won', loss: 'No prize this round', push: 'Round tied', payoutSummary: 'Payout {payout} DG · {multiplier}x',
      payoutRules: 'Triple-match payouts', ruleVersion: 'Rule', theoreticalRtp: 'Theoretical RTP', payoutRuleHint: 'A payout requires three identical symbols. Probability is calculated from the active server rule; all other combinations pay 0x.',
      confirmExchangeTitle: 'Confirm balance exchange', confirmExchangeMessage: 'Exchange {amount} using "{direction}" at 1:1?', transactions: 'Wallet activity', rounds: 'Game rounds', historyFailed: 'Could not load history', emptyHistory: 'No history yet', betValue: 'Bet {value} DG', transactionTypes: { balance_to_dg: 'Main balance deposit', dg_to_balance: 'Return to main balance', bet: 'Game bet', payout: 'Game payout' },
      symbols: { star: 'Star', seven: 'Seven', diamond: 'Diamond', cherry: 'Cherry', bell: 'Bell', lemon: 'Lemon', orange: 'Orange', grape: 'Grape', bar: 'BAR' },
      errors: { featureDisabled: 'This feature is currently unavailable', userDisabled: 'Your game hall access has been disabled by an administrator', exchangeReturnDisabled: 'DG cannot currently be returned to the main balance', exchangeDailyLimit: 'Your daily two-way exchange limit has been reached', exchangeOutOfRange: 'The exchange amount is outside the current per-exchange range' },
    },
  },
  zh: {
    gameHall: {
      title: '娱乐大厅', description: '使用独立 DG 钱包参与站内娱乐玩法', mainBalance: '主余额', dgBalance: 'DG 娱乐余额', jackpot: '公共奖池',
      exchangeTitle: '余额兑换', exchangeHint: '主余额与 DG 按 1:1 兑换，每次操作均产生可审计流水。', toDG: '主余额转 DG', toMain: 'DG 转回主余额', amount: '兑换金额', rate: '兑换比例', afterExchange: '兑换后 主余额 / DG', exchangeAction: '确认兑换', exchangeSuccess: '兑换成功', exchangeRange: '单次范围 {min} - {max}', exchangeDailyRemaining: '今日双向合计剩余 {remaining} / {limit}', exchangeDailyUnlimited: '今日双向兑换不限额', unlimited: '不限', loadFailed: '娱乐大厅加载失败，请重试', exchangeFailed: '兑换失败，请检查余额或功能状态', playFailed: '本局未能完成结算，请稍后重试',
      safetyTitle: '资金与玩法说明', safetyWallet: 'DG 钱包与主余额隔离，游戏不会直接扣除主余额。', safetySettlement: '游戏结果由服务端生成并原子结算。', riskNotice: '娱乐玩法存在损失 DG 的可能，请合理控制投入。',
      slotsTitle: '三轴老虎机', betRange: '单局投注 {min} - {max} DG', betAmount: '投注金额', serverResult: '点击开始后，页面只展示服务端已结算结果。', playAction: '开始一局', noGames: '当前没有开放的游戏', win: '本局获胜', loss: '本局未中奖', push: '本局持平', payoutSummary: '派彩 {payout} DG · 倍率 {multiplier}x',
      payoutRules: '三连派彩规则', ruleVersion: '规则', theoreticalRtp: '理论 RTP', payoutRuleHint: '三个相同符号才会派彩。概率按当前服务端规则计算，其余组合派彩为 0x。',
      confirmExchangeTitle: '确认余额兑换', confirmExchangeMessage: '确认按 1:1 将 {amount} 执行“{direction}”？', transactions: '钱包流水', rounds: '游戏记录', historyFailed: '历史记录加载失败', emptyHistory: '暂无历史记录', betValue: '投注 {value} DG', transactionTypes: { balance_to_dg: '主余额转入', dg_to_balance: '转回主余额', bet: '游戏投注', payout: '游戏派彩' },
      symbols: { star: '星耀', seven: '七号', diamond: '钻石', cherry: '樱桃', bell: '铃铛', lemon: '柠檬', orange: '橙子', grape: '葡萄', bar: 'BAR' },
      errors: { featureDisabled: '该功能当前未开放', userDisabled: '您的娱乐大厅权限已被管理员停用', exchangeReturnDisabled: '当前不允许将 DG 转回主余额', exchangeDailyLimit: '今日双向兑换额度已用完', exchangeOutOfRange: '兑换金额不符合当前单次限额' },
    },
  },
} as const
