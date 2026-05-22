import { executeScenario } from './helpers.js';

const scenarioDefinitions = {
  health: {
    requestName: 'health',
    pathname: '/health',
    assertBody: (response) => response.json('status') === 'ok',
  },
  pricing: {
    requestName: 'public-pricing',
    pathname: '/api/v1/public/pricing',
    assertBody: (response) => {
      const code = response.json('code');
      const groups = response.json('data.groups');
      return code === 0 && Array.isArray(groups);
    },
  },
  'monitoring-summary': {
    requestName: 'monitoring-summary',
    pathname: '/api/v1/monitoring/summary',
    assertBody: (response) => {
      const code = response.json('code');
      const groups = response.json('data.groups');
      return code === 0 && Array.isArray(groups);
    },
  },
};

export function getScenarioDefinition(name) {
  const definition = scenarioDefinitions[name];
  if (!definition) {
    throw new Error(`Unknown perf scenario: ${name}`);
  }

  return {
    scenario: name,
    ...definition,
  };
}

export function executeNamedScenario(name, options = {}) {
  const definition = getScenarioDefinition(name);

  return executeScenario({
    scenario: options.scenarioTag || name,
    name: definition.requestName,
    pathname: definition.pathname,
    assertBody: definition.assertBody,
    expectedStatus: options.expectedStatus,
  });
}
