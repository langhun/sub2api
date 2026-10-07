<template>
  <div class="routing-editor">
    <div class="form-section">
      <label class="form-label">
        <input
          v-model="localConfig.enabled"
          type="checkbox"
          class="form-checkbox"
        />
        <span>{{ t('keys.routing.enableSmartRouting') }}</span>
      </label>
      <p class="form-hint">{{ t('keys.routing.smartRoutingHint') }}</p>
    </div>

    <div v-if="localConfig.enabled" class="form-section">
      <label class="form-label">{{ t('keys.routing.strategy') }}</label>
      <select v-model="localConfig.strategy" class="form-select">
        <option value="auto">{{ t('keys.routing.strategyAuto') }}</option>
        <option value="price">{{ t('keys.routing.strategyPrice') }}</option>
        <option value="speed">{{ t('keys.routing.strategySpeed') }}</option>
        <option value="success">{{ t('keys.routing.strategySuccess') }}</option>
        <option value="custom">{{ t('keys.routing.strategyCustom') }}</option>
      </select>
    </div>

    <div v-if="localConfig.enabled && localConfig.strategy === 'custom'" class="form-section">
      <label class="form-label">{{ t('keys.routing.customWeights') }}</label>
      <WeightSliders
        v-model:price-weight="localConfig.price_weight"
        v-model:speed-weight="localConfig.speed_weight"
        v-model:success-weight="localConfig.success_weight"
      />
    </div>

    <div v-if="localConfig.enabled" class="form-section">
      <label class="form-label">{{ t('keys.routing.explorationRate') }}</label>
      <input
        v-model.number="localConfig.exploration_rate"
        type="number"
        step="0.001"
        min="0"
        :max="MAX_EXPLORATION_RATE"
        class="form-input"
      />
      <p class="form-hint">{{ t('keys.routing.explorationRateHint') }}</p>
    </div>

    <div v-if="localConfig.enabled" class="form-section">
      <label class="form-label">{{ t('keys.routing.candidateGroups') }}</label>
      <div class="candidates-container">
        <p v-if="!localConfig.group_ids.length" class="empty-hint">
          {{ t('keys.routing.noCandidatesSelected') }}
        </p>
        <div v-else class="selected-groups">
          <div
            v-for="(groupId, index) in localConfig.group_ids"
            :key="groupId"
            class="selected-group-item"
          >
            <span class="group-order">{{ index + 1 }}</span>
            <span class="group-name">{{ getGroupName(groupId) }}</span>
            <button
              @click="removeGroup(groupId)"
              class="btn-remove"
              :title="t('common.remove')"
            >
              <Icon name="x" size="sm" />
            </button>
          </div>
        </div>
      </div>
      <RoutingCandidateOptions
        :options="availableCandidates"
        :selected-values="localConfig.group_ids"
        :disabled="!localConfig.enabled"
        @select="addGroup"
      />
    </div>

    <div class="form-actions">
      <button
        @click="handleSave"
        :disabled="!isValid || saving"
        class="btn btn-primary"
      >
        <Icon v-if="saving" name="loader" size="sm" class="animate-spin" />
        <span>{{ t('common.save') }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/common/Icon.vue'
import WeightSliders from './WeightSliders.vue'
import RoutingCandidateOptions from './RoutingCandidateOptions.vue'
import {
  getSmartRouting,
  updateSmartRouting,
  VersionConflictError,
  MAX_EXPLORATION_RATE,
  type SmartRoutingProfile,
  type SmartRoutingUpdate
} from './api'

const props = defineProps<{
  keyId: number
  availableGroups: Array<{ value: number; label: string; platform?: string }>
}>()

const emit = defineEmits<{
  (event: 'saved'): void
  (event: 'error', error: string): void
}>()

const { t } = useI18n()

const localConfig = ref<SmartRoutingUpdate>({
  version: 0,
  enabled: false,
  mode: 'smart',
  strategy: 'auto',
  price_weight: 40,
  speed_weight: 30,
  success_weight: 30,
  sticky_ttl_seconds: 0,
  exploration_rate: 0.01,
  group_ids: []
})

const saving = ref(false)
const loading = ref(true)

const weightSum = computed(() =>
  localConfig.value.price_weight +
  localConfig.value.speed_weight +
  localConfig.value.success_weight
)

const isValid = computed(() => {
  if (!localConfig.value.enabled) return true

  // 自定义策略权重必须总和为100
  if (localConfig.value.strategy === 'custom' && Math.abs(weightSum.value - 100) > 0.1) {
    return false
  }

  // 必须选择至少一个候选组
  if (localConfig.value.group_ids.length === 0) {
    return false
  }

  // 探索率范围检查
  if (
    localConfig.value.exploration_rate < 0 ||
    localConfig.value.exploration_rate > MAX_EXPLORATION_RATE
  ) {
    return false
  }

  return true
})

const availableCandidates = computed(() => {
  return props.availableGroups.filter(g => !localConfig.value.group_ids.includes(g.value))
})

function getGroupName(groupId: number): string {
  const group = props.availableGroups.find(g => g.value === groupId)
  return group?.label || `Group #${groupId}`
}

function addGroup(groupId: number) {
  if (!localConfig.value.group_ids.includes(groupId)) {
    localConfig.value.group_ids.push(groupId)
  }
}

function removeGroup(groupId: number) {
  const index = localConfig.value.group_ids.indexOf(groupId)
  if (index > -1) {
    localConfig.value.group_ids.splice(index, 1)
  }
}

async function loadConfig() {
  try {
    loading.value = true
    const profile = await getSmartRouting(props.keyId)
    localConfig.value = {
      version: profile.version,
      enabled: profile.enabled,
      mode: profile.mode,
      strategy: profile.strategy,
      price_weight: profile.price_weight,
      speed_weight: profile.speed_weight,
      success_weight: profile.success_weight,
      sticky_ttl_seconds: profile.sticky_ttl_seconds || 0,
      exploration_rate: profile.exploration_rate,
      group_ids: profile.groups.map(g => g.group_id)
    }
  } catch (err: any) {
    emit('error', err.message || 'Failed to load configuration')
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!isValid.value) return

  try {
    saving.value = true
    await updateSmartRouting(props.keyId, localConfig.value)
    emit('saved')
    // 重新加载以获取新version
    await loadConfig()
  } catch (err: any) {
    if (err instanceof VersionConflictError) {
      if (confirm(t('keys.routing.versionConflictPrompt'))) {
        await loadConfig()
        emit('error', t('keys.routing.versionConflictReloaded'))
      }
    } else {
      emit('error', err.message || 'Failed to save configuration')
    }
  } finally {
    saving.value = false
  }
}

// 初始加载
loadConfig()
</script>

<style scoped>
.routing-editor {
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-width: 800px;
}

.form-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.form-checkbox {
  width: 18px;
  height: 18px;
  cursor: pointer;
}

.form-hint {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0;
}

.form-select,
.form-input {
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  background: var(--bg-primary);
  color: var(--text-primary);
}

.form-select:focus,
.form-input:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: -1px;
}

.candidates-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.empty-hint {
  padding: 16px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 14px;
  background: var(--bg-secondary);
  border-radius: 6px;
}

.selected-groups {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.selected-group-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
}

.group-order {
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--primary-color);
  color: white;
  border-radius: 50%;
  font-size: 12px;
  font-weight: 600;
}

.group-name {
  flex: 1;
  font-size: 14px;
  color: var(--text-primary);
}

.btn-remove {
  flex-shrink: 0;
  padding: 4px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.2s;
}

.btn-remove:hover {
  background: var(--error-bg);
  color: var(--error-text);
}

.form-actions {
  display: flex;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-color);
}

.btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary {
  background: var(--primary-color);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: var(--primary-hover);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.animate-spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
