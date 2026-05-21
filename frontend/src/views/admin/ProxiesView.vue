<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <ProxiesToolbar
          :active-tab="activeTab"
          :search-query="searchQuery"
          :filters="filters"
          :protocol-options="protocolOptions"
          :status-options="statusOptions"
          :runtime-status-options="runtimeStatusOptions"
          :loading="loadingState.loading"
          :loading-subscriptions="loadingState.loadingSubscriptions"
          :batch-testing="testingState.batchTesting"
          :batch-quality-checking="testingState.batchQualityChecking"
          :selected-count="selectedCount"
          :show-column-dropdown="dropdownState.showColumnDropdown"
          :show-proxy-tools-dropdown="dropdownState.showProxyToolsDropdown"
          :show-proxy-batch-dropdown="dropdownState.showProxyBatchDropdown"
          :toggleable-columns="toggleableColumns"
          :is-column-visible="isColumnVisible"
          @set-tab="handleToolbarSetTab"
          @update:search-query="updateToolbarSearch"
          @update:filters="updateToolbarFilters"
          @reload-proxies="loadProxies"
          @reload-subscriptions="loadProxySubscriptions"
          @create-subscription="openCreateSubscriptionDialog"
          @toggle-column-dropdown="dropdownState.showColumnDropdown = !dropdownState.showColumnDropdown"
          @toggle-tools-dropdown="dropdownState.showProxyToolsDropdown = !dropdownState.showProxyToolsDropdown; dropdownState.showProxyBatchDropdown = false"
          @toggle-batch-dropdown="dropdownState.showProxyBatchDropdown = !dropdownState.showProxyBatchDropdown; dropdownState.showProxyToolsDropdown = false"
          @toggle-column="toggleColumn"
          @open-import="modalState.showImportData = true; dropdownState.showProxyToolsDropdown = false"
          @open-export="modalState.showExportDataDialog = true; dropdownState.showProxyToolsDropdown = false"
          @open-pool="openPoolDialog(); dropdownState.showProxyToolsDropdown = false"
          @batch-test="handleBatchTest(); dropdownState.showProxyBatchDropdown = false"
          @batch-quality-check="handleBatchQualityCheck(); dropdownState.showProxyBatchDropdown = false"
          @batch-enable-pool="handleBatchPoolMembership(true); dropdownState.showProxyBatchDropdown = false"
          @batch-disable-pool="handleBatchPoolMembership(false); dropdownState.showProxyBatchDropdown = false"
          @batch-clear-cooldown="handleClearCooldown(Array.from(selectedProxyIds)); dropdownState.showProxyBatchDropdown = false"
          @batch-assign="modalState.showAssignAccounts = true; dropdownState.showProxyBatchDropdown = false"
          @batch-unassign="openBatchUnassign(); dropdownState.showProxyBatchDropdown = false"
          @batch-delete="openBatchDelete(); dropdownState.showProxyBatchDropdown = false"
          @create-proxy="modalState.showCreateModal = true"
        />
      </template>

      <template #table>
        <ProxySubscriptionsPanel
          v-if="activeTab === 'subscriptions'"
          :loading="loadingState.loadingSubscriptions"
          :items="dataState.proxySubscriptions"
          @refresh="handleRefreshSubscription"
          @edit="handleEditSubscription"
          @view-nodes="handleViewSubscriptionNodes"
          @delete="handleDeleteSubscription"
        />
        <div ref="proxyTableRef" class="flex min-h-0 flex-1 flex-col overflow-hidden">
        <template v-if="activeTab === 'proxies'">
        <!-- Bulk Actions Bar -->
        <ProxyBulkActionsBar
          v-if="selectedCount > 0"
          :selected-count="selectedCount"
          :batch-testing="testingState.batchTesting"
          :batch-quality-checking="testingState.batchQualityChecking"
          @test="handleBatchTest"
          @quality-check="handleBatchQualityCheck"
          @enable-pool="handleBatchPoolMembership(true)"
          @disable-pool="handleBatchPoolMembership(false)"
          @clear-cooldown="handleClearCooldown(Array.from(selectedProxyIds))"
          @assign="modalState.showAssignAccounts = true"
          @unassign="openBatchUnassign"
          @delete="openBatchDelete"
          @clear="clearSelectedProxies"
        />
        <DataTable
          :columns="columns"
          :data="dataState.proxies"
          :loading="loadingState.loading"
          :server-side-sort="true"
          default-sort-key="id"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #header-select>
            <input
              type="checkbox"
              class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="allVisibleSelected"
              @click.stop
              @change="toggleSelectAllVisible($event)"
            />
          </template>

          <template #cell-select="{ row }">
            <input
              type="checkbox"
              class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="selectedProxyIds.has(row.id)"
              @click.stop
              @change="toggleSelectRow(row.id, $event)"
            />
          </template>

          <template #cell-name="{ value, row }">
            <div class="flex flex-col gap-1">
                <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
              <div v-if="row.managed_by_subscription" class="flex flex-wrap items-center gap-1">
                <span class="badge badge-warning">{{ t('admin.proxies.subscriptions.managedBadge') }}</span>
                <span v-if="row.subscription_source_name" class="text-xs text-gray-500 dark:text-gray-400">
                  {{ row.subscription_source_name }}
                </span>
                <span v-if="row.subscription_node_type" class="text-xs text-gray-500 dark:text-gray-400">
                  [{{ row.subscription_node_type }}]
                </span>
              </div>
            </div>
          </template>

          <template #cell-protocol="{ value }">
            <span
              v-if="value"
              :class="['badge', value.startsWith('socks5') ? 'badge-primary' : 'badge-gray']"
            >
              {{ value.toUpperCase() }}
            </span>
            <span v-else class="text-sm text-gray-400">-</span>
          </template>

          <template #cell-address="{ row }">
            <div class="flex items-center gap-1.5">
              <code class="code text-xs">{{ row.host }}:{{ row.port }}</code>
              <div class="relative">
                <button
                  type="button"
                  class="rounded p-0.5 text-gray-400 hover:text-primary-600 dark:hover:text-primary-400"
                  :title="t('admin.proxies.copyProxyUrl')"
                  @click.stop="copyProxyUrl(row)"
                  @contextmenu.prevent="toggleCopyMenu(row.id)"
                >
                  <Icon name="copy" size="sm" />
                </button>
                <!-- Context menu for alternate copy formats -->
                <div
                  v-if="dropdownState.copyMenuProxyId === row.id"
                  class="absolute left-0 top-full z-50 mt-1 w-auto min-w-[180px] rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-500 dark:bg-dark-700"
                >
                  <button
                    v-for="fmt in getCopyFormats(row)"
                    :key="fmt.label"
                    class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs hover:bg-gray-100 dark:hover:bg-dark-600"
                    @click.stop="copyFormat(fmt.value)"
                  >
                    <span class="truncate font-mono text-gray-600 dark:text-gray-300">{{ fmt.label }}</span>
                  </button>
                </div>
              </div>
            </div>
          </template>

          <template #cell-auth="{ row }">
            <div v-if="row.username || row.password" class="flex items-center gap-1.5">
              <div class="flex flex-col text-xs">
                <span v-if="row.username" class="text-gray-700 dark:text-gray-200">{{ row.username }}</span>
                <span v-if="row.password" class="font-mono text-gray-500 dark:text-gray-400">
                  {{ passwordState.visiblePasswordIds.has(row.id) ? row.password : '••••••' }}
                </span>
              </div>
              <button
                v-if="row.password"
                type="button"
                class="ml-1 rounded p-0.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                @click.stop="passwordState.visiblePasswordIds.has(row.id) ? passwordState.visiblePasswordIds.delete(row.id) : passwordState.visiblePasswordIds.add(row.id)"
              >
                <Icon :name="passwordState.visiblePasswordIds.has(row.id) ? 'eyeOff' : 'eye'" size="sm" />
              </button>
            </div>
            <span v-else class="text-sm text-gray-400">-</span>
          </template>

          <template #cell-location="{ row }">
            <div class="flex items-center gap-2">
              <span
                v-if="countryFlagEmoji(row.country_code)"
                class="inline-flex h-4 w-6 items-center justify-center text-sm leading-none"
                role="img"
                :aria-label="row.country || normalizedCountryCode(row.country_code)"
              >
                {{ countryFlagEmoji(row.country_code) }}
              </span>
              <span
                v-else-if="normalizedCountryCode(row.country_code)"
                class="inline-flex min-w-6 items-center justify-center rounded-sm bg-gray-100 px-1 py-0.5 text-[10px] font-medium uppercase leading-none text-gray-500 dark:bg-dark-600 dark:text-gray-300"
              >
                {{ normalizedCountryCode(row.country_code) }}
              </span>
              <span v-if="formatLocation(row)" class="text-sm text-gray-700 dark:text-gray-200">
                {{ formatLocation(row) }}
              </span>
              <span v-else class="text-sm text-gray-400">-</span>
            </div>
          </template>

          <template #cell-account_count="{ row, value }">
            <button
              v-if="(value || 0) > 0"
              type="button"
              class="inline-flex items-center rounded bg-gray-100 px-2 py-0.5 text-xs font-medium text-primary-700 hover:bg-gray-200 dark:bg-dark-600 dark:text-primary-300 dark:hover:bg-dark-500"
              @click="openAccountsModal(row)"
            >
              {{ t('admin.groups.accountsCount', { count: value || 0 }) }}
            </button>
            <span
              v-else
              class="inline-flex items-center rounded bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-800 dark:bg-dark-600 dark:text-gray-300"
            >
              {{ t('admin.groups.accountsCount', { count: 0 }) }}
            </span>
          </template>

          <template #cell-pool="{ row }">
            <div class="flex flex-col items-start gap-1">
              <span :class="['badge', row.auto_failover_pool_enabled ? 'badge-primary' : 'badge-gray']">
                {{ row.auto_failover_pool_enabled ? t('admin.proxies.poolEnabled') : t('admin.proxies.poolDisabled') }}
              </span>
              <span
                v-if="typeof row.failover_switch_count === 'number' && row.failover_switch_count > 0"
                class="text-xs text-gray-500 dark:text-gray-400"
              >
                {{ t('admin.proxies.failoverSwitchCount', { count: row.failover_switch_count }) }}
              </span>
            </div>
          </template>

          <template #cell-latency="{ row }">
            <div class="flex flex-col gap-1">
              <span
                v-if="row.latency_status === 'failed'"
                class="badge badge-danger"
                :title="row.latency_message || undefined"
              >
                {{ t('admin.proxies.latencyFailed') }}
              </span>
              <span
                v-else-if="typeof row.latency_ms === 'number'"
                :class="['badge', row.latency_ms < 200 ? 'badge-success' : 'badge-warning']"
              >
                {{ row.latency_ms }}ms
              </span>
              <span v-else class="text-sm text-gray-400">-</span>
              <div
                v-if="typeof row.quality_checked === 'number'"
                class="flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400"
                :title="row.quality_summary || undefined"
              >
                <span>{{ t('admin.proxies.qualityInline', { grade: row.quality_grade || '-', score: row.quality_score ?? '-' }) }}</span>
                <span class="badge" :class="qualityOverallClass(row.quality_status)">
                  {{ qualityOverallLabel(row.quality_status) }}
                </span>
              </div>
            </div>
          </template>

          <template #cell-status="{ row, value }">
            <div class="flex flex-col items-start gap-1">
              <div class="flex flex-wrap items-center gap-1">
                <span :class="['badge', value === 'active' ? 'badge-success' : 'badge-danger']">
                  {{ t('admin.accounts.status.' + value) }}
                </span>
                <span v-if="row.health_status" :class="['badge', healthStatusClass(row.health_status)]">
                  {{ healthStatusLabel(row.health_status) }}
                </span>
              </div>
              <div v-if="row.health_status === 'cooldown' && row.cooldown_until_unix" class="text-xs text-amber-600 dark:text-amber-400">
                {{ t('admin.proxies.cooldownUntil', { time: formatCooldownCountdown(row.cooldown_until_unix) }) }}
              </div>
              <div
                v-if="row.last_fail_reason"
                class="text-xs text-gray-500 dark:text-gray-400"
                :title="`${row.last_fail_reason}\n${formatRuntimeTime(row.last_fail_at_unix)}`"
              >
                {{ row.last_fail_reason }}
              </div>
              <div
                v-if="typeof row.failover_switch_count === 'number' && row.failover_switch_count > 0"
                class="text-xs text-gray-500 dark:text-gray-400"
              >
                {{ t('admin.proxies.failoverSwitchCount', { count: row.failover_switch_count }) }}
              </div>
            </div>
          </template>

          <template #cell-actions="{ row }">
            <div class="relative flex items-center justify-end gap-1" @click.stop>
              <button
                @click="handleTestConnection(row)"
                :disabled="testingProxyIds.has(row.id)"
                class="inline-flex h-8 w-8 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-emerald-50 hover:text-emerald-600 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-emerald-900/20 dark:hover:text-emerald-400"
                :title="t('admin.proxies.testConnection')"
                :aria-label="t('admin.proxies.testConnection')"
              >
                <svg
                  v-if="testingProxyIds.has(row.id)"
                  class="h-4 w-4 animate-spin"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    class="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    stroke-width="4"
                  ></circle>
                  <path
                    class="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  ></path>
                </svg>
                <Icon v-else name="checkCircle" size="sm" />
                <span class="sr-only">{{ t('admin.proxies.testConnection') }}</span>
              </button>
              <button
                @click="handleQualityCheck(row)"
                :disabled="qualityCheckingProxyIds.has(row.id)"
                class="inline-flex h-8 w-8 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
                :title="t('admin.proxies.qualityCheck')"
                :aria-label="t('admin.proxies.qualityCheck')"
              >
                <svg
                  v-if="qualityCheckingProxyIds.has(row.id)"
                  class="h-4 w-4 animate-spin"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    class="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    stroke-width="4"
                  ></circle>
                  <path
                    class="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  ></path>
                </svg>
                <Icon v-else name="shield" size="sm" />
                <span class="sr-only">{{ t('admin.proxies.qualityCheck') }}</span>
              </button>
              <button
                @click="handleEdit(row)"
                class="inline-flex h-8 w-8 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
                :title="t('common.edit')"
                :aria-label="t('common.edit')"
              >
                <Icon name="edit" size="sm" />
                <span class="sr-only">{{ t('common.edit') }}</span>
              </button>
              <button
                type="button"
                @click="toggleRowActionMenu(row.id, $event)"
                class="inline-flex h-8 w-8 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:hover:bg-dark-700 dark:hover:text-white"
                :class="{ 'bg-gray-100 text-gray-900 dark:bg-dark-700 dark:text-white': dropdownState.activeRowActionMenuId === row.id }"
                :title="t('common.more')"
                :aria-label="t('common.more')"
                :aria-expanded="dropdownState.activeRowActionMenuId === row.id"
              >
                <Icon name="more" size="sm" />
                <span class="sr-only">{{ t('common.more') }}</span>
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.proxies.noProxiesYet')"
              :description="t('admin.proxies.createFirstProxy')"
              :action-text="t('admin.proxies.createProxy')"
              @action="modalState.showCreateModal = true"
            />
          </template>
        </DataTable>
        </template>
        </div>
      </template>

      <template #pagination>
        <Pagination
          v-if="activeTab === 'proxies' && pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- Create Proxy Modal -->
    <BaseDialog
      :show="modalState.showCreateModal"
      :title="t('admin.proxies.createProxy')"
      width="normal"
      @close="closeCreateModal"
    >
      <!-- Tab Switch -->
      <div class="mb-6 flex border-b border-gray-200 dark:border-dark-600">
        <button
          type="button"
          @click="createMode = 'standard'"
          :class="[
            '-mb-px border-b-2 px-4 py-2 text-sm font-medium transition-colors',
            createMode === 'standard'
              ? 'border-primary-500 text-primary-600 dark:text-primary-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
          ]"
        >
          <Icon name="plus" size="sm" class="mr-1.5 inline" />
          {{ t('admin.proxies.standardAdd') }}
        </button>
        <button
          type="button"
          @click="createMode = 'batch'"
          :class="[
            '-mb-px border-b-2 px-4 py-2 text-sm font-medium transition-colors',
            createMode === 'batch'
              ? 'border-primary-500 text-primary-600 dark:text-primary-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
          ]"
        >
          <svg
            class="mr-1.5 inline h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="1.5"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M3.75 12h16.5m-16.5 3.75h16.5M3.75 19.5h16.5M5.625 4.5h12.75a1.875 1.875 0 010 3.75H5.625a1.875 1.875 0 010-3.75z"
            />
          </svg>
          {{ t('admin.proxies.batchAdd') }}
        </button>
      </div>

      <!-- Standard Add Form -->
      <form
        v-if="createMode === 'standard'"
        id="create-proxy-form"
        @submit.prevent="handleCreateProxy"
        class="space-y-5"
      >
        <div>
          <label class="input-label">{{ t('admin.proxies.name') }}</label>
          <input
            v-model="createForm.name"
            type="text"
            required
            class="input"
            :placeholder="t('admin.proxies.enterProxyName')"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.proxies.protocol') }}</label>
          <Select v-model="createForm.protocol" :options="protocolSelectOptions" />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('admin.proxies.host') }}</label>
            <input
              v-model="createForm.host"
              type="text"
              required
              :placeholder="t('admin.proxies.form.hostPlaceholder')"
              class="input"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.proxies.port') }}</label>
            <input
              v-model.number="createForm.port"
              type="number"
              required
              min="1"
              max="65535"
              :placeholder="t('admin.proxies.form.portPlaceholder')"
              class="input"
            />
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('admin.proxies.username') }}</label>
          <input
            v-model="createForm.username"
            type="text"
            class="input"
            :placeholder="t('admin.proxies.optionalAuth')"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.proxies.password') }}</label>
          <div class="relative">
            <input
              v-model="createForm.password"
              :type="passwordState.createPasswordVisible ? 'text' : 'password'"
              class="input pr-10"
              :placeholder="t('admin.proxies.optionalAuth')"
            />
            <button
              type="button"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              @click="passwordState.createPasswordVisible = !passwordState.createPasswordVisible"
            >
              <Icon :name="passwordState.createPasswordVisible ? 'eyeOff' : 'eye'" size="md" />
            </button>
          </div>
        </div>
        <label class="flex items-start gap-3 rounded-lg border border-gray-200 p-3 text-sm dark:border-dark-600">
          <input
            v-model="createForm.auto_failover_pool_enabled"
            type="checkbox"
            class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <span class="space-y-1">
            <span class="font-medium text-gray-900 dark:text-white">{{ t('admin.proxies.poolToggleLabel') }}</span>
            <span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.proxies.poolToggleHint') }}</span>
          </span>
        </label>

      </form>

      <!-- Batch Add Form -->
      <div v-else class="space-y-5">
        <div>
          <label class="input-label">{{ t('admin.proxies.batchInput') }}</label>
          <textarea
            v-model="batchInput"
            rows="10"
            class="input font-mono text-sm"
            :placeholder="t('admin.proxies.batchInputPlaceholder')"
            @input="parseBatchInput"
          ></textarea>
          <p class="input-hint mt-2">
            {{ t('admin.proxies.batchInputHint') }}
          </p>
        </div>

        <!-- Parse Result -->
        <div v-if="batchParseResult.total > 0" class="rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
            <div class="flex items-center gap-4 text-sm">
              <div class="flex items-center gap-1.5">
              <Icon name="checkCircle" size="sm" :stroke-width="2" class="text-primary-500" />
              <span class="text-gray-700 dark:text-gray-300">
                {{ t('admin.proxies.parsedCount', { count: batchParseResult.valid }) }}
              </span>
            </div>
            <div v-if="batchParseResult.invalid > 0" class="flex items-center gap-1.5">
              <Icon
                name="exclamationCircle"
                size="sm"
                :stroke-width="2"
                class="text-amber-500"
              />
              <span class="text-amber-600 dark:text-amber-400">
                {{ t('admin.proxies.invalidCount', { count: batchParseResult.invalid }) }}
              </span>
            </div>
            <div v-if="batchParseResult.duplicate > 0" class="flex items-center gap-1.5">
              <svg
                class="h-4 w-4 text-gray-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75"
                />
              </svg>
              <span class="text-gray-500 dark:text-gray-400">
                {{ t('admin.proxies.duplicateCount', { count: batchParseResult.duplicate }) }}
              </span>
            </div>
          </div>
        </div>

      </div>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeCreateModal" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            v-if="createMode === 'standard'"
            type="submit"
            form="create-proxy-form"
            :disabled="loadingState.submitting"
            class="btn btn-primary"
          >
            <svg
              v-if="loadingState.submitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{ loadingState.submitting ? t('admin.proxies.creating') : t('common.create') }}
          </button>
          <button
            v-else
            @click="handleBatchCreate"
            type="button"
            :disabled="loadingState.submitting || batchParseResult.valid === 0"
            class="btn btn-primary"
          >
            <svg
              v-if="loadingState.submitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{
              loadingState.submitting
                ? t('admin.proxies.importing')
                : t('admin.proxies.importProxies', { count: batchParseResult.valid })
            }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Edit Proxy Modal -->
    <BaseDialog
      :show="modalState.showEditModal"
      :title="t('admin.proxies.editProxy')"
      width="normal"
      @close="closeEditModal"
    >
      <form
        v-if="currentItems.editingProxy"
        id="edit-proxy-form"
        @submit.prevent="handleUpdateProxy"
        class="space-y-5"
      >
        <div>
          <label class="input-label">{{ t('admin.proxies.name') }}</label>
          <input v-model="editForm.name" type="text" required class="input" :disabled="currentItems.editingProxy?.managed_by_subscription" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.proxies.protocol') }}</label>
          <Select v-model="editForm.protocol" :options="protocolSelectOptions" :disabled="currentItems.editingProxy?.managed_by_subscription" />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('admin.proxies.host') }}</label>
            <input v-model="editForm.host" type="text" required class="input" :disabled="currentItems.editingProxy?.managed_by_subscription" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.proxies.port') }}</label>
            <input
              v-model.number="editForm.port"
              type="number"
              required
              min="1"
              max="65535"
              class="input"
              :disabled="currentItems.editingProxy?.managed_by_subscription"
            />
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('admin.proxies.username') }}</label>
          <input v-model="editForm.username" type="text" class="input" :disabled="currentItems.editingProxy?.managed_by_subscription" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.proxies.password') }}</label>
          <div class="relative">
            <input
              v-model="editForm.password"
              :type="passwordState.editPasswordVisible ? 'text' : 'password'"
              :placeholder="t('admin.proxies.leaveEmptyToKeep')"
              class="input pr-10"
              @input="passwordState.editPasswordDirty = true"
              :disabled="currentItems.editingProxy?.managed_by_subscription"
            />
            <button
              type="button"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              @click="passwordState.editPasswordVisible = !passwordState.editPasswordVisible"
            >
              <Icon :name="passwordState.editPasswordVisible ? 'eyeOff' : 'eye'" size="md" />
            </button>
          </div>
        </div>
        <div
          v-if="currentItems.editingProxy?.managed_by_subscription"
          class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-400"
        >
          {{ t('admin.proxies.subscriptions.managedReadonlyHint') }}
        </div>
        <div>
          <label class="input-label">{{ t('admin.proxies.status') }}</label>
          <Select v-model="editForm.status" :options="editStatusOptions" />
        </div>
        <label class="flex items-start gap-3 rounded-lg border border-gray-200 p-3 text-sm dark:border-dark-600">
          <input
            v-model="editForm.auto_failover_pool_enabled"
            type="checkbox"
            class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <span class="space-y-1">
            <span class="font-medium text-gray-900 dark:text-white">{{ t('admin.proxies.poolToggleLabel') }}</span>
            <span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.proxies.poolToggleHint') }}</span>
          </span>
        </label>

      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeEditModal" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            v-if="currentItems.editingProxy"
            type="submit"
            form="edit-proxy-form"
            :disabled="loadingState.submitting"
            class="btn btn-primary"
          >
            <svg
              v-if="loadingState.submitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{ loadingState.submitting ? t('admin.proxies.updating') : t('common.update') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="modalState.showDeleteDialog"
      :title="t('admin.proxies.deleteProxy')"
      :message="t('admin.proxies.deleteConfirm', { name: currentItems.deletingProxy?.name })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="modalState.showDeleteDialog = false"
    />

    <!-- Batch Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="modalState.showBatchDeleteDialog"
      :title="t('admin.proxies.batchDelete')"
      :message="t('admin.proxies.batchDeleteConfirm', { count: selectedCount })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmBatchDelete"
      @cancel="modalState.showBatchDeleteDialog = false"
    />
    <ConfirmDialog
      :show="modalState.showBatchUnassignDialog"
      :title="t('admin.proxies.quickUnassign')"
      :message="t('admin.proxies.quickUnassignConfirm', { count: selectedCount })"
      :confirm-text="t('admin.proxies.quickUnassignConfirmButton')"
      :cancel-text="t('common.cancel')"
      @confirm="confirmBatchUnassign"
      @cancel="modalState.showBatchUnassignDialog = false"
    />
    <ConfirmDialog
      :show="modalState.showExportDataDialog"
      :title="t('admin.proxies.dataExport')"
      :message="t('admin.proxies.dataExportConfirmMessage')"
      :confirm-text="t('admin.proxies.dataExportConfirm')"
      :cancel-text="t('common.cancel')"
      @confirm="handleExportData"
      @cancel="modalState.showExportDataDialog = false"
    />

    <ImportDataModal
      :show="modalState.showImportData"
      @close="modalState.showImportData = false"
      @imported="handleDataImported"
    />

    <AssignAccountsModal
      :show="modalState.showAssignAccounts"
      :proxy-ids="Array.from(selectedProxyIds)"
      :groups="dataState.accountGroups"
      @close="modalState.showAssignAccounts = false"
      @assigned="loadProxies"
    />

    <BaseDialog
      :show="modalState.showQualityReportDialog"
      :title="t('admin.proxies.qualityReportTitle')"
      width="wide"
      @close="closeQualityReportDialog"
    >
      <div v-if="currentItems.qualityReport" class="space-y-4">
        <div class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-700/80">
          <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div class="min-w-0 space-y-3">
              <div>
                <div class="text-sm text-gray-500 dark:text-gray-400">
                {{ currentItems.qualityReportProxy?.name || '-' }}
                </div>
                <div class="mt-1 text-base font-medium text-gray-900 dark:text-white">
                  {{ currentItems.qualityReport.summary }}
                </div>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <span class="badge" :class="qualityOverallClass(qualityOverallStatus(currentItems.qualityReport))">
                  {{ qualityOverallLabel(qualityOverallStatus(currentItems.qualityReport)) }}
                </span>
                <span
                  v-for="stat in qualityReportBreakdown(currentItems.qualityReport)"
                  :key="stat.key"
                  class="inline-flex items-center gap-1 rounded-full border border-gray-200 bg-white px-2.5 py-1 text-xs text-gray-600 dark:border-dark-500 dark:bg-dark-800 dark:text-gray-300"
                >
                  <span class="font-medium text-gray-900 dark:text-white">{{ stat.label }}</span>
                  <span>{{ stat.value }}</span>
                </span>
              </div>
            </div>
            <div class="grid shrink-0 grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-2 xl:grid-cols-4">
              <div class="rounded-lg bg-white px-4 py-3 text-center shadow-sm dark:bg-dark-800">
                <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.proxies.qualityScoreLabel') }}
                </div>
                <div class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">
                  {{ currentItems.qualityReport.score }}
                </div>
              </div>
              <div class="rounded-lg bg-white px-4 py-3 text-center shadow-sm dark:bg-dark-800">
                <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.proxies.qualityGradeLabel') }}
                </div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ currentItems.qualityReport.grade }}
                </div>
              </div>
              <div class="rounded-lg bg-white px-4 py-3 text-center shadow-sm dark:bg-dark-800">
                <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.proxies.qualityBaseLatency') }}
                </div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ typeof currentItems.qualityReport.base_latency_ms === 'number' ? `${currentItems.qualityReport.base_latency_ms}ms` : '-' }}
                </div>
              </div>
              <div class="rounded-lg bg-white px-4 py-3 text-center shadow-sm dark:bg-dark-800">
                <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.proxies.qualityCountry') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ currentItems.qualityReport.country || '-' }}
                </div>
              </div>
            </div>
          </div>
          <div class="mt-4 grid gap-3 text-xs text-gray-600 dark:text-gray-300 sm:grid-cols-2 xl:grid-cols-4">
            <div class="rounded-lg border border-gray-200 bg-white px-3 py-2 dark:border-dark-500 dark:bg-dark-800">
              <div class="text-[11px] uppercase tracking-wide text-gray-400 dark:text-gray-500">{{ t('admin.proxies.qualityExitIP') }}</div>
              <div class="mt-1 break-all text-sm text-gray-900 dark:text-white">{{ currentItems.qualityReport.exit_ip || '-' }}</div>
            </div>
            <div class="rounded-lg border border-gray-200 bg-white px-3 py-2 dark:border-dark-500 dark:bg-dark-800">
              <div class="text-[11px] uppercase tracking-wide text-gray-400 dark:text-gray-500">{{ t('admin.proxies.qualityCountry') }}</div>
              <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ currentItems.qualityReport.country || '-' }}</div>
            </div>
            <div class="rounded-lg border border-gray-200 bg-white px-3 py-2 dark:border-dark-500 dark:bg-dark-800">
              <div class="text-[11px] uppercase tracking-wide text-gray-400 dark:text-gray-500">{{ t('admin.proxies.qualityCheckedAt') }}</div>
              <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ formatQualityCheckedAt(currentItems.qualityReport.checked_at) }}</div>
            </div>
            <div class="rounded-lg border border-gray-200 bg-white px-3 py-2 dark:border-dark-500 dark:bg-dark-800">
              <div class="text-[11px] uppercase tracking-wide text-gray-400 dark:text-gray-500">{{ t('admin.proxies.qualityInterpretation') }}</div>
              <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ qualityInterpretationLabel(qualityOverallStatus(currentItems.qualityReport)) }}</div>
            </div>
          </div>
        </div>

        <div class="max-h-80 overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
          <table class="min-w-full table-fixed divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-xs uppercase text-gray-500 dark:bg-dark-800 dark:text-dark-400">
              <tr>
                <th class="w-32 px-3 py-2 text-left">{{ t('admin.proxies.qualityTableTarget') }}</th>
                <th class="w-28 px-3 py-2 text-left">{{ t('admin.proxies.qualityTableStatus') }}</th>
                <th class="w-20 px-3 py-2 text-left">HTTP</th>
                <th class="w-24 px-3 py-2 text-left">{{ t('admin.proxies.qualityTableLatency') }}</th>
                <th class="px-3 py-2 text-left">{{ t('admin.proxies.qualityTableMessage') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="item in currentItems.qualityReport.items" :key="item.target">
                <td class="px-3 py-3 align-top text-gray-900 dark:text-white">{{ qualityTargetLabel(item.target) }}</td>
                <td class="px-3 py-2">
                  <span class="badge" :class="qualityStatusClass(item.status)">{{ qualityStatusLabel(item.status) }}</span>
                </td>
                <td class="px-3 py-3 align-top text-gray-600 dark:text-gray-300">{{ item.http_status ?? '-' }}</td>
                <td class="px-3 py-3 align-top text-gray-600 dark:text-gray-300">
                  {{ typeof item.latency_ms === 'number' ? `${item.latency_ms}ms` : '-' }}
                </td>
                <td class="px-3 py-3 align-top text-gray-600 dark:text-gray-300">
                  <div class="space-y-1">
                    <div>{{ qualityItemMessage(item) }}</div>
                    <div v-if="item.cf_ray" class="text-xs text-gray-400">CF-Ray: {{ item.cf_ray }}</div>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end">
          <button @click="closeQualityReportDialog" class="btn btn-secondary">
            {{ t('common.close') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Proxy Accounts Dialog -->
    <BaseDialog
      :show="modalState.showAccountsModal"
      :title="t('admin.proxies.accountsTitle', { name: currentItems.accountsProxy?.name || '' })"
      width="normal"
      @close="closeAccountsModal"
    >
      <div v-if="loadingState.accountsLoading" class="flex items-center justify-center py-8 text-sm text-gray-500">
        <Icon name="refresh" size="md" class="mr-2 animate-spin" />
        {{ t('common.loading') }}
      </div>
      <div v-else-if="dataState.proxyAccounts.length === 0" class="py-6 text-center text-sm text-gray-500">
        {{ t('admin.proxies.accountsEmpty') }}
      </div>
      <div v-else class="max-h-80 overflow-auto">
        <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
          <thead class="bg-gray-50 text-xs uppercase text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-4 py-2 text-left">{{ t('admin.proxies.accountName') }}</th>
              <th class="px-4 py-2 text-left">{{ t('admin.accounts.columns.platformType') }}</th>
              <th class="px-4 py-2 text-left">{{ t('admin.proxies.accountNotes') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
            <tr v-for="account in dataState.proxyAccounts" :key="account.id">
              <td class="px-4 py-2 font-medium text-gray-900 dark:text-white">{{ account.name }}</td>
              <td class="px-4 py-2">
                <PlatformTypeBadge :platform="account.platform" :type="account.type" />
              </td>
              <td class="px-4 py-2 text-gray-600 dark:text-gray-300">
                {{ account.notes || '-' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <template #footer>
        <div class="flex justify-end">
          <button @click="closeAccountsModal" class="btn btn-secondary">
            {{ t('common.close') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <PoolMembersDialog
      :show="modalState.showPoolDialog"
      :loading="loadingState.poolDialogLoading"
      :rows="dataState.poolDialogRows"
      @close="modalState.showPoolDialog = false"
    />

    <Teleport to="body">
      <div
        v-if="dropdownState.activeRowActionMenuId !== null && dropdownState.rowActionMenuPosition"
        class="fixed z-[200] w-44 overflow-hidden rounded-xl bg-white shadow-lg ring-1 ring-black/5 dark:bg-dark-800 dark:ring-white/10"
        :style="{
          top: `${dropdownState.rowActionMenuPosition.top}px`,
          left: `${dropdownState.rowActionMenuPosition.left}px`
        }"
        @click.stop
      >
        <div class="py-1">
          <button
            v-if="activeRowActionMenuRow"
            @click="handleToggleStatus(activeRowActionMenuRow); closeRowActionMenu()"
            class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
          >
            <Icon
              :name="activeRowActionMenuRow.status === 'active' ? 'ban' : 'play'"
              size="sm"
              :class="activeRowActionMenuRow.status === 'active' ? 'text-amber-500' : 'text-emerald-500'"
            />
            {{ activeRowActionMenuRow.status === 'active' ? t('admin.proxies.disableAction') : t('admin.proxies.enableAction') }}
          </button>
          <button
            v-if="activeRowActionMenuRow"
            @click="handleTogglePoolMembership(activeRowActionMenuRow); closeRowActionMenu()"
            class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
          >
            <Icon :name="activeRowActionMenuRow.auto_failover_pool_enabled ? 'x' : 'plus'" size="sm" class="text-violet-500" />
            {{ activeRowActionMenuRow.auto_failover_pool_enabled ? t('admin.proxies.poolDisableAction') : t('admin.proxies.poolEnableAction') }}
          </button>
          <button
            v-if="activeRowActionMenuRow"
            @click="handleClearCooldown([activeRowActionMenuRow.id]); closeRowActionMenu()"
            class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
          >
            <Icon name="refresh" size="sm" class="text-amber-500" />
            {{ t('admin.proxies.clearCooldownAction') }}
          </button>
          <div class="my-1 border-t border-gray-100 dark:border-dark-700"></div>
          <button
            v-if="activeRowActionMenuRow"
            @click="handleDelete(activeRowActionMenuRow); closeRowActionMenu()"
            class="flex w-full items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
          >
            <Icon name="trash" size="sm" />
            {{ t('common.delete') }}
          </button>
        </div>
      </div>
    </Teleport>

    <SubscriptionSourceDialog
      :show="modalState.showCreateSubscriptionModal"
      :editing="!!currentItems.editingSubscription"
      :submitting="loadingState.submittingSubscription"
      :form="subscriptionForm"
      :format-options="subscriptionFormatOptions"
      @close="modalState.showCreateSubscriptionModal = false"
      @submit="handleSubmitSubscription"
      @update:form="updateSubscriptionForm"
    />

    <BaseDialog
      :show="modalState.showSubscriptionNodesModal"
      :title="t('admin.proxies.subscriptions.nodesTitle')"
      width="normal"
      @close="modalState.showSubscriptionNodesModal = false"
    >
      <div v-if="loadingState.subscriptionNodesLoading" class="p-2 text-sm text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</div>
      <div v-else-if="dataState.subscriptionNodes.length === 0" class="p-2 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.proxies.subscriptions.nodesEmpty') }}</div>
      <div v-else class="max-h-[60vh] space-y-2 overflow-y-auto">
        <div v-for="node in dataState.subscriptionNodes" :key="node.id" class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <div class="flex items-center gap-2">
            <span class="font-medium text-gray-900 dark:text-white">{{ node.display_name || `${node.server}:${node.port}` }}</span>
            <span class="badge badge-gray">{{ node.node_type }}</span>
            <span class="badge" :class="node.landing_status === 'active' ? 'badge-success' : 'badge-warning'">{{ node.landing_status }}</span>
          </div>
          <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ node.server }}:{{ node.port }}</div>
          <div v-if="node.last_error" class="mt-2 text-xs text-red-500 dark:text-red-400">{{ node.last_error }}</div>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end">
          <button class="btn btn-secondary" type="button" @click="modalState.showSubscriptionNodesModal = false">{{ t('common.close') }}</button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type {
  Proxy,
  ProxyAccountSummary,
  ProxyProtocol,
  ProxyQualityCheckItem,
  ProxyQualityCheckResult,
  ProxySubscriptionSource,
  ProxySubscriptionNode
} from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ImportDataModal from '@/components/admin/proxy/ImportDataModal.vue'
import AssignAccountsModal from '@/components/admin/proxy/AssignAccountsModal.vue'
import PoolMembersDialog from '@/components/admin/proxy/PoolMembersDialog.vue'
import ProxiesToolbar from '@/components/admin/proxy/ProxiesToolbar.vue'
import ProxySubscriptionsPanel from '@/components/admin/proxy/ProxySubscriptionsPanel.vue'
import SubscriptionSourceDialog from '@/components/admin/proxy/SubscriptionSourceDialog.vue'
import ProxyBulkActionsBar from '@/components/admin/proxy/ProxyBulkActionsBar.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useSwipeSelect } from '@/composables/useSwipeSelect'
import { useTableSelection } from '@/composables/useTableSelection'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useProxyTesting } from '@/composables/useProxyTesting'
import { useProxyResultHandler } from '@/composables/useProxyResultHandler'
import { useKeyboardShortcuts } from '@/composables/useKeyboardShortcuts'
import type { AdminGroup } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()
const activeTab = ref<'proxies' | 'subscriptions'>('proxies')
const searchInputRef = ref<HTMLInputElement | null>(null)

const {
  testingProxyIds,
  qualityCheckingProxyIds,
  testSingleProxy,
  checkSingleProxyQuality
} = useProxyTesting()

const {
  applyLatencyResult,
  applyQualityResult,
  extractBaseConnectivityResult,
  summarizeQualityStatus
} = useProxyResultHandler(computed(() => dataState.proxies))

const allColumns = computed<Column[]>(() => [
  { key: 'select', label: '', sortable: false, class: 'w-[52px] min-w-[52px]' },
  { key: 'name', label: t('admin.proxies.columns.name'), sortable: true, class: 'min-w-[160px]' },
  { key: 'protocol', label: t('admin.proxies.columns.protocol'), sortable: true, class: 'min-w-[88px]' },
  { key: 'address', label: t('admin.proxies.columns.address'), sortable: false, class: 'min-w-[200px]' },
  { key: 'auth', label: t('admin.proxies.columns.auth'), sortable: false, class: 'min-w-[160px]' },
  { key: 'location', label: t('admin.proxies.columns.location'), sortable: false, class: 'min-w-[180px]' },
  { key: 'account_count', label: t('admin.proxies.columns.accounts'), sortable: true, class: 'min-w-[100px]' },
  { key: 'pool', label: t('admin.proxies.columns.pool'), sortable: false, class: 'min-w-[120px]' },
  { key: 'latency', label: t('admin.proxies.columns.latency'), sortable: false, class: 'min-w-[140px]' },
  { key: 'status', label: t('admin.proxies.columns.status'), sortable: true, class: 'min-w-[160px]' },
  { key: 'actions', label: t('admin.proxies.columns.actions'), sortable: false, class: 'w-[176px] min-w-[176px]' }
])
const toggleableColumns = computed(() =>
  allColumns.value.filter((column) => column.key !== 'select' && column.key !== 'actions')
)
const hiddenColumns = reactive<Set<string>>(new Set())
const HIDDEN_COLUMNS_KEY = 'admin-proxies-hidden-columns'

const loadSavedColumns = () => {
  hiddenColumns.clear()
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    if (!saved) return
    const parsed = JSON.parse(saved) as string[]
    const toggleableKeys = new Set(toggleableColumns.value.map((column) => column.key))
    parsed
      .filter((key) => toggleableKeys.has(key))
      .forEach((key) => hiddenColumns.add(key))
  } catch (error) {
    console.error('Failed to load saved proxy columns:', error)
  }
}

const saveColumnsToStorage = () => {
  try {
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
  } catch (error) {
    console.error('Failed to save proxy columns:', error)
  }
}

const toggleColumn = (key: string) => {
  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key)
  } else {
    hiddenColumns.add(key)
  }
  saveColumnsToStorage()
}

const isColumnVisible = (key: string) => !hiddenColumns.has(key)

const columns = computed<Column[]>(() =>
  allColumns.value.filter((column) =>
    column.key === 'select' || column.key === 'actions' || !hiddenColumns.has(column.key)
  )
)

// Filter options
const protocolOptions = computed(() => [
  { value: '', label: t('admin.proxies.allProtocols') },
  { value: 'http', label: 'HTTP' },
  { value: 'https', label: 'HTTPS' },
  { value: 'socks5', label: 'SOCKS5' },
  { value: 'socks5h', label: 'SOCKS5H' }
])

const statusOptions = computed(() => [
  { value: '', label: t('admin.proxies.allStatus') },
  { value: 'active', label: t('admin.accounts.status.active') },
  { value: 'inactive', label: t('admin.accounts.status.inactive') }
])

const runtimeStatusOptions = computed(() => [
  { value: '', label: t('admin.proxies.allRuntimeStatus') },
  { value: 'healthy', label: t('admin.proxies.healthHealthy') },
  { value: 'cooldown', label: t('admin.proxies.healthCooldown') },
  { value: 'warn', label: t('admin.proxies.qualityStatusWarn') },
  { value: 'challenge', label: t('admin.proxies.qualityStatusChallenge') },
  { value: 'failed', label: t('admin.proxies.failedStatus') }
])

// Form options
const protocolSelectOptions = computed(() => [
  { value: 'http', label: t('admin.proxies.protocols.http') },
  { value: 'https', label: t('admin.proxies.protocols.https') },
  { value: 'socks5', label: t('admin.proxies.protocols.socks5') },
  { value: 'socks5h', label: t('admin.proxies.protocols.socks5h') }
])

const editStatusOptions = computed(() => [
  { value: 'active', label: t('admin.accounts.status.active') },
  { value: 'inactive', label: t('admin.accounts.status.inactive') }
])

// Data state
const dataState = reactive({
  proxies: [] as Proxy[],
  proxySubscriptions: [] as ProxySubscriptionSource[],
  accountGroups: [] as AdminGroup[],
  proxyAccounts: [] as ProxyAccountSummary[],
  subscriptionNodes: [] as ProxySubscriptionNode[],
  poolDialogRows: [] as Proxy[]
})

// UI state - Modals and dialogs
const modalState = reactive({
  showCreateModal: false,
  showEditModal: false,
  showImportData: false,
  showAssignAccounts: false,
  showDeleteDialog: false,
  showBatchDeleteDialog: false,
  showBatchUnassignDialog: false,
  showExportDataDialog: false,
  showAccountsModal: false,
  showPoolDialog: false,
  showCreateSubscriptionModal: false,
  showSubscriptionNodesModal: false,
  showQualityReportDialog: false
})

// UI state - Dropdowns and menus
const dropdownState = reactive({
  showColumnDropdown: false,
  showProxyToolsDropdown: false,
  showProxyBatchDropdown: false,
  activeRowActionMenuId: null as number | null,
  rowActionMenuPosition: null as { top: number; left: number } | null,
  copyMenuProxyId: null as number | null
})

// UI state - Password visibility
const passwordState = reactive({
  visiblePasswordIds: new Set<number>(),
  createPasswordVisible: false,
  editPasswordVisible: false,
  editPasswordDirty: false
})

// Loading state
const loadingState = reactive({
  loading: false,
  loadingSubscriptions: false,
  submitting: false,
  submittingSubscription: false,
  exportingData: false,
  accountsLoading: false,
  poolDialogLoading: false,
  subscriptionNodesLoading: false
})

// Testing state
const testingState = reactive({
  batchTesting: false,
  batchQualityChecking: false
})

// Filter and search state
const searchQuery = ref('')
const filters = reactive({
  protocol: '',
  status: '',
  runtime_status: ''
})
const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})
const sortState = reactive({
  sort_by: 'id',
  sort_order: 'desc' as 'asc' | 'desc'
})

// Refs
const proxyTableRef = ref<HTMLElement | null>(null)
// Current editing/viewing items
const currentItems = reactive({
  accountsProxy: null as Proxy | null,
  editingProxy: null as Proxy | null,
  deletingProxy: null as Proxy | null,
  qualityReportProxy: null as Proxy | null,
  qualityReport: null as ProxyQualityCheckResult | null,
  editingSubscription: null as ProxySubscriptionSource | null
})

const activeRowActionMenuRow = computed(() => {
  if (dropdownState.activeRowActionMenuId === null) return null
  return dataState.proxies.find((row) => row.id === dropdownState.activeRowActionMenuId) || null
})

const {
  selectedSet: selectedProxyIds,
  selectedCount,
  allVisibleSelected,
  isSelected,
  select,
  deselect,
  clear: clearSelectedProxies,
  removeMany: removeSelectedProxies,
  toggleVisible,
  batchUpdate
} = useTableSelection<Proxy>({
  rows: computed(() => dataState.proxies),
  getId: (proxy) => proxy.id
})

useSwipeSelect(proxyTableRef, {
  isSelected,
  select,
  deselect,
  batchUpdate
})
const subscriptionFormatOptions = [
  { value: 'auto' as const, label: t('admin.proxies.subscriptions.formats.auto') },
  { value: 'direct_list' as const, label: t('admin.proxies.subscriptions.formats.directList') },
  { value: 'uri_list' as const, label: t('admin.proxies.subscriptions.formats.uriList') },
  { value: 'clash_yaml' as const, label: t('admin.proxies.subscriptions.formats.clashYaml') }
]
const subscriptionForm = reactive({
  name: '',
  url: '',
  source_format: 'auto' as 'auto' | 'direct_list' | 'uri_list' | 'clash_yaml',
  enabled: true,
  refresh_interval_hours: 6,
  target_entry_count: 3,
  auto_add_to_pool: true
})

const switchToSubscriptions = async () => {
  activeTab.value = 'subscriptions'
  if (dataState.proxySubscriptions.length === 0) {
    await loadProxySubscriptions()
  }
}

const handleToolbarSetTab = async (tab: 'proxies' | 'subscriptions') => {
  if (tab === 'subscriptions') {
    await switchToSubscriptions()
    return
  }
  activeTab.value = 'proxies'
}

const updateToolbarSearch = (value: string) => {
  searchQuery.value = value
  handleSearch()
}

const updateToolbarFilters = (nextFilters: { protocol: string; status: string; runtime_status: string }) => {
  filters.protocol = nextFilters.protocol
  filters.status = nextFilters.status
  filters.runtime_status = nextFilters.runtime_status
}

const updateSubscriptionForm = (nextForm: typeof subscriptionForm) => {
  subscriptionForm.name = nextForm.name
  subscriptionForm.url = nextForm.url
  subscriptionForm.source_format = nextForm.source_format
  subscriptionForm.enabled = nextForm.enabled
  subscriptionForm.refresh_interval_hours = nextForm.refresh_interval_hours
  subscriptionForm.target_entry_count = nextForm.target_entry_count
  subscriptionForm.auto_add_to_pool = nextForm.auto_add_to_pool
}

const openCreateSubscriptionDialog = () => {
  currentItems.editingSubscription = null
  subscriptionForm.name = ''
  subscriptionForm.url = ''
  subscriptionForm.source_format = 'auto'
  subscriptionForm.enabled = true
  subscriptionForm.refresh_interval_hours = 6
  subscriptionForm.target_entry_count = 3
  subscriptionForm.auto_add_to_pool = true
  modalState.showCreateSubscriptionModal = true
}

const handleEditSubscription = (item: ProxySubscriptionSource) => {
  currentItems.editingSubscription = item
  subscriptionForm.name = item.name
  subscriptionForm.url = item.url
  subscriptionForm.source_format = item.source_format
  subscriptionForm.enabled = item.enabled
  subscriptionForm.refresh_interval_hours = item.refresh_interval_hours
  subscriptionForm.target_entry_count = item.target_entry_count || 3
  subscriptionForm.auto_add_to_pool = item.auto_add_to_pool
  modalState.showCreateSubscriptionModal = true
}

// Batch import state
const createMode = ref<'standard' | 'batch'>('standard')
const batchInput = ref('')
const batchParseResult = reactive({
  total: 0,
  valid: 0,
  invalid: 0,
  duplicate: 0,
  proxies: [] as Array<{
    protocol: ProxyProtocol
    host: string
    port: number
    username: string
    password: string
  }>
})

const createForm = reactive({
  name: '',
  protocol: 'http' as ProxyProtocol,
  host: '',
  port: 8080,
  username: '',
  password: '',
  auto_failover_pool_enabled: false
})

const editForm = reactive({
  name: '',
  protocol: 'http' as ProxyProtocol,
  host: '',
  port: 8080,
  username: '',
  password: '',
  status: 'active' as 'active' | 'inactive',
  auto_failover_pool_enabled: false
})

let abortController: AbortController | null = null

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== 'object') return false
  const maybeError = error as { name?: string; code?: string }
  return maybeError.name === 'AbortError' || maybeError.code === 'ERR_CANCELED'
}

