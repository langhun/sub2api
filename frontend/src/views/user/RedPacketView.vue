<template>
  <AppLayout>
    <div class="mx-auto max-w-2xl space-y-6">
      <!-- Header Card -->
      <div class="card overflow-hidden">
        <div class="bg-gradient-to-br from-red-500 to-red-600 px-6 py-8 text-center">
          <div class="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-white/20 backdrop-blur-sm">
            <Icon name="gift" size="xl" class="text-white" />
          </div>
          <p class="text-2xl font-bold text-white">{{ t('redpacket.title', '红包中心') }}</p>
          <p class="mt-2 text-sm text-red-100">{{ t('redpacket.subtitle', '发红包、领红包，分享快乐') }}</p>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="grid grid-cols-2 gap-4">
        <button @click="showCreate = true; createError = ''; createdRp = null"
          class="card flex items-center justify-center gap-2 p-4 text-red-600 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20">
          <Icon name="plus" size="md" />
          <span class="text-sm font-medium">{{ t('redpacket.create', '发红包') }}</span>
        </button>
        <button @click="showClaim = true; claimError = ''; claimResult = null; claimCode = ''"
          class="card flex items-center justify-center gap-2 p-4 text-amber-600 transition-colors hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-900/20">
          <Icon name="gift" size="md" />
          <span class="text-sm font-medium">{{ t('redpacket.claim', '领红包') }}</span>
        </button>
      </div>

      <!-- Create Red Packet -->
      <transition name="fade">
        <div v-if="showCreate" class="card border-red-200 bg-red-50 dark:border-red-800/50 dark:bg-red-900/20">
          <div class="p-6">
            <div class="flex items-start gap-4">
              <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-red-100 dark:bg-red-900/30">
                <Icon name="sparkles" size="md" class="text-red-600 dark:text-red-400" />
              </div>
              <div class="flex-1">
                <h3 class="text-sm font-semibold text-red-800 dark:text-red-300">{{ t('redpacket.create', '发红包') }}</h3>
                <form @submit.prevent="handleCreate" class="mt-3 space-y-3">
                  <div>
                    <label class="input-label">{{ t('redpacket.totalAmount', '总金额') }}</label>
                    <input v-model.number="createForm.total_amount" type="number" step="0.01" min="0.01" required
                      :disabled="createLoading" class="input mt-1 w-full" />
                  </div>
                  <div>
                    <label class="input-label">{{ t('redpacket.count', '份数') }}</label>
                    <input v-model.number="createForm.count" type="number" min="1" max="100" required
                      :disabled="createLoading" class="input mt-1 w-full" />
                  </div>
                  <div>
                    <label class="input-label">{{ t('redpacket.type', '类型') }}</label>
                    <select v-model="createForm.redpacket_type" :disabled="createLoading" class="input mt-1 w-full">
                      <option value="equal">{{ t('redpacket.equal', '等分红包') }}</option>
                      <option value="random">{{ t('redpacket.random', '拼手气红包') }}</option>
                    </select>
                  </div>
                  <div>
                    <label class="input-label">{{ t('redpacket.memo', '附言') }}</label>
                    <input v-model="createForm.memo" type="text" maxlength="100"
                      :placeholder="t('redpacket.memoPlaceholder', '可选附言')"
                      :disabled="createLoading" class="input mt-1 w-full" />
                  </div>
                  <p v-if="createError" class="text-sm text-red-600 dark:text-red-400">{{ createError }}</p>
                  <div class="flex gap-2">
                    <button type="submit" :disabled="createLoading" class="btn btn-primary flex-1">
                      <svg v-if="createLoading" class="-ml-1 mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                      {{ t('redpacket.create', '发红包') }}
                    </button>
                    <button type="button" @click="showCreate = false" class="btn btn-secondary flex-1">{{ t('common.cancel', '取消') }}</button>
                  </div>
                </form>

                <div v-if="createdRp" class="mt-4 rounded-lg bg-emerald-50 p-4 dark:bg-emerald-900/20">
                  <div class="flex items-start gap-3">
                    <Icon name="checkCircle" size="md" class="mt-0.5 text-emerald-600 dark:text-emerald-400" />
                    <div>
                      <p class="text-sm font-medium text-emerald-700 dark:text-emerald-300">{{ t('redpacket.created', '红包已创建！') }}</p>
                      <p class="mt-1 font-mono text-lg select-all text-emerald-800 dark:text-emerald-200">{{ createdRp.code }}</p>
                      <p class="text-xs text-emerald-600/70 dark:text-emerald-400/50">{{ t('redpacket.shareCode', '将此口令分享给好友即可领取') }}</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </transition>

      <!-- Claim Red Packet -->
      <transition name="fade">
        <div v-if="showClaim" class="card border-amber-200 bg-amber-50 dark:border-amber-800/50 dark:bg-amber-900/20">
          <div class="p-6">
            <div class="flex items-start gap-4">
              <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-amber-100 dark:bg-amber-900/30">
                <Icon name="gift" size="md" class="text-amber-600 dark:text-amber-400" />
              </div>
              <div class="flex-1">
                <h3 class="text-sm font-semibold text-amber-800 dark:text-amber-300">{{ t('redpacket.claim', '领红包') }}</h3>
                <form @submit.prevent="handleClaim" class="mt-3 space-y-3">
                  <div>
                    <label class="input-label">{{ t('redpacket.code', '红包口令') }}</label>
                    <input v-model="claimCode" type="text" required
                      :placeholder="t('redpacket.codePlaceholder', '输入红包口令')"
                      :disabled="claimLoading" class="input mt-1 w-full" />
                  </div>
                  <div v-if="claimResult" class="rounded-lg bg-emerald-50 p-3 text-center dark:bg-emerald-900/20">
                    <p class="text-2xl font-bold text-emerald-600 dark:text-emerald-400">+${{ claimResult.amount.toFixed(4) }}</p>
                  </div>
                  <p v-if="claimError" class="text-sm text-red-600 dark:text-red-400">{{ claimError }}</p>
                  <div class="flex gap-2">
                    <button type="submit" :disabled="claimLoading || !claimCode" class="btn btn-primary flex-1">
                      <svg v-if="claimLoading" class="-ml-1 mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                      {{ t('redpacket.claim', '领取') }}
                    </button>
                    <button type="button" @click="showClaim = false; claimResult = null; claimError = ''" class="btn btn-secondary flex-1">{{ t('common.cancel', '取消') }}</button>
                  </div>
                </form>
              </div>
            </div>
          </div>
        </div>
      </transition>

      <!-- My Red Packets -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('redpacket.myPackets', '我的红包') }}</h2>
        </div>
        <div class="p-6">
          <div v-if="loadingPackets" class="flex items-center justify-center py-8">
            <svg class="h-6 w-6 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          </div>

          <div v-else-if="myPackets.length > 0" class="space-y-3">
            <div v-for="rp in myPackets" :key="rp.id" class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
              <div class="flex items-start justify-between">
                <div class="flex items-start gap-3">
                  <div :class="['flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg', rpStatusStyle(rp.status).bg]">
                    <Icon name="gift" size="sm" :class="rpStatusStyle(rp.status).icon" />
                  </div>
                  <div>
                    <div class="flex items-center gap-2">
                      <p class="text-sm font-medium text-gray-900 dark:text-white">
                        {{ rp.redpacket_type === 'equal' ? t('redpacket.equal', '等分') : t('redpacket.random', '拼手气') }}
                      </p>
                      <span class="rounded-full px-2 py-0.5 text-[10px] font-semibold" :class="rpStatusStyle(rp.status).badge">
                        {{ rp.status }}
                      </span>
                    </div>
                    <p class="mt-0.5 text-xs text-gray-400 dark:text-dark-500">
                      {{ rp.total_count }}{{ t('redpacket.copies', '份') }}
                      · {{ rp.remaining_count }}{{ t('redpacket.remaining', '剩余') }}
                    </p>
                    <p v-if="rp.memo" class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ rp.memo }}</p>
                  </div>
                </div>
                <div class="text-right">
                  <p class="text-sm font-semibold text-red-600 dark:text-red-400">${{ rp.total_amount.toFixed(4) }}</p>
                  <p v-if="rp.remaining_amount > 0" class="text-xs text-gray-400 dark:text-dark-500">
                    {{ t('redpacket.remainingAmount', '剩余') }}: ${{ rp.remaining_amount.toFixed(4) }}
                  </p>
                </div>
              </div>
            </div>
          </div>

          <div v-else class="empty-state py-8">
            <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-800">
              <Icon name="gift" size="xl" class="text-gray-400 dark:text-dark-500" />
            </div>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('redpacket.noPackets', '暂无红包记录') }}</p>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { createRedPacket, claimRedPacket, getMyRedPackets } from '@/api/transfer'
