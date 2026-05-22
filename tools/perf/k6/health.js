import { createOptions, getScenarioConfig } from './lib/config.js';
import { executeScenario } from './lib/helpers.js';
import { createSummary } from './lib/summary.js';

const scenario = 'health';
const config = getScenarioConfig(scenario);

export const options = createOptions({
  suite: 'perf-baseline',
  scenario,
});

export default function () {
  executeScenario({
    scenario,
    name: 'health',
    pathname: '/health',
    assertBody: (response) => response.json('status') === 'ok',
  });
}

export function handleSummary(data) {
  return createSummary(data, {
    scenario,
    target: `${config.baseUrl}/health`,
    vus: config.vus,
    duration: config.duration,
  });
}
