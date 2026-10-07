<template>
  <div class="weight-sliders">
    <div class="weight-control">
      <label class="weight-label">
        <span>{{ t('keys.routing.priceWeight') }}</span>
        <span class="weight-value">{{ priceWeight }}%</span>
      </label>
      <input
        :value="priceWeight"
        type="range"
        min="0"
        max="100"
        step="1"
        class="weight-slider"
        @input="onPriceChange($event.target.value)"
      />
    </div>

    <div class="weight-control">
      <label class="weight-label">
        <span>{{ t('keys.routing.speedWeight') }}</span>
        <span class="weight-value">{{ speedWeight }}%</span>
      </label>
      <input
        :value="speedWeight"
        type="range"
        min="0"
        max="100"
        step="1"
        class="weight-slider"
        @input="onSpeedChange($event.target.value)"
      />
    </div>

    <div class="weight-control">
      <label class="weight-label">
        <span>{{ t('keys.routing.successWeight') }}</span>
        <span class="weight-value">{{ successWeight }}%</span>
      </label>
      <input
        :value="successWeight"
        type="range"
        min="0"
        max="100"
        step="1"
        class="weight-slider"
        @input="onSuccessChange($event.target.value)"
      />
    </div>

    <div v-if="Math.abs(weightSum - 100) > 0.1" class="weight-error">
      <Icon name="alert-triangle" size="sm" />
      <span>{{ t('keys.routing.weightSumError', { sum: weightSum }) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/common/Icon.vue'

const props = defineProps<{
  priceWeight: number
  speedWeight: number
  successWeight: number
}>()

const emit = defineEmits<{
  (event: 'update:priceWeight', value: number): void
  (event: 'update:speedWeight', value: number): void
  (event: 'update:successWeight', value: number): void
}>()

const { t } = useI18n()

const weightSum = computed(() =>
  props.priceWeight + props.speedWeight + props.successWeight
)

function normalizeWeights(changed: 'price' | 'speed' | 'success', newValue: number) {
  const others: ('price' | 'speed' | 'success')[] = ['price', 'speed', 'success'].filter(w => w !== changed) as ('price' | 'speed' | 'success')[]
  const remaining = 100 - newValue

  // 计算其他两个权重的总和
  const otherTotal = others.reduce((sum, w) => {
    if (w === 'price') return sum + props.priceWeight
    if (w === 'speed') return sum + props.speedWeight
    return sum + props.successWeight
  }, 0)

  // 按比例分配剩余权重
  if (otherTotal > 0) {
    others.forEach(w => {
      const currentValue = w === 'price' ? props.priceWeight : w === 'speed' ? props.speedWeight : props.successWeight
      const newWeight = Math.round((currentValue / otherTotal) * remaining)
      if (w === 'price') emit('update:priceWeight', newWeight)
      else if (w === 'speed') emit('update:speedWeight', newWeight)
      else emit('update:successWeight', newWeight)
    })
  } else {
    // 均分
    const equalWeight = Math.floor(remaining / 2)
    if (others[0] === 'price') emit('update:priceWeight', equalWeight)
    else if (others[0] === 'speed') emit('update:speedWeight', equalWeight)
    else emit('update:successWeight', equalWeight)

    if (others[1] === 'price') emit('update:priceWeight', remaining - equalWeight)
    else if (others[1] === 'speed') emit('update:speedWeight', remaining - equalWeight)
    else emit('update:successWeight', remaining - equalWeight)
  }
}

function onPriceChange(value: string | number) {
  const numValue = typeof value === 'string' ? parseInt(value, 10) : value
  emit('update:priceWeight', numValue)
  normalizeWeights('price', numValue)
}

function onSpeedChange(value: string | number) {
  const numValue = typeof value === 'string' ? parseInt(value, 10) : value
  emit('update:speedWeight', numValue)
  normalizeWeights('speed', numValue)
}

function onSuccessChange(value: string | number) {
  const numValue = typeof value === 'string' ? parseInt(value, 10) : value
  emit('update:successWeight', numValue)
  normalizeWeights('success', numValue)
}
</script>

<style scoped>
.weight-sliders {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.weight-control {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.weight-label {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.weight-value {
  font-variant-numeric: tabular-nums;
  color: var(--text-secondary);
}

.weight-slider {
  width: 100%;
  height: 6px;
  border-radius: 3px;
  background: var(--bg-secondary);
  outline: none;
  -webkit-appearance: none;
}

.weight-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--primary-color);
  cursor: pointer;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

.weight-slider::-moz-range-thumb {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--primary-color);
  cursor: pointer;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
  border: none;
}

.weight-slider:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

.weight-error {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: var(--error-bg);
  border: 1px solid var(--error-border);
  border-radius: 6px;
  color: var(--error-text);
  font-size: 14px;
}
</style>