import type { RedPacketRecord, RedPacketClaimRecord } from '@/api/transfer'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const showCreate = ref(false)
const showClaim = ref(false)
const claimCode = ref('')
const claimResult = ref<RedPacketClaimRecord | null>(null)
const claimError = ref('')
const claimLoading = ref(false)
const createError = ref('')
const createLoading = ref(false)
const createdRp = ref<RedPacketRecord | null>(null)
const myPackets = ref<RedPacketRecord[]>([])
const loadingPackets = ref(false)

const createForm = reactive({
  total_amount: 0,
  count: 1,
  redpacket_type: 'equal' as 'equal' | 'random',
  memo: '',
})

function rpStatusStyle(status: string) {
  switch (status) {
    case 'active': return {
      bg: 'bg-red-100 dark:bg-red-900/30',
      icon: 'text-red-600 dark:text-red-400',
      badge: 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300',
    }
    case 'exhausted': return {
      bg: 'bg-gray-100 dark:bg-dark-800',
      icon: 'text-gray-500 dark:text-dark-400',
      badge: 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-400',
    }
    case 'expired': return {
      bg: 'bg-gray-100 dark:bg-dark-800',
      icon: 'text-gray-400 dark:text-dark-500',
      badge: 'bg-orange-100 text-orange-600 dark:bg-orange-900/50 dark:text-orange-300',
    }
    default: return {
      bg: 'bg-gray-100 dark:bg-dark-800',
      icon: 'text-gray-500 dark:text-dark-400',
      badge: 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-400',
    }
  }
}

