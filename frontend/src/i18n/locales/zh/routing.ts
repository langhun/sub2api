export default {
  enableSmartRouting: '启用智能路由',
  smartRoutingHint: '智能路由会根据价格、速度和成功率自动选择最优分组',

  strategy: '路由策略',
  strategyAuto: '自动（推荐）',
  strategyPrice: '优先价格',
  strategySpeed: '优先速度',
  strategySuccess: '优先成功率',
  strategyCustom: '自定义权重',

  customWeights: '自定义权重',
  priceWeight: '价格权重',
  speedWeight: '速度权重',
  successWeight: '成功率权重',
  weightSumError: '权重总和必须为100%（当前: {sum}%）',

  explorationRate: '探索率',
  explorationRateHint: '建议保持0.5-1%之间，用于冷启动分组数据收集',

  candidateGroups: '候选分组',
  noCandidatesSelected: '尚未选择候选分组',

  versionConflictPrompt: '配置已被其他用户修改，是否重新加载？',
  versionConflictReloaded: '配置已重新加载，请重新编辑',

  configSaved: '路由配置已保存',
}
