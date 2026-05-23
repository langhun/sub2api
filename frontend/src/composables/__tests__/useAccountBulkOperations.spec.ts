import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAccountBulkOperations } from '../useAccountBulkOperations'
import { useAppStore } from '@/stores/app'

vi.mock('@/api/admin/system', () => ({
  checkUpdates: vi.fn()
}))
vi.mock('@/api/auth', () => ({
  getPublicSettings: vi.fn()
}))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (key === 'admin.accounts.bulkActions.partialSuccess') {
          return `partial ${params?.success}/${params?.failed}`
        }
        if (key === 'common.error') {
          return 'common.error'
        }
        return key
      }
    })
  }
})

describe('useAccountBulkOperations', () => {
  let appStore: ReturnType<typeof useAppStore>
  let bulkOps: ReturnType<typeof useAccountBulkOperations>
  let confirmSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    setActivePinia(createPinia())
    appStore = useAppStore()
    bulkOps = useAccountBulkOperations()
    confirmSpy = vi.spyOn(window, 'confirm')
    vi.clearAllMocks()
  })

  describe('handleBulkOperation', () => {
    it('should execute operation successfully', async () => {
      confirmSpy.mockReturnValue(true)
      const operation = vi.fn().mockResolvedValue({
        success: 5,
        failed: 0
      })
      const showSuccessSpy = vi.spyOn(appStore, 'showSuccess')

      const result = await bulkOps.handleBulkOperation(
        operation,
        {
          confirmMessage: 'Are you sure?',
          successMessage: 'Operation completed'
        }
      )

      expect(confirmSpy).toHaveBeenCalledWith('Are you sure?')
      expect(operation).toHaveBeenCalled()
      expect(showSuccessSpy).toHaveBeenCalledWith('Operation completed')
      expect(result).toEqual({ success: 5, failed: 0 })
      expect(bulkOps.operating.value).toBe(false)
    })

    it('should skip confirmation when skipConfirm is true', async () => {
      const operation = vi.fn().mockResolvedValue({
        success: 3,
        failed: 0
      })

      await bulkOps.handleBulkOperation(
        operation,
        {
          confirmMessage: 'Are you sure?',
          successMessage: 'Done',
          skipConfirm: true
        }
      )

      expect(confirmSpy).not.toHaveBeenCalled()
      expect(operation).toHaveBeenCalled()
    })

    it('should return null when user cancels confirmation', async () => {
      confirmSpy.mockReturnValue(false)
      const operation = vi.fn()

      const result = await bulkOps.handleBulkOperation(
        operation,
        {
          confirmMessage: 'Are you sure?',
          successMessage: 'Done'
        }
      )

      expect(result).toBeNull()
      expect(operation).not.toHaveBeenCalled()
    })

    it('should handle partial success', async () => {
      confirmSpy.mockReturnValue(true)
      const operation = vi.fn().mockResolvedValue({
        success: 3,
        failed: 2
      })
      const showErrorSpy = vi.spyOn(appStore, 'showError')

      const result = await bulkOps.handleBulkOperation(
        operation,
        {
          confirmMessage: 'Are you sure?',
          successMessage: 'All done',
          errorMessage: 'Some failed'
        }
      )

      expect(showErrorSpy).toHaveBeenCalledWith('Some failed')
      expect(result).toEqual({ success: 3, failed: 2 })
    })

    it('should use function for success message', async () => {
      confirmSpy.mockReturnValue(true)
      const operation = vi.fn().mockResolvedValue({
        success: 5,
        failed: 0
      })
      const showSuccessSpy = vi.spyOn(appStore, 'showSuccess')

      await bulkOps.handleBulkOperation(
        operation,
        {
          confirmMessage: 'Are you sure?',
          successMessage: (result) => `Processed ${result.success} items`
        }
      )

      expect(showSuccessSpy).toHaveBeenCalledWith('Processed 5 items')
    })

    it('should use function for error message', async () => {
      confirmSpy.mockReturnValue(true)
      const operation = vi.fn().mockResolvedValue({
        success: 3,
        failed: 2
      })
      const showErrorSpy = vi.spyOn(appStore, 'showError')

      await bulkOps.handleBulkOperation(
        operation,
        {
          confirmMessage: 'Are you sure?',
          successMessage: 'Done',
          errorMessage: (result) => `Failed: ${result.failed}`
        }
      )

      expect(showErrorSpy).toHaveBeenCalledWith('Failed: 2')
    })

    it('should handle operation error', async () => {
      confirmSpy.mockReturnValue(true)
      const error = new Error('Network error')
      const operation = vi.fn().mockRejectedValue(error)
      const showErrorSpy = vi.spyOn(appStore, 'showError')

      const result = await bulkOps.handleBulkOperation(
        operation,
        {
          confirmMessage: 'Are you sure?',
          successMessage: 'Done',
          errorMessage: 'Operation failed'
        }
      )

      expect(showErrorSpy).toHaveBeenCalledWith('Network error')
      expect(result).toBeNull()
      expect(bulkOps.operating.value).toBe(false)
    })

    it('should extract plain-object api error message when operation fails', async () => {
      confirmSpy.mockReturnValue(true)
      const operation = vi.fn().mockRejectedValue({
        status: 400,
        code: 400,
        message: 'Cannot set privacy: missing access_token'
      })
      const showErrorSpy = vi.spyOn(appStore, 'showError')

      const result = await bulkOps.handleBulkOperation(
        operation,
        {
          confirmMessage: 'Are you sure?',
          successMessage: 'Done'
        }
      )

      expect(showErrorSpy).toHaveBeenCalledWith('Cannot set privacy: missing access_token')
      expect(result).toBeNull()
    })

    it('should call onSuccess callback', async () => {
      confirmSpy.mockReturnValue(true)
      const operation = vi.fn().mockResolvedValue({
        success: 5,
        failed: 0
      })
      const onSuccess = vi.fn()

      await bulkOps.handleBulkOperation(
        operation,
        {
          confirmMessage: 'Are you sure?',
          successMessage: 'Done'
        },
        { onSuccess }
      )

      expect(onSuccess).toHaveBeenCalledWith({ success: 5, failed: 0 })
    })

    it('should call onError callback', async () => {
      confirmSpy.mockReturnValue(true)
      const error = new Error('Failed')
      const operation = vi.fn().mockRejectedValue(error)
      const onError = vi.fn()

      await bulkOps.handleBulkOperation(
        operation,
        {
          confirmMessage: 'Are you sure?',
          successMessage: 'Done'
        },
        { onError }
      )

      expect(onError).toHaveBeenCalledWith(error)
    })

    it('should call onFinally callback', async () => {
      confirmSpy.mockReturnValue(true)
      const operation = vi.fn().mockResolvedValue({
        success: 5,
        failed: 0
      })
      const onFinally = vi.fn()

      await bulkOps.handleBulkOperation(
        operation,
        {
          confirmMessage: 'Are you sure?',
          successMessage: 'Done'
        },
        { onFinally }
      )

      expect(onFinally).toHaveBeenCalled()
    })

    it('should set operating flag during operation', async () => {
      confirmSpy.mockReturnValue(true)
      let operatingDuringExecution = false
      const operation = vi.fn().mockImplementation(async () => {
        operatingDuringExecution = bulkOps.operating.value
        return { success: 1, failed: 0 }
      })

      expect(bulkOps.operating.value).toBe(false)

      await bulkOps.handleBulkOperation(
        operation,
        {
          confirmMessage: 'Are you sure?',
          successMessage: 'Done'
        }
      )

      expect(operatingDuringExecution).toBe(true)
      expect(bulkOps.operating.value).toBe(false)
    })
  })
})
