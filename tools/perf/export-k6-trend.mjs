import { promises as fs } from 'node:fs';
import path from 'node:path';
import {
  applyThresholds,
  buildThresholdMarkdown,
  loadThresholdConfig,
} from './thresholds.mjs';

const SCENARIO_ORDER = ['health', 'pricing', 'monitoring-summary', 'mixed'];
const CSV_COLUMNS = [
  'scenario',
  'requests',
  'rps',
  'error_rate',
  'timeout_rate',
  'p95',
  'p99',
  'threshold_status',
  'threshold_failed_metrics',
  'threshold_checked_metrics',
  'error_rate_threshold',
  'error_rate_check',
  'timeout_rate_threshold',
  'timeout_rate_check',
  'p95_threshold',
  'p95_check',
  'p99_threshold',
  'p99_check',
  'commit',
  'date',
];
const LATEST_TREND_METRICS = [
  { key: 'requests', label: 'requests', digits: 0, unit: '' },
  { key: 'rps', label: 'rps', digits: 2, unit: '' },
  { key: 'p95', label: 'p95', digits: 2, unit: ' ms' },
  { key: 'p99', label: 'p99', digits: 2, unit: ' ms' },
  { key: 'error_rate', label: 'error_rate', digits: 3, unit: '%', scale: 100 },
  { key: 'timeout_rate', label: 'timeout_rate', digits: 3, unit: '%', scale: 100 },
];

function parseArgs(argv) {
  const args = {
    'input-dir': '',
    'output-dir': '',
    metadata: '',
    thresholds: '',
    'thresholds-json': '',
    'history-csv': '',
  };

  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (!token.startsWith('--')) {
      continue;
    }

    const key = token.slice(2);
    const value = argv[index + 1];
    if (!value || value.startsWith('--')) {
      throw new Error(`Missing value for --${key}`);
    }

    args[key] = value;
    index += 1;
  }

  if (!args['input-dir']) {
    throw new Error('Missing required argument: --input-dir');
  }

  if (!args['output-dir']) {
    throw new Error('Missing required argument: --output-dir');
  }

  return args;
}

function getMetric(summary, metricName, valueName) {
  const metric = summary?.metrics?.[metricName];
  if (!metric?.values) {
    return null;
  }

  const value = metric.values[valueName];
  return Number.isFinite(value) ? value : null;
}

function inferScenario(fileName, summary) {
  const fromSummary = summary?.options?.tags?.scenario;
  if (typeof fromSummary === 'string' && fromSummary.trim()) {
    return fromSummary.trim();
  }

  return fileName.replace(/\.summary\.json$/i, '');
}

function sortRows(rows) {
  return [...rows].sort((left, right) => {
    const leftIndex = SCENARIO_ORDER.indexOf(left.scenario);
    const rightIndex = SCENARIO_ORDER.indexOf(right.scenario);
    const normalizedLeft = leftIndex === -1 ? Number.MAX_SAFE_INTEGER : leftIndex;
    const normalizedRight = rightIndex === -1 ? Number.MAX_SAFE_INTEGER : rightIndex;

    if (normalizedLeft !== normalizedRight) {
      return normalizedLeft - normalizedRight;
    }

    return left.scenario.localeCompare(right.scenario);
  });
}

