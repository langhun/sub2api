import { promises as fs } from 'node:fs';
import path from 'node:path';

const SCENARIO_ORDER = ['health', 'pricing', 'monitoring-summary', 'mixed'];

function parseArgs(argv) {
  const args = {
    'input-dir': '',
    'output-dir': '',
    metadata: '',
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
    '| scenario | requests | rps | error_rate | timeout_rate | p95 | p99 | commit | date |',
    '| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- |',
  );

  for (const row of rows) {
    lines.push(
      `| ${row.scenario} | ${formatNumber(row.requests, 0)} | ${formatNumber(row.rps, 2)} | ${formatPercent(row.error_rate)} | ${formatPercent(row.timeout_rate)} | ${formatNumber(row.p95, 2)} ms | ${formatNumber(row.p99, 2)} ms | ${row.commit || 'n/a'} | ${row.date || 'n/a'} |`,
    );
  }

  lines.push('');
  return lines.join('\n');
}

function buildCsv(rows) {
  const header = ['scenario', 'requests', 'rps', 'error_rate', 'timeout_rate', 'p95', 'p99', 'commit', 'date'];
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
  const rows = await loadRows(inputDir, metadata);
  const markdown = buildMarkdown(rows, metadata);
  const csv = buildCsv(rows);

  await fs.mkdir(outputDir, { recursive: true });
  await Promise.all([
    fs.writeFile(path.join(outputDir, 'perf-trend.md'), markdown, 'utf8'),
    fs.writeFile(path.join(outputDir, 'perf-trend.csv'), csv, 'utf8'),
  ]);

  process.stdout.write(`Generated ${rows.length} trend rows in ${outputDir}\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.message}\n`);
  process.exitCode = 1;
});
