import http from 'k6/http';
import { check } from 'k6';
import { Counter, Rate } from 'k6/metrics';
import { getScenarioConfig } from './config.js';

export const scenarioErrors = new Rate('scenario_error_rate');
export const scenarioTimeouts = new Rate('scenario_timeout_rate');
export const scenarioRequests = new Counter('scenario_requests_total');

export function buildUrl(pathname, defaultScenario) {
  const config = getScenarioConfig(defaultScenario);
  return `${config.baseUrl}${pathname}`;
}

function classifyAsTimeout(response) {
  if (!response || response.error_code) {
    const code = String(response?.error_code || '').toUpperCase();
    return code.includes('TIMEOUT') || code.includes('DEADLINE');
  }

  return false;
}

export function runGetRequest({
  scenario,
  name,
  pathname,
  expectedStatus,
  timeout,
  headers,
  assertBody,
}) {
  const url = buildUrl(pathname, scenario);
  const response = http.get(url, {
    headers,
    timeout,
    tags: {
      scenario,
      endpoint: name,
    },
  });

  scenarioRequests.add(1);

  const statusOk = check(response, {
    [`${name} returned expected status`]: (res) => res.status === expectedStatus,
  });

  const bodyOk = assertBody
    ? check(response, {
        [`${name} response body is valid`]: (res) => assertBody(res),
      })
    : true;

  const success = statusOk && bodyOk;
  scenarioErrors.add(!success);
  scenarioTimeouts.add(classifyAsTimeout(response));

  return response;
}

export function executeScenario(definition) {
  const config = getScenarioConfig(definition.scenario);
  return runGetRequest({
    scenario: definition.scenario,
    name: definition.name,
    pathname: definition.pathname,
    expectedStatus: definition.expectedStatus || config.expectedStatus,
    timeout: config.timeout,
    headers: config.headers,
    assertBody: definition.assertBody,
  });
}