function sortHistoryRows(rows) {
  return [...rows].sort((left, right) => {
    const leftTime = Date.parse(left.date || '');
    const rightTime = Date.parse(right.date || '');
    const leftHasTime = Number.isFinite(leftTime);
    const rightHasTime = Number.isFinite(rightTime);

    if (leftHasTime && rightHasTime && leftTime !== rightTime) {
      return leftTime - rightTime;
    }

    if (leftHasTime !== rightHasTime) {
      return leftHasTime ? -1 : 1;
    }

    const dateCompare = String(left.date || '').localeCompare(String(right.date || ''));
    if (dateCompare !== 0) {
      return dateCompare;
    }

    const commitCompare = String(left.commit || '').localeCompare(String(right.commit || ''));
    if (commitCompare !== 0) {
      return commitCompare;
    }

    const leftIndex = SCENARIO_ORDER.indexOf(left.scenario);
    const rightIndex = SCENARIO_ORDER.indexOf(right.scenario);
    const normalizedLeft = leftIndex === -1 ? Number.MAX_SAFE_INTEGER : leftIndex;
    const normalizedRight = rightIndex === -1 ? Number.MAX_SAFE_INTEGER : rightIndex;

    if (normalizedLeft !== normalizedRight) {
      return normalizedLeft - normalizedRight;
    }

    return left.scenario.localeCompare(right.scenario);
  });
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

function toCsvValue(value) {
  const stringValue = value ?? '';
  if (typeof stringValue === 'number') {
    return String(stringValue);
  }

  const normalized = String(stringValue);
  if (!/[",\n]/.test(normalized)) {
    return normalized;
  }

  return `"${normalized.replace(/"/g, '""')}"`;
}

function parseNumericValue(value) {
  if (value === null || value === undefined) {
    return null;
  }

  const normalized = String(value).trim();
  if (!normalized || normalized.toLowerCase() === 'n/a') {
    return null;
  }

  const parsed = Number(normalized);
  return Number.isFinite(parsed) ? parsed : null;
}

function normalizeTextValue(value, fallback = '') {
  if (value === null || value === undefined) {
    return fallback;
  }

  const normalized = String(value).trim();
  return normalized || fallback;
}

function normalizeTrendRow(row) {
  return {
    scenario: normalizeTextValue(row.scenario),
    requests: parseNumericValue(row.requests),
    rps: parseNumericValue(row.rps),
    error_rate: parseNumericValue(row.error_rate),
    timeout_rate: parseNumericValue(row.timeout_rate),
    p95: parseNumericValue(row.p95),
    p99: parseNumericValue(row.p99),
    threshold_status: normalizeTextValue(row.threshold_status, 'not_configured'),
    threshold_failed_metrics: normalizeTextValue(row.threshold_failed_metrics),
    threshold_checked_metrics: normalizeTextValue(row.threshold_checked_metrics),
    error_rate_threshold: parseNumericValue(row.error_rate_threshold),
    error_rate_check: normalizeTextValue(row.error_rate_check, 'not_configured'),
    timeout_rate_threshold: parseNumericValue(row.timeout_rate_threshold),
    timeout_rate_check: normalizeTextValue(row.timeout_rate_check, 'not_configured'),
    p95_threshold: parseNumericValue(row.p95_threshold),
    p95_check: normalizeTextValue(row.p95_check, 'not_configured'),
    p99_threshold: parseNumericValue(row.p99_threshold),
    p99_check: normalizeTextValue(row.p99_check, 'not_configured'),
    commit: normalizeTextValue(row.commit),
    date: normalizeTextValue(row.date),
  };
}

function parseCsv(content) {
  const rows = [];
  let currentRow = [];
  let currentValue = '';
  let inQuotes = false;

  for (let index = 0; index < content.length; index += 1) {
    const character = content[index];

    if (inQuotes) {
      if (character === '"') {
        if (content[index + 1] === '"') {
          currentValue += '"';
          index += 1;
        } else {
          inQuotes = false;
        }
      } else {
        currentValue += character;
      }
      continue;
    }

    if (character === '"') {
      inQuotes = true;
      continue;
    }

    if (character === ',') {
      currentRow.push(currentValue);
      currentValue = '';
      continue;
    }

    if (character === '\n') {
      currentRow.push(currentValue);
      rows.push(currentRow);
      currentRow = [];
      currentValue = '';
      continue;
    }

    if (character === '\r') {
      continue;
    }

    currentValue += character;
  }

  if (currentValue.length > 0 || currentRow.length > 0) {
    currentRow.push(currentValue);
    rows.push(currentRow);
  }

  if (rows.length === 0) {
    return [];
  }

  const [header, ...dataRows] = rows;
  return dataRows
    .filter((row) => row.some((value) => String(value || '').trim()))
    .map((row) => {
      const record = {};
      for (let index = 0; index < header.length; index += 1) {
        const key = header[index];
        if (!key) {
          continue;
        }
        record[key] = row[index] ?? '';
      }
      return record;
    });
}

async function loadMetadata(metadataPath) {
  if (!metadataPath) {
    return {};
  }

  const content = await fs.readFile(metadataPath, 'utf8');
  return JSON.parse(content);
}

async function loadRows(inputDir, metadata) {
  const entries = await fs.readdir(inputDir, { withFileTypes: true });
  const summaryFiles = entries
    .filter((entry) => entry.isFile() && entry.name.endsWith('.summary.json'))
    .map((entry) => entry.name);

  if (summaryFiles.length === 0) {
    throw new Error(`No *.summary.json files found in ${inputDir}`);
  }

  const rows = await Promise.all(
    summaryFiles.map(async (fileName) => {
      const fullPath = path.join(inputDir, fileName);
      const content = await fs.readFile(fullPath, 'utf8');
      const summary = JSON.parse(content);
      const scenario = inferScenario(fileName, summary);
      const requests = getMetric(summary, 'scenario_requests_total', 'count')
        ?? getMetric(summary, 'http_reqs', 'count')
        ?? 0;
      const rps = getMetric(summary, 'http_reqs', 'rate');
      const errorRate = getMetric(summary, 'scenario_error_rate', 'rate');
      const timeoutRate = getMetric(summary, 'scenario_timeout_rate', 'rate');
      const p95 = getMetric(summary, 'http_req_duration', 'p(95)');
      const p99 = getMetric(summary, 'http_req_duration', 'p(99)');

      return {
        scenario,
        requests,
        rps,
        error_rate: errorRate,
        timeout_rate: timeoutRate,
        p95,
        p99,
        commit: metadata.commit || process.env.GITHUB_SHA || '',
        date: metadata.date || process.env.RUN_DATE_UTC || '',
      };
    }),
  );

  return sortRows(rows);
}

async function loadHistoryRows(historyCsvPath) {
  if (!historyCsvPath) {
    return [];
  }

  try {
    const content = await fs.readFile(historyCsvPath, 'utf8');
    return sortHistoryRows(parseCsv(content).map((row) => normalizeTrendRow(row)).filter((row) => row.scenario));
  } catch (error) {
    if (error && error.code === 'ENOENT') {
      return [];
    }
    throw error;
  }
}

function buildMarkdown(rows, metadata) {
  const lines = [
    '# k6 Nightly Trend',
    '',
  ];

  if (metadata.date) {
    lines.push(`- Date: \`${metadata.date}\``);
  }
  if (metadata.commit) {
    lines.push(`- Commit: \`${metadata.commit}\``);
  }
  if (metadata.base_url) {
    lines.push(`- Base URL: \`${metadata.base_url}\``);
  }
  if (metadata.vus || metadata.duration || metadata.timeout || metadata.rps) {
    lines.push(
      `- Config: \`vus=${metadata.vus || 'n/a'} duration=${metadata.duration || 'n/a'} timeout=${metadata.timeout || 'n/a'} rps=${metadata.rps || 'n/a'}\``,
    );
  }

  lines.push(
    '',
    '| scenario | requests | rps | error_rate | timeout_rate | p95 | p99 | threshold_result | commit | date |',
    '| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- | --- |',
  );

  for (const row of rows) {
    lines.push(
      `| ${row.scenario} | ${formatNumber(row.requests, 0)} | ${formatNumber(row.rps, 2)} | ${formatPercent(row.error_rate)} | ${formatPercent(row.timeout_rate)} | ${formatNumber(row.p95, 2)} ms | ${formatNumber(row.p99, 2)} ms | ${row.threshold_status} | ${row.commit || 'n/a'} | ${row.date || 'n/a'} |`,
    );
  }

  lines.push('');
  return lines.join('\n');
}

function buildCsv(rows) {
  const lines = [CSV_COLUMNS.join(',')];

  for (const row of rows) {
    lines.push([
      row.scenario,
      formatNumber(row.requests, 0),
      formatNumber(row.rps, 6),
      formatNumber(row.error_rate, 6),
      formatNumber(row.timeout_rate, 6),
      formatNumber(row.p95, 2),
      formatNumber(row.p99, 2),
      row.threshold_status,
      row.threshold_failed_metrics,
      row.threshold_checked_metrics,
      formatNumber(row.error_rate_threshold, 6),
      row.error_rate_check,
      formatNumber(row.timeout_rate_threshold, 6),
      row.timeout_rate_check,
      formatNumber(row.p95_threshold, 2),
      row.p95_check,
      formatNumber(row.p99_threshold, 2),
      row.p99_check,
      row.commit,
      row.date,
    ].map(toCsvValue).join(','));
  }

  lines.push('');
  return lines.join('\n');
}

function formatLatestMetricValue(metric, value) {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return 'n/a';
  }

  const scaled = metric.scale ? Number(value) * metric.scale : Number(value);
  return `${scaled.toFixed(metric.digits)}${metric.unit}`;
}

function formatLatestMetricDelta(metric, currentValue, previousValue) {
  if (
    currentValue === null
    || currentValue === undefined
    || Number.isNaN(currentValue)
    || previousValue === null
    || previousValue === undefined
    || Number.isNaN(previousValue)
  ) {
    return 'n/a';
  }

  const scale = metric.scale || 1;
  const delta = (Number(currentValue) - Number(previousValue)) * scale;
  const prefix = delta > 0 ? '+' : '';
  return `${prefix}${delta.toFixed(metric.digits)}${metric.unit}`;
}

function buildLatestMetricCell(metric, currentRow, previousRow) {
  const currentValue = formatLatestMetricValue(metric, currentRow[metric.key]);

  if (!previousRow) {
    return `${currentValue} (first sample)`;
  }

  const deltaValue = formatLatestMetricDelta(metric, currentRow[metric.key], previousRow[metric.key]);
  return `${currentValue} (${deltaValue})`;
}

function findLatestHistoryRowsByScenario(historyRows) {
  const latestRows = new Map();

  for (const row of historyRows) {
    if (!row.scenario) {
      continue;
    }

    const previousRow = latestRows.get(row.scenario);
    if (!previousRow) {
      latestRows.set(row.scenario, row);
      continue;
    }

    const candidateRows = sortHistoryRows([previousRow, row]);
    latestRows.set(row.scenario, candidateRows[candidateRows.length - 1]);
  }

  return latestRows;
}

function buildLatestMarkdown(currentRows, historyRows, metadata) {
  const previousRowsByScenario = findLatestHistoryRowsByScenario(historyRows);
  const lines = [
    '# k6 Trend Snapshot',
    '',
  ];

  if (metadata.date) {
    lines.push(`- Current date: \`${metadata.date}\``);
  }
  if (metadata.commit) {
    lines.push(`- Current commit: \`${metadata.commit}\``);
  }
  lines.push(`- History rows loaded: \`${historyRows.length}\``);
  lines.push('- Delta legend: values are `current - previous` for the same scenario.');
  lines.push('- Interpretation: latency/error deltas below `0` are better; throughput deltas above `0` are better.');
  lines.push(
    '',
    '| scenario | previous sample | requests | rps | p95 | p99 | error_rate | timeout_rate | threshold_result |',
    '| --- | --- | --- | --- | --- | --- | --- | --- | --- |',
  );

  for (const row of sortRows(currentRows)) {
    const previousRow = previousRowsByScenario.get(row.scenario);
    const previousSample = previousRow
      ? `${previousRow.date || 'n/a'} / ${previousRow.commit || 'n/a'}`
      : 'first sample';

    lines.push(
      `| ${row.scenario} | ${previousSample} | ${buildLatestMetricCell(LATEST_TREND_METRICS[0], row, previousRow)} | ${buildLatestMetricCell(LATEST_TREND_METRICS[1], row, previousRow)} | ${buildLatestMetricCell(LATEST_TREND_METRICS[2], row, previousRow)} | ${buildLatestMetricCell(LATEST_TREND_METRICS[3], row, previousRow)} | ${buildLatestMetricCell(LATEST_TREND_METRICS[4], row, previousRow)} | ${buildLatestMetricCell(LATEST_TREND_METRICS[5], row, previousRow)} | ${row.threshold_status} |`,
    );
  }

  lines.push('');
  return lines.join('\n');
}

function getTrendRowIdentity(row) {
  return `${row.date || ''}::${row.commit || ''}::${row.scenario || ''}`;
}

function mergeHistoryRows(historyRows, currentRows) {
  const mergedRows = new Map();

  for (const row of historyRows) {
    if (!row.scenario) {
      continue;
    }
    mergedRows.set(getTrendRowIdentity(row), normalizeTrendRow(row));
  }

  for (const row of currentRows) {
    if (!row.scenario) {
      continue;
    }
    mergedRows.set(getTrendRowIdentity(row), normalizeTrendRow(row));
  }

  return sortHistoryRows([...mergedRows.values()]);
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const inputDir = path.resolve(args['input-dir']);
  const outputDir = path.resolve(args['output-dir']);
  const historyCsvPath = args['history-csv'] ? path.resolve(args['history-csv']) : '';
  const metadata = await loadMetadata(args.metadata);
  const thresholdConfig = await loadThresholdConfig({
    thresholdsPath: args.thresholds,
    thresholdsJson: args['thresholds-json'],
  });
  const baseRows = await loadRows(inputDir, metadata);
  const { rows, report } = applyThresholds(baseRows, thresholdConfig);
  const markdown = buildMarkdown(rows, metadata);
  const csv = buildCsv(rows);
  const thresholdMarkdown = buildThresholdMarkdown(rows, report);
  const currentRows = rows.map((row) => normalizeTrendRow(row));
  const historyModeEnabled = Boolean(args['history-csv']);
  const historyRows = historyModeEnabled ? await loadHistoryRows(historyCsvPath) : [];
  const mergedHistoryRows = historyModeEnabled ? mergeHistoryRows(historyRows, currentRows) : [];
  const latestMarkdown = historyModeEnabled ? buildLatestMarkdown(currentRows, historyRows, metadata) : '';
  const historyCsv = historyModeEnabled ? buildCsv(mergedHistoryRows) : '';
  const thresholdReport = {
    configured: report.configured,
    checked: report.checked,
    source: report.source,
    overall_status: report.overall_status,
    has_failures: report.has_failures,
    checked_scenarios: report.checked_scenarios,
    failed_scenarios: report.failed_scenarios,
    rows: rows.map((row) => ({
      scenario: row.scenario,
      threshold_status: row.threshold_status,
      threshold_failed_metrics: row.threshold_failed_metrics
        ? row.threshold_failed_metrics.split('|').filter(Boolean)
        : [],
      threshold_checked_metrics: row.threshold_checked_metrics
        ? row.threshold_checked_metrics.split('|').filter(Boolean)
        : [],
      checks: row.checks,
    })),
  };

  await fs.mkdir(outputDir, { recursive: true });
  const writes = [
    fs.writeFile(path.join(outputDir, 'perf-trend.md'), markdown, 'utf8'),
    fs.writeFile(path.join(outputDir, 'perf-trend.csv'), csv, 'utf8'),
    fs.writeFile(path.join(outputDir, 'perf-thresholds.md'), thresholdMarkdown, 'utf8'),
    fs.writeFile(path.join(outputDir, 'perf-threshold-report.json'), `${JSON.stringify(thresholdReport, null, 2)}\n`, 'utf8'),
  ];

  if (historyModeEnabled) {
    writes.push(
      fs.writeFile(path.join(outputDir, 'perf-trend-history.csv'), historyCsv, 'utf8'),
      fs.writeFile(path.join(outputDir, 'perf-trend-latest.md'), latestMarkdown, 'utf8'),
    );
  }

  await Promise.all(writes);

  process.stdout.write(`Generated ${rows.length} trend rows in ${outputDir}\n`);
  if (historyModeEnabled) {
    process.stdout.write(`Generated ${mergedHistoryRows.length} merged history rows from ${historyRows.length} historical rows\n`);
  }
  process.stdout.write(`Threshold status: ${report.overall_status}\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.message}\n`);
  process.exitCode = 1;
});