const toggleSelectRow = (id: number, event: Event) => {
  const target = event.target as HTMLInputElement
  if (target.checked) {
    select(id)
    return
  }
  deselect(id)
}

const toggleSelectAllVisible = (event: Event) => {
  const target = event.target as HTMLInputElement
  toggleVisible(target.checked)
}

const buildProxyQueryFilters = () => ({
  protocol: filters.protocol || undefined,
  status: (filters.status || undefined) as 'active' | 'inactive' | undefined,
  runtime_status: (filters.runtime_status || undefined) as 'failed' | 'cooldown' | 'healthy' | 'warn' | 'challenge' | undefined,
  search: searchQuery.value || undefined,
  sort_by: sortState.sort_by,
  sort_order: sortState.sort_order
})

const loadProxySubscriptions = async () => {
  loadingState.loadingSubscriptions = true
  try {
    const response = await adminAPI.proxySubscriptions.list(1, 100)
    dataState.proxySubscriptions = response.items
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.subscriptions.loadFailed'))
    console.error('Error loading proxy subscriptions:', error)
  } finally {
    loadingState.loadingSubscriptions = false
  }
}

const loadProxies = async () => {
  if (abortController) {
    abortController.abort()
  }
  const currentAbortController = new AbortController()
  abortController = currentAbortController
  loadingState.loading = true
  try {
    const response = await adminAPI.proxies.list(
      pagination.page,
      pagination.page_size,
      buildProxyQueryFilters(),
      { signal: currentAbortController.signal }
    )
    if (currentAbortController.signal.aborted || abortController !== currentAbortController) {
      return
    }
    dataState.proxies = response.items
    pagination.total = response.total
    pagination.pages = response.pages
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    appStore.showError(t('admin.proxies.failedToLoad'))
    console.error('Error loading proxies:', error)
  } finally {
    if (abortController === currentAbortController) {
      loadingState.loading = false
      abortController = null
    }
  }
}

