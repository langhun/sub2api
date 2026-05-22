import { promises as fs } from 'node:fs';
import path from 'node:path';
import {
  applyThresholds,
  buildThresholdMarkdown,
  loadThresholdConfig,
} from './thresholds.mjs';

const SCENARIO_ORDER = ['health', 'pricing', 'monitoring-summary', 'mixed'];

function parseArgs(argv) {
  const args = {
    'input-dir': '',
    'output-dir': '',
    metadata: '',
    thresholds: '',
    'thresholds-json': '',
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
  const header = [
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
  const lines = [header.join(',')];

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

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const inputDir = path.resolve(args['input-dir']);
  const outputDir = path.resolve(args['output-dir']);
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
  await Promise.all([
    fs.writeFile(path.join(outputDir, 'perf-trend.md'), markdown, 'utf8'),
    fs.writeFile(path.join(outputDir, 'perf-trend.csv'), csv, 'utf8'),
    fs.writeFile(path.join(outputDir, 'perf-thresholds.md'), thresholdMarkdown, 'utf8'),
    fs.writeFile(path.join(outputDir, 'perf-threshold-report.json'), `${JSON.stringify(thresholdReport, null, 2)}\n`, 'utf8'),
  ]);

  process.stdout.write(`Generated ${rows.length} trend rows in ${outputDir}\n`);
  process.stdout.write(`Threshold status: ${report.overall_status}\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.message}\n`);
  process.exitCode = 1;
});
