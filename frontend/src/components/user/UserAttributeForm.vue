<template>
  <div v-if="attributes.length > 0" class="space-y-4">
    <div v-for="attr in attributes" :key="attr.id">
      <label class="input-label">
        {{ attr.name }}
        <span v-if="attr.required" class="text-red-500">*</span>
      </label>

      <!-- Text Input -->
      <input
        v-if="attr.type === 'text' || attr.type === 'email' || attr.type === 'url'"
        v-model="localValues[attr.id]"
        :type="attr.type === 'text' ? 'text' : attr.type"
        :required="attr.required"
        :placeholder="attr.placeholder"
        class="input"
        @input="emitChange"
      />

      <!-- Number Input -->
      <input
        v-else-if="attr.type === 'number'"
        v-model.number="localValues[attr.id]"
        type="number"
        :required="attr.required"
        :placeholder="attr.placeholder"
        :min="attr.validation?.min"
        :max="attr.validation?.max"
        class="input"
        @input="emitChange"
      />

      <!-- Date Input -->
      <input
        v-else-if="attr.type === 'date'"
        v-model="localValues[attr.id]"
        type="date"
        :required="attr.required"
        class="input"
        @input="emitChange"
      />

      <!-- Textarea -->
      <textarea
        v-else-if="attr.type === 'textarea'"
        v-model="localValues[attr.id]"
        :required="attr.required"
        :placeholder="attr.placeholder"
        rows="3"
        class="input"
        @input="emitChange"
      />

      <!-- Select -->
      <Select
        v-else-if="attr.type === 'select'"
        v-model="localValues[attr.id]"
        :options="attr.options || []"
        @change="emitChange"
      />

      <!-- Multi-Select (Checkboxes) -->
      <div v-else-if="attr.type === 'multi_select'" class="space-y-2">
        <label
          v-for="opt in attr.options"
          :key="opt.value"
          class="flex items-center gap-2"
        >
          <input
            type="checkbox"
            :value="opt.value"
            :checked="isOptionSelected(attr.id, opt.value)"
            @change="toggleMultiSelectOption(attr.id, opt.value)"
            class="h-4 w-4 rounded border-gray-300 text-primary-600"
          />
          <span class="text-sm text-gray-700 dark:text-gray-300">{{ opt.label }}</span>
        </label>
      </div>

      <!-- Description -->
      <p v-if="attr.description" class="input-hint">{{ attr.description }}</p>
    </div>
  </div>

  <!-- Loading State -->
  <div v-else-if="loading" class="flex justify-center py-4">
    <svg class="h-5 w-5 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
    </svg>
  </div>

  <div v-else-if="loadFailed" class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-900/20 dark:text-red-300">
    <p>{{ loadErrorMessage }}</p>
    <button type="button" class="mt-2 text-xs font-medium underline" @click="retryLoad">
      {{ retryLabel }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { UserAttributeDefinition, UserAttributeValuesMap } from '@/types'
import Select from '@/components/common/Select.vue'

interface Props {
  userId?: number
  modelValue: UserAttributeValuesMap
}

interface Emits {
  (e: 'update:modelValue', value: UserAttributeValuesMap): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const { t } = useI18n()

const loading = ref(false)
const loadFailed = ref(false)
const attributes = ref<UserAttributeDefinition[]>([])
const localValues = ref<UserAttributeValuesMap>({})
let latestLoadRequestId = 0

const loadErrorMessage = computed(() => t('admin.settings.attributes.failedToLoad'))
const retryLabel = computed(() => t('common.retry'))

const markLoadFailed = () => {
  loadFailed.value = true
}

const clearLoadFailed = () => {
  loadFailed.value = false
}

const loadAttributes = async () => {
  loading.value = true
  clearLoadFailed()
  try {
    attributes.value = await adminAPI.userAttributes.listEnabledDefinitions()
  } catch (error) {
    attributes.value = []
    markLoadFailed()
    console.error('Failed to load attributes:', error)
  } finally {
    loading.value = false
  }
}

const loadUserValues = async () => {
  if (!props.userId) return
  const requestId = ++latestLoadRequestId
  const targetUserId = props.userId

  try {
    const values = await adminAPI.userAttributes.getUserAttributeValues(targetUserId)
    if (requestId !== latestLoadRequestId || props.userId !== targetUserId) {
      return
    }
    const valuesMap: UserAttributeValuesMap = {}
    values.forEach(v => {
      valuesMap[v.attribute_id] = v.value
    })
    localValues.value = { ...valuesMap }
    emit('update:modelValue', localValues.value)
  } catch (error) {
    if (requestId === latestLoadRequestId) {
      markLoadFailed()
    }
    console.error('Failed to load user attribute values:', error)
  }
}

const retryLoad = async () => {
  await loadAttributes()
  if (props.userId) {
    await loadUserValues()
  }
}

const emitChange = () => {
  emit('update:modelValue', { ...localValues.value })
}

const isOptionSelected = (attrId: number, optionValue: string): boolean => {
  const value = localValues.value[attrId]
  if (!value) return false
  try {
    const arr = JSON.parse(value)
    return Array.isArray(arr) && arr.includes(optionValue)
  } catch {
    return false
  }
}

const toggleMultiSelectOption = (attrId: number, optionValue: string) => {
  let arr: string[] = []
  const value = localValues.value[attrId]
  if (value) {
    try {
      arr = JSON.parse(value)
      if (!Array.isArray(arr)) arr = []
    } catch {
      arr = []
    }
  }

  const index = arr.indexOf(optionValue)
  if (index > -1) {
    arr.splice(index, 1)
  } else {
    arr.push(optionValue)
  }

  localValues.value[attrId] = JSON.stringify(arr)
  emitChange()
}

watch(() => props.modelValue, (newVal) => {
  localValues.value = newVal ? { ...newVal } : {}
}, { immediate: true })

watch(() => props.userId, (newUserId) => {
  if (newUserId) {
    localValues.value = {}
    emitChange()
    loadUserValues()
  } else {
    // Reset for new user
    latestLoadRequestId += 1
    localValues.value = {}
    emitChange()
  }
}, { immediate: true })

onMounted(() => {
  loadAttributes()
})
</script>
