import { createOptions, getScenarioConfig } from './lib/config.js';
import { executeScenario } from './lib/helpers.js';
import { createSummary } from './lib/summary.js';

const scenario = 'monitoring-summary';
const config = getScenarioConfig(scenario);

export const options = createOptions({
  suite: 'perf-baseline',
  scenario,
});

export default function () {
  executeScenario({
    scenario,
    name: 'monitoring-summary',
    pathname: '/api/v1/monitoring/summary',
    assertBody: (response) => {
      const code = response.json('code');
      const groups = response.json('data.groups');
      return code === 0 && Array.isArray(groups);
    },
  });
}

export function handleSummary(data) {
  return createSummary(data, {
    scenario,
    target: `${config.baseUrl}/api/v1/monitoring/summary`,
    vus: config.vus,
    duration: config.duration,
  });
}
