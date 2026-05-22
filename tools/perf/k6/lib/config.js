import { fail } from 'k6';

const DEFAULT_BASE_URL = 'http://127.0.0.1:18808';
const DEFAULT_TIMEOUT = '5s';
const DEFAULT_DURATION = '30s';
const DEFAULT_VUS = 5;
const DEFAULT_EXPECTED_STATUS = 200;
const DEFAULT_AUTH_HEADER = 'Authorization';
const DEFAULT_AUTH_SCHEME = 'Bearer';
const SUPPORTED_SCENARIOS = ['health', 'pricing', 'monitoring-summary'];

function parsePositiveInt(name, fallbackValue) {
  const raw = __ENV[name];
  if (!raw) {
    return fallbackValue;
  }

  const value = Number.parseInt(raw, 10);
  if (!Number.isFinite(value) || value < 0) {
    fail(`${name} must be a non-negative integer, got: ${raw}`);
  }

  return value;
}

function parseBoolean(name, fallbackValue = false) {
  const raw = __ENV[name];
  if (!raw) {
    return fallbackValue;
  }

  return ['1', 'true', 'yes', 'on'].includes(String(raw).trim().toLowerCase());
}

function parseJsonObject(name) {
  const raw = __ENV[name];
  if (!raw) {
    return {};
  }

  try {
    const value = JSON.parse(raw);
    if (!value || typeof value !== 'object' || Array.isArray(value)) {
      fail(`${name} must be a JSON object, got: ${raw}`);
    }
    return value;
  } catch (error) {
    fail(`${name} must be valid JSON, parse failed: ${error.message}`);
  }
}

function normalizeBaseUrl(rawBaseUrl) {
  return String(rawBaseUrl || DEFAULT_BASE_URL).replace(/\/+$/, '');
}

export function getScenarioName(defaultScenario, options = {}) {
  const { allowEnvOverride = false } = options;
  const rawScenario = allowEnvOverride ? __ENV.SCENARIO || defaultScenario || 'health' : defaultScenario || 'health';
  const scenario = rawScenario.trim();
  if (!SUPPORTED_SCENARIOS.includes(scenario)) {
    fail(`SCENARIO must be one of ${SUPPORTED_SCENARIOS.join(', ')}, got: ${scenario}`);
  }
  return scenario;
}

export function getScenarioConfig(defaultScenario, options = {}) {
  const scenario = getScenarioName(defaultScenario, options);
  const baseUrl = normalizeBaseUrl(__ENV.BASE_URL || DEFAULT_BASE_URL);
  const expectedStatus = parsePositiveInt('EXPECTED_STATUS', DEFAULT_EXPECTED_STATUS);
  const timeout = __ENV.K6_TIMEOUT || DEFAULT_TIMEOUT;
  const vus = parsePositiveInt('K6_VUS', DEFAULT_VUS);
  const duration = __ENV.K6_DURATION || DEFAULT_DURATION;
  const rps = parsePositiveInt('K6_RPS', 0);
  const insecureSkipTLSVerify = parseBoolean('K6_INSECURE_SKIP_TLS_VERIFY', false);
  const authHeader = (__ENV.AUTH_HEADER || DEFAULT_AUTH_HEADER).trim();
  const authScheme = __ENV.AUTH_SCHEME === undefined ? DEFAULT_AUTH_SCHEME : String(__ENV.AUTH_SCHEME).trim();
  const authToken = (__ENV.AUTH_TOKEN || '').trim();
  const extraHeaders = parseJsonObject('EXTRA_HEADERS');

  const headers = {
    Accept: 'application/json',
    ...extraHeaders,
  };

  if (authToken && authHeader) {
    headers[authHeader] = authScheme ? `${authScheme} ${authToken}`.trim() : authToken;
  }

  return {
    scenario,
    baseUrl,
    expectedStatus,
    timeout,
    vus,
    duration,
    rps,
    insecureSkipTLSVerify,
    headers,
  };
}

export function createOptions(tags = {}, configOptions = {}) {
  const config = getScenarioConfig(tags.scenario, configOptions);
  const options = {
    vus: config.vus,
    duration: config.duration,
    insecureSkipTLSVerify: config.insecureSkipTLSVerify,
    summaryTrendStats: (__ENV.SUMMARY_TREND_STATS || 'avg,min,med,max,p(50),p(95),p(99)')
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean),
    tags,
  };

  if (config.rps > 0) {
    options.rps = config.rps;
  }

  return options;
}