const loadAccountGroups = async () => {
  try {
    dataState.accountGroups = await adminAPI.groups.getAll()
  } catch (error) {
    console.error('Error loading account groups:', error)
  }
}

let searchTimeout: ReturnType<typeof setTimeout>
const handleSearch = () => {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    pagination.page = 1
    loadProxies()
  }, 300)
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadProxies()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize
  pagination.page = 1
  loadProxies()
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  loadProxies()
}

const closeCreateModal = () => {
  modalState.showCreateModal = false
  createMode.value = 'standard'
  createForm.name = ''
  createForm.protocol = 'http'
  createForm.host = ''
  createForm.port = 8080
  createForm.username = ''
  createForm.password = ''
  createForm.auto_failover_pool_enabled = false
  passwordState.createPasswordVisible = false
  batchInput.value = ''
  batchParseResult.total = 0
  batchParseResult.valid = 0
  batchParseResult.invalid = 0
  batchParseResult.duplicate = 0
  batchParseResult.proxies = []
}

const handleDataImported = () => {
  modalState.showImportData = false
  loadProxies()
}

// Parse proxy URL: protocol://user:pass@host:port or protocol://host:port
const parseProxyUrl = (
  line: string
): {
  protocol: ProxyProtocol
  host: string
  port: number
  username: string
  password: string
} | null => {
  const trimmed = line.trim()
  if (!trimmed) return null

  // Regex to parse proxy URL (supports http, https, socks5, socks5h)
  const regex = /^(https?|socks5h?):\/\/(?:([^:@]+):([^@]+)@)?([^:]+):(\d+)$/i
  const match = trimmed.match(regex)

  if (!match) return null

  const [, protocol, username, password, host, port] = match
  const portNum = parseInt(port, 10)

  if (portNum < 1 || portNum > 65535) return null

  return {
    protocol: protocol.toLowerCase() as ProxyProtocol,
    host: host.trim(),
    port: portNum,
    username: username?.trim() || '',
    password: password?.trim() || ''
  }
}

