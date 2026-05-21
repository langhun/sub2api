import { ref, reactive, toRefs } from 'vue'
import { useAppStore } from '@/stores/app'
import { useI18n } from 'vue-i18n'

export interface SettingsCardOptions<T> {
  loadFn: () => Promise<T>
  saveFn: (data: T) => Promise<void>
  successMessage?: string
  errorMessage?: string
}

export function useSettingsCard<T extends Record<string, any>>(
  options: SettingsCardOptions<T>
) {
  const { t } = useI18n()
  const appStore = useAppStore()

  const loading = ref(true)
  const saving = ref(false)
  const form = reactive<T>({} as T)

  async function load() {
    loading.value = true
    try {
      const data = await options.loadFn()
      Object.assign(form, data)
    } catch (error: any) {
      console.error('Failed to load settings:', error)
      appStore.showError(
        options.errorMessage || error?.message || t('admin.settings.loadFailed')
      )
    } finally {
      loading.value = false
    }
  }

  async function save() {
    saving.value = true
    try {
      await options.saveFn(form)
      appStore.showSuccess(
        options.successMessage || t('admin.settings.saveSuccess')
      )
    } catch (error: any) {
      console.error('Failed to save settings:', error)
      appStore.showError(
        options.errorMessage || error?.message || t('admin.settings.saveFailed')
      )
    } finally {
      saving.value = false
    }
  }

  return {
    loading,
    saving,
    form,
    load,
    save
  }
}
