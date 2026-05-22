import { createOptions, getMixedScenarioConfig } from './lib/config.js';
import {
  applyMixedPacing,
  createWeightedMixPlan,
  formatMixedPacing,
  formatMixedWeights,
  selectFromMixPlan,
} from './lib/mixed.js';
import { executeNamedScenario } from './lib/scenarios.js';
import { createSummary } from './lib/summary.js';

const scenario = 'mixed';
const config = getMixedScenarioConfig();
const mixPlan = createWeightedMixPlan(config.weights);

export const options = createOptions({
  suite: 'perf-baseline',
  scenario,
});

export default function () {
  const selectedScenario = selectFromMixPlan(mixPlan);
  executeNamedScenario(selectedScenario, {
    scenarioTag: scenario,
  });
  applyMixedPacing(config.pacing);
}

export function handleSummary(data) {
  return createSummary(data, {
    scenario,
    target: `${config.baseUrl} (weighted health/pricing/monitoring-summary)`,
    vus: config.vus,
    duration: config.duration,
    extraLines: [
      `mix_weights=${formatMixedWeights(config.weights)}`,
      `mix_pacing_ms=${formatMixedPacing(config.pacing)}`,
    ],
  });
}