const parseBatchInput = () => {
  const lines = batchInput.value.split('\n').filter((l) => l.trim())
  const seen = new Set<string>()
  const proxies: typeof batchParseResult.proxies = []
  let invalid = 0
  let duplicate = 0

  for (const line of lines) {
    const parsed = parseProxyUrl(line)
    if (!parsed) {
      invalid++
      continue
    }

    // Check for duplicates (same host:port:username:password)
    const key = `${parsed.host}:${parsed.port}:${parsed.username}:${parsed.password}`
    if (seen.has(key)) {
      duplicate++
      continue
    }
    seen.add(key)
    proxies.push(parsed)
  }

  batchParseResult.total = lines.length
  batchParseResult.valid = proxies.length
  batchParseResult.invalid = invalid
  batchParseResult.duplicate = duplicate
  batchParseResult.proxies = proxies
}

const handleBatchCreate = async () => {
  if (batchParseResult.valid === 0) return

  loadingState.submitting = true
  try {
    const result = await adminAPI.proxies.batchCreate(batchParseResult.proxies)
    const created = result.created || 0
    const skipped = result.skipped || 0

    if (created > 0) {
      appStore.showSuccess(t('admin.proxies.batchImportSuccess', { created, skipped }))
    } else {
      appStore.showInfo(t('admin.proxies.batchImportAllSkipped', { skipped }))
    }

    closeCreateModal()
    loadProxies()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.failedToImport'))
    console.error('Error batch creating proxies:', error)
  } finally {
    loadingState.submitting = false
  }
}

