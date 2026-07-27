export const activityAdminLocaleMessages = {
  en: {
    audit: {
      actionSegments: {
        blindbox: 'Blind box',
        redpacket: 'Red packet',
        checkin: 'Check-in',
        leaderboard: 'Leaderboard',
      },
    },
    settings: {
      tabs: { balanceFeatures: 'Balance Features' },
      balanceFeatures: {
        entrySwitchesTitle: 'Entry and activity switches', entrySwitchesDescription: 'Control user entries, leaderboard tabs, and API access.',
        checkinTitle: 'Check-in Settings', checkinDescription: 'Configure normal and lucky check-in rewards', normalCheckin: 'Enable normal check-in', luckCheckin: 'Enable lucky check-in',
        minReward: 'Minimum reward', maxReward: 'Maximum reward', minMultiplier: 'Minimum multiplier', maxMultiplier: 'Maximum multiplier',
      },
      checkin: {
        blindboxTitle: 'Check-in Blind Box', blindboxDescription: 'Configure triggers and maintain the prize pool', blindboxTriggerType: 'Trigger type', blindboxTriggerStreak: 'Consecutive check-ins',
        blindboxTriggerTotal: 'Total check-ins', blindboxInterval: 'Trigger interval', blindboxIntervalHint: 'Open a blind box after the configured number of check-ins',
      },
    },
    resources: {
      redeem: {
        types: {
          checkin: 'Check-in',
          checkin_luck: 'Lucky check-in',
          checkin_blindbox: 'Blind box check-in',
        },
      },
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
      deliveryLoadFailed: 'Could not load reward deliveries', deliveryTotal: '{total} deliveries',
    },
    codeFormat: {
      title: 'Code and phrase formats', description: 'Configure newly generated redemption, invitation, and packet codes. Existing codes remain valid.', balance: 'Balance codes', concurrency: 'Concurrency codes', subscription: 'Subscription codes', invitation: 'Invitation codes', redpacket: 'Packet phrases', prefix: 'Prefix', characterSet: 'Characters', separator: 'Separator', custom: 'Custom', customSeparator: 'Custom character', combinationSpace: 'Available combinations: {count}', groupLength: 'Group size', groupCount: 'Groups', length: 'Example length {length}', none: 'None', uppercase: 'Uppercase', numeric: 'Numeric', alphanumeric: 'Alphanumeric', hex: 'Hexadecimal', invalidCharset: 'Select a valid character set', invalidGroupLength: 'Group size must be an integer from 1 to 32', invalidGroupCount: 'Groups must be an integer from 1 to 16', invalidSeparator: 'Separator must be one printable ASCII character', invalidPrefix: 'Prefix must be printable ASCII and cannot contain the separator',
    },
  },
  zh: {
    audit: {
      actionSegments: {
        blindbox: '盲盒',
        redpacket: '红包',
        checkin: '签到',
        leaderboard: '排行榜',
      },
    },
    settings: {
      tabs: { balanceFeatures: '余额功能' },
      balanceFeatures: {
        entrySwitchesTitle: '入口与活动开关', entrySwitchesDescription: '集中控制用户端入口、排行榜标签及其接口访问。',
        checkinTitle: '签到设置', checkinDescription: '配置普通签到和运气签到奖励', normalCheckin: '启用普通签到', luckCheckin: '启用运气签到',
        minReward: '最小奖励', maxReward: '最大奖励', minMultiplier: '最小倍率', maxMultiplier: '最大倍率',
      },
      checkin: {
        blindboxTitle: '签到盲盒', blindboxDescription: '配置触发规则并维护盲盒奖池', blindboxTriggerType: '触发方式', blindboxTriggerStreak: '连续签到',
        blindboxTriggerTotal: '累计签到', blindboxInterval: '触发间隔', blindboxIntervalHint: '达到指定签到天数时触发盲盒',
      },
    },
    resources: {
      redeem: {
        types: {
          checkin: '签到',
          checkin_luck: '运气签到',
          checkin_blindbox: '盲盒签到',
        },
      },
    },
    blindbox: {
      createItem: '新增奖品', editItem: '编辑奖品', totalItems: '奖品总数', enabledItems: '已启用', totalDraws: '累计抽取', empty: '暂无奖品',
      colName: '名称', colRarity: '稀有度', colRewardType: '奖励类型', colReward: '奖励', colWeight: '权重', colStatus: '状态', colActions: '操作',
      namePlaceholder: '奖品名称', rewardBalance: '余额', rewardConcurrency: '并发', rewardSubscription: '订阅', rewardInvitation: '邀请码', minValue: '最小值', maxValue: '最大值',
      concurrencyValue: '并发值', subscriptionGroup: '订阅分组', selectGroup: '选择分组', subscriptionDays: '订阅天数', weightHint: '权重越高越容易抽中', days: '天', confirmDelete: '确定删除此奖品吗？',
      deliveryTitle: '奖励投递运维', deliveryDescription: '查看盲盒奖励投递状态，处理达到重试上限的失败任务。', deliveryStatusFilter: '投递状态', deliveryAllStatuses: '全部状态',
      deliveryStatus: { pending: '待投递', delivering: '投递中', delivered: '已送达', failed: '投递失败', compensated: '已补偿' },
      deliveryEmpty: '当前筛选条件下没有投递记录', deliveryColSource: '来源', deliveryColUser: '用户', deliveryColReward: '奖励', deliveryColStatus: '投递状态', deliveryColAttempts: '尝试次数', deliveryColTime: '更新时间',
      deliveryRetry: '重新投递', deliveryRetryTitle: '确认重新投递', deliveryRetryMessage: '任务将回到待投递队列，并由后台 worker 再次处理。', deliveryRetryFailed: '重新投递失败',
      deliveryCompensate: '标记补偿', deliveryCompensateTitle: '登记人工补偿', deliveryCompensateMessage: '仅在已通过其他方式完成补偿后登记，系统不会重复发放奖励。', deliveryCompensateReason: '补偿说明', deliveryConfirmCompensate: '确认已补偿', deliveryCompensateFailed: '补偿登记失败',
      deliveryLoadFailed: '投递记录加载失败', deliveryTotal: '共 {total} 条',
    },
    codeFormat: {
      title: '兑换码与口令格式', description: '统一配置新生成的兑换码、邀请码和红包口令；历史代码不受影响。', balance: '余额兑换码', concurrency: '并发兑换码', subscription: '订阅兑换码', invitation: '邀请码', redpacket: '红包口令', prefix: '前缀', characterSet: '字符集', separator: '分隔符', custom: '自定义', customSeparator: '自定义单字符', combinationSpace: '可用组合空间：{count}', groupLength: '组长', groupCount: '组数', length: '示例长度 {length}', none: '无', uppercase: '大写字母', numeric: '数字', alphanumeric: '字母数字', hex: '十六进制', invalidCharset: '请选择有效字符集', invalidGroupLength: '组长必须是 1 到 32 的整数', invalidGroupCount: '组数必须是 1 到 16 的整数', invalidSeparator: '分隔符必须是单个可打印 ASCII 字符', invalidPrefix: '前缀必须使用可打印 ASCII，且不能包含分隔符',
    },
  },
} as const
