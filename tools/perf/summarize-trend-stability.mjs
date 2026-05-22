import { promises as fs } from 'node:fs';
import path from 'node:path';

const SCENARIO_ORDER = ['health', 'pricing', 'monitoring-summary', 'mixed'];

function parseArgs(argv) {
  const args = {
    input: '',
    'output-md': '',
    'output-json': '',
    'recent-runs': '10',
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

  if (!args.input) {
    throw new Error('Missing required argument: --input');
  }

  if (!args['output-md']) {
    throw new Error('Missing required argument: --output-md');
  }

  return args;
}

function parsePositiveInt(rawValue, label) {
  const value = Number.parseInt(String(rawValue), 10);
  if (!Number.isFinite(value) || value <= 0) {
    throw new Error(`${label} must be a positive integer, got: ${rawValue}`);
  }

  return value;
}

function parseCsv(text) {
  const rows = [];
  let row = [];
  let field = '';
  let inQuotes = false;

  for (let index = 0; index < text.length; index += 1) {
    const char = text[index];
    const next = text[index + 1];

    if (char === '"') {
      if (inQuotes && next === '"') {
        field += '"';
        index += 1;
        continue;
      }

      inQuotes = !inQuotes;
      continue;
    }

    if (!inQuotes && char === ',') {
      row.push(field);
      field = '';
      continue;
    }

    if (!inQuotes && (char === '\n' || char === '\r')) {
      if (char === '\r' && next === '\n') {
        index += 1;
      }

      row.push(field);
      field = '';

      if (row.length > 1 || row[0] !== '') {
        rows.push(row);
      }

      row = [];
      continue;
    }

    field += char;
  }

  if (field !== '' || row.length > 0) {
    row.push(field);
    if (row.length > 1 || row[0] !== '') {
      rows.push(row);
    }
  }

  return rows;
}

function parseCsvRecords(text) {
  const rows = parseCsv(text);
  if (rows.length === 0) {
    return [];
  }

  const [header, ...body] = rows;
  return body.map((values) => {
    const record = {};
    for (let index = 0; index < header.length; index += 1) {
      record[header[index]] = values[index] ?? '';
    }
    return record;
  });
}

function parseNullableNumber(rawValue) {
  if (rawValue === null || rawValue === undefined) {
    return null;
  }

  const normalized = String(rawValue).trim();
  if (!normalized || normalized.toLowerCase() === 'n/a') {
    return null;
  }

  const value = Number(normalized);
  return Number.isFinite(value) ? value : null;
}

function formatNumber(value, digits = 2) {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return 'n/a';
  }

  return Number(value).toFixed(digits);
}

function formatPercentFromRatio(value) {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return 'n/a';
  }

  return `${(Number(value) * 100).toFixed(1)}%`;
}

function formatScenarioList(items) {
  if (items.length === 0) {
    return 'none';
  }

  return items.map((item) => {
    const failedMetrics = item.failed_metrics.length > 0
      ? ` [${item.failed_metrics.join(', ')}]`
      : '';
    return `\`${item.scenario}\`${failedMetrics}`;
  }).join(', ');
}

function normalizeRow(record, index) {
  return {
    row_index: index,
    scenario: String(record.scenario || '').trim(),
    threshold_status: String(record.threshold_status || 'not_configured').trim() || 'not_configured',
    threshold_failed_metrics: String(record.threshold_failed_metrics || '')
      .split('|')
      .map((item) => item.trim())
      .filter(Boolean),
    p95: parseNullableNumber(record.p95),
    p99: parseNullableNumber(record.p99),
    commit: String(record.commit || '').trim(),
    date: String(record.date || '').trim(),
  };
}

function compareScenarios(left, right) {
  const leftIndex = SCENARIO_ORDER.indexOf(left);
  const rightIndex = SCENARIO_ORDER.indexOf(right);
  const normalizedLeft = leftIndex === -1 ? Number.MAX_SAFE_INTEGER : leftIndex;
  const normalizedRight = rightIndex === -1 ? Number.MAX_SAFE_INTEGER : rightIndex;

  if (normalizedLeft !== normalizedRight) {
    return normalizedLeft - normalizedRight;
  }

  return left.localeCompare(right);
}

function buildRunKey(row) {
  return `${row.date || 'unknown-date'}::${row.commit || 'unknown-commit'}`;
}

function getTimestamp(rawDate) {
  const value = Date.parse(rawDate);
  return Number.isFinite(value) ? value : null;
}

function summarizeRun(run) {
  const failRows = run.rows.filter((row) => row.threshold_status === 'fail');
  const passRows = run.rows.filter((row) => row.threshold_status === 'pass');
  const checkedRows = run.rows.filter((row) => row.threshold_status === 'pass' || row.threshold_status === 'fail');
  const overallStatus = failRows.length > 0
    ? 'fail'
    : checkedRows.length === 0
      ? 'not_configured'
      : passRows.length === run.rows.length
        ? 'pass'
        : 'partial';

  return {
    ...run,
    checked: checkedRows.length > 0,
    overall_status: overallStatus,
    failed_scenarios: failRows
      .map((row) => ({
        scenario: row.scenario,
        failed_metrics: row.threshold_failed_metrics,
      }))
      .sort((left, right) => compareScenarios(left.scenario, right.scenario)),
  };
}