const handleCreateProxy = async () => {
  if (!createForm.name.trim()) {
    appStore.showError(t('admin.proxies.nameRequired'))
    return
  }
  if (!createForm.host.trim()) {
    appStore.showError(t('admin.proxies.hostRequired'))
    return
  }
  if (createForm.port < 1 || createForm.port > 65535) {
    appStore.showError(t('admin.proxies.portInvalid'))
    return
  }
  loadingState.submitting = true
  try {
    await adminAPI.proxies.create({
      name: createForm.name.trim(),
      protocol: createForm.protocol,
      host: createForm.host.trim(),
      port: createForm.port,
      username: createForm.username.trim() || null,
      password: createForm.password.trim() || null,
      auto_failover_pool_enabled: createForm.auto_failover_pool_enabled
    })
    appStore.showSuccess(t('admin.proxies.proxyCreated'))
    closeCreateModal()
    loadProxies()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.failedToCreate'))
    console.error('Error creating proxy:', error)
  } finally {
    loadingState.submitting = false
  }
}

const handleEdit = (proxy: Proxy) => {
  currentItems.editingProxy = proxy
  editForm.name = proxy.name
  editForm.protocol = proxy.protocol
  editForm.host = proxy.host
  editForm.port = proxy.port
  editForm.username = proxy.username || ''
  editForm.password = proxy.password || ''
  editForm.status = proxy.status
  editForm.auto_failover_pool_enabled = !!proxy.auto_failover_pool_enabled
  passwordState.editPasswordVisible = false
  passwordState.editPasswordDirty = false
  modalState.showEditModal = true
}

