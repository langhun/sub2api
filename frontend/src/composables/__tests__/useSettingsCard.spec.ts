import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSettingsCard } from '../useSettingsCard'
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
      t: (key: string) => key
    })
  }
})

describe('useSettingsCard', () => {
  let appStore: ReturnType<typeof useAppStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    appStore = useAppStore()
    vi.clearAllMocks()
  })

  describe('load', () => {
    it('should load settings successfully', async () => {
      const mockData = {
        setting1: 'value1',
        setting2: 42,
        setting3: true
      }
      const loadFn = vi.fn().mockResolvedValue(mockData)
      const saveFn = vi.fn()

      const card = useSettingsCard({ loadFn, saveFn })

      expect(card.loading.value).toBe(true)

      await card.load()

      expect(loadFn).toHaveBeenCalled()
      expect(card.form).toEqual(mockData)
      expect(card.loading.value).toBe(false)
    })

    it('should handle load error', async () => {
      const error = new Error('Load failed')
      const loadFn = vi.fn().mockRejectedValue(error)
      const saveFn = vi.fn()
      const showErrorSpy = vi.spyOn(appStore, 'showError')

      const card = useSettingsCard({
        loadFn,
        saveFn,
        errorMessage: 'Failed to load'
      })

      await card.load()

      expect(showErrorSpy).toHaveBeenCalledWith('Failed to load')
      expect(card.loading.value).toBe(false)
    })

    it('should use error message from exception', async () => {
      const error = { message: 'Network error' }
      const loadFn = vi.fn().mockRejectedValue(error)
      const saveFn = vi.fn()
      const showErrorSpy = vi.spyOn(appStore, 'showError')

      const card = useSettingsCard({ loadFn, saveFn })

      await card.load()

      expect(showErrorSpy).toHaveBeenCalledWith('Network error')
    })

    it('should remove stale keys before applying a narrower payload', async () => {
      const loadFn = vi
        .fn()
        .mockResolvedValueOnce({ setting1: 'value1', setting2: 42 })
        .mockResolvedValueOnce({ setting1: 'value2' })
      const saveFn = vi.fn()

      const card = useSettingsCard({ loadFn, saveFn })

      await card.load()
      expect(card.form).toEqual({ setting1: 'value1', setting2: 42 })

      await card.load()

      expect(card.form).toEqual({ setting1: 'value2' })
      expect('setting2' in card.form).toBe(false)
    })
  })

  describe('save', () => {
    it('should save settings successfully', async () => {
      const mockData = { setting1: 'value1' }
      const loadFn = vi.fn().mockResolvedValue(mockData)
      const saveFn = vi.fn().mockResolvedValue(undefined)
      const showSuccessSpy = vi.spyOn(appStore, 'showSuccess')

      const card = useSettingsCard({
        loadFn,
        saveFn,
        successMessage: 'Saved successfully'
      })

      await card.load()
      await card.save()

      expect(saveFn).toHaveBeenCalledWith(mockData)
      expect(showSuccessSpy).toHaveBeenCalledWith('Saved successfully')
      expect(card.saving.value).toBe(false)
    })

    it('should handle save error', async () => {
      const mockData = { setting1: 'value1' }
      const loadFn = vi.fn().mockResolvedValue(mockData)
      const error = new Error('Save failed')
      const saveFn = vi.fn().mockRejectedValue(error)
      const showErrorSpy = vi.spyOn(appStore, 'showError')

      const card = useSettingsCard({
        loadFn,
        saveFn,
        errorMessage: 'Failed to save'
      })

      await card.load()
      await card.save()

      expect(showErrorSpy).toHaveBeenCalledWith('Failed to save')
      expect(card.saving.value).toBe(false)
    })

    it('should use error message from exception', async () => {
      const mockData = { setting1: 'value1' }
      const loadFn = vi.fn().mockResolvedValue(mockData)
      const error = { message: 'Network error' }
      const saveFn = vi.fn().mockRejectedValue(error)
      const showErrorSpy = vi.spyOn(appStore, 'showError')

      const card = useSettingsCard({ loadFn, saveFn })

      await card.load()
      await card.save()

      expect(showErrorSpy).toHaveBeenCalledWith('Network error')
    })

    it('should set saving flag during save', async () => {
      const mockData = { setting1: 'value1' }
      const loadFn = vi.fn().mockResolvedValue(mockData)
      let savingDuringExecution = false
      const saveFn = vi.fn().mockImplementation(async () => {
        savingDuringExecution = card.saving.value
      })

      const card = useSettingsCard({ loadFn, saveFn })

      await card.load()
      expect(card.saving.value).toBe(false)

      await card.save()

      expect(savingDuringExecution).toBe(true)
      expect(card.saving.value).toBe(false)
    })
  })

  describe('form reactivity', () => {
    it('should allow modifying form data', async () => {
      const mockData = {
        setting1: 'value1',
        setting2: 42
      }
      const loadFn = vi.fn().mockResolvedValue(mockData)
      const saveFn = vi.fn()

      const card = useSettingsCard({ loadFn, saveFn })

      await card.load()

      card.form.setting1 = 'new value'
      card.form.setting2 = 100

      expect(card.form.setting1).toBe('new value')
      expect(card.form.setting2).toBe(100)

      await card.save()

      expect(saveFn).toHaveBeenCalledWith({
        setting1: 'new value',
        setting2: 100
      })
    })

    it('should preserve form data between saves', async () => {
      const mockData = { setting1: 'value1' }
      const loadFn = vi.fn().mockResolvedValue(mockData)
      const saveFn = vi.fn().mockResolvedValue(undefined)

      const card = useSettingsCard({ loadFn, saveFn })

      await card.load()
      card.form.setting1 = 'modified'

      await card.save()
      expect(card.form.setting1).toBe('modified')

      await card.save()
      expect(saveFn).toHaveBeenLastCalledWith({ setting1: 'modified' })
    })
  })
})
