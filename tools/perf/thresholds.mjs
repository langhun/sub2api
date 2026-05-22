import { promises as fs } from 'node:fs';
import path from 'node:path';

export const THRESHOLD_METRICS = ['error_rate', 'timeout_rate', 'p95', 'p99'];

function isPlainObject(value) {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function parseThresholdValue(metric, rawValue, sourceLabel) {
  const value = Number(rawValue);
  if (!Number.isFinite(value) || value < 0) {
    throw new Error(`Invalid ${metric} threshold in ${sourceLabel}: ${rawValue}`);
  }

  return value;
}

function normalizeThresholdSet(rawThresholds, sourceLabel) {
  if (!isPlainObject(rawThresholds)) {
    return {};
  }

  const normalized = {};
  for (const metric of THRESHOLD_METRICS) {
    if (rawThresholds[metric] === undefined) {
      continue;
    }

    normalized[metric] = parseThresholdValue(metric, rawThresholds[metric], sourceLabel);
  }

  return normalized;
}

function normalizeThresholdConfig(rawConfig, sourceLabel) {
  if (!isPlainObject(rawConfig)) {
    throw new Error(`Threshold config from ${sourceLabel} must be a JSON object`);
  }

  const defaultThresholds = {
    ...normalizeThresholdSet(rawConfig, sourceLabel),
    ...normalizeThresholdSet(rawConfig.default, `${sourceLabel}.default`),
  };
  const scenarios = {};

  if (rawConfig.scenarios !== undefined) {
    if (!isPlainObject(rawConfig.scenarios)) {
      throw new Error(`Threshold config "scenarios" from ${sourceLabel} must be a JSON object`);
    }

    for (const [scenario, scenarioThresholds] of Object.entries(rawConfig.scenarios)) {
      scenarios[scenario] = normalizeThresholdSet(scenarioThresholds, `${sourceLabel}.scenarios.${scenario}`);
    }
  }

  return {
    configured: Object.keys(defaultThresholds).length > 0 || Object.keys(scenarios).length > 0,
    source: sourceLabel,
    default: defaultThresholds,
    scenarios,
  };
}

function parseThresholdConfigJson(rawJson, sourceLabel) {
  try {
    return normalizeThresholdConfig(JSON.parse(rawJson), sourceLabel);
  } catch (error) {
    if (error instanceof SyntaxError) {
      throw new Error(`Threshold config from ${sourceLabel} is not valid JSON: ${error.message}`);
    }
    throw error;
  }
}

export async function loadThresholdConfig({ thresholdsPath = '', thresholdsJson = '' } = {}) {
  if (thresholdsJson) {
    return parseThresholdConfigJson(thresholdsJson, 'cli:--thresholds-json');
  }

  if (thresholdsPath) {
    const resolvedPath = path.resolve(thresholdsPath);
    const content = await fs.readFile(resolvedPath, 'utf8');
    return parseThresholdConfigJson(content, `file:${resolvedPath}`);
  }

  const envThresholds = process.env.PERF_NIGHTLY_THRESHOLDS || '';
  if (envThresholds.trim()) {
    return parseThresholdConfigJson(envThresholds, 'env:PERF_NIGHTLY_THRESHOLDS');
  }

  return {
    configured: false,
    source: null,
    default: {},
    scenarios: {},
  };
}

function evaluateMetric(metric, actualValue, thresholdValue) {
  if (thresholdValue === undefined) {
    return {
      threshold: null,
      actual: actualValue,
      pass: null,
      status: 'not_configured',
    };
  }

  if (actualValue === null || actualValue === undefined || Number.isNaN(actualValue)) {
    return {
      threshold: thresholdValue,
      actual: actualValue,
      pass: false,
      status: 'fail',
      reason: 'missing_metric',
    };
  }

  const pass = Number(actualValue) <= thresholdValue;
  return {
    threshold: thresholdValue,
    actual: Number(actualValue),
    pass,
    status: pass ? 'pass' : 'fail',
  };
}

function decorateRow(row, thresholdConfig) {
  const scenarioThresholds = {
    ...thresholdConfig.default,
    ...(thresholdConfig.scenarios[row.scenario] || {}),
  };
  const checks = {};
  const checkedMetrics = [];
  const failedMetrics = [];

  for (const metric of THRESHOLD_METRICS) {
    const check = evaluateMetric(metric, row[metric], scenarioThresholds[metric]);
    checks[metric] = check;

    if (check.status === 'not_configured') {
      continue;
    }

    checkedMetrics.push(metric);
    if (!check.pass) {
      failedMetrics.push(metric);
    }
  }

  const thresholdStatus = checkedMetrics.length === 0
    ? 'not_configured'
    : failedMetrics.length === 0
      ? 'pass'
      : 'fail';

  return {
    ...row,
    threshold_status: thresholdStatus,
    threshold_failed_metrics: failedMetrics.join('|'),
    threshold_checked_metrics: checkedMetrics.join('|'),
    checks,
    error_rate_threshold: checks.error_rate.threshold,
    error_rate_check: checks.error_rate.status,
    timeout_rate_threshold: checks.timeout_rate.threshold,
    timeout_rate_check: checks.timeout_rate.status,
    p95_threshold: checks.p95.threshold,
    p95_check: checks.p95.status,
    p99_threshold: checks.p99.threshold,
    p99_check: checks.p99.status,
  };
}

export function applyThresholds(rows, thresholdConfig) {
  if (!thresholdConfig.configured) {
    return {
      rows: rows.map((row) => ({
        ...row,
        threshold_status: 'not_configured',
        threshold_failed_metrics: '',
        threshold_checked_metrics: '',
        checks: {
          error_rate: { threshold: null, actual: row.error_rate, pass: null, status: 'not_configured' },
          timeout_rate: { threshold: null, actual: row.timeout_rate, pass: null, status: 'not_configured' },
          p95: { threshold: null, actual: row.p95, pass: null, status: 'not_configured' },
          p99: { threshold: null, actual: row.p99, pass: null, status: 'not_configured' },
        },
        error_rate_threshold: null,
        error_rate_check: 'not_configured',
        timeout_rate_threshold: null,
        timeout_rate_check: 'not_configured',
        p95_threshold: null,
        p95_check: 'not_configured',
        p99_threshold: null,
        p99_check: 'not_configured',
      })),
      report: {
        configured: false,
        checked: false,
        source: null,
        overall_status: 'not_configured',
        has_failures: false,
        checked_scenarios: [],
        failed_scenarios: [],
      },
    };
  }

  const decoratedRows = rows.map((row) => decorateRow(row, thresholdConfig));
  const checkedScenarios = decoratedRows
    .filter((row) => row.threshold_status !== 'not_configured')
    .map((row) => row.scenario);
  const failedScenarios = decoratedRows
    .filter((row) => row.threshold_status === 'fail')
    .map((row) => row.scenario);

  return {
    rows: decoratedRows,
    report: {
      configured: true,
      checked: checkedScenarios.length > 0,
      source: thresholdConfig.source,
      overall_status: failedScenarios.length > 0
        ? 'fail'
        : checkedScenarios.length > 0
          ? 'pass'
          : 'not_configured',
      has_failures: failedScenarios.length > 0,
      checked_scenarios: checkedScenarios,
      failed_scenarios: failedScenarios,
    },
  };
}

function formatNumber(value, digits = 2) {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return 'n/a';
  }

  return Number(value).toFixed(digits);
}