const closeEditModal = () => {
  modalState.showEditModal = false
  currentItems.editingProxy = null
  passwordState.editPasswordVisible = false
  passwordState.editPasswordDirty = false
}

const handleUpdateProxy = async () => {
  if (!currentItems.editingProxy) return
  if (!editForm.name.trim()) {
    appStore.showError(t('admin.proxies.nameRequired'))
    return
  }
  if (!editForm.host.trim()) {
    appStore.showError(t('admin.proxies.hostRequired'))
    return
  }
  if (editForm.port < 1 || editForm.port > 65535) {
    appStore.showError(t('admin.proxies.portInvalid'))
    return
  }

  loadingState.submitting = true
  try {
    const updateData: any = {
      name: editForm.name.trim(),
      protocol: editForm.protocol,
      host: editForm.host.trim(),
      port: editForm.port,
      username: editForm.username.trim() || null,
      status: editForm.status,
      auto_failover_pool_enabled: editForm.auto_failover_pool_enabled
    }

    // Only include password if user actually modified the field
    if (passwordState.editPasswordDirty) {
      updateData.password = editForm.password.trim() || null
    }

    await adminAPI.proxies.update(currentItems.editingProxy.id, updateData)
    appStore.showSuccess(t('admin.proxies.proxyUpdated'))
    closeEditModal()
    loadProxies()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.failedToUpdate'))
    console.error('Error updating proxy:', error)
  } finally {
    loadingState.submitting = false
  }
}

const formatLocation = (proxy: Proxy) => {
  const parts = [proxy.country, proxy.city].filter(Boolean) as string[]
  return parts.join(' · ')
}

const formatRuntimeTime = (unix?: number) => {
  if (!unix) return '-'
  return new Date(unix * 1000).toLocaleString()
}

const formatCooldownCountdown = (unix?: number) => {
  if (!unix) return ''
  const remaining = unix * 1000 - Date.now()
  if (remaining <= 0) return t('admin.proxies.cooldownExpired')
  const minutes = Math.floor(remaining / 60000)
  const seconds = Math.floor((remaining % 60000) / 1000)
  return minutes > 0 ? `${minutes}m ${seconds}s` : `${seconds}s`
}

const healthStatusClass = (status?: Proxy['health_status']) => {
  if (status === 'healthy') return 'badge-success'
  if (status === 'cooldown') return 'badge-warning'
  if (status === 'failed') return 'badge-danger'
  return 'badge-gray'
}

const healthStatusLabel = (status?: Proxy['health_status']) => {
  if (status === 'healthy') return t('admin.proxies.healthHealthy')
  if (status === 'cooldown') return t('admin.proxies.healthCooldown')
  if (status === 'failed') return t('admin.proxies.healthFailed')
  return t('admin.proxies.healthUnknown')
}

const normalizedCountryCode = (code?: string) => {
  const normalized = code?.trim().toUpperCase() ?? ''
  return /^[A-Z]{2}$/.test(normalized) ? normalized : ''
}

const countryFlagEmoji = (code?: string) => {
  const normalized = normalizedCountryCode(code)
  if (!normalized) return ''
  return String.fromCodePoint(
    ...Array.from(normalized, (char) => 0x1f1a5 + char.charCodeAt(0))
  )
}

const runProxyTest = async (proxyId: number, notify: boolean) => {
  const result = await testSingleProxy(proxyId)
  if (!result) return null

  applyLatencyResult(proxyId, result)

  if (notify) {
    if (result.success) {
      const message = result.latency_ms
        ? t('admin.proxies.proxyWorkingWithLatency', { latency: result.latency_ms })
        : t('admin.proxies.proxyWorking')
      appStore.showSuccess(message)
    } else {
      appStore.showError(result.message || t('admin.proxies.proxyTestFailed'))
    }
  }

  return result
}

const handleTestConnection = async (proxy: Proxy) => {
  await runProxyTest(proxy.id, true)
}

const handleQualityCheck = async (proxy: Proxy) => {
  const result = await checkSingleProxyQuality(proxy.id)
  if (!result) {
    appStore.showError(t('admin.proxies.qualityCheckFailed'))
    return
  }

  currentItems.qualityReportProxy = proxy
  currentItems.qualityReport = result
  modalState.showQualityReportDialog = true

  const baseLatency = extractBaseConnectivityResult(result)
  if (baseLatency) {
    applyLatencyResult(proxy.id, baseLatency)
  }

  applyQualityResult(proxy.id, result)
  appStore.showSuccess(
    t('admin.proxies.qualityCheckDone', { score: result.score, grade: result.grade })
  )
}

const runBatchProxyQualityChecks = async (ids: number[]) => {
  if (ids.length === 0) {
    return { total: 0, healthy: 0, warn: 0, challenge: 0, failed: 0 }
  }

  const concurrency = 3
  let index = 0
  let healthy = 0
  let warn = 0
  let challenge = 0
  let failed = 0

  const worker = async () => {
    while (index < ids.length) {
      const current = ids[index]
      index++

      const result = await checkSingleProxyQuality(current)
      if (!result) {
        failed++
        continue
      }

      const baseLatency = extractBaseConnectivityResult(result)
      if (baseLatency) {
        applyLatencyResult(current, baseLatency)
      }
      applyQualityResult(current, result)

      const status = summarizeQualityStatus(result)
      if (status === 'challenge') {
        challenge++
      } else if (status === 'failed') {
        failed++
      } else if (status === 'warn') {
        warn++
      } else {
        healthy++
      }
    }
  }

  const workers = Array.from(
    { length: Math.min(concurrency, ids.length) },
    () => worker()
  )
  await Promise.all(workers)

  return {
    total: ids.length,
    healthy,
    warn,
    challenge,
    failed
  }
}

