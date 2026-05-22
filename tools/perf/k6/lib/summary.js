function safeMetric(metrics, key, nestedKey) {
  const metric = metrics?.[key];
  if (!metric) {
    return null;
  }

  if (nestedKey) {
    return metric.values?.[nestedKey] ?? null;
  }

  return metric.values ?? null;
}

function formatNumber(value, digits = 2) {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return 'n/a';
  }

  return Number(value).toFixed(digits);
}

export function createTextSummary(data, context) {
  const durationMs = safeMetric(data.metrics, 'iteration_duration', 'avg');
  const totalRequests = safeMetric(data.metrics, 'http_reqs', 'count') || 0;
  const throughput = safeMetric(data.metrics, 'http_reqs', 'rate');
  const errorRate = safeMetric(data.metrics, 'scenario_error_rate', 'rate');
  const timeoutRate = safeMetric(data.metrics, 'scenario_timeout_rate', 'rate');
  const p50 = safeMetric(data.metrics, 'http_req_duration', 'p(50)');
  const p95 = safeMetric(data.metrics, 'http_req_duration', 'p(95)');
  const p99 = safeMetric(data.metrics, 'http_req_duration', 'p(99)');
  const extraLines = context.extraLines || [];

  return [
    '',
    'PERF_BASELINE_SUMMARY',
    `scenario=${context.scenario}`,
    `target=${context.target}`,
    `vus=${context.vus}`,
    `duration=${context.duration}`,
    `requests=${formatNumber(totalRequests, 0)}`,
    `throughput_rps=${formatNumber(throughput)}`,
    `error_rate=${formatNumber((errorRate || 0) * 100)}%`,
    `timeout_rate=${formatNumber((timeoutRate || 0) * 100)}%`,
    `p50_ms=${formatNumber(p50)}`,
    `p95_ms=${formatNumber(p95)}`,
    `p99_ms=${formatNumber(p99)}`,
    `iteration_avg_ms=${formatNumber(durationMs)}`,
    ...extraLines,
    '',
  ].join('\n');
}

export function createSummary(data, context) {
  return {
    stdout: `${createTextSummary(data, context)}\n`,
  };
}
