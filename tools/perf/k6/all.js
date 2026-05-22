import health, { handleSummary as healthSummary, options as healthOptions } from './health.js';
import pricing, { handleSummary as pricingSummary, options as pricingOptions } from './pricing.js';
import monitoringSummary, {
  handleSummary as monitoringSummaryHandler,
  options as monitoringOptions,
} from './monitoring-summary.js';
import { getScenarioName } from './lib/config.js';

const scenarioHandlers = {
  health: {
    run: health,
    summary: healthSummary,
    options: healthOptions,
  },
  pricing: {
    run: pricing,
    summary: pricingSummary,
    options: pricingOptions,
  },
  'monitoring-summary': {
    run: monitoringSummary,
    summary: monitoringSummaryHandler,
    options: monitoringOptions,
  },
};

const selectedScenario = getScenarioName('health', { allowEnvOverride: true });
const selected = scenarioHandlers[selectedScenario];

export const options = selected.options;

export default function () {
  selected.run();
}

export function handleSummary(data) {
  return selected.summary(data);
}