const closeQualityReportDialog = () => {
  modalState.showQualityReportDialog = false
  currentItems.qualityReportProxy = null
  currentItems.qualityReport = null
}

const qualityStatusClass = (status: string) => {
  if (status === 'pass') return 'badge-success'
  if (status === 'warn') return 'badge-warning'
  if (status === 'challenge') return 'badge-purple'
  return 'badge-danger'
}

const qualityStatusLabel = (status: string) => {
  if (status === 'pass') return t('admin.proxies.qualityStatusPass')
  if (status === 'warn') return t('admin.proxies.qualityStatusWarn')
  if (status === 'challenge') return t('admin.proxies.qualityStatusChallenge')
  return t('admin.proxies.qualityStatusFail')
}

const qualityOverallClass = (status?: string) => {
  if (status === 'healthy') return 'badge-success'
  if (status === 'warn') return 'badge-warning'
  if (status === 'challenge') return 'badge-purple'
  return 'badge-danger'
}

const qualityOverallLabel = (status?: string) => {
  if (status === 'healthy') return t('admin.proxies.qualityStatusHealthy')
  if (status === 'warn') return t('admin.proxies.qualityStatusWarn')
  if (status === 'challenge') return t('admin.proxies.qualityStatusChallenge')
  return t('admin.proxies.qualityStatusFail')
}

const qualityTargetLabel = (target: string) => {
  switch (target) {
    case 'base_connectivity':
      return t('admin.proxies.qualityTargetBase')
    case 'openai':
      return 'OpenAI'
    case 'anthropic':
      return 'Anthropic'
    case 'gemini':
      return 'Gemini'
    default:
      return target
  }
}

const qualityOverallStatus = (result?: ProxyQualityCheckResult | null): Proxy['quality_status'] => {
  if (!result) return 'failed'
  return summarizeQualityStatus(result)
}

const formatQualityCheckedAt = (checkedAt?: number) => {
  if (!checkedAt) return '-'
  return new Date(checkedAt * 1000).toLocaleString()
}

const qualityReportBreakdown = (result: ProxyQualityCheckResult) => [
  { key: 'pass', label: t('admin.proxies.qualityStatusPass'), value: result.passed_count },
  { key: 'warn', label: t('admin.proxies.qualityStatusWarn'), value: result.warn_count },
  { key: 'challenge', label: t('admin.proxies.qualityStatusChallenge'), value: result.challenge_count },
  { key: 'fail', label: t('admin.proxies.qualityStatusFail'), value: result.failed_count }
]

const qualityInterpretationLabel = (status?: Proxy['quality_status']) => {
  if (status === 'healthy') return t('admin.proxies.qualityInterpretationHealthy')
  if (status === 'warn') return t('admin.proxies.qualityInterpretationWarn')
  if (status === 'challenge') return t('admin.proxies.qualityInterpretationChallenge')
  return t('admin.proxies.qualityInterpretationFail')
}

const qualityItemMessage = (item: ProxyQualityCheckItem) => {
  if (item.message) return item.message
  return t('admin.proxies.qualityItemMessageEmpty')
}

const fetchAllProxiesForBatch = async (respectCurrentFilters: boolean = true): Promise<Proxy[]> => {
  const pageSize = 200
  const result: Proxy[] = []
  let page = 1
  let totalPages = 1

  while (page <= totalPages) {
    const response = await adminAPI.proxies.list(
      page,
      pageSize,
      respectCurrentFilters
        ? {
            protocol: filters.protocol || undefined,
            status: filters.status as any,
            runtime_status: filters.runtime_status as any,
            search: searchQuery.value || undefined,
            sort_by: sortState.sort_by,
            sort_order: sortState.sort_order
          }
        : {
            sort_by: sortState.sort_by,
            sort_order: sortState.sort_order
          }
    )
    result.push(...response.items)
    totalPages = response.pages || 1
    page++
  }

  return result
}

const runBatchProxyTests = async (ids: number[]) => {
  if (ids.length === 0) return

  const concurrency = 5
  let index = 0

  const worker = async () => {
    while (index < ids.length) {
      const current = ids[index]
      index++
      const result = await testSingleProxy(current)
      if (result) {
        applyLatencyResult(current, result)
      }
    }
  }

  const workers = Array.from(
    { length: Math.min(concurrency, ids.length) },
    () => worker()
  )
  await Promise.all(workers)
}

const handleBatchTest = async () => {
  if (testingState.batchTesting) return

  testingState.batchTesting = true
  try {
    let ids: number[] = []
    if (selectedCount.value > 0) {
      ids = Array.from(selectedProxyIds.value)
    } else {
      const allProxies = await fetchAllProxiesForBatch()
      ids = allProxies.map((proxy) => proxy.id)
    }

    if (ids.length === 0) {
      appStore.showInfo(t('admin.proxies.batchTestEmpty'))
      return
    }

    await runBatchProxyTests(ids)
    appStore.showSuccess(t('admin.proxies.batchTestDone', { count: ids.length }))
    loadProxies()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.batchTestFailed'))
    console.error('Error batch testing proxies:', error)
  } finally {
    testingState.batchTesting = false
  }
}

const handleBatchQualityCheck = async () => {
  if (testingState.batchQualityChecking) return

  testingState.batchQualityChecking = true
  try {
    let ids: number[] = []
    if (selectedCount.value > 0) {
      ids = Array.from(selectedProxyIds.value)
    } else {
      const allProxies = await fetchAllProxiesForBatch()
      ids = allProxies.map((proxy) => proxy.id)
    }

    if (ids.length === 0) {
      appStore.showInfo(t('admin.proxies.batchQualityEmpty'))
      return
    }

    const summary = await runBatchProxyQualityChecks(ids)
    appStore.showSuccess(
      t('admin.proxies.batchQualityDone', {
        count: summary.total,
        healthy: summary.healthy,
        warn: summary.warn,
        challenge: summary.challenge,
        failed: summary.failed
      })
    )
    loadProxies()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.batchQualityFailed'))
    console.error('Error batch checking quality:', error)
  } finally {
    testingState.batchQualityChecking = false
  }
}

const handleBatchPoolMembership = async (enabled: boolean) => {
  const ids = Array.from(selectedProxyIds.value)
  if (ids.length === 0) return

  try {
    await adminAPI.proxies.updatePoolMembership(ids, enabled)
    appStore.showSuccess(
      enabled
        ? t('admin.proxies.poolBatchEnabled', { count: ids.length })
        : t('admin.proxies.poolBatchDisabled', { count: ids.length })
    )
    clearSelectedProxies()
    loadProxies()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.poolUpdateFailed'))
    console.error('Error updating proxy pool membership:', error)
  }
}

const handleTogglePoolMembership = async (proxy: Proxy) => {
  const enabled = !proxy.auto_failover_pool_enabled
  try {
    await adminAPI.proxies.updatePoolMembership([proxy.id], enabled)
    proxy.auto_failover_pool_enabled = enabled
    appStore.showSuccess(
      enabled ? t('admin.proxies.poolSingleEnabled', { name: proxy.name }) : t('admin.proxies.poolSingleDisabled', { name: proxy.name })
    )
    if (modalState.showPoolDialog) {
      await openPoolDialog()
    }
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.poolUpdateFailed'))
    console.error('Error toggling proxy pool membership:', error)
  }
}

const handleToggleStatus = async (proxy: Proxy) => {
  const nextStatus: 'active' | 'inactive' = proxy.status === 'active' ? 'inactive' : 'active'
  try {
    await adminAPI.proxies.update(proxy.id, { status: nextStatus })
    proxy.status = nextStatus
    appStore.showSuccess(
      nextStatus === 'active' ? t('admin.proxies.statusEnabled') : t('admin.proxies.statusDisabled')
    )

    await loadProxies()
    if (modalState.showPoolDialog) {
      await openPoolDialog()
    }
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.failedToToggle'))
    console.error('Error toggling proxy status:', error)
  }
}

const handleClearCooldown = async (ids: number[]) => {
  if (ids.length === 0) return
  try {
    await adminAPI.proxies.clearCooldown(ids)
    appStore.showSuccess(
      ids.length === 1
        ? t('admin.proxies.cooldownClearedSingle')
        : t('admin.proxies.cooldownClearedBatch', { count: ids.length })
    )
    loadProxies()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.cooldownClearFailed'))
    console.error('Error clearing proxy cooldown:', error)
  }
}

const openPoolDialog = async () => {
  modalState.showPoolDialog = true
  loadingState.poolDialogLoading = true
  try {
    const allProxies = await fetchAllProxiesForBatch(false)
    dataState.poolDialogRows = allProxies.filter((proxy) => proxy.auto_failover_pool_enabled)
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.failedToLoad'))
    console.error('Error loading proxy pool members:', error)
    dataState.poolDialogRows = []
  } finally {
    loadingState.poolDialogLoading = false
  }
}

const formatExportTimestamp = () => {
  const now = new Date()
  const pad2 = (value: number) => String(value).padStart(2, '0')
  return `${now.getFullYear()}${pad2(now.getMonth() + 1)}${pad2(now.getDate())}${pad2(now.getHours())}${pad2(now.getMinutes())}${pad2(now.getSeconds())}`
}