function formatPercent(value) {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return 'n/a';
  }

  return `${(Number(value) * 100).toFixed(3)}%`;
}

function formatThresholdValue(metric, value) {
  if (metric === 'error_rate' || metric === 'timeout_rate') {
    return formatPercent(value);
  }

  return `${formatNumber(value, 2)} ms`;
}

function formatCheck(metric, check) {
  if (check.status === 'not_configured') {
    return 'not configured';
  }

  const actual = formatThresholdValue(metric, check.actual);
  const threshold = formatThresholdValue(metric, check.threshold);
  return `${actual} <= ${threshold} (${check.status})`;
}

export function buildThresholdMarkdown(rows, report) {
  const lines = [
    '## Perf Threshold Guard',
    '',
  ];

  if (!report.configured) {
    lines.push('- Status: `not_configured`');
    lines.push('- Configure `PERF_NIGHTLY_THRESHOLDS` or pass `--thresholds` / `--thresholds-json` to enable checks.');
    lines.push('');
    return lines.join('\n');
  }

  lines.push(`- Status: \`${report.overall_status}\``);
  lines.push(`- Source: \`${report.source || 'unknown'}\``);
  lines.push(`- Checked scenarios: ${report.checked_scenarios.length > 0 ? report.checked_scenarios.map((item) => `\`${item}\``).join(', ') : 'none'}`);
  lines.push(`- Failed scenarios: ${report.failed_scenarios.length > 0 ? report.failed_scenarios.map((item) => `\`${item}\``).join(', ') : 'none'}`);
  lines.push(
    '',
    '| scenario | error_rate | timeout_rate | p95 | p99 | result |',
    '| --- | --- | --- | --- | --- | --- |',
  );

  for (const row of rows) {
    lines.push(
      `| ${row.scenario} | ${formatCheck('error_rate', row.checks.error_rate)} | ${formatCheck('timeout_rate', row.checks.timeout_rate)} | ${formatCheck('p95', row.checks.p95)} | ${formatCheck('p99', row.checks.p99)} | ${row.threshold_status} |`,
    );
  }

  lines.push('');
  return lines.join('\n');
}
