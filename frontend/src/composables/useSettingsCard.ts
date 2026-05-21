/**
 * @fileoverview 设置卡片 Composable
 *
 * 提供统一的设置加载和保存逻辑，简化设置页面的状态管理。
 *
 * @example
 * ```typescript
 * const { loading, saving, form, load, save } = useSettingsCard({
 *   loadFn: () => adminAPI.settings.getGeneral(),
 *   saveFn: (data) => adminAPI.settings.updateGeneral(data),
 *   successMessage: t('admin.settings.saveSuccess')
 * })
 *
 * onMounted(() => load())
 * ```
 */

import { ref, reactive, toRefs } from 'vue'
import { useAppStore } from '@/stores/app'
import { useI18n } from 'vue-i18n'

/**
 * 设置卡片配置选项
 * @template T 表单数据类型
 */
export interface SettingsCardOptions<T> {
  /** 加载数据的函数 */
  loadFn: () => Promise<T>
  /** 保存数据的函数 */
  saveFn: (data: T) => Promise<void>
  /** 保存成功提示消息 */
  successMessage?: string
  /** 错误提示消息 */
  errorMessage?: string
}

/**
 * 设置卡片 Composable
 *
 * 提供统一的加载/保存逻辑和状态管理，配合 SettingsCard 组件使用。
 *
 * @template T 表单数据类型，必须是对象类型
 * @param options 配置选项
 * @returns 包含加载状态、保存状态、表单数据和操作方法的对象
 *
 * @example
 * ```typescript
 * interface GeneralSettings {
 *   site_name: string
 *   site_url: string
 * }
 *
 * const { loading, saving, form, load, save } = useSettingsCard<GeneralSettings>({
 *   loadFn: async () => {
 *     const response = await adminAPI.settings.getGeneral()
 *     return response.data
 *   },
 *   saveFn: async (data) => {
 *     await adminAPI.settings.updateGeneral(data)
 *   }
 * })
 * ```
 */
export function useSettingsCard<T extends Record<string, any>>(
  options: SettingsCardOptions<T>
) {
  const { t } = useI18n()
  const appStore = useAppStore()

  const loading = ref(true)
  const saving = ref(false)
  const form = reactive<T>({} as T)

  /**
   * 加载设置数据
   * 自动处理加载状态和错误提示
   */
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

  /**
   * 保存设置数据
   * 自动处理保存状态和成功/错误提示
   */
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
