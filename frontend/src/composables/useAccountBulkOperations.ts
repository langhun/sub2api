/**
 * @fileoverview 账户批量操作 Composable
 *
 * 提供统一的批量操作处理逻辑，包括确认对话框、错误处理、部分成功处理和结果反馈。
 *
 * @example
 * ```typescript
 * const { operating, handleBulkOperation } = useAccountBulkOperations()
 *
 * async function handleBatchDelete() {
 *   await handleBulkOperation(
 *     () => adminAPI.accounts.batchDelete(selectedIds.value),
 *     {
 *       confirmMessage: t('admin.accounts.confirmDelete'),
 *       successMessage: (result) => t('admin.accounts.deleteSuccess', { count: result.success })
 *     },
 *     {
 *       onSuccess: () => {
 *         clearSelection()
 *         loadAccounts()
 *       }
 *     }
 *   )
 * }
 * ```
 */

import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

/**
 * 批量操作结果
 */
export interface BulkOperationResult {
  /** 成功数量 */
  success: number
  /** 失败数量 */
  failed: number
  /** 跳过数量（可选） */
  skipped?: number
  /** 成功的 ID 列表（可选） */
  success_ids?: number[]
  /** 失败的 ID 列表（可选） */
  failed_ids?: number[]
}

/**
 * 批量操作配置选项
 */
export interface BulkOperationOptions {
  /** 确认对话框消息 */
  confirmMessage: string
  /** 成功提示消息（可以是字符串或根据结果生成消息的函数） */
  successMessage: string | ((result: BulkOperationResult) => string)
  /** 错误提示消息（可以是字符串或根据结果生成消息的函数） */
  errorMessage?: string | ((result: BulkOperationResult) => string)
  /** 是否跳过确认对话框 */
  skipConfirm?: boolean
}

/**
 * 账户批量操作 Composable
 *
 * 提供统一的批量操作处理逻辑，支持：
 * - 自动确认对话框
 * - 部分成功处理（区分成功和失败的项）
 * - 统一的错误处理和提示
 * - 支持动态消息（函数形式）
 * - 回调钩子支持
 *
 * @returns 包含操作状态和处理方法的对象
 */
export function useAccountBulkOperations() {
  const { t } = useI18n()
  const appStore = useAppStore()
  const operating = ref(false)

  /**
   * 处理批量操作
   *
   * @template T 操作结果类型，必须继承自 BulkOperationResult
   * @param operation 要执行的批量操作函数
   * @param options 操作配置选项
   * @param callbacks 可选的回调函数
   * @param callbacks.onSuccess 操作成功时的回调
   * @param callbacks.onError 操作失败时的回调
   * @param callbacks.onFinally 操作完成时的回调（无论成功或失败）
   * @returns 操作结果，如果用户取消确认则返回 null
   *
   * @example
   * ```typescript
   * const result = await handleBulkOperation(
   *   () => adminAPI.accounts.batchEnable(selectedIds.value),
   *   {
   *     confirmMessage: '确认启用选中的账户？',
   *     successMessage: (result) => `成功启用 ${result.success} 个账户`,
   *     errorMessage: (result) => `启用失败：成功 ${result.success}，失败 ${result.failed}`
   *   },
   *   {
   *     onSuccess: (result) => {
   *       console.log('Enabled accounts:', result.success_ids)
   *     }
   *   }
   * )
   * ```
   */
  async function handleBulkOperation<T extends BulkOperationResult>(
    operation: () => Promise<T>,
    options: BulkOperationOptions,
    callbacks?: {
      onSuccess?: (result: T) => void
      onError?: (error: any) => void
      onFinally?: () => void
    }
  ): Promise<T | null> {
    if (!options.skipConfirm && !confirm(options.confirmMessage)) {
      return null
    }

    operating.value = true
    try {
      const result = await operation()

      // Handle partial success
      if (result.failed > 0) {
        const message =
          typeof options.errorMessage === 'function'
            ? options.errorMessage(result)
            : options.errorMessage ||
              t('admin.accounts.bulkActions.partialSuccess', {
                success: result.success,
                failed: result.failed
              })
        appStore.showError(message)
      } else {
        const message =
          typeof options.successMessage === 'function'
            ? options.successMessage(result)
            : options.successMessage
        appStore.showSuccess(message)
      }

      callbacks?.onSuccess?.(result)
      return result
    } catch (error: any) {
      console.error('Bulk operation failed:', error)
      const fallback =
        typeof options.errorMessage === 'string' && options.errorMessage.trim().length > 0
          ? options.errorMessage
          : t('common.error')
      const message = extractApiErrorMessage(error, fallback)
      appStore.showError(message)
      callbacks?.onError?.(error)
      return null
    } finally {
      operating.value = false
      callbacks?.onFinally?.()
    }
  }

  return {
    operating,
    handleBulkOperation
  }
}
