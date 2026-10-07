<template>
  <section class="card" data-testid="support-contacts-settings">
    <div class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <div>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.supportContacts.title') }}</h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.supportContacts.description') }}</p>
      </div>
      <div class="flex gap-2">
        <button type="button" class="btn btn-secondary btn-sm" :disabled="loading || saving" @click="load">{{ t('common.refresh') }}</button>
        <button type="button" class="btn btn-primary btn-sm" :disabled="saving" @click="saveAll">{{ t('common.save') }}</button>
      </div>
    </div>

    <div class="space-y-4 p-6">
      <div v-if="loading" class="py-8 text-center text-sm text-gray-500">{{ t('common.loading') }}</div>
      <template v-else>
        <p v-if="items.length === 0" class="rounded-xl border border-dashed border-gray-300 p-4 text-sm text-gray-500 dark:border-dark-600">
          {{ t('admin.supportContacts.empty') }}
        </p>

        <article v-for="(item, index) in items" :key="item.id || index" class="space-y-4 rounded-xl border border-gray-200 p-4 dark:border-dark-600">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="flex items-center gap-3">
              <span class="rounded-md bg-primary-50 px-2 py-1 text-xs font-semibold uppercase text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">{{ channelLabel(item.channel) }}</span>
              <span class="text-sm text-gray-500">{{ item.id ? t('admin.supportContacts.saved') : t('admin.supportContacts.newContact') }}</span>
            </div>
            <div class="flex gap-2">
              <button type="button" class="btn btn-secondary btn-sm" :disabled="index === 0 || saving" @click="move(index, -1)">{{ t('admin.supportContacts.moveUp') }}</button>
              <button type="button" class="btn btn-secondary btn-sm" :disabled="index === items.length - 1 || saving" @click="move(index, 1)">{{ t('admin.supportContacts.moveDown') }}</button>
              <button type="button" class="btn btn-secondary btn-sm text-red-600" :disabled="saving" @click="remove(item, index)">{{ t('common.delete') }}</button>
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.supportContacts.channel') }}
              <select v-model="item.channel" class="input mt-1">
                <option v-for="channel in channels" :key="channel" :value="channel">{{ channelLabel(channel) }}</option>
              </select>
            </label>
            <label class="flex items-end justify-between gap-3 rounded-xl border border-gray-200 px-3 py-2 text-sm text-gray-700 dark:border-dark-600 dark:text-gray-200">
              <span>{{ t('admin.supportContacts.enabled') }}</span>
              <input v-model="item.enabled" type="checkbox" class="h-4 w-4 accent-primary-600" />
            </label>
            <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.supportContacts.name') }}
              <input v-model.trim="item.display_name" class="input mt-1" :placeholder="channelLabel(item.channel)" />
            </label>
            <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.supportContacts.contact') }}
              <input v-model.trim="item.contact" class="input mt-1" :placeholder="t('admin.supportContacts.contactPlaceholder')" />
            </label>
          </div>

          <label class="block text-sm text-gray-600 dark:text-gray-300">{{ t('admin.supportContacts.link') }}
            <input v-model.trim="item.link" class="input mt-1" :placeholder="t('admin.supportContacts.linkPlaceholder')" />
            <span class="mt-1 block text-xs text-gray-500">{{ t('admin.supportContacts.linkHint') }}</span>
          </label>

          <div class="grid gap-4 border-t border-gray-100 pt-4 dark:border-dark-700 md:grid-cols-[minmax(0,1fr)_180px]">
            <div class="space-y-3">
              <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.supportContacts.qrCode') }}</p>
              <div class="flex flex-wrap gap-2">
                <button type="button" class="btn btn-secondary btn-sm" :class="item.qr_source === 'none' ? 'ring-2 ring-primary-500/40' : ''" @click="setQRSource(item, 'none')">{{ t('admin.supportContacts.noQr') }}</button>
                <button type="button" class="btn btn-secondary btn-sm" :class="item.qr_source === 'url' ? 'ring-2 ring-primary-500/40' : ''" @click="setQRSource(item, 'url')">{{ t('admin.supportContacts.imageUrl') }}</button>
                <button type="button" class="btn btn-secondary btn-sm" :class="item.qr_source === 'upload' ? 'ring-2 ring-primary-500/40' : ''" :disabled="!item.id" @click="setQRSource(item, 'upload')">{{ t('admin.supportContacts.uploadImage') }}</button>
              </div>
              <label v-if="item.qr_source === 'url'" class="block text-sm text-gray-600 dark:text-gray-300">{{ t('admin.supportContacts.qrImageUrl') }}
                <input v-model.trim="item.qr_external_url" class="input mt-1" placeholder="https://example.com/qr.png" />
              </label>
              <label v-if="item.qr_source === 'upload'" class="btn btn-secondary btn-sm inline-flex cursor-pointer" :class="{ 'pointer-events-none opacity-60': !item.id || uploadingId === item.id }">
                <input type="file" accept="image/png,image/jpeg,image/webp" class="hidden" @change="upload(item, $event)" />
                {{ uploadingId === item.id ? t('common.uploading') : t('admin.supportContacts.selectImage') }}
              </label>
              <p class="text-xs text-gray-500">{{ t('admin.supportContacts.qrHint') }}</p>
            </div>
            <div class="flex min-h-36 items-center justify-center rounded-xl border border-dashed border-gray-300 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-800">
              <img v-if="previewURL(item)" :src="previewURL(item)" :alt="t('admin.supportContacts.qrCode')" class="max-h-40 max-w-full object-contain" />
              <span v-else class="text-xs text-gray-400">{{ t('admin.supportContacts.noQr') }}</span>
            </div>
          </div>
        </article>

        <button v-if="items.length < 12" type="button" class="btn btn-secondary" :disabled="saving" @click="add">{{ t('admin.supportContacts.add') }}</button>
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { supportContactsAPI, type SupportContact, type SupportContactChannel, type SupportContactInput } from './api'

