import { promises as fs } from 'node:fs';
import path from 'node:path';

function parseArgs(argv) {
  const args = {
    report: '',
    enforce: '',
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

  if (!args.report) {
    throw new Error('Missing required argument: --report');
  }

  return args;
}

function parseBoolean(rawValue) {
  if (!rawValue) {
    return false;
  }

  return ['1', 'true', 'yes', 'on'].includes(String(rawValue).trim().toLowerCase());
}

function buildFailureMessage(report) {
  const failures = (report.rows || [])
    .filter((row) => row.threshold_status === 'fail')
    .map((row) => `${row.scenario} [${(row.threshold_failed_metrics || []).join(', ')}]`);

  if (failures.length === 0) {
    return 'Threshold failures detected';
  }

  return `Threshold failures detected: ${failures.join('; ')}`;
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const reportPath = path.resolve(args.report);
  const enforce = parseBoolean(args.enforce || process.env.PERF_NIGHTLY_ENFORCE || '');
  const content = await fs.readFile(reportPath, 'utf8');
  const report = JSON.parse(content);

  process.stdout.write(`Threshold report status: ${report.overall_status}\n`);

  if (report.overall_status === 'fail') {
    const message = buildFailureMessage(report);
    process.stdout.write(`${message}\n`);
    if (enforce) {
      process.stderr.write(`::error::${message}\n`);
      process.stderr.write(`${message}\n`);
      process.exitCode = 1;
      return;
    }

    process.stdout.write(`::warning::${message}\n`);
    process.stdout.write('Threshold guard is running in warning mode.\n');
    return;
  }

  if (report.overall_status === 'pass') {
    process.stdout.write('All configured thresholds passed.\n');
    return;
  }

  process.stdout.write('Threshold guard not configured; nothing to enforce.\n');
}

main().catch((error) => {
  process.stderr.write(`${error.message}\n`);
  process.exitCode = 1;
});