function groupRuns(rows) {
  const runs = new Map();

  for (const row of rows) {
    const key = buildRunKey(row);
    if (!runs.has(key)) {
      runs.set(key, {
        key,
        date: row.date,
        commit: row.commit,
        first_row_index: row.row_index,
        rows: [],
      });
    }

    runs.get(key).rows.push(row);
  }

  return Array.from(runs.values())
    .map((run) => summarizeRun(run))
    .sort((left, right) => {
      const leftTimestamp = getTimestamp(left.date);
      const rightTimestamp = getTimestamp(right.date);

      if (leftTimestamp !== null && rightTimestamp !== null && leftTimestamp !== rightTimestamp) {
        return rightTimestamp - leftTimestamp;
      }

      if (leftTimestamp !== null && rightTimestamp === null) {
        return -1;
      }

      if (leftTimestamp === null && rightTimestamp !== null) {
        return 1;
      }

      return right.first_row_index - left.first_row_index;
    });
}

function average(values) {
  const filtered = values.filter((value) => value !== null && value !== undefined && !Number.isNaN(value));
  if (filtered.length === 0) {
    return null;
  }

  return filtered.reduce((sum, value) => sum + Number(value), 0) / filtered.length;
}

function buildScenarioSummaries(runs) {
  const byScenario = new Map();

  for (const run of runs) {
    for (const row of run.rows) {
      if (!row.scenario) {
        continue;
      }

      if (!byScenario.has(row.scenario)) {
        byScenario.set(row.scenario, []);
      }

      byScenario.get(row.scenario).push(row);
    }
  }

  return Array.from(byScenario.entries())
    .map(([scenario, rows]) => {
      const checkedRows = rows.filter((row) => row.threshold_status === 'pass' || row.threshold_status === 'fail');
      const passedRows = checkedRows.filter((row) => row.threshold_status === 'pass');

      return {
        scenario,
        samples: rows.length,
        checked_samples: checkedRows.length,
        pass_rate: checkedRows.length > 0 ? passedRows.length / checkedRows.length : null,
        avg_p95: average(rows.map((row) => row.p95)),
        avg_p99: average(rows.map((row) => row.p99)),
      };
    })
    .sort((left, right) => compareScenarios(left.scenario, right.scenario));
}

function buildSummary(runs, recentRunsRequested) {
  const windowRuns = runs.slice(0, recentRunsRequested);
  const checkedRuns = windowRuns.filter((run) => run.checked);
  const passedRuns = checkedRuns.filter((run) => run.overall_status === 'pass');
  const passRate = checkedRuns.length > 0 ? passedRuns.length / checkedRuns.length : null;

  let currentPassStreak = 0;
  for (const run of windowRuns) {
    if (run.overall_status !== 'pass') {
      break;
    }
    currentPassStreak += 1;
  }

  const latestFailureRun = windowRuns.find((run) => run.failed_scenarios.length > 0) || null;

  return {
    recent_runs_requested: recentRunsRequested,
    recent_runs_considered: windowRuns.length,
    threshold_checked_runs: checkedRuns.length,
    passed_runs: passedRuns.length,
    pass_rate: passRate,
    current_pass_streak: currentPassStreak,
    latest_failure: latestFailureRun
      ? {
          date: latestFailureRun.date,
          commit: latestFailureRun.commit,
          scenarios: latestFailureRun.failed_scenarios,
        }
      : null,
    scenarios: buildScenarioSummaries(windowRuns),
  };
}

function buildMarkdown(summary) {
  const lines = [
    '## Perf Stability Summary',
    '',
    `- Window: recent \`${summary.recent_runs_requested}\` runs requested, \`${summary.recent_runs_considered}\` runs found`,
    `- Threshold-checked runs: \`${summary.threshold_checked_runs}\``,
    `- Rolling pass rate: \`${summary.passed_runs}/${summary.threshold_checked_runs}\` (${formatPercentFromRatio(summary.pass_rate)})`,
    `- Current pass streak: \`${summary.current_pass_streak}\``,
  ];

  if (summary.latest_failure) {
    lines.push(
      `- Latest threshold failure: date=\`${summary.latest_failure.date || 'n/a'}\` commit=\`${summary.latest_failure.commit || 'n/a'}\` scenarios=${formatScenarioList(summary.latest_failure.scenarios)}`,
    );
  } else {
    lines.push('- Latest threshold failure: none');
  }

  lines.push(
    '',
    '| scenario | samples | checked_samples | pass_rate | avg_p95 | avg_p99 |',
    '| --- | ---: | ---: | ---: | ---: | ---: |',
  );

  for (const scenario of summary.scenarios) {
    lines.push(
      `| ${scenario.scenario} | ${formatNumber(scenario.samples, 0)} | ${formatNumber(scenario.checked_samples, 0)} | ${formatPercentFromRatio(scenario.pass_rate)} | ${formatNumber(scenario.avg_p95, 2)} ms | ${formatNumber(scenario.avg_p99, 2)} ms |`,
    );
  }

  lines.push('');
  return lines.join('\n');
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const inputPath = path.resolve(args.input);
  const outputMdPath = path.resolve(args['output-md']);
  const outputJsonPath = args['output-json'] ? path.resolve(args['output-json']) : '';
  const recentRuns = parsePositiveInt(args['recent-runs'], '--recent-runs');
  const content = await fs.readFile(inputPath, 'utf8');
  const rows = parseCsvRecords(content).map((record, index) => normalizeRow(record, index));
  const runs = groupRuns(rows);
  const summary = buildSummary(runs, recentRuns);
  const markdown = buildMarkdown(summary);

  await fs.mkdir(path.dirname(outputMdPath), { recursive: true });
  await fs.writeFile(outputMdPath, markdown, 'utf8');

  if (outputJsonPath) {
    await fs.mkdir(path.dirname(outputJsonPath), { recursive: true });
    await fs.writeFile(outputJsonPath, `${JSON.stringify(summary, null, 2)}\n`, 'utf8');
  }

  process.stdout.write(`Generated stability summary from ${summary.recent_runs_considered} run(s)\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.message}\n`);
  process.exitCode = 1;
});
