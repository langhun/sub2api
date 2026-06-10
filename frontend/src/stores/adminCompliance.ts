import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export interface AdminComplianceMetadata {
  [key: string]: string | undefined
}

export const useAdminComplianceStore = defineStore('adminCompliance', () => {
  const initialized = ref(false)
  const acknowledgementRequired = ref(false)
  const metadata = ref<AdminComplianceMetadata | null>(null)

  const documentVersion = computed(() => metadata.value?.document_version || '')
  const confirmationText = computed(() => metadata.value?.confirmation_text || '')
  const documentUrl = computed(() => metadata.value?.document_url || '')

  async function fetchStatus(): Promise<void> {
    initialized.value = true
    acknowledgementRequired.value = false
  }

  function requireAcknowledgement(nextMetadata?: Record<string, string>): void {
    initialized.value = true
    acknowledgementRequired.value = true
    metadata.value = nextMetadata ? { ...nextMetadata } : null
  }

  function $reset(): void {
    initialized.value = false
    acknowledgementRequired.value = false
    metadata.value = null
  }

  return {
    initialized,
    acknowledgementRequired,
    metadata,
    documentVersion,
    confirmationText,
    documentUrl,
    fetchStatus,
    requireAcknowledgement,
    $reset
  }
})
