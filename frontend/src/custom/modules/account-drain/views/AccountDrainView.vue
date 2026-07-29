<template>
  <AppLayout>
    <main class="mx-auto max-w-6xl space-y-6">
      <section class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-gray-100">账号定向消耗</h1>
          <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">计划只优先分配新会话，现有粘性会话保持原账号。</p>
        </div>
        <button class="btn-secondary" :disabled="loading" title="刷新计划和账号池" @click="load">刷新</button>
      </section>

      <section class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.8fr)]">
        <div class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
          <div class="border-b border-gray-200 px-5 py-4 dark:border-gray-700">
            <h2 class="font-medium text-gray-900 dark:text-gray-100">消耗计划</h2>
          </div>
          <div v-if="loading" class="px-5 py-8 text-sm text-gray-500">正在加载...</div>
          <div v-else-if="plans.length === 0" class="px-5 py-8 text-sm text-gray-500">暂无计划，账号池按正常规则调度。</div>
          <ul v-else class="divide-y divide-gray-100 dark:divide-gray-700">
            <li v-for="plan in plans" :key="plan.id" class="flex items-center gap-4 px-5 py-4">
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <strong class="truncate text-sm text-gray-900 dark:text-gray-100">{{ plan.name }}</strong>
                  <span :class="statusClass(plan.status)">{{ statusLabel(plan.status) }}</span>
                </div>
                <p class="mt-1 text-xs text-gray-500">账号 ID: {{ plan.account_ids.join(', ') }}</p>
                <p class="mt-1 text-xs text-gray-500">{{ expiryLabel(plan.expires_at) }}</p>
              </div>
              <button
                v-if="plan.status === 'active'"
                class="btn-secondary text-sm text-red-600 dark:text-red-400"
                :disabled="stoppingId === plan.id"
                @click="stop(plan)"
              >
                {{ stoppingId === plan.id ? '停止中...' : '停止' }}
              </button>
            </li>
          </ul>
        </div>

        <form class="space-y-4 rounded-lg border border-gray-200 bg-white p-5 dark:border-gray-700 dark:bg-gray-800" @submit.prevent="create">
          <div>
            <h2 class="font-medium text-gray-900 dark:text-gray-100">新建计划</h2>
            <p class="mt-1 text-xs text-gray-500">目标账号不可用、满并发或不支持模型时会自动回退到正常账号池。</p>
          </div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
            计划名称
            <input v-model.trim="form.name" class="input-field mt-1 w-full" maxlength="120" placeholder="例如：优先消耗新购 GPT Pro" required />
          </label>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
            截止时间（可选）
            <input v-model="form.expiresAt" class="input-field mt-1 w-full" type="datetime-local" />
          </label>
          <div>
            <div class="flex items-center justify-between gap-3">
              <span class="text-sm font-medium text-gray-700 dark:text-gray-300">目标 OpenAI 账号</span>
              <input v-model.trim="search" class="input-field w-36 text-xs" placeholder="筛选名称或 ID" />
            </div>
            <div class="mt-2 max-h-64 overflow-y-auto rounded border border-gray-200 dark:border-gray-700">
              <label v-for="account in filteredAccounts" :key="account.id" class="flex cursor-pointer items-center gap-3 border-b border-gray-100 px-3 py-2 text-sm last:border-0 dark:border-gray-700">
                <input v-model="form.accountIds" type="checkbox" :value="account.id" class="h-4 w-4" />
                <span class="min-w-0 flex-1 truncate">{{ account.name }}</span>
                <span class="text-xs text-gray-500">#{{ account.id }}</span>
              </label>
              <p v-if="filteredAccounts.length === 0" class="px-3 py-4 text-sm text-gray-500">没有可选的 OpenAI 账号。</p>
            </div>
          </div>
          <button class="btn-primary w-full" :disabled="creating || form.accountIds.length === 0">
            {{ creating ? '创建中...' : '启动定向消耗' }}
          </button>
        </form>
      </section>
    </main>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { list as listAccounts } from '@/api/admin/accounts'
import { useAppStore } from '@/stores/app'
import type { Account } from '@/types'
import {
  createAccountDrainPlan,
  listAccountDrainPlans,
  stopAccountDrainPlan,
  type AccountDrainPlan,
} from '../api'

const appStore = useAppStore()
const loading = ref(false)
const creating = ref(false)
const stoppingId = ref<number | null>(null)
const plans = ref<AccountDrainPlan[]>([])
const accounts = ref<Account[]>([])
const search = ref('')
const form = reactive({ name: '', accountIds: [] as number[], expiresAt: '' })

const filteredAccounts = computed(() => {
  const keyword = search.value.toLowerCase()
  return accounts.value.filter((account) => !keyword || account.name.toLowerCase().includes(keyword) || String(account.id).includes(keyword))
})

onMounted(() => { void load() })

async function load() {
  loading.value = true
  try {
    const [nextPlans, accountPage] = await Promise.all([
      listAccountDrainPlans(),
      listAccounts(1, 500, { platform: 'openai', status: 'active' }),
    ])
    plans.value = nextPlans
    accounts.value = accountPage.items.filter((account) => account.schedulable)
  } catch (error) {
    console.error('Failed to load account drain plans', error)
    appStore.showError('加载定向消耗计划失败')
  } finally {
    loading.value = false
  }
}

async function create() {
  if (!form.name || form.accountIds.length === 0) return
  creating.value = true
  try {
    await createAccountDrainPlan({
      name: form.name,
      account_ids: form.accountIds,
      expires_at: form.expiresAt ? new Date(form.expiresAt).toISOString() : null,
    })
    form.name = ''
    form.accountIds = []
    form.expiresAt = ''
    await load()
    appStore.showSuccess('定向消耗计划已启动')
  } catch (error) {
    console.error('Failed to create account drain plan', error)
    appStore.showError('启动定向消耗计划失败')
  } finally {
    creating.value = false
  }
}

async function stop(plan: AccountDrainPlan) {
  stoppingId.value = plan.id
  try {
    await stopAccountDrainPlan(plan.id)
    await load()
    appStore.showSuccess('定向消耗计划已停止')
  } catch (error) {
    console.error('Failed to stop account drain plan', error)
    appStore.showError('停止定向消耗计划失败')
  } finally {
    stoppingId.value = null
  }
}

function statusLabel(status: AccountDrainPlan['status']) {
  return status === 'active' ? '生效中' : status === 'expired' ? '已到期' : '已停止'
}

function statusClass(status: AccountDrainPlan['status']) {
  return status === 'active'
    ? 'rounded bg-green-100 px-2 py-0.5 text-xs text-green-700 dark:bg-green-900/30 dark:text-green-300'
    : 'rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300'
}

function expiryLabel(expiresAt: string | null) {
  return expiresAt ? `截止：${new Date(expiresAt).toLocaleString()}` : '截止：手动停止'
}
</script>
