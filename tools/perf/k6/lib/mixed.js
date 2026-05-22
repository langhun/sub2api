import { sleep } from 'k6';
import { BASELINE_SCENARIOS } from './config.js';

export function createWeightedMixPlan(weights) {
  let totalWeight = 0;
  const entries = [];

  for (const name of BASELINE_SCENARIOS) {
    const weight = weights[name] || 0;
    if (weight <= 0) {
      continue;
    }

    totalWeight += weight;
    entries.push({
      name,
      threshold: totalWeight,
    });
  }

  return {
    entries,
    totalWeight,
  };
}

export function selectFromMixPlan(plan, randomValue = Math.random()) {
  const point = randomValue * plan.totalWeight;
  const selected = plan.entries.find((entry) => point < entry.threshold);
  return selected ? selected.name : plan.entries[plan.entries.length - 1].name;
}

export function applyMixedPacing(pacing) {
  const delayMs = pickPacingMs(pacing);
  if (delayMs > 0) {
    sleep(delayMs / 1000);
  }
}

export function formatMixedWeights(weights) {
  return BASELINE_SCENARIOS.map((name) => `${name}:${weights[name] || 0}`).join(',');
}

export function formatMixedPacing(pacing) {
  if (pacing.minMs === pacing.maxMs) {
    return String(pacing.minMs);
  }

  return `${pacing.minMs}-${pacing.maxMs}`;
}

function pickPacingMs(pacing) {
  if (pacing.maxMs <= pacing.minMs) {
    return pacing.minMs;
  }

  return pacing.minMs + Math.random() * (pacing.maxMs - pacing.minMs);
}
