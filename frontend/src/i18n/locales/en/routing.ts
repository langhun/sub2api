export default {
  enableSmartRouting: 'Enable Smart Routing',
  smartRoutingHint: 'Smart routing automatically selects the best group based on price, speed, and success rate',

  strategy: 'Routing Strategy',
  strategyAuto: 'Auto (Recommended)',
  strategyPrice: 'Price Priority',
  strategySpeed: 'Speed Priority',
  strategySuccess: 'Success Priority',
  strategyCustom: 'Custom Weights',

  customWeights: 'Custom Weights',
  priceWeight: 'Price Weight',
  speedWeight: 'Speed Weight',
  successWeight: 'Success Rate Weight',
  weightSumError: 'Weights must sum to 100% (current: {sum}%)',

  explorationRate: 'Exploration Rate',
  explorationRateHint: 'Keep between 0.5-1% for cold-start group data collection',

  candidateGroups: 'Candidate Groups',
  noCandidatesSelected: 'No candidate groups selected',

  versionConflictPrompt: 'Configuration was modified by another user. Reload?',
  versionConflictReloaded: 'Configuration reloaded, please re-edit',

  configSaved: 'Routing configuration saved',
}