type QRSource = 'none' | 'url' | 'upload'
type Draft = SupportContact & { qr_source: QRSource; qr_external_url: string }

defineProps<{ modelValue?: Record<string, unknown> }>()
const { t } = useI18n()
const channels: SupportContactChannel[] = ['wechat', 'qq', 'telegram', 'email', 'link']
const items = ref<Draft[]>([])
const loading = ref(false)
const saving = ref(false)
const uploadingId = ref('')

function channelLabel(channel: SupportContactChannel) { return t(`supportContacts.channels.${channel}`) }
function message(error: unknown, fallback: string) { return error instanceof Error ? error.message : (error as { message?: string })?.message || fallback }
function notify(kind: 'success' | 'error', text: string) {
  try {
    const store = useAppStore()
    if (kind === 'success') store.showSuccess(text)
    else store.showError(text)
  } catch {
    // SettingsView's isolated render tests intentionally mount without Pinia.
  }
}
function source(item: SupportContact): QRSource { return item.has_qr_image ? 'upload' : item.qr_image_url ? 'url' : 'none' }
function draft(item: SupportContact): Draft { const qr_source = source(item); return { ...item, qr_source, qr_external_url: qr_source === 'url' ? item.qr_image_url : '' } }
function input(item: Draft): SupportContactInput { return { channel: item.channel, display_name: item.display_name, contact: item.contact, link: item.link, qr_image_url: item.qr_source === 'url' ? item.qr_external_url : '', enabled: item.enabled, sort_order: item.sort_order } }
function previewURL(item: Draft) { return item.qr_source === 'url' ? item.qr_external_url : item.qr_source === 'upload' ? item.qr_image_url : '' }

async function load() {
  loading.value = true
  try { const response = await supportContactsAPI.listAdmin(); items.value = response.data.map(draft) }
  catch (error) { notify('error', message(error, t('admin.supportContacts.loadFailed'))) }
  finally { loading.value = false }
}

function add() {
  items.value.push({ id: '', channel: 'wechat', display_name: '', contact: '', link: '', qr_image_url: '', has_qr_image: false, enabled: true, sort_order: items.value.length * 10, qr_source: 'none', qr_external_url: '' })
}

function move(index: number, direction: number) {
  const target = index + direction
  if (target < 0 || target >= items.value.length) return
  const [item] = items.value.splice(index, 1)
  items.value.splice(target, 0, item)
  items.value.forEach((draft, order) => { draft.sort_order = order * 10 })
}

function setQRSource(item: Draft, next: QRSource) {
  item.qr_source = next
  if (next !== 'url') item.qr_external_url = ''
}

async function saveAll() {
  saving.value = true
  try {
    for (let index = 0; index < items.value.length; index++) {
      const item = items.value[index]
      item.sort_order = index * 10
      if (item.id && item.qr_source === 'none' && item.has_qr_image) {
        await supportContactsAPI.removeQR(item.id)
        item.has_qr_image = false
        item.qr_image_url = ''
      }
      const response = item.id ? await supportContactsAPI.update(item.id, input(item)) : await supportContactsAPI.create(input(item))
      items.value[index] = draft(response.data)
    }
    notify('success', t('admin.supportContacts.savedSuccess'))
  } catch (error) { notify('error', message(error, t('admin.supportContacts.saveFailed'))) }
  finally { saving.value = false }
}

async function remove(item: Draft, index: number) {
  try {
    if (item.id) await supportContactsAPI.remove(item.id)
    items.value.splice(index, 1)
    items.value.forEach((draft, order) => { draft.sort_order = order * 10 })
    notify('success', t('admin.supportContacts.deleted'))
  } catch (error) { notify('error', message(error, t('admin.supportContacts.deleteFailed'))) }
}

async function upload(item: Draft, event: Event) {
  const inputElement = event.target as HTMLInputElement
  const file = inputElement.files?.[0]
  inputElement.value = ''
  if (!file || !item.id) return
  if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type) || file.size > 2 * 1024 * 1024) { notify('error', t('admin.supportContacts.invalidImage')); return }
  uploadingId.value = item.id
  try {
    const response = await supportContactsAPI.uploadQR(item.id, file)
    item.qr_source = 'upload'; item.qr_external_url = ''; item.qr_image_url = response.data.qr_image_url; item.has_qr_image = true
    notify('success', t('admin.supportContacts.uploaded'))
  } catch (error) { notify('error', message(error, t('admin.supportContacts.uploadFailed'))) }
  finally { uploadingId.value = '' }
}

onMounted(load)
</script>
