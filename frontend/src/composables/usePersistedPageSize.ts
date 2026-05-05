import { getConfiguredTableDefaultPageSize, normalizeTablePageSize } from '@/utils/tablePreferences'

const STORAGE_KEY = 'table-page-size'
const STORAGE_CONFIG_KEY = 'table-page-size-config'

function getPageSizeConfigSignature(): string {
  return String(normalizeTablePageSize(getConfiguredTableDefaultPageSize()))
}

export function getPersistedPageSize(fallback = getConfiguredTableDefaultPageSize()): number {
  const configuredDefault = normalizeTablePageSize(getConfiguredTableDefaultPageSize() || fallback)
  if (typeof window !== 'undefined') {
    try {
      const stored = window.localStorage.getItem(STORAGE_KEY)
      const storedConfig = window.localStorage.getItem(STORAGE_CONFIG_KEY)
      if (stored !== null && storedConfig === getPageSizeConfigSignature()) {
        const parsed = Number(stored)
        if (Number.isFinite(parsed)) {
          return normalizeTablePageSize(parsed)
        }
      }
    } catch (error) {
      console.warn('Failed to read persisted page size:', error)
    }
  }
  return configuredDefault
}

export function setPersistedPageSize(size: number): void {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(STORAGE_KEY, String(size))
    window.localStorage.setItem(STORAGE_CONFIG_KEY, getPageSizeConfigSignature())
  } catch (error) {
    console.warn('Failed to persist page size:', error)
  }
}
