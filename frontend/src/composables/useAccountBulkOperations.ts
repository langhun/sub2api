import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'

export interface BulkOperationResult {
  success: number
  failed: number
  skipped?: number
  success_ids?: number[]
  failed_ids?: number[]
}

export interface BulkOperationOptions {
  confirmMessage: string
  successMessage: string | ((result: BulkOperationResult) => string)
  errorMessage?: string | ((result: BulkOperationResult) => string)
  skipConfirm?: boolean
}

export function useAccountBulkOperations() {
  const { t } = useI18n()
  const appStore = useAppStore()
  const operating = ref(false)

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
      const message =
        typeof options.errorMessage === 'function'
          ? options.errorMessage({ success: 0, failed: 0 })
          : options.errorMessage || String(error)
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