async function loadMyPackets() {
  loadingPackets.value = true
  try {
    const res = await getMyRedPackets({ page: 1, page_size: 20 })
    myPackets.value = res.items || []
  } catch {} finally {
    loadingPackets.value = false
  }
}

async function handleCreate() {
  createError.value = ''
  createLoading.value = true
  try {
    createdRp.value = await createRedPacket({
      total_amount: createForm.total_amount,
      count: createForm.count,
      redpacket_type: createForm.redpacket_type,
      memo: createForm.memo || undefined,
    })
    appStore.showSuccess(t('redpacket.created', '红包创建成功'))
    await Promise.all([loadMyPackets(), authStore.refreshUser()])
  } catch (e: any) {
    createError.value = e?.response?.data?.error || t('redpacket.createFailed', '创建失败')
  } finally {
    createLoading.value = false
  }
}

async function handleClaim() {
  claimError.value = ''
  claimResult.value = null
  claimLoading.value = true
  try {
    claimResult.value = await claimRedPacket(claimCode.value)
    appStore.showSuccess(t('redpacket.claimSuccess', '领取成功！'))
    await Promise.all([loadMyPackets(), authStore.refreshUser()])
  } catch (e: any) {
    claimError.value = e?.response?.data?.error || t('redpacket.claimFailed', '领取失败')
  } finally {
    claimLoading.value = false
  }
}

onMounted(loadMyPackets)
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
