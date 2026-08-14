<template>
  <template v-if="placement === 'header'">
    <div v-if="hasContacts" class="border-t border-gray-100 px-4 py-2.5 dark:border-dark-700">
      <button type="button" class="flex w-full items-center gap-2 text-left text-xs text-gray-500 hover:text-primary-600 dark:text-gray-400 dark:hover:text-primary-300" @click="visible = true">
        <Icon name="chat" size="sm" />
        <span class="font-medium">{{ t('common.contactSupport') }}</span>
      </button>
    </div>
    <div v-else-if="legacyContactInfo" class="border-t border-gray-100 px-4 py-2.5 dark:border-dark-700">
      <div class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400"><Icon name="chat" size="sm" /><span>{{ t('common.contactSupport') }}:</span><span class="font-medium text-gray-700 dark:text-gray-300">{{ legacyContactInfo }}</span></div>
    </div>
  </template>

  <template v-else-if="placement === 'profile'">
    <div v-if="hasContacts" class="card border-primary-200 bg-primary-50 p-6 dark:bg-primary-900/20">
      <button type="button" class="flex w-full items-center gap-4 text-left" @click="visible = true">
        <div class="rounded-xl bg-primary-100 p-3 text-primary-600"><Icon name="chat" size="lg" /></div>
        <div><h3 class="font-semibold text-primary-800 dark:text-primary-200">{{ t('common.contactSupport') }}</h3><p class="mt-1 text-sm text-primary-700 dark:text-primary-300">{{ t('supportContacts.openHint') }}</p></div>
      </button>
    </div>
    <div v-else-if="legacyContactInfo" class="card border-primary-200 bg-primary-50 p-6 dark:bg-primary-900/20">
      <div class="flex items-center gap-4"><div class="rounded-xl bg-primary-100 p-3 text-primary-600"><Icon name="chat" size="lg" /></div><div><h3 class="font-semibold text-primary-800 dark:text-primary-200">{{ t('common.contactSupport') }}</h3><p class="text-sm font-medium">{{ legacyContactInfo }}</p></div></div>
    </div>
  </template>

  <li v-else-if="hasContacts">
    {{ t('redeem.codeRule3') }}
    <button type="button" class="ml-1.5 inline-flex items-center rounded-md bg-primary-200/50 px-2 py-0.5 text-xs font-medium text-primary-800 hover:bg-primary-200 dark:bg-primary-800/40 dark:text-primary-200" @click="visible = true">{{ t('common.contactSupport') }}</button>
  </li>
  <li v-else-if="legacyContactInfo">{{ t('redeem.codeRule3') }} <span class="ml-1.5 inline-flex items-center rounded-md bg-primary-200/50 px-2 py-0.5 text-xs font-medium text-primary-800 dark:bg-primary-800/40 dark:text-primary-200">{{ legacyContactInfo }}</span></li>

  <BaseDialog :show="visible" :title="t('common.contactSupport')" width="wide" @close="visible = false">
    <div class="space-y-3">
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('supportContacts.dialogHint') }}</p>
      <article v-for="contact in contacts" :key="contact.id" class="grid gap-4 rounded-lg border border-gray-200 p-4 dark:border-dark-600 sm:grid-cols-[1fr_148px]">
        <div class="min-w-0 space-y-3">
          <div>
            <div class="flex flex-wrap items-center gap-2">
              <span class="rounded-md bg-primary-50 px-2 py-0.5 text-xs font-semibold text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">{{ channelLabel(contact) }}</span>
              <h4 v-if="contact.display_name" class="font-semibold text-gray-900 dark:text-white">{{ contact.display_name }}</h4>
            </div>
            <p class="mt-2 break-all text-sm text-gray-600 dark:text-gray-300">{{ contact.contact }}</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" @click="copy(contact.contact)">{{ t('supportContacts.copy') }}</button>
            <a v-if="contact.link" class="btn btn-secondary btn-sm" :href="contact.link" target="_blank" rel="noopener noreferrer">{{ t('supportContacts.open') }}</a>
          </div>
        </div>
        <a v-if="contact.qr_image_url" :href="contact.qr_image_url" target="_blank" rel="noopener noreferrer" class="flex min-h-32 items-center justify-center rounded-md bg-gray-50 p-2 dark:bg-dark-800"><img :src="contact.qr_image_url" :alt="`${contactLabel(contact)} QR`" class="max-h-36 max-w-full object-contain" /></a>
      </article>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { supportContactsAPI, type SupportContact } from './api'

defineProps<{ placement: 'header' | 'profile' | 'redeem'; legacyContactInfo: string }>()
const { t } = useI18n()
const appStore = useAppStore()
const contacts = ref<SupportContact[]>([])
const loaded = ref(false)
const visible = ref(false)
const hasContacts = computed(() => loaded.value && contacts.value.length > 0)

function channelLabel(contact: SupportContact) { return t(`supportContacts.channels.${contact.channel}`) }
function contactLabel(contact: SupportContact) { return contact.display_name || channelLabel(contact) }
async function load() { try { const response = await supportContactsAPI.listPublic(); contacts.value = response.data } catch { contacts.value = [] } finally { loaded.value = true } }
async function copy(value: string) { try { await navigator.clipboard.writeText(value); appStore.showSuccess(t('supportContacts.copied')) } catch { appStore.showError(t('common.copyFailed')) } }
onMounted(load)
</script>