const handleExportData = async () => {
  if (loadingState.exportingData) return
  loadingState.exportingData = true
  try {
    const dataPayload = await adminAPI.proxies.exportData(
      selectedCount.value > 0
        ? { ids: Array.from(selectedProxyIds.value) }
        : {
            filters: buildProxyQueryFilters()
          }
    )
    const timestamp = formatExportTimestamp()
    const filename = `sub2api-proxy-${timestamp}.json`
    const blob = new Blob([JSON.stringify(dataPayload, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    link.click()
    URL.revokeObjectURL(url)
    appStore.showSuccess(t('admin.proxies.dataExported'))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.proxies.dataExportFailed'))
  } finally {
    loadingState.exportingData = false
    modalState.showExportDataDialog = false
  }
}

const handleRefreshSubscription = async (id: number) => {
  try {
    const result = await adminAPI.proxySubscriptions.refresh(id)
    appStore.showSuccess(t('admin.proxies.subscriptions.refreshSuccess', {
      nodes: result.node_count,
      proxies: result.materialized_proxy_count
    }))
    await loadProxySubscriptions()
    if (activeTab.value === 'proxies') {
      await loadProxies()
    }
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.subscriptions.refreshFailed'))
    console.error('Error refreshing proxy subscription:', error)
  }
}

const handleViewSubscriptionNodes = async (id: number) => {
  modalState.showSubscriptionNodesModal = true
  loadingState.subscriptionNodesLoading = true
  try {
    dataState.subscriptionNodes = await adminAPI.proxySubscriptions.listNodes(id)
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.subscriptions.nodesLoadFailed'))
    console.error('Error loading subscription nodes:', error)
  } finally {
    loadingState.subscriptionNodesLoading = false
  }
}

const handleDeleteSubscription = async (id: number) => {
  try {
    await adminAPI.proxySubscriptions.delete(id)
    appStore.showSuccess(t('admin.proxies.subscriptions.deleteSuccess'))
    await loadProxySubscriptions()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.subscriptions.deleteFailed'))
    console.error('Error deleting proxy subscription:', error)
  }
}

const handleSubmitSubscription = async () => {
  loadingState.submittingSubscription = true
  try {
    if (currentItems.editingSubscription) {
      await adminAPI.proxySubscriptions.update(currentItems.editingSubscription.id, {
        name: subscriptionForm.name.trim(),
        url: subscriptionForm.url.trim(),
        source_format: subscriptionForm.source_format,
        enabled: subscriptionForm.enabled,
        refresh_interval_hours: subscriptionForm.refresh_interval_hours,
        target_entry_count: subscriptionForm.target_entry_count,
        auto_add_to_pool: subscriptionForm.auto_add_to_pool
      })
      appStore.showSuccess(t('admin.proxies.subscriptions.updateSuccess'))
    } else {
      await adminAPI.proxySubscriptions.create({
        name: subscriptionForm.name.trim(),
        url: subscriptionForm.url.trim(),
        source_format: subscriptionForm.source_format,
        enabled: subscriptionForm.enabled,
        refresh_interval_hours: subscriptionForm.refresh_interval_hours,
        target_entry_count: subscriptionForm.target_entry_count,
        auto_add_to_pool: subscriptionForm.auto_add_to_pool
      })
      appStore.showSuccess(t('admin.proxies.subscriptions.createSuccess'))
    }
    modalState.showCreateSubscriptionModal = false
    currentItems.editingSubscription = null
    await loadProxySubscriptions()
  } catch (error: any) {
    appStore.showError(
      error.response?.data?.detail ||
      (currentItems.editingSubscription
        ? t('admin.proxies.subscriptions.updateFailed')
        : t('admin.proxies.subscriptions.createFailed'))
    )
    console.error('Error submitting proxy subscription:', error)
  } finally {
    loadingState.submittingSubscription = false
  }
}

const handleDelete = (proxy: Proxy) => {
  if ((proxy.account_count || 0) > 0) {
    appStore.showError(t('admin.proxies.deleteBlockedInUse'))
    return
  }
  currentItems.deletingProxy = proxy
  modalState.showDeleteDialog = true
}

const openBatchDelete = () => {
  if (selectedCount.value === 0) {
    return
  }
  modalState.showBatchDeleteDialog = true
}

const openBatchUnassign = () => {
  if (selectedCount.value === 0) {
    return
  }
  modalState.showBatchUnassignDialog = true
}

const confirmDelete = async () => {
  if (!currentItems.deletingProxy) return

  try {
    await adminAPI.proxies.delete(currentItems.deletingProxy.id)
    appStore.showSuccess(t('admin.proxies.proxyDeleted'))
    modalState.showDeleteDialog = false
    removeSelectedProxies([currentItems.deletingProxy.id])
    currentItems.deletingProxy = null
    loadProxies()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.failedToDelete'))
    console.error('Error deleting proxy:', error)
  }
}

const confirmBatchDelete = async () => {
  const ids = Array.from(selectedProxyIds.value)
  if (ids.length === 0) {
    modalState.showBatchDeleteDialog = false
    return
  }

  try {
    const result = await adminAPI.proxies.batchDelete(ids)
    const deleted = result.deleted_ids?.length || 0
    const skipped = result.skipped?.length || 0

    if (deleted > 0) {
      appStore.showSuccess(t('admin.proxies.batchDeleteDone', { deleted, skipped }))
    } else if (skipped > 0) {
      appStore.showInfo(t('admin.proxies.batchDeleteSkipped', { skipped }))
    }

    clearSelectedProxies()
    modalState.showBatchDeleteDialog = false
    loadProxies()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.batchDeleteFailed'))
    console.error('Error batch deleting proxies:', error)
  }
}

const confirmBatchUnassign = async () => {
  const ids = Array.from(selectedProxyIds.value)
  if (ids.length === 0) {
    modalState.showBatchUnassignDialog = false
    return
  }

  try {
    const result = await adminAPI.proxies.unassignAccounts(ids)
    appStore.showSuccess(
      t('admin.proxies.quickUnassignDone', {
        matched: result.matched_accounts,
        unassigned: result.unassigned_accounts
      })
    )
    clearSelectedProxies()
    modalState.showBatchUnassignDialog = false
    loadProxies()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.quickUnassignFailed'))
    console.error('Error unassigning proxy accounts:', error)
  }
}

const openAccountsModal = async (proxy: Proxy) => {
  currentItems.accountsProxy = proxy
  dataState.proxyAccounts = []
  loadingState.accountsLoading = true
  modalState.showAccountsModal = true

  try {
    dataState.proxyAccounts = await adminAPI.proxies.getProxyAccounts(proxy.id)
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.accountsFailed'))
    console.error('Error loading proxy accounts:', error)
  } finally {
    loadingState.accountsLoading = false
  }
}

const closeAccountsModal = () => {
  modalState.showAccountsModal = false
  currentItems.accountsProxy = null
  dataState.proxyAccounts = []
}

// ── Proxy URL copy ──
function buildAuthPart(row: any): string {
  const user = row.username ? encodeURIComponent(row.username) : ''
  const pass = row.password ? encodeURIComponent(row.password) : ''
  if (user && pass) return `${user}:${pass}@`
  if (user) return `${user}@`
  if (pass) return `:${pass}@`
  return ''
}

function buildProxyUrl(row: any): string {
  return `${row.protocol}://${buildAuthPart(row)}${row.host}:${row.port}`
}

function getCopyFormats(row: any) {
  const hasAuth = row.username || row.password
  const fullUrl = buildProxyUrl(row)
  const formats = [
    { label: fullUrl, value: fullUrl },
  ]
  if (hasAuth) {
    const withoutProtocol = fullUrl.replace(/^[^:]+:\/\//, '')
    formats.push({ label: withoutProtocol, value: withoutProtocol })
  }
  formats.push({ label: `${row.host}:${row.port}`, value: `${row.host}:${row.port}` })
  return formats
}

function copyProxyUrl(row: any) {
  copyToClipboard(buildProxyUrl(row), t('admin.proxies.urlCopied'))
  dropdownState.copyMenuProxyId = null
}

function toggleCopyMenu(id: number) {
  dropdownState.copyMenuProxyId = dropdownState.copyMenuProxyId === id ? null : id
}

function copyFormat(value: string) {
  copyToClipboard(value, t('admin.proxies.urlCopied'))
  dropdownState.copyMenuProxyId = null
}

function toggleRowActionMenu(id: number, event: MouseEvent) {
  dropdownState.showColumnDropdown = false
  dropdownState.showProxyToolsDropdown = false
  dropdownState.showProxyBatchDropdown = false
  if (dropdownState.activeRowActionMenuId === id) {
    closeRowActionMenu()
    return
  }
  const trigger = event.currentTarget as HTMLElement | null
  if (!trigger) return
  const rect = trigger.getBoundingClientRect()
  const menuWidth = 176
  dropdownState.rowActionMenuPosition = {
    top: rect.bottom + 8,
    left: Math.max(8, rect.right - menuWidth)
  }
  dropdownState.activeRowActionMenuId = id
}

function closeRowActionMenu() {
  dropdownState.activeRowActionMenuId = null
  dropdownState.rowActionMenuPosition = null
}

function closeFloatingMenus() {
  dropdownState.copyMenuProxyId = null
  dropdownState.showColumnDropdown = false
  dropdownState.showProxyToolsDropdown = false
  dropdownState.showProxyBatchDropdown = false
  closeRowActionMenu()
}

// 键盘快捷键
const isAnyModalOpen = computed(() => {
  return (
    modalState.showCreateModal ||
    modalState.showEditModal ||
    modalState.showDeleteDialog ||
    modalState.showBatchDeleteDialog ||
    modalState.showBatchUnassignDialog ||
    modalState.showExportDataDialog ||
    modalState.showImportData ||
    modalState.showAssignAccounts ||
    modalState.showQualityReportDialog ||
    modalState.showAccountsModal ||
    modalState.showPoolDialog ||
    modalState.showCreateSubscriptionModal ||
    modalState.showSubscriptionNodesModal
  )
})

useKeyboardShortcuts({
  searchInputRef,
  onRefresh: () => {
    if (!isAnyModalOpen.value) {
      loadProxies()
    }
  },
  onSelectAll: () => {
    if (!isAnyModalOpen.value && activeTab.value === 'proxies' && dataState.proxies.length > 0) {
      const allIds = dataState.proxies.map(p => p.id)
      allIds.forEach(id => select(id))
    }
  },
  onClearSelection: () => {
    if (!isAnyModalOpen.value && selectedCount.value > 0) {
      clearSelectedProxies()
    }
  },
  onDelete: () => {
    if (!isAnyModalOpen.value && selectedCount.value > 0) {
      openBatchDelete()
    }
  },
  onEscape: () => {
    if (selectedCount.value > 0) {
      clearSelectedProxies()
    }
    closeFloatingMenus()
  },
  disabled: computed(() => isAnyModalOpen.value && selectedCount.value === 0)
})

onMounted(() => {
  loadSavedColumns()
  loadProxies()
  loadAccountGroups()
  document.addEventListener('click', closeFloatingMenus)
  window.addEventListener('scroll', closeRowActionMenu, true)
  window.addEventListener('resize', closeRowActionMenu)
})

onUnmounted(() => {
  clearTimeout(searchTimeout)
  abortController?.abort()
  document.removeEventListener('click', closeFloatingMenus)
  window.removeEventListener('scroll', closeRowActionMenu, true)
  window.removeEventListener('resize', closeRowActionMenu)
})
</script>
