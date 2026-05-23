import { promises as fs } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(scriptDir, '..', '..');

function buildRule(description, fragments) {
  return { description, fragments };
}

function buildSharedRules(workflow) {
  return [
    buildRule('workflow_dispatch exposes history_source with all supported options', [
      { label: 'history_source input', snippet: '      history_source:' },
      { label: "history_source default", snippet: "        default: 'previous-run-artifact'" },
      { label: 'history_source choice type', snippet: '        type: choice' },
      { label: 'history_source option previous-run-artifact', snippet: '          - previous-run-artifact' },
      { label: 'history_source option artifact', snippet: '          - artifact' },
      { label: 'history_source option path', snippet: '          - path' },
      { label: 'history_source option none', snippet: '          - none' },
    ]),
    buildRule('workflow_dispatch exposes history_run_id, history_artifact_name, and history_csv_path inputs', [
      { label: 'history_run_id input', snippet: '      history_run_id:' },
      { label: 'history_artifact_name input', snippet: '      history_artifact_name:' },
      { label: 'history_csv_path input', snippet: '      history_csv_path:' },
    ]),
    buildRule('non-dispatch runs force history_source back to previous-run-artifact', [
      { label: 'dispatch guard', snippet: 'if [ "${GITHUB_EVENT_NAME}" != "workflow_dispatch" ]; then' },
      { label: 'history_source fallback', snippet: 'HISTORY_SOURCE="previous-run-artifact"' },
    ]),
    buildRule('history artifact lookup uses the expected trend artifact prefix', [
      { label: 'artifact auto-detect prefix', snippet: `.startsWith('${workflow.artifactPrefix}-trend-')` },
      { label: 'history artifact step', snippet: '- name: Resolve history artifact metadata' },
    ]),
    buildRule('history download step emits fallback_reason outputs for artifact failures and successes', [
      { label: 'download history step', snippet: '- name: Download history artifact' },
      { label: 'download failed fallback reason', snippet: 'echo "fallback_reason=download_failed" >> "$GITHUB_OUTPUT"' },
      { label: 'download ok fallback reason', snippet: 'echo "fallback_reason=download_ok" >> "$GITHUB_OUTPUT"' },
      { label: 'csv missing fallback reason', snippet: 'echo "fallback_reason=csv_not_found_inside_artifact" >> "$GITHUB_OUTPUT"' },
    ]),
    buildRule('history path step emits fallback_reason outputs for missing and valid CSV paths', [
      { label: 'resolve history path step', snippet: '- name: Resolve history CSV path' },
      { label: 'path input missing fallback reason', snippet: 'echo "fallback_reason=path_input_missing" >> "$GITHUB_OUTPUT"' },
      { label: 'path not found fallback reason', snippet: 'echo "fallback_reason=path_not_found" >> "$GITHUB_OUTPUT"' },
      { label: 'path ok fallback reason', snippet: 'echo "fallback_reason=path_ok" >> "$GITHUB_OUTPUT"' },
    ]),
    buildRule('final history step exports history CSV path and fallback_reason', [
      { label: 'finalize history step', snippet: '- name: Finalize history input' },
      { label: 'history csv env export', snippet: 'echo "HISTORY_CSV_PATH=$HISTORY_CSV_PATH" >> "$GITHUB_ENV"' },
      { label: 'history csv output export', snippet: 'echo "csv_path=$HISTORY_CSV_PATH" >> "$GITHUB_OUTPUT"' },
      { label: 'final fallback reason output', snippet: 'echo "fallback_reason=$FALLBACK_REASON" >> "$GITHUB_OUTPUT"' },
      { label: 'summary final history csv path', snippet: 'echo "- Final history CSV path: ' },
      { label: 'summary final fallback reason', snippet: 'echo "- Final fallback reason: ' },
    ]),
    buildRule('trend export step forwards HISTORY_CSV_PATH via --history-csv', [
      { label: 'trend build step', snippet: '- name: Build trend markdown and csv' },
      { label: 'history csv guard', snippet: 'if [ -n "${HISTORY_CSV_PATH:-}" ]; then' },
      { label: 'history csv flag', snippet: 'cmd+=(--history-csv "$HISTORY_CSV_PATH")' },
    ]),
    buildRule('step summary appends perf-trend.md and perf-trend-latest.md when present', [
      { label: 'attach summary step', snippet: '- name: Attach trend summary to workflow' },
      { label: 'trend markdown append', snippet: 'cat "$TREND_DIR/perf-trend.md" >> "$GITHUB_STEP_SUMMARY"' },
      { label: 'latest markdown guard', snippet: 'if [ -f "$TREND_DIR/perf-trend-latest.md" ]; then' },
      { label: 'latest markdown append', snippet: 'cat "$TREND_DIR/perf-trend-latest.md" >> "$GITHUB_STEP_SUMMARY"' },
      { label: 'threshold markdown append', snippet: 'cat "$TREND_DIR/perf-thresholds.md" >> "$GITHUB_STEP_SUMMARY"' },
    ]),
    buildRule('upload steps keep raw and trend artifact names on the expected prefix contract', [
      { label: 'upload raw step', snippet: '- name: Upload raw perf artifacts' },
      { label: 'raw artifact name', snippet: `name: ${workflow.artifactPrefix}-raw-\${{ steps.resolve.outputs.artifact_suffix }}` },
      { label: 'upload trend step', snippet: '- name: Upload trend artifacts' },
      { label: 'trend artifact name', snippet: `name: ${workflow.artifactPrefix}-trend-\${{ steps.resolve.outputs.artifact_suffix }}` },
    ]),
  ];
}

const workflows = [
  {
    label: 'Perf Nightly',
    file: '.github/workflows/perf-nightly.yml',
    artifactPrefix: 'perf-nightly',
  },
  {
    label: 'Perf Long Run',
    file: '.github/workflows/perf-long-run.yml',
    artifactPrefix: 'perf-long-run',
  },
];

function evaluateRule(content, rule) {
  const missing = rule.fragments
    .filter((fragment) => !content.includes(fragment.snippet))
    .map((fragment) => fragment.label);

  return {
    description: rule.description,
    ok: missing.length === 0,
    missing,
  };
}

async function loadWorkflowContent(relativePath) {
  const absolutePath = path.join(repoRoot, relativePath);
  const content = await fs.readFile(absolutePath, 'utf8');
  return { absolutePath, content };
}

async function main() {
  let totalChecks = 0;
  let passedChecks = 0;
  let failedChecks = 0;

  process.stdout.write('Perf workflow readiness verification\n\n');

  for (const workflow of workflows) {
    const { absolutePath, content } = await loadWorkflowContent(workflow.file);
    const results = buildSharedRules(workflow).map((rule) => evaluateRule(content, rule));

    process.stdout.write(`${workflow.label}\n`);
    process.stdout.write(`  File: ${path.relative(repoRoot, absolutePath)}\n`);

    for (const result of results) {
      totalChecks += 1;
      if (result.ok) {
        passedChecks += 1;
        process.stdout.write(`  [PASS] ${result.description}\n`);
        continue;
      }

      failedChecks += 1;
      process.stdout.write(`  [FAIL] ${result.description}\n`);
      process.stdout.write(`         Missing: ${result.missing.join(', ')}\n`);
    }

    process.stdout.write('\n');
  }

  if (failedChecks > 0) {
    process.stderr.write(`Readiness check failed: ${failedChecks}/${totalChecks} checks failed.\n`);
    process.exitCode = 1;
    return;
  }

  process.stdout.write(`Readiness check passed: ${passedChecks}/${totalChecks} checks passed.\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.message}\n`);
  process.exitCode = 1;
});
