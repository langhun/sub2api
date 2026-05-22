import { createOptions, getScenarioConfig } from './lib/config.js';
import { executeNamedScenario, getScenarioDefinition } from './lib/scenarios.js';
import { createSummary } from './lib/summary.js';

const scenario = 'monitoring-summary';
const config = getScenarioConfig(scenario);
const definition = getScenarioDefinition(scenario);

export const options = createOptions({
  suite: 'perf-baseline',
  scenario,
});

export default function () {
  executeNamedScenario(scenario);
}

export function handleSummary(data) {
  return createSummary(data, {
    scenario,
    target: `${config.baseUrl}${definition.pathname}`,
    vus: config.vus,
    duration: config.duration,
  });
}
