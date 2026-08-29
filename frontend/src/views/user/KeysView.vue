<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col gap-3">
          <div class="flex flex-wrap items-center gap-3">
            <SearchInput
              v-model="filterSearch"
              :placeholder="t('keys.searchPlaceholder')"
              class="w-full sm:w-64"
              @search="onFilterChange"
            />
            <Select
              :model-value="filterGroupId"
              class="w-40"
              :options="groupFilterOptions"
              @update:model-value="onGroupFilterChange"
            />
            <Select
              :model-value="filterStatus"
              class="w-40"
              :options="statusFilterOptions"
              @update:model-value="onStatusFilterChange"
            />
          </div>
          <EndpointPopover
            v-if="publicSettings?.api_base_url || (publicSettings?.custom_endpoints?.length ?? 0) > 0"
            :api-base-url="publicSettings?.api_base_url || ''"
            :custom-endpoints="publicSettings?.custom_endpoints || []"
          />
        </div>
      </template>

      <template #actions>
        <div class="flex justify-end gap-3">
          <button
            @click="loadApiKeys"
            :disabled="loading"
            class="btn btn-secondary"
            :title="t('common.refresh')"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
          <div class="relative" ref="columnDropdownRef">
            <button
              @click="showColumnDropdown = !showColumnDropdown"
              class="btn btn-secondary px-2 md:px-3"
              :title="t('keys.columnSettings')"
            >
              <svg class="h-4 w-4 md:mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 4.5v15m6-15v15m-10.875 0h15.75c.621 0 1.125-.504 1.125-1.125V5.625c0-.621-.504-1.125-1.125-1.125H4.125C3.504 4.5 3 5.004 3 5.625v12.75c0 .621.504 1.125 1.125 1.125z" />
              </svg>
              <span class="hidden md:inline">{{ t('keys.columnSettings') }}</span>
            </button>
            <div
              v-if="showColumnDropdown"
              class="absolute right-0 top-full z-50 mt-1 max-h-80 w-48 overflow-y-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-600 dark:bg-dark-800"
            >
              <button
                v-for="col in toggleableColumns"
                :key="col.key"
                @click="toggleColumn(col.key)"
                class="flex w-full items-center justify-between px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
              >
                <span>{{ col.label }}</span>
                <Icon
                  v-if="isColumnVisible(col.key)"
                  name="check"
                  size="sm"
                  class="text-primary-500"
                  :stroke-width="2"
                />
              </button>
            </div>
          </div>
          <button @click="openCreateModal" class="btn btn-primary" data-tour="keys-create-btn">
            <Icon name="plus" size="md" class="mr-2" />
            {{ t('keys.createKey') }}
          </button>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="apiKeys"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="created_at"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-id="{ value }">
            <span class="font-mono text-xs text-gray-500 dark:text-gray-400">#{{ value }}</span>
          </template>

          <template #cell-key="{ value, row }">
            <div class="flex items-center gap-2">
              <code class="code text-xs">
                {{ maskApiKey(value) }}
              </code>
              <button
                @click="copyToClipboard(value, row.id)"
                class="rounded-lg p-1 transition-colors hover:bg-gray-100 dark:hover:bg-dark-700"
                :class="
                  copiedKeyId === row.id
                    ? 'text-green-500'
                    : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300'
                "
                :title="copiedKeyId === row.id ? t('keys.copied') : t('keys.copyToClipboard')"
              >
                <Icon
                  v-if="copiedKeyId === row.id"
                  name="check"
                  size="sm"
                  :stroke-width="2"
                />
                <Icon v-else name="clipboard" size="sm" />
              </button>
            </div>
          </template>

          <template #cell-name="{ value, row }">
            <div class="flex items-center gap-1.5">
              <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
              <Icon
                v-if="row.ip_whitelist?.length > 0 || row.ip_blacklist?.length > 0"
                name="shield"
                size="sm"
                class="text-blue-500"
                :title="t('keys.ipRestrictionEnabled')"
              />
            </div>
          </template>

          <template #cell-group="{ row }">
            <div class="group/dropdown relative">
              <button
                :ref="(el) => setGroupButtonRef(row.id, el)"
                @click="openGroupSelector(row)"
                class="-mx-2 -my-1 flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1 transition-all duration-200 hover:bg-gray-100 dark:hover:bg-dark-700"
                :title="t('keys.clickToChangeGroup')"
              >
                <GroupBadge
                  v-if="row.group"
                  :name="row.group.name"
                  :platform="row.group.platform"
                  :subscription-type="row.group.subscription_type"
                  :rate-multiplier="row.group.rate_multiplier"
                  :user-rate-multiplier="userGroupRates[row.group.id]"
                  :peak-rate-enabled="row.group.peak_rate_enabled"
                  :peak-start="row.group.peak_start"
                  :peak-end="row.group.peak_end"
                  :peak-rate-multiplier="row.group.peak_rate_multiplier"
                />
                <span v-else class="text-sm text-gray-400 dark:text-dark-500">{{
                  t('keys.noGroup')
                }}</span>
                <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('keys.selectGroup') }}</span>
                <svg
                  class="h-3.5 w-3.5 text-gray-400 opacity-60 transition-opacity group-hover/dropdown:opacity-100"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  stroke-width="2"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M8.25 15L12 18.75 15.75 15m-7.5-6L12 5.25 15.75 9"
                  />
                </svg>
              </button>
            </div>
          </template>

          <template #cell-routing="{ row }">
            <div class="key-routing-cell" :aria-label="t('keys.routingListLabel')">
              <template v-if="routingListSummaries[row.id]?.loading">
                <span class="key-routing-badge key-routing-badge-muted">{{ t('keys.routingSummaryLoading') }}</span>
              </template>
              <template v-else-if="routingListSummaries[row.id]?.error">
                <span class="key-routing-badge key-routing-badge-muted">{{ t('keys.routingSummaryUnavailable') }}</span>
              </template>
              <template v-else>
                <span
                  class="key-routing-badge"
                  :class="routingListSummaryIsSmart(routingListSummaries[row.id])
                    ? 'key-routing-badge-smart'
                    : 'key-routing-badge-single'"
                >
                  <Icon :name="routingListSummaryIsSmart(routingListSummaries[row.id]) ? 'sparkles' : 'server'" size="xs" />
                  {{ routingListSummaryIsSmart(routingListSummaries[row.id]) ? t('keys.smartRouting') : t('keys.singleGroup') }}
                </span>
                <span v-if="routingListSummaryIsSmart(routingListSummaries[row.id])" class="key-routing-count">
                  {{ routingListSummaryCandidateCount(routingListSummaries[row.id]) }}{{ t('keys.routingCandidateCountSuffix') }}
                </span>
                <span class="key-routing-status" :class="routingListSummaryStatusClass(routingListSummaries[row.id])">
                  <span class="key-routing-status-dot" aria-hidden="true" />
                  {{ routingListSummaryStatusLabel(routingListSummaries[row.id]) }}
                </span>
              </template>
            </div>
          </template>

          <template #cell-current_concurrency="{ value }">
            <span
              :class="[
                'inline-flex min-w-8 items-center justify-center rounded px-2 py-1 text-sm font-semibold tabular-nums',
                (value ?? 0) > 0
                  ? 'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-900/25 dark:text-emerald-300 dark:ring-emerald-800'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-400'
              ]"
            >
              {{ value ?? 0 }}
            </span>
          </template>

          <template #cell-usage="{ row }">
            <div class="text-sm">
              <div class="flex items-center gap-1.5">
                <span class="text-gray-500 dark:text-gray-400">{{ t('keys.today') }}:</span>
                <span class="font-medium text-gray-900 dark:text-white">
                  ${{ (usageStats[row.id]?.today_actual_cost ?? 0).toFixed(4) }}
                </span>
              </div>
              <div class="mt-0.5 flex items-center gap-1.5">
                <span class="text-gray-500 dark:text-gray-400">{{ t('keys.total') }}:</span>
                <span class="font-medium text-gray-900 dark:text-white">
                  ${{ (usageStats[row.id]?.total_actual_cost ?? 0).toFixed(4) }}
                </span>
              </div>
              <!-- Quota progress (if quota is set) -->
              <div v-if="row.quota > 0" class="mt-1.5">
                <div class="flex items-center gap-1.5">
                  <span class="text-gray-500 dark:text-gray-400">{{ t('keys.quota') }}:</span>
                  <span :class="[
                    'font-medium',
                    row.quota_used >= row.quota ? 'text-red-500' :
                    row.quota_used >= row.quota * 0.8 ? 'text-yellow-500' :
                    'text-gray-900 dark:text-white'
                  ]">
                    ${{ row.quota_used?.toFixed(2) || '0.00' }} / ${{ row.quota?.toFixed(2) }}
                  </span>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      row.quota_used >= row.quota ? 'bg-red-500' :
                      row.quota_used >= row.quota * 0.8 ? 'bg-yellow-500' :
                      'bg-primary-500'
                    ]"
                    :style="{ width: Math.min((row.quota_used / row.quota) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>
          </template>

          <template #cell-rate_limit="{ row }">
            <div v-if="row.rate_limit_5h > 0 || row.rate_limit_1d > 0 || row.rate_limit_7d > 0" class="space-y-1.5 min-w-[140px]">
              <!-- 5h window -->
              <div v-if="row.rate_limit_5h > 0">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-gray-500 dark:text-gray-400">5h</span>
                  <span :class="[
                    'font-medium tabular-nums',
                    row.usage_5h >= row.rate_limit_5h ? 'text-red-500' :
                    row.usage_5h >= row.rate_limit_5h * 0.8 ? 'text-yellow-500' :
                    'text-gray-700 dark:text-gray-300'
                  ]">
                    ${{ row.usage_5h?.toFixed(2) || '0.00' }}/${{ row.rate_limit_5h?.toFixed(2) }}
                  </span>
                </div>
                <div class="h-1 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      row.usage_5h >= row.rate_limit_5h ? 'bg-red-500' :
                      row.usage_5h >= row.rate_limit_5h * 0.8 ? 'bg-yellow-500' :
                      'bg-emerald-500'
                    ]"
                    :style="{ width: Math.min((row.usage_5h / row.rate_limit_5h) * 100, 100) + '%' }"
                  />
                </div>
                <div v-if="row.reset_5h_at && formatResetTime(row.reset_5h_at)" class="text-[10px] text-gray-400 dark:text-gray-500 tabular-nums">
                  ⟳ {{ formatResetTime(row.reset_5h_at) }}
                </div>
              </div>
              <!-- 1d window -->
              <div v-if="row.rate_limit_1d > 0">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-gray-500 dark:text-gray-400">1d</span>
                  <span :class="[
                    'font-medium tabular-nums',
                    row.usage_1d >= row.rate_limit_1d ? 'text-red-500' :
                    row.usage_1d >= row.rate_limit_1d * 0.8 ? 'text-yellow-500' :
                    'text-gray-700 dark:text-gray-300'
                  ]">
                    ${{ row.usage_1d?.toFixed(2) || '0.00' }}/${{ row.rate_limit_1d?.toFixed(2) }}
                  </span>
                </div>
                <div class="h-1 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      row.usage_1d >= row.rate_limit_1d ? 'bg-red-500' :
                      row.usage_1d >= row.rate_limit_1d * 0.8 ? 'bg-yellow-500' :
                      'bg-emerald-500'
                    ]"
                    :style="{ width: Math.min((row.usage_1d / row.rate_limit_1d) * 100, 100) + '%' }"
                  />
                </div>
                <div v-if="row.reset_1d_at && formatResetTime(row.reset_1d_at)" class="text-[10px] text-gray-400 dark:text-gray-500 tabular-nums">
                  ⟳ {{ formatResetTime(row.reset_1d_at) }}
                </div>
              </div>
              <!-- 7d window -->
              <div v-if="row.rate_limit_7d > 0">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-gray-500 dark:text-gray-400">7d</span>
                  <span :class="[
                    'font-medium tabular-nums',
                    row.usage_7d >= row.rate_limit_7d ? 'text-red-500' :
                    row.usage_7d >= row.rate_limit_7d * 0.8 ? 'text-yellow-500' :
                    'text-gray-700 dark:text-gray-300'
                  ]">
                    ${{ row.usage_7d?.toFixed(2) || '0.00' }}/${{ row.rate_limit_7d?.toFixed(2) }}
                  </span>
                </div>
                <div class="h-1 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      row.usage_7d >= row.rate_limit_7d ? 'bg-red-500' :
                      row.usage_7d >= row.rate_limit_7d * 0.8 ? 'bg-yellow-500' :
                      'bg-emerald-500'
                    ]"
                    :style="{ width: Math.min((row.usage_7d / row.rate_limit_7d) * 100, 100) + '%' }"
                  />
                </div>
                <div v-if="row.reset_7d_at && formatResetTime(row.reset_7d_at)" class="text-[10px] text-gray-400 dark:text-gray-500 tabular-nums">
                  ⟳ {{ formatResetTime(row.reset_7d_at) }}
                </div>
              </div>
              <!-- Reset button -->
              <button
                v-if="row.usage_5h > 0 || row.usage_1d > 0 || row.usage_7d > 0"
                @click.stop="confirmResetRateLimitFromTable(row)"
                class="mt-0.5 inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
                :title="t('keys.resetRateLimitUsage')"
              >
                <Icon name="refresh" size="xs" />
                {{ t('keys.resetUsage') }}
              </button>
            </div>
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
          </template>

          <template #cell-expires_at="{ value }">
            <span v-if="value" :class="[
              'text-sm',
              new Date(value) < new Date() ? 'text-red-500 dark:text-red-400' : 'text-gray-500 dark:text-dark-400'
            ]">
              {{ formatDateTime(value) }}
            </span>
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">{{ t('keys.noExpiration') }}</span>
          </template>

          <template #cell-status="{ value }">
            <span :class="[
              'badge',
              value === 'active' ? 'badge-success' :
              value === 'quota_exhausted' ? 'badge-warning' :
              value === 'expired' ? 'badge-danger' :
              'badge-gray'
            ]">
              {{ t('keys.status.' + value) }}
            </span>
          </template>

          <template #cell-last_used_at="{ value }">
            <span v-if="value" class="text-sm text-gray-500 dark:text-dark-400">
              {{ formatDateTime(value) }}
            </span>
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
          </template>

          <template #cell-last_used_ip="{ value }">
            <span v-if="value" class="text-sm text-gray-500 dark:text-dark-400">
              {{ value }}
            </span>
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <!-- Use Key Button -->
              <button
                @click="openUseKeyModal(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/20 dark:hover:text-green-400"
              >
                <Icon name="terminal" size="sm" />
                <span class="text-xs">{{ t('keys.useKey') }}</span>
              </button>
              <!-- Import to CC Switch Button -->
              <button
                v-if="!publicSettings?.hide_ccs_import_button"
                @click="importToCcswitch(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
              >
                <Icon name="upload" size="sm" />
                <span class="text-xs">{{ t('keys.importToCcSwitch') }}</span>
              </button>
              <!-- Toggle Status Button -->
              <button
                @click="toggleKeyStatus(row)"
                :class="[
                  'flex flex-col items-center gap-0.5 rounded-lg p-1.5 transition-colors',
                  row.status === 'active'
                    ? 'text-gray-500 hover:bg-yellow-50 hover:text-yellow-600 dark:hover:bg-yellow-900/20 dark:hover:text-yellow-400'
                    : 'text-gray-500 hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/20 dark:hover:text-green-400'
                ]"
              >
                <Icon v-if="row.status === 'active'" name="ban" size="sm" />
                <Icon v-else name="checkCircle" size="sm" />
                <span class="text-xs">{{ row.status === 'active' ? t('keys.disable') : t('keys.enable') }}</span>
              </button>
              <!-- Edit Button -->
              <button
                @click="editKey(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
              >
                <Icon name="edit" size="sm" />
                <span class="text-xs">{{ t('common.edit') }}</span>
              </button>
              <!-- Delete Button -->
              <button
                @click="confirmDelete(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
              >
                <Icon name="trash" size="sm" />
                <span class="text-xs">{{ t('common.delete') }}</span>
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('keys.noKeysYet')"
              :description="t('keys.createFirstKey')"
              :action-text="t('keys.createKey')"
              @action="openCreateModal"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- Create/Edit Modal -->
    <BaseDialog
      :show="showCreateModal || showEditModal"
      :title="showEditModal ? t('keys.editKey') : t('keys.createKey')"
      width="normal"
      @close="closeModals"
    >
      <form id="key-form" @submit.prevent="handleSubmit" class="key-editor-form space-y-5" :class="{ 'routing-smart-form': formData.routing_mode === 'smart' }">
        <div class="routing-name-field">
          <label class="input-label">{{ t('keys.nameLabel') }}</label>
          <input
            v-model="formData.name"
            type="text"
            required
            class="input"
            :placeholder="t('keys.namePlaceholder')"
            data-tour="key-form-name"
          />
        </div>

        <div class="routing-primary-group">
          <label class="input-label">{{ t('keys.groupLabel') }}</label>
          <Select
            v-model="formData.group_id"
            :options="groupOptions"
            :disabled="routingLoading"
            :placeholder="t('keys.selectGroup')"
            :searchable="true"
            :search-placeholder="t('keys.searchGroup')"
            data-tour="key-form-group"
          >
            <template #selected="{ option }">
              <GroupBadge
                v-if="option"
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                :peak-rate-enabled="(option as unknown as GroupOption).peakRateEnabled"
                :peak-start="(option as unknown as GroupOption).peakStart"
                :peak-end="(option as unknown as GroupOption).peakEnd"
                :peak-rate-multiplier="(option as unknown as GroupOption).peakRateMultiplier"
              />
              <span v-else class="text-gray-400">{{ t('keys.selectGroup') }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                :peak-rate-enabled="(option as unknown as GroupOption).peakRateEnabled"
                :peak-start="(option as unknown as GroupOption).peakStart"
                :peak-end="(option as unknown as GroupOption).peakEnd"
                :peak-rate-multiplier="(option as unknown as GroupOption).peakRateMultiplier"
                :description="(option as unknown as GroupOption).description"
                :selected="selected"
              />
            </template>
          </Select>
        </div>

        <section v-if="showEditModal || showCreateModal" class="routing-panel">
          <div class="routing-panel-header">
            <label class="input-label mb-0">{{ t('keys.routingMode') }}</label>
            <div class="routing-segmented" role="tablist" :aria-label="t('keys.routingMode')">
              <button type="button" role="tab" :aria-selected="formData.routing_mode === 'single'" :tabindex="formData.routing_mode === 'single' ? 0 : -1" data-routing-mode="single" :class="{ active: formData.routing_mode === 'single' }" :disabled="routingLoading" @click="formData.routing_mode = 'single'" @keydown="handleRoutingModeKeydown($event, 'single')">{{ t('keys.singleGroup') }}</button>
              <button type="button" role="tab" :aria-selected="formData.routing_mode === 'smart'" :tabindex="formData.routing_mode === 'smart' ? 0 : -1" data-routing-mode="smart" :class="{ active: formData.routing_mode === 'smart' }" :disabled="routingLoading" @click="formData.routing_mode = 'smart'" @keydown="handleRoutingModeKeydown($event, 'smart')">{{ t('keys.smartRouting') }}</button>
            </div>
          </div>
          <p class="routing-mode-hint">{{ formData.routing_mode === 'smart' ? t('keys.routingModeHint') : t('keys.routingSingleModeHint') }}</p>
          <div v-if="routingProfileLoadError" class="routing-profile-error" role="alert">
            <span class="routing-error-copy"><Icon name="exclamationTriangle" size="sm" /><span>{{ t('keys.routingLoadFailed') }}</span></span>
            <button type="button" class="routing-retry-button" :disabled="routingLoading || !selectedKey" @click="selectedKey && retryRoutingProfile(selectedKey.id)"><Icon name="refresh" size="xs" /> {{ t('keys.routingRetry') }}</button>
          </div>

          <div v-if="formData.routing_mode === 'smart'" class="routing-workspace" :aria-busy="routingLoading">
            <p v-if="routingLoading" class="routing-loading-hint">{{ t('keys.routingLoading') }}</p>
            <div class="routing-strategy-block">
              <div class="routing-workspace-header"><label class="input-label mb-0">{{ t('keys.routingStrategy') }}</label></div>
              <p class="routing-field-hint">{{ t('keys.routingStrategyHint') }}</p>
              <div class="routing-strategy-grid" role="radiogroup" :aria-label="t('keys.routingStrategy')">
                <button
                  v-for="strategy in routingStrategies"
                  :key="strategy.value"
                  type="button"
                  role="radio"
                  class="routing-strategy-card"
                  :class="[{ selected: formData.routing_strategy === strategy.value }, strategy.value === 'custom' && 'custom']"
                  :aria-checked="formData.routing_strategy === strategy.value"
                  :tabindex="formData.routing_strategy === strategy.value ? 0 : -1"
                  :data-routing-strategy="strategy.value"
                  :disabled="routingLoading"
                  @click="selectRoutingStrategy(strategy.value)"
                  @keydown="handleRoutingStrategyKeydown($event, strategy.value)"
                >
                  <span class="routing-strategy-icon" :class="strategy.tone"><Icon :name="strategy.icon" size="sm" /></span>
                  <span class="routing-strategy-copy"><strong>{{ strategy.label }}</strong><small>{{ strategy.description }}</small><em>{{ strategy.weightSummary }}</em></span>
                </button>
              </div>
              <div class="routing-mobile-strategy-select">
                <Select
                  v-model="formData.routing_strategy"
                  :options="routingStrategyOptions"
                  :disabled="routingLoading"
                  :aria-label="t('keys.routingStrategy')"
                  @update:model-value="handleRoutingStrategySelect"
                />
              </div>
              <div v-if="formData.routing_strategy === 'custom'" class="routing-custom-weights" :class="{ invalid: !routingWeightsValid }">
                <div class="routing-custom-weight-total" :class="routingWeightsValid ? 'valid' : 'invalid'" aria-live="polite">{{ t('keys.routingWeightTotal') }}: {{ routingWeightTotal }}%</div>
                <label v-for="field in routingWeightFields" :key="field.key"><span>{{ t(field.label) }}</span><div class="routing-number-input"><input v-model.number="formData[field.key]" :disabled="routingLoading" type="number" min="0" max="100" step="1" :aria-label="t(field.label)" /><b>%</b></div></label>
                <p v-if="!routingWeightsValid" class="routing-validation-error" role="alert">{{ t('keys.routingWeightInvalid') }}</p>
              </div>
              <div v-if="formData.routing_strategy === 'custom'" class="routing-mobile-weight-editor" :class="{ invalid: !routingWeightsValid }">
                <div class="routing-custom-weight-total" :class="routingWeightsValid ? 'valid' : 'invalid'" aria-live="polite">{{ t('keys.routingWeightTotal') }}: {{ routingWeightTotal }}%</div>
                <label v-for="field in routingWeightFields" :key="`mobile-${field.key}`"><span>{{ t(field.label) }}</span><div class="routing-number-input"><input v-model.number="formData[field.key]" :disabled="routingLoading" type="number" min="0" max="100" step="1" :aria-label="t(field.label)" @focus="handleMobileWeightFocus" /><b>%</b></div></label>
                <p v-if="!routingWeightsValid" class="routing-validation-error" role="alert">{{ t('keys.routingWeightInvalid') }}</p>
              </div>
            </div>

            <div class="routing-candidates-block">
              <div class="routing-workspace-header">
                <label class="input-label mb-0">{{ t('keys.routingCandidates') }}</label>
                <span class="routing-count" aria-live="polite">{{ formData.routing_group_ids.length }}{{ t('keys.routingCandidateCountSuffix') }}</span>
              </div>
              <p class="routing-field-hint">{{ t('keys.routingCandidatesHint') }}</p>
              <div v-if="selectedRoutingGroups.length" class="routing-selected-list" role="list" :aria-label="t('keys.routingCandidates')">
                <div v-for="(group, index) in selectedRoutingGroups" :key="group.id" class="routing-selected-row" :class="{ invalid: !group.eligible, dragging: routingDraggedGroupId === group.id, 'drag-over-before': routingDragOverGroupId === group.id && routingDragOverPlacement === 'before', 'drag-over-after': routingDragOverGroupId === group.id && routingDragOverPlacement === 'after' }" role="listitem" :aria-setsize="selectedRoutingGroups.length" :aria-posinset="index + 1" @dragover="handleRoutingGroupDragOver(group.id, $event)" @dragleave="handleRoutingGroupDragLeave(group.id, $event)" @drop.prevent="dropRoutingGroup(group.id)">
                  <span class="routing-order">{{ index + 1 }}</span>
                  <button type="button" class="routing-drag-handle" :aria-label="t('keys.dragRoutingCandidate')" :title="t('keys.dragRoutingCandidate')" :aria-keyshortcuts="'ArrowUp ArrowDown Home End'" :disabled="routingLoading" draggable="true" @dragstart="startRoutingGroupDrag(group.id, $event)" @dragend="clearRoutingGroupDrag" @keydown="handleRoutingCandidateKeydown($event, group.id)"><Icon name="sort" size="xs" /></button>
                  <span class="routing-selected-copy"><strong>{{ group.label }}</strong><small>{{ group.description }}</small></span>
                  <span class="routing-group-status" :class="group.eligible ? 'eligible' : 'invalid'" :title="group.eligible ? t('keys.routingCandidateReady') : t('keys.routingCandidateInvalidShort')"><Icon :name="group.eligible ? 'checkCircle' : 'exclamationTriangle'" size="xs" /><span>{{ group.eligible ? t('keys.routingCandidateReady') : t('keys.routingCandidateInvalidShort') }}</span></span>
                  <button type="button" class="routing-icon-button" :disabled="routingLoading || index === 0" :aria-label="t('keys.moveRoutingCandidateUp')" :title="t('keys.moveRoutingCandidateUp')" @click="moveRoutingGroup(group.id, -1)"><Icon name="chevronUp" size="xs" /></button>
                  <button type="button" class="routing-icon-button" :disabled="routingLoading || index === selectedRoutingGroups.length - 1" :aria-label="t('keys.moveRoutingCandidateDown')" :title="t('keys.moveRoutingCandidateDown')" @click="moveRoutingGroup(group.id, 1)"><Icon name="chevronDown" size="xs" /></button>
                  <button type="button" class="routing-icon-button danger" :disabled="routingLoading" :aria-label="`${t('common.delete')} ${group.label}`" :title="t('common.delete')" @click="removeRoutingGroup(group.id)"><Icon name="x" size="xs" /></button>
                </div>
              </div>
              <div v-else class="routing-empty">{{ t('keys.routingCandidatesEmpty') }}</div>
              <Select v-model="routingCandidateToAdd" class="routing-add-select" :options="availableRoutingGroupOptions" :placeholder="t('keys.addRoutingCandidate')" :searchable="true" :search-placeholder="t('keys.searchRoutingCandidates')" :empty-text="t('keys.routingCandidatesNoMatch')" :disabled="routingLoading || routingProfileLoadError || formData.routing_group_ids.length >= 8 || availableRoutingGroupOptions.length === 0" :aria-label="t('keys.selectRoutingCandidates')" @update:model-value="addRoutingGroup">
                <template #option="{ option }"><span class="routing-select-option"><strong>{{ option.label }}</strong><small>{{ option.description }}</small></span></template>
              </Select>
              <p v-if="routingSubmitAttempted && routingCandidateValidationError" class="routing-validation-error" role="alert">{{ routingCandidateValidationError }}</p>
            </div>

            <div class="routing-safety-note" role="note"><Icon name="shield" size="sm" /><span><strong>{{ t('keys.routingSafetyTitle') }}</strong> {{ t('keys.routingSafetyHint') }}</span></div>

          </div>
        </section>

        <!-- Custom Key Section (only for create) -->
        <div v-if="!showEditModal" class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.customKeyLabel') }}</label>
            <button
              type="button"
              @click="formData.use_custom_key = !formData.use_custom_key"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.use_custom_key ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  formData.use_custom_key ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          <div v-if="formData.use_custom_key">
            <input
              v-model="formData.custom_key"
              type="text"
              class="input font-mono"
              :placeholder="t('keys.customKeyPlaceholder')"
              :class="{ 'border-red-500 dark:border-red-500': customKeyError }"
            />
            <p v-if="customKeyError" class="mt-1 text-sm text-red-500">{{ customKeyError }}</p>
            <p v-else class="input-hint">{{ t('keys.customKeyHint') }}</p>
          </div>
        </div>

        <div v-if="showEditModal">
          <label class="input-label">{{ t('keys.statusLabel') }}</label>
          <Select
            v-model="formData.status"
            :options="statusOptions"
            :placeholder="t('keys.selectStatus')"
          />
        </div>

        <!-- IP Restriction Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.ipRestriction') }}</label>
            <button
              type="button"
              @click="formData.enable_ip_restriction = !formData.enable_ip_restriction"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_ip_restriction ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_ip_restriction ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_ip_restriction" class="space-y-4 pt-2">
            <div>
              <label class="input-label">{{ t('keys.ipWhitelist') }}</label>
              <textarea
                v-model="formData.ip_whitelist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipWhitelistPlaceholder')"
              />
              <p class="input-hint">{{ t('keys.ipWhitelistHint') }}</p>
            </div>

            <div>
              <label class="input-label">{{ t('keys.ipBlacklist') }}</label>
              <textarea
                v-model="formData.ip_blacklist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipBlacklistPlaceholder')"
              />
              <p class="input-hint">{{ t('keys.ipBlacklistHint') }}</p>
            </div>
          </div>
        </div>

        <!-- Quota Limit Section -->
        <div class="space-y-3">
          <label class="input-label">{{ t('keys.quotaLimit') }}</label>
          <!-- Switch commented out - always show input, 0 = unlimited
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.quotaLimit') }}</label>
            <button
              type="button"
              @click="formData.enable_quota = !formData.enable_quota"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_quota ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_quota ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          -->

          <div class="space-y-4">
            <div>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
                <input
                  v-model.number="formData.quota"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="t('keys.quotaAmountPlaceholder')"
                />
              </div>
              <p class="input-hint">{{ t('keys.quotaAmountHint') }}</p>
            </div>

            <!-- Quota used display (only in edit mode) -->
            <div v-if="showEditModal && selectedKey && selectedKey.quota > 0">
              <label class="input-label">{{ t('keys.quotaUsed') }}</label>
              <div class="flex items-center gap-2">
                <div class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700">
                  <span class="font-medium text-gray-900 dark:text-white">
                    ${{ selectedKey.quota_used?.toFixed(4) || '0.0000' }}
                  </span>
                  <span class="mx-2 text-gray-400">/</span>
                  <span class="text-gray-500 dark:text-gray-400">
                    ${{ selectedKey.quota?.toFixed(2) || '0.00' }}
                  </span>
                </div>
                <button
                  type="button"
                  @click="confirmResetQuota"
                  class="btn btn-secondary text-sm"
                  :title="t('keys.resetQuotaUsed')"
                >
                  {{ t('keys.reset') }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Rate Limit Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.rateLimitSection') }}</label>
            <button
              type="button"
              @click="formData.enable_rate_limit = !formData.enable_rate_limit"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_rate_limit ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_rate_limit ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_rate_limit" class="space-y-4 pt-2">
            <p class="input-hint -mt-2">{{ t('keys.rateLimitHint') }}</p>
            <!-- 5-Hour Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit5h') }}</label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
                <input
                  v-model.number="formData.rate_limit_5h"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_5h > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700 text-sm">
                    <span :class="[
                      'font-medium',
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h ? 'text-red-500' :
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h * 0.8 ? 'text-yellow-500' :
                      'text-gray-900 dark:text-white'
                    ]">
                      ${{ selectedKey.usage_5h?.toFixed(4) || '0.0000' }}
                    </span>
                    <span class="mx-2 text-gray-400">/</span>
                    <span class="text-gray-500 dark:text-gray-400">
                      ${{ selectedKey.rate_limit_5h?.toFixed(2) || '0.00' }}
                    </span>
                  </div>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h ? 'bg-red-500' :
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h * 0.8 ? 'bg-yellow-500' :
                      'bg-green-500'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_5h / selectedKey.rate_limit_5h) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- Daily Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit1d') }}</label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
                <input
                  v-model.number="formData.rate_limit_1d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_1d > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700 text-sm">
                    <span :class="[
                      'font-medium',
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d ? 'text-red-500' :
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d * 0.8 ? 'text-yellow-500' :
                      'text-gray-900 dark:text-white'
                    ]">
                      ${{ selectedKey.usage_1d?.toFixed(4) || '0.0000' }}
                    </span>
                    <span class="mx-2 text-gray-400">/</span>
                    <span class="text-gray-500 dark:text-gray-400">
                      ${{ selectedKey.rate_limit_1d?.toFixed(2) || '0.00' }}
                    </span>
                  </div>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d ? 'bg-red-500' :
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d * 0.8 ? 'bg-yellow-500' :
                      'bg-green-500'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_1d / selectedKey.rate_limit_1d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- 7-Day Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit7d') }}</label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
                <input
                  v-model.number="formData.rate_limit_7d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_7d > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700 text-sm">
                    <span :class="[
                      'font-medium',
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d ? 'text-red-500' :
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d * 0.8 ? 'text-yellow-500' :
                      'text-gray-900 dark:text-white'
                    ]">
                      ${{ selectedKey.usage_7d?.toFixed(4) || '0.0000' }}
                    </span>
                    <span class="mx-2 text-gray-400">/</span>
                    <span class="text-gray-500 dark:text-gray-400">
                      ${{ selectedKey.rate_limit_7d?.toFixed(2) || '0.00' }}
                    </span>
                  </div>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d ? 'bg-red-500' :
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d * 0.8 ? 'bg-yellow-500' :
                      'bg-green-500'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_7d / selectedKey.rate_limit_7d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- Reset Rate Limit button (edit mode only) -->
            <div v-if="showEditModal && selectedKey && (selectedKey.rate_limit_5h > 0 || selectedKey.rate_limit_1d > 0 || selectedKey.rate_limit_7d > 0)">
              <button
                type="button"
                @click="confirmResetRateLimit"
                class="btn btn-secondary text-sm"
              >
                {{ t('keys.resetRateLimitUsage') }}
              </button>
            </div>
          </div>
        </div>

        <!-- Expiration Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.expiration') }}</label>
            <button
              type="button"
              @click="formData.enable_expiration = !formData.enable_expiration"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_expiration ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_expiration ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_expiration" class="space-y-4 pt-2">
            <!-- Quick select buttons (for both create and edit mode) -->
            <div class="flex flex-wrap gap-2">
              <button
                v-for="days in ['7', '30', '90']"
                :key="days"
                type="button"
                @click="setExpirationDays(parseInt(days))"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm transition-colors',
                  formData.expiration_preset === days
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600'
                ]"
              >
                {{ showEditModal ? t('keys.extendDays', { days }) : t('keys.expiresInDays', { days }) }}
              </button>
              <button
                type="button"
                @click="formData.expiration_preset = 'custom'"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm transition-colors',
                  formData.expiration_preset === 'custom'
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600'
                ]"
              >
                {{ t('keys.customDate') }}
              </button>
            </div>

            <!-- Date picker (always show for precise adjustment) -->
            <div>
              <label class="input-label">{{ t('keys.expirationDate') }}</label>
              <input
                v-model="formData.expiration_date"
                type="datetime-local"
                class="input"
              />
              <p class="input-hint">{{ t('keys.expirationDateHint') }}</p>
            </div>

            <!-- Current expiration display (only in edit mode) -->
            <div v-if="showEditModal && selectedKey?.expires_at" class="text-sm">
              <span class="text-gray-500 dark:text-gray-400">{{ t('keys.currentExpiration') }}: </span>
              <span class="font-medium text-gray-900 dark:text-white">
                {{ formatDateTime(selectedKey.expires_at) }}
              </span>
            </div>
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeModals" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            form="key-form"
            type="submit"
            :disabled="submitting || routingLoading || routingProfileLoadError"
            class="btn btn-primary"
            data-tour="key-form-submit"
          >
            <svg
              v-if="submitting"
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
              submitting
                ? t('keys.saving')
                : showEditModal
                  ? t('common.update')
                  : t('common.create')
            }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('keys.deleteKey')"
      :message="t('keys.deleteConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="handleDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Reset Quota Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetQuotaDialog"
      :title="t('keys.resetQuotaTitle')"
      :message="t('keys.resetQuotaConfirmMessage', { name: selectedKey?.name, used: selectedKey?.quota_used?.toFixed(4) })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetQuotaUsed"
      @cancel="showResetQuotaDialog = false"
    />

    <!-- Reset Rate Limit Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetRateLimitDialog"
      :title="t('keys.resetRateLimitTitle')"
      :message="t('keys.resetRateLimitConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetRateLimitUsage"
      @cancel="showResetRateLimitDialog = false"
    />

    <!-- Use Key Modal -->
    <UseKeyModal
      :show="showUseKeyModal"
      :api-key="selectedKey?.key || ''"
      :base-url="publicSettings?.api_base_url || ''"
      :platform="selectedKey?.group?.platform || null"
      :allow-messages-dispatch="selectedKey?.group?.allow_messages_dispatch || false"
      @close="closeUseKeyModal"
    />

    <!-- CCS Client Selection Dialog for Antigravity -->
    <BaseDialog
      :show="showCcsClientSelect"
      :title="t('keys.ccsClientSelect.title')"
      width="narrow"
      @close="closeCcsClientSelect"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ t('keys.ccsClientSelect.description') }}
	        </p>
	        <div class="grid grid-cols-2 gap-3">
	          <button
	            @click="handleCcsClientSelect('claude')"
	            class="flex flex-col items-center gap-2 p-4 rounded-xl border-2 border-gray-200 dark:border-dark-600 hover:border-primary-500 dark:hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
	          >
	            <Icon name="terminal" size="xl" class="text-gray-600 dark:text-gray-400" />
	            <span class="font-medium text-gray-900 dark:text-white">{{
	              t('keys.ccsClientSelect.claudeCode')
	            }}</span>
	            <span class="text-xs text-gray-500 dark:text-gray-400">{{
	              t('keys.ccsClientSelect.claudeCodeDesc')
	            }}</span>
	          </button>
	          <button
	            @click="handleCcsClientSelect('gemini')"
	            class="flex flex-col items-center gap-2 p-4 rounded-xl border-2 border-gray-200 dark:border-dark-600 hover:border-primary-500 dark:hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
	          >
	            <Icon name="sparkles" size="xl" class="text-gray-600 dark:text-gray-400" />
	            <span class="font-medium text-gray-900 dark:text-white">{{
	              t('keys.ccsClientSelect.geminiCli')
	            }}</span>
	            <span class="text-xs text-gray-500 dark:text-gray-400">{{
	              t('keys.ccsClientSelect.geminiCliDesc')
	            }}</span>
	          </button>
	        </div>
	      </div>
      <template #footer>
        <div class="flex justify-end">
          <button @click="closeCcsClientSelect" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Group Selector Dropdown (Teleported to body to avoid overflow clipping) -->
    <Teleport to="body">
      <div
        v-if="groupSelectorKeyId !== null && dropdownPosition"
        ref="dropdownRef"
        class="animate-in fade-in slide-in-from-top-2 fixed z-[100000020] w-max max-w-[calc(100vw-16px)] overflow-hidden rounded-xl bg-white shadow-lg ring-1 ring-black/5 duration-200 sm:min-w-[380px] dark:bg-dark-800 dark:ring-white/10"
        style="pointer-events: auto !important;"
        :style="{
          top: dropdownPosition.top !== undefined ? dropdownPosition.top + 'px' : undefined,
          bottom: dropdownPosition.bottom !== undefined ? dropdownPosition.bottom + 'px' : undefined,
          left: dropdownPosition.left + 'px'
        }"
      >
        <!-- Search box -->
        <div class="border-b border-gray-100 p-2 dark:border-dark-700">
          <div class="relative">
            <svg class="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              v-model="groupSearchQuery"
              type="text"
              class="w-full rounded-lg border border-gray-200 bg-gray-50 py-1.5 pl-8 pr-3 text-sm text-gray-900 placeholder-gray-400 outline-none focus:border-primary-300 focus:ring-1 focus:ring-primary-300 dark:border-dark-600 dark:bg-dark-700 dark:text-white dark:placeholder-gray-500 dark:focus:border-primary-600 dark:focus:ring-primary-600"
              :placeholder="t('keys.searchGroup')"
              @click.stop
            />
          </div>
        </div>
        <!-- Group list -->
        <div class="max-h-80 overflow-y-auto p-1.5">
          <button
            v-for="option in filteredGroupOptions"
            :key="option.value ?? 'null'"
            @click="changeGroup(selectedKeyForGroup!, option.value)"
            :class="[
              'flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-sm transition-colors',
              'border-b border-gray-100 last:border-0 dark:border-dark-700',
              selectedKeyForGroup?.group_id === option.value ||
              (!selectedKeyForGroup?.group_id && option.value === null)
                ? 'bg-primary-50 dark:bg-primary-900/20'
                : 'hover:bg-gray-100 dark:hover:bg-dark-700'
            ]"
            :title="option.description || undefined"
          >
            <GroupOptionItem
              :name="option.label"
              :platform="option.platform"
              :subscription-type="option.subscriptionType"
              :rate-multiplier="option.rate"
              :user-rate-multiplier="option.userRate"
              :peak-rate-enabled="option.peakRateEnabled"
              :peak-start="option.peakStart"
              :peak-end="option.peakEnd"
              :peak-rate-multiplier="option.peakRateMultiplier"
              :description="option.description"
              :selected="
                selectedKeyForGroup?.group_id === option.value ||
                (!selectedKeyForGroup?.group_id && option.value === null)
              "
            />
          </button>
          <!-- Empty state when search has no results -->
          <div v-if="filteredGroupOptions.length === 0" class="py-4 text-center text-sm text-gray-400 dark:text-gray-500">
            {{ t('keys.noGroupFound') }}
          </div>
        </div>
      </div>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
	import { ref, reactive, computed, nextTick, onMounted, onUnmounted, type ComponentPublicInstance } from 'vue'
	import { useI18n } from 'vue-i18n'
	import { useAppStore } from '@/stores/app'
	import { useOnboardingStore } from '@/stores/onboarding'
	import { useClipboard } from '@/composables/useClipboard'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'

const { t } = useI18n()
import { keysAPI, authAPI, usageAPI, userGroupsAPI } from '@/api'
import { getSmartRouting, updateSmartRouting, type SmartRoutingProfile } from '@/custom/modules/smart-routing/api'
import { reorderRoutingGroups, reorderRoutingGroupsByKeyboard, type RoutingDropPlacement, type RoutingKeyboardMove } from '@/custom/modules/smart-routing/reorder'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
	import DataTable from '@/components/common/DataTable.vue'
	import Pagination from '@/components/common/Pagination.vue'
	import BaseDialog from '@/components/common/BaseDialog.vue'
	import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
	import EmptyState from '@/components/common/EmptyState.vue'
	import Select from '@/components/common/Select.vue'
	import SearchInput from '@/components/common/SearchInput.vue'
	import Icon from '@/components/icons/Icon.vue'
	import UseKeyModal from '@/components/keys/UseKeyModal.vue'
	import EndpointPopover from '@/components/keys/EndpointPopover.vue'
	import GroupBadge from '@/components/common/GroupBadge.vue'
	import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
	import type { ApiKey, Group, PublicSettings, SubscriptionType, GroupPlatform, UpdateApiKeyRequest } from '@/types'
import type { Column } from '@/components/common/types'
import type { BatchApiKeyUsageStats } from '@/api/usage'
import { formatDateTime } from '@/utils/format'
import { maskApiKey } from '@/utils/maskApiKey'
import {
  buildCcSwitchImportDeeplink,
  type CcSwitchClientType
} from '@/utils/ccswitchImport'
import { extractApiErrorMessage } from '@/utils/apiError'

// Helper to format date for datetime-local input
const formatDateTimeLocal = (isoDate: string): string => {
  const date = new Date(isoDate)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

interface GroupOption {
  value: number
  label: string
  description: string | null
  rate: number
  userRate: number | null
  peakRateEnabled: boolean
  peakStart: string
  peakEnd: string
  peakRateMultiplier: number
  subscriptionType: SubscriptionType
  platform: GroupPlatform
}

const appStore = useAppStore()
const onboardingStore = useOnboardingStore()
const { copyToClipboard: clipboardCopy } = useClipboard()

const allColumns = computed<Column[]>(() => [
  { key: 'name', label: t('common.name'), sortable: true },
  { key: 'id', label: t('keys.id'), sortable: true },
  { key: 'key', label: t('keys.apiKey'), sortable: false },
  { key: 'group', label: t('keys.group'), sortable: false },
  { key: 'routing', label: t('keys.routingListLabel'), sortable: false },
  { key: 'current_concurrency', label: t('keys.currentConcurrency'), sortable: true },
  { key: 'usage', label: t('keys.usage'), sortable: false },
  { key: 'rate_limit', label: t('keys.rateLimitColumn'), sortable: false },
  { key: 'expires_at', label: t('keys.expiresAt'), sortable: true },
  { key: 'status', label: t('common.status'), sortable: true },
  { key: 'last_used_at', label: t('keys.lastUsedAt'), sortable: true },
  { key: 'last_used_ip', label: t('keys.lastUsedIP'), sortable: false },
  { key: 'created_at', label: t('keys.created'), sortable: true },
  { key: 'actions', label: t('common.actions'), sortable: false }
])

const ALWAYS_VISIBLE_COLUMNS = new Set(['name', 'actions'])
const DEFAULT_HIDDEN_COLUMNS = ['id', 'rate_limit', 'last_used_at', 'last_used_ip']
const HIDDEN_COLUMNS_KEY = 'api-key-hidden-columns'
const COLUMN_SETTINGS_VERSION_KEY = 'api-key-column-settings-version'
const COLUMN_SETTINGS_VERSION = 3
const VERSION_NEW_HIDDEN_COLUMNS: Record<number, string[]> = {
  2: ['last_used_ip'],
  3: ['id']
}

const toggleableColumns = computed(() =>
  allColumns.value.filter((col) => !ALWAYS_VISIBLE_COLUMNS.has(col.key))
)

const hiddenColumns = reactive<Set<string>>(new Set())

const saveColumnsToStorage = () => {
  try {
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
    localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
  } catch (error) {
    console.error('Failed to save API key table columns:', error)
  }
}

const loadSavedColumns = () => {
  hiddenColumns.clear()
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    if (saved) {
      const parsed = JSON.parse(saved) as string[]
      const validColumnKeys = new Set(allColumns.value.map((col) => col.key))
      parsed
        .filter((key) =>
          typeof key === 'string' &&
          validColumnKeys.has(key) &&
          !ALWAYS_VISIBLE_COLUMNS.has(key)
        )
        .forEach((key) => hiddenColumns.add(key))
      const storedVersion = Number(localStorage.getItem(COLUMN_SETTINGS_VERSION_KEY) ?? '1')
      if (storedVersion < COLUMN_SETTINGS_VERSION) {
        for (let v = storedVersion + 1; v <= COLUMN_SETTINGS_VERSION; v++) {
          for (const key of VERSION_NEW_HIDDEN_COLUMNS[v] ?? []) {
            if (validColumnKeys.has(key) && !ALWAYS_VISIBLE_COLUMNS.has(key)) {
              hiddenColumns.add(key)
            }
          }
        }
        saveColumnsToStorage()
      } else {
        localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
      }
    } else {
      DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key))
      localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
    }
  } catch (error) {
    console.error('Failed to load API key table columns:', error)
    DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key))
  }
}

const toggleColumn = (key: string) => {
  if (ALWAYS_VISIBLE_COLUMNS.has(key)) return
  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key)
  } else {
    hiddenColumns.add(key)
  }
  saveColumnsToStorage()
}

const isColumnVisible = (key: string) => !hiddenColumns.has(key)

const columns = computed<Column[]>(() =>
  allColumns.value.filter((col) => ALWAYS_VISIBLE_COLUMNS.has(col.key) || !hiddenColumns.has(col.key))
)

const apiKeys = ref<ApiKey[]>([])

interface RoutingListSummary {
  profile: SmartRoutingProfile | null
  loading: boolean
  error: boolean
  loadedAt: number
}

const routingListSummaries = ref<Record<number, RoutingListSummary>>({})
const routingListGeneration = ref(0)
const routingSummaryCacheTtlMs = 30_000

const routingListSummaryIsSmart = (summary?: RoutingListSummary) =>
  Boolean(summary?.profile?.enabled && summary.profile.mode === 'smart')

const routingListSummaryCandidateCount = (summary?: RoutingListSummary) =>
  summary?.profile?.groups?.filter((group) => group.enabled).length ?? 0

const routingListSummaryStatusLabel = (summary?: RoutingListSummary) => {
  if (!routingListSummaryIsSmart(summary)) return t('keys.routingStatusDisabled')
  return routingListSummaryCandidateCount(summary) > 0
    ? t('keys.routingStatusActive')
    : t('keys.routingStatusNoCandidates')
}

const routingListSummaryStatusClass = (summary?: RoutingListSummary) => {
  if (!routingListSummaryIsSmart(summary)) return 'disabled'
  return routingListSummaryCandidateCount(summary) > 0 ? 'active' : 'warning'
}
const groups = ref<Group[]>([])
const loading = ref(false)
const submitting = ref(false)
const now = ref(new Date())
let resetTimer: ReturnType<typeof setInterval> | null = null
const usageStats = ref<Record<string, BatchApiKeyUsageStats>>({})
const userGroupRates = ref<Record<number, number>>({})

const pagination = ref({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})
const sortState = ref({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

// Filter state
const filterSearch = ref('')
const filterStatus = ref('')
const filterGroupId = ref<string | number>('')

const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteDialog = ref(false)
const showResetQuotaDialog = ref(false)
const showResetRateLimitDialog = ref(false)
const showUseKeyModal = ref(false)
const showCcsClientSelect = ref(false)
const showColumnDropdown = ref(false)
const pendingCcsRow = ref<ApiKey | null>(null)
const selectedKey = ref<ApiKey | null>(null)
const routingProfile = ref<SmartRoutingProfile | null>(null)
const routingLoading = ref(false)
const routingProfileLoadError = ref(false)
const routingCandidateToAdd = ref<number | null>(null)
const routingSubmitAttempted = ref(false)
const routingDraggedGroupId = ref<number | null>(null)
const routingDragOverGroupId = ref<number | null>(null)
const routingDragOverPlacement = ref<RoutingDropPlacement | null>(null)
let routingRequestGeneration = 0
// Preview calls are independent from profile loading.  A request can outlive
// the editor (or be superseded by a newer model), so only the latest request
// for the still-open key may update the preview state.
const copiedKeyId = ref<number | null>(null)
const groupSelectorKeyId = ref<number | null>(null)
const publicSettings = ref<PublicSettings | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const columnDropdownRef = ref<HTMLElement | null>(null)
const dropdownPosition = ref<{ top?: number; bottom?: number; left: number } | null>(null)
const groupButtonRefs = ref<Map<number, HTMLElement>>(new Map())
let abortController: AbortController | null = null

// Get the currently selected key for group change
const selectedKeyForGroup = computed(() => {
  if (groupSelectorKeyId.value === null) return null
  return apiKeys.value.find((k) => k.id === groupSelectorKeyId.value) || null
})

const setGroupButtonRef = (keyId: number, el: Element | ComponentPublicInstance | null) => {
  if (el instanceof HTMLElement) {
    groupButtonRefs.value.set(keyId, el)
  } else {
    groupButtonRefs.value.delete(keyId)
  }
}

const formData = ref({
  name: '',
  group_id: null as number | null,
  status: 'active' as 'active' | 'inactive',
  use_custom_key: false,
  custom_key: '',
  enable_ip_restriction: false,
  ip_whitelist: '',
  ip_blacklist: '',
  // Quota settings (empty = unlimited)
  enable_quota: false,
  quota: null as number | null,
  // Rate limit settings
  enable_rate_limit: false,
  rate_limit_5h: null as number | null,
  rate_limit_1d: null as number | null,
  rate_limit_7d: null as number | null,
  enable_expiration: false,
  expiration_preset: '30' as '7' | '30' | '90' | 'custom',
  expiration_date: ''
  ,routing_mode: 'single' as 'single' | 'smart'
  ,routing_enabled: false
  ,routing_strategy: 'auto' as SmartRoutingProfile['strategy']
  ,routing_price_weight: 40
  ,routing_speed_weight: 30
  ,routing_success_weight: 30
  ,routing_group_ids: [] as number[]
})

type RoutingWeightKey = 'routing_price_weight' | 'routing_speed_weight' | 'routing_success_weight'
type RoutingStrategy = SmartRoutingProfile['strategy']

const routingWeightFields: Array<{ key: RoutingWeightKey; label: string }> = [
  { key: 'routing_price_weight', label: 'keys.priceWeight' },
  { key: 'routing_speed_weight', label: 'keys.speedWeight' },
  { key: 'routing_success_weight', label: 'keys.successWeight' }
]

const routingWeightValue = (value: number | null | undefined) => {
  const numeric = Number(value)
  return Number.isFinite(numeric) ? numeric : 0
}

const routingWeightTotal = computed(() =>
  routingWeightValue(formData.value.routing_price_weight) +
  routingWeightValue(formData.value.routing_speed_weight) +
  routingWeightValue(formData.value.routing_success_weight)
)

const routingWeightsValid = computed(() => {
  const weights = routingWeightFields.map(({ key }) => routingWeightValue(formData.value[key]))
  return weights.every((weight) => Number.isInteger(weight) && weight >= 0 && weight <= 100) && routingWeightTotal.value === 100
})

const primaryRoutingGroup = computed(() => groups.value.find((group) => group.id === formData.value.group_id) || null)

const isRoutingCompatibleGroup = (group: Group) => {
  const primary = primaryRoutingGroup.value
  return Boolean(
    primary &&
    group.id !== primary.id &&
    group.status === 'active' &&
    group.platform !== 'composite' &&
    primary.platform !== 'composite' &&
    group.platform === primary.platform &&
    group.subscription_type === primary.subscription_type
  )
}

interface RoutingGroupOption {
  [key: string]: unknown
  value: number
  label: string
  description: string
  platform: GroupPlatform
  subscriptionType: SubscriptionType
  rate: number
}

const routingGroupOptions = computed<RoutingGroupOption[]>(() =>
  groups.value
    .filter(isRoutingCompatibleGroup)
    .map((group) => ({
      value: group.id,
      label: group.name,
      description: `${group.platform} · ${group.subscription_type} · ${group.rate_multiplier}x`,
      platform: group.platform,
      subscriptionType: group.subscription_type,
      rate: group.rate_multiplier
    }))
)

const availableRoutingGroupOptions = computed(() =>
  routingGroupOptions.value.filter((option) => !formData.value.routing_group_ids.includes(option.value))
)

const selectedRoutingGroups = computed(() =>
  formData.value.routing_group_ids.map((groupId) => {
    const option = routingGroupOptions.value.find((candidate) => candidate.value === groupId)
    const group = groups.value.find((candidate) => candidate.id === groupId)
    return {
      id: groupId,
      label: option?.label || group?.name || `#${groupId}`,
      description: option?.description || (group ? `${group.platform} · ${group.subscription_type} · ${group.rate_multiplier}x` : t('keys.routingCandidateUnavailable')),
      eligible: Boolean(option)
    }
  })
)

const routingCandidateValidationError = computed(() => {
  if (routingLoading.value) return ''
  if (!formData.value.routing_group_ids.length) return t('keys.routingCandidatesRequired')
  if (formData.value.routing_group_ids.length > 8) return t('keys.routingCandidatesLimit')
  if (selectedRoutingGroups.value.some((group) => !group.eligible)) return t('keys.routingCandidateInvalid')
  return ''
})

const routingStrategies = computed(() => [
  { value: 'auto' as const, label: t('keys.strategyAuto'), description: t('keys.strategyAutoDesc'), weightSummary: `${t('keys.priceWeight')} 40% · ${t('keys.speedWeight')} 30% · ${t('keys.successWeight')} 30%`, icon: 'sparkles', tone: 'teal' },
  { value: 'price' as const, label: t('keys.strategyPrice'), description: t('keys.strategyPriceDesc'), weightSummary: `${t('keys.priceWeight')} 100% · ${t('keys.speedWeight')} 0% · ${t('keys.successWeight')} 0%`, icon: 'dollar', tone: 'green' },
  { value: 'speed' as const, label: t('keys.strategySpeed'), description: t('keys.strategySpeedDesc'), weightSummary: `${t('keys.priceWeight')} 15% · ${t('keys.speedWeight')} 70% · ${t('keys.successWeight')} 15%`, icon: 'bolt', tone: 'amber' },
  { value: 'success' as const, label: t('keys.strategySuccess'), description: t('keys.strategySuccessDesc'), weightSummary: `${t('keys.priceWeight')} 15% · ${t('keys.speedWeight')} 15% · ${t('keys.successWeight')} 70%`, icon: 'shield', tone: 'rose' },
  { value: 'custom' as const, label: t('keys.strategyCustom'), description: t('keys.strategyCustomDesc'), weightSummary: `${t('keys.priceWeight')} ${routingWeightValue(formData.value.routing_price_weight)}% · ${t('keys.speedWeight')} ${routingWeightValue(formData.value.routing_speed_weight)}% · ${t('keys.successWeight')} ${routingWeightValue(formData.value.routing_success_weight)}%`, icon: 'chart', tone: 'violet' }
] as const)

const routingStrategyOptions = computed(() => routingStrategies.value.map(({ value, label }) => ({ value, label })))

const selectRoutingStrategy = (strategy: RoutingStrategy) => {
  formData.value.routing_strategy = strategy
  if (strategy === 'auto') {
    formData.value.routing_price_weight = 40
    formData.value.routing_speed_weight = 30
    formData.value.routing_success_weight = 30
  } else if (strategy === 'price') {
    formData.value.routing_price_weight = 100
    formData.value.routing_speed_weight = 0
    formData.value.routing_success_weight = 0
  } else if (strategy === 'speed') {
    formData.value.routing_price_weight = 15
    formData.value.routing_speed_weight = 70
    formData.value.routing_success_weight = 15
  } else if (strategy === 'success') {
    formData.value.routing_price_weight = 15
    formData.value.routing_speed_weight = 15
    formData.value.routing_success_weight = 70
  }
}

const routingStrategyValues: RoutingStrategy[] = ['auto', 'price', 'speed', 'success', 'custom']

const handleRoutingStrategySelect = (value: unknown) => {
  if (typeof value !== 'string' || !routingStrategyValues.includes(value as RoutingStrategy)) return
  selectRoutingStrategy(value as RoutingStrategy)
}

const handleMobileWeightFocus = () => {
  if (formData.value.routing_strategy !== 'custom') {
    selectRoutingStrategy('custom')
  }
}

const focusRoutingStrategy = (strategy: RoutingStrategy) => {
  void nextTick(() => {
    document.querySelector<HTMLElement>(`[data-routing-strategy="${strategy}"]`)?.focus()
  })
}

const handleRoutingStrategyKeydown = (event: KeyboardEvent, current: RoutingStrategy) => {
  if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const currentIndex = routingStrategyValues.indexOf(current)
  const previous = event.key === 'ArrowLeft' || event.key === 'ArrowUp'
  const nextIndex = event.key === 'Home'
    ? 0
    : event.key === 'End'
      ? routingStrategyValues.length - 1
      : (currentIndex + (previous ? -1 : 1) + routingStrategyValues.length) % routingStrategyValues.length
  const nextStrategy = routingStrategyValues[nextIndex]
  selectRoutingStrategy(nextStrategy)
  focusRoutingStrategy(nextStrategy)
}

const routingModeValues = ['single', 'smart'] as const

const focusRoutingMode = (mode: (typeof routingModeValues)[number]) => {
  void nextTick(() => {
    document.querySelector<HTMLElement>(`[data-routing-mode="${mode}"]`)?.focus()
  })
}

const handleRoutingModeKeydown = (event: KeyboardEvent, current: (typeof routingModeValues)[number]) => {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const currentIndex = routingModeValues.indexOf(current)
  const nextIndex = event.key === 'Home'
    ? 0
    : event.key === 'End'
      ? routingModeValues.length - 1
      : (currentIndex + (event.key === 'ArrowLeft' ? -1 : 1) + routingModeValues.length) % routingModeValues.length
  const nextMode = routingModeValues[nextIndex]
  formData.value.routing_mode = nextMode
  focusRoutingMode(nextMode)
}

const addRoutingGroup = (value?: number | string | boolean | null) => {
  const rawGroupId = value ?? routingCandidateToAdd.value
  if (typeof rawGroupId === 'boolean') {
    routingCandidateToAdd.value = null
    return
  }
  const groupId = rawGroupId === null || rawGroupId === undefined || rawGroupId === '' ? null : Number(rawGroupId)
  if (groupId !== null && Number.isInteger(groupId) && availableRoutingGroupOptions.value.some((option) => option.value === groupId) && formData.value.routing_group_ids.length < 8) {
    formData.value.routing_group_ids.push(groupId)
  }
  routingCandidateToAdd.value = null
}

const removeRoutingGroup = (groupId: number) => {
  formData.value.routing_group_ids = formData.value.routing_group_ids.filter((id) => id !== groupId)
}

const moveRoutingGroup = (groupId: number, direction: -1 | 1) => {
  if (routingLoading.value) return
  const currentIndex = formData.value.routing_group_ids.indexOf(groupId)
  const nextIndex = currentIndex + direction
  if (currentIndex < 0 || nextIndex < 0 || nextIndex >= formData.value.routing_group_ids.length) return
  const targetGroupId = formData.value.routing_group_ids[nextIndex]
  formData.value.routing_group_ids = reorderRoutingGroups(
    formData.value.routing_group_ids,
    groupId,
    targetGroupId,
    direction === 1 ? 'after' : 'before'
  )
}

const handleRoutingCandidateKeydown = (event: KeyboardEvent, groupId: number) => {
  if (routingLoading.value || !['ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const movementByKey: Record<string, RoutingKeyboardMove> = {
    ArrowUp: 'up',
    ArrowDown: 'down',
    Home: 'home',
    End: 'end'
  }
  formData.value.routing_group_ids = reorderRoutingGroupsByKeyboard(
    formData.value.routing_group_ids,
    groupId,
    movementByKey[event.key]
  )
}

const startRoutingGroupDrag = (groupId: number, event: DragEvent) => {
  if (routingLoading.value) {
    event.preventDefault()
    return
  }
  routingDraggedGroupId.value = groupId
  routingDragOverGroupId.value = null
  routingDragOverPlacement.value = null
  event.dataTransfer?.setData('text/plain', String(groupId))
  if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move'
}

const handleRoutingGroupDragOver = (targetGroupId: number, event: DragEvent) => {
  if (routingLoading.value || routingDraggedGroupId.value === null || routingDraggedGroupId.value === targetGroupId) return
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'move'
  const target = event.currentTarget as HTMLElement | null
  const placement: RoutingDropPlacement = target && event.clientY > target.getBoundingClientRect().top + target.getBoundingClientRect().height / 2
    ? 'after'
    : 'before'
  routingDragOverGroupId.value = targetGroupId
  routingDragOverPlacement.value = placement
}

const handleRoutingGroupDragLeave = (targetGroupId: number, event: DragEvent) => {
  if (routingDragOverGroupId.value !== targetGroupId) return
  const target = event.currentTarget as HTMLElement | null
  const related = event.relatedTarget as Node | null
  if (target && related && target.contains(related)) return
  routingDragOverGroupId.value = null
  routingDragOverPlacement.value = null
}

const dropRoutingGroup = (targetGroupId: number) => {
  const draggedGroupId = routingDraggedGroupId.value
  const placement = routingDragOverPlacement.value || 'before'
  clearRoutingGroupDrag()
  if (routingLoading.value || draggedGroupId === null || draggedGroupId === targetGroupId) return
  formData.value.routing_group_ids = reorderRoutingGroups(
    formData.value.routing_group_ids,
    draggedGroupId,
    targetGroupId,
    placement
  )
}

const clearRoutingGroupDrag = () => {
  routingDraggedGroupId.value = null
  routingDragOverGroupId.value = null
  routingDragOverPlacement.value = null
}

const loadRoutingProfile = async (keyId: number) => {
  const generation = ++routingRequestGeneration
  routingLoading.value = true
  routingProfileLoadError.value = false
  try {
    const profile = await getSmartRouting(keyId)
    if (generation !== routingRequestGeneration || !showEditModal.value || selectedKey.value?.id !== keyId) return
    const profileGroups = Array.isArray(profile.groups) ? profile.groups : []
    routingProfile.value = { ...profile, groups: profileGroups }
    // A persisted smart profile can be intentionally disabled by operations.
    // Do not turn it back on merely because a user edits an unrelated key field.
    formData.value.routing_mode = profile.enabled && profile.mode === 'smart' ? 'smart' : 'single'
    formData.value.routing_enabled = profile.enabled && profile.mode === 'smart'
    formData.value.routing_strategy = profile.strategy
    formData.value.routing_price_weight = profile.price_weight
    formData.value.routing_speed_weight = profile.speed_weight
    formData.value.routing_success_weight = profile.success_weight
    formData.value.routing_group_ids = profileGroups.filter((group) => group.enabled).map((group) => group.group_id)
  } catch {
    if (generation === routingRequestGeneration && showEditModal.value && selectedKey.value?.id === keyId) {
      routingProfile.value = null
      routingProfileLoadError.value = true
    }
  } finally {
    if (generation === routingRequestGeneration) routingLoading.value = false
  }
}

const retryRoutingProfile = (keyId: number) => {
  if (routingLoading.value) return
  routingProfileLoadError.value = false
  void loadRoutingProfile(keyId)
}

// 自定义Key验证
const customKeyError = computed(() => {
  if (!formData.value.use_custom_key || !formData.value.custom_key) {
    return ''
  }
  const key = formData.value.custom_key
  if (key.length < 16) {
    return t('keys.customKeyTooShort')
  }
  // 检查字符：只允许字母、数字、下划线、连字符
  if (!/^[a-zA-Z0-9_-]+$/.test(key)) {
    return t('keys.customKeyInvalidChars')
  }
  return ''
})

const statusOptions = computed(() => [
  { value: 'active', label: t('common.active') },
  { value: 'inactive', label: t('common.inactive') }
])

const shouldSubmitEditStatus = (key: ApiKey, status: 'active' | 'inactive') => {
  if (key.status === 'quota_exhausted' || key.status === 'expired') {
    return status === 'active'
  }
  return true
}

// Filter dropdown options
const groupFilterOptions = computed(() => [
  { value: '', label: t('keys.allGroups') },
  { value: 0, label: t('keys.noGroup') },
  ...groups.value.map((g) => ({ value: g.id, label: g.name }))
])

const statusFilterOptions = computed(() => [
  { value: '', label: t('keys.allStatus') },
  { value: 'active', label: t('keys.status.active') },
  { value: 'inactive', label: t('keys.status.inactive') },
  { value: 'quota_exhausted', label: t('keys.status.quota_exhausted') },
  { value: 'expired', label: t('keys.status.expired') }
])

const onFilterChange = () => {
  pagination.value.page = 1
  loadApiKeys()
}

const onGroupFilterChange = (value: string | number | boolean | null) => {
  filterGroupId.value = value as string | number
  onFilterChange()
}

const onStatusFilterChange = (value: string | number | boolean | null) => {
  filterStatus.value = value as string
  onFilterChange()
}

// Convert groups to Select options format with rate multiplier and subscription type
const groupOptions = computed(() =>
  groups.value.map((group) => ({
    value: group.id,
    label: group.name,
    description: group.description,
    rate: group.rate_multiplier,
    userRate: userGroupRates.value[group.id] ?? null,
    peakRateEnabled: group.peak_rate_enabled,
    peakStart: group.peak_start,
    peakEnd: group.peak_end,
    peakRateMultiplier: group.peak_rate_multiplier,
    subscriptionType: group.subscription_type,
    platform: group.platform
  }))
)

// Group dropdown search
const groupSearchQuery = ref('')
const filteredGroupOptions = computed(() => {
  const query = groupSearchQuery.value.trim().toLowerCase()
  if (!query) return groupOptions.value
  return groupOptions.value.filter((opt) => {
    return opt.label.toLowerCase().includes(query) ||
      (opt.description && opt.description.toLowerCase().includes(query))
  })
})

const copyToClipboard = async (text: string, keyId: number) => {
  const success = await clipboardCopy(text, t('keys.copied'))
  if (success) {
    copiedKeyId.value = keyId
    setTimeout(() => {
      copiedKeyId.value = null
    }, 800)
  }
}

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== 'object') return false
  const { name, code } = error as { name?: string; code?: string }
  return name === 'AbortError' || code === 'ERR_CANCELED'
}

const loadRoutingListSummaries = (items: ApiKey[]) => {
  const generation = ++routingListGeneration.value
  const now = Date.now()
  const previous = routingListSummaries.value
  const next: Record<number, RoutingListSummary> = {}

  for (const key of items) {
    const cached = previous[key.id]
    if (cached && !cached.error && now - cached.loadedAt < routingSummaryCacheTtlMs) {
      next[key.id] = cached
    } else {
      next[key.id] = {
        profile: null,
        loading: true,
        error: false,
        loadedAt: cached?.loadedAt ?? 0
      }
    }
  }
  routingListSummaries.value = next

  // Route profiles are an optional overlay and must not delay the API-key list.
  void Promise.all(items.map(async (key) => {
    const cached = previous[key.id]
    if (cached && !cached.error && now - cached.loadedAt < routingSummaryCacheTtlMs) return
    try {
      const profile = await getSmartRouting(key.id)
      if (generation !== routingListGeneration.value) return
      routingListSummaries.value = {
        ...routingListSummaries.value,
        [key.id]: {
          profile: { ...profile, groups: Array.isArray(profile.groups) ? profile.groups : [] },
          loading: false,
          error: false,
          loadedAt: Date.now()
        }
      }
    } catch {
      if (generation !== routingListGeneration.value) return
      routingListSummaries.value = {
        ...routingListSummaries.value,
        [key.id]: {
          profile: null,
          loading: false,
          error: true,
          loadedAt: Date.now()
        }
      }
    }
  }))
}

// Keep the list summary visible while an editor is merely opened or closed.
// Invalidate only after a routing-affecting write so unrelated rows do not
// briefly fall through to the misleading single-group display.
const invalidateRoutingListSummary = (keyId: number | null | undefined) => {
  if (!keyId) return
  const current = routingListSummaries.value[keyId]
  if (!current) return
  routingListSummaries.value = {
    ...routingListSummaries.value,
    [keyId]: { ...current, loading: true, loadedAt: 0 }
  }
}

const loadApiKeys = async () => {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  const { signal } = controller
  loading.value = true
  try {
    // Build filters
    const filters: {
      search?: string
      status?: string
      group_id?: number | string
      sort_by?: string
      sort_order?: 'asc' | 'desc'
    } = {}
    if (filterSearch.value) filters.search = filterSearch.value
    if (filterStatus.value) filters.status = filterStatus.value
    if (filterGroupId.value !== '') filters.group_id = filterGroupId.value
    filters.sort_by = sortState.value.sort_by
    filters.sort_order = sortState.value.sort_order

    const response = await keysAPI.list(pagination.value.page, pagination.value.page_size, filters, {
      signal
    })
    if (signal.aborted) return
    apiKeys.value = response.items
    pagination.value.total = response.total
    pagination.value.pages = response.pages
    loadRoutingListSummaries(response.items)

    // Load usage stats for all API keys in the list
    if (response.items.length > 0) {
      const keyIds = response.items.map((k) => k.id)
      try {
        const usageResponse = await usageAPI.getDashboardApiKeysUsage(keyIds, { signal })
        if (signal.aborted) return
        usageStats.value = usageResponse.stats
      } catch (e) {
        if (!isAbortError(e)) {
          console.error('Failed to load usage stats:', e)
        }
      }
    }
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    appStore.showError(t('keys.failedToLoad'))
  } finally {
    if (abortController === controller) {
      loading.value = false
    }
  }
}

const loadGroups = async () => {
  try {
    groups.value = await userGroupsAPI.getAvailable()
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

const loadUserGroupRates = async () => {
  try {
    userGroupRates.value = await userGroupsAPI.getUserGroupRates()
  } catch (error) {
    console.error('Failed to load user group rates:', error)
  }
}

const loadPublicSettings = async () => {
  try {
    publicSettings.value = await authAPI.getPublicSettings()
  } catch (error) {
    console.error('Failed to load public settings:', error)
  }
}

const openUseKeyModal = (key: ApiKey) => {
  selectedKey.value = key
  showUseKeyModal.value = true
}

const closeUseKeyModal = () => {
  showUseKeyModal.value = false
  selectedKey.value = null
}

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadApiKeys()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.page_size = pageSize
  pagination.value.page = 1
  loadApiKeys()
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  sortState.value.sort_by = key
  sortState.value.sort_order = order
  pagination.value.page = 1
  loadApiKeys()
}

const editKey = (key: ApiKey) => {
  routingProfile.value = null
  routingProfileLoadError.value = false
  routingCandidateToAdd.value = null
  routingSubmitAttempted.value = false
  clearRoutingGroupDrag()
  selectedKey.value = key
  const hasIPRestriction = (key.ip_whitelist?.length > 0) || (key.ip_blacklist?.length > 0)
  const hasExpiration = !!key.expires_at
  formData.value = {
    name: key.name,
    group_id: key.group_id,
    status: key.status === 'quota_exhausted' || key.status === 'expired' ? 'inactive' : key.status,
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: hasIPRestriction,
    ip_whitelist: (key.ip_whitelist || []).join('\n'),
    ip_blacklist: (key.ip_blacklist || []).join('\n'),
    enable_quota: key.quota > 0,
    quota: key.quota > 0 ? key.quota : null,
    enable_rate_limit: (key.rate_limit_5h > 0) || (key.rate_limit_1d > 0) || (key.rate_limit_7d > 0),
    rate_limit_5h: key.rate_limit_5h || null,
    rate_limit_1d: key.rate_limit_1d || null,
    rate_limit_7d: key.rate_limit_7d || null,
    enable_expiration: hasExpiration,
    expiration_preset: 'custom',
    expiration_date: key.expires_at ? formatDateTimeLocal(key.expires_at) : '',
    routing_mode: 'single',
    routing_enabled: false,
    routing_strategy: 'auto',
    routing_price_weight: 40,
    routing_speed_weight: 30,
    routing_success_weight: 30,
    routing_group_ids: []
  }
  showEditModal.value = true
  void loadRoutingProfile(key.id)
}

const toggleKeyStatus = async (key: ApiKey) => {
  const newStatus = key.status === 'active' ? 'inactive' : 'active'
  try {
    await keysAPI.toggleStatus(key.id, newStatus)
    appStore.showSuccess(
      newStatus === 'active' ? t('keys.keyEnabledSuccess') : t('keys.keyDisabledSuccess')
    )
    loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToUpdateStatus'))
  }
}

const openGroupSelector = (key: ApiKey) => {
  if (groupSelectorKeyId.value === key.id) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  } else {
    const buttonEl = groupButtonRefs.value.get(key.id)
    if (buttonEl) {
      const rect = buttonEl.getBoundingClientRect()
      const dropdownEstHeight = 400 // estimated max dropdown height
      const dropdownEstWidth = Math.min(380, window.innerWidth - 16)
      const spaceBelow = window.innerHeight - rect.bottom
      const spaceAbove = rect.top
      // 夹取 left，避免窄屏下浮层超出视口右缘
      const left = Math.max(8, Math.min(rect.left, window.innerWidth - dropdownEstWidth - 8))

      if (spaceBelow < dropdownEstHeight && spaceAbove > spaceBelow) {
        // Not enough space below, pop upward
        dropdownPosition.value = {
          bottom: window.innerHeight - rect.top + 4,
          left
        }
      } else {
        // Default: pop downward
        dropdownPosition.value = {
          top: rect.bottom + 4,
          left
        }
      }
    }
    groupSelectorKeyId.value = key.id
    groupSearchQuery.value = ''
  }
}

const changeGroup = async (key: ApiKey, newGroupId: number | null) => {
  groupSelectorKeyId.value = null
  dropdownPosition.value = null
  if (key.group_id === newGroupId) return

  let previousRouting: SmartRoutingProfile | null = null
  let safeRouting: SmartRoutingProfile | null = null
  let groupUpdateSucceeded = false
  try {
    // The inline group picker bypasses the edit modal. Disable an active smart
    // profile first so candidates validated against the old primary group can
    // never be used while the key's primary group is changing.
    previousRouting = await getSmartRouting(key.id)
    const routingActive = previousRouting.enabled && previousRouting.mode === 'smart'
    if (routingActive) {
      safeRouting = await updateSmartRouting(key.id, {
        version: previousRouting.version,
        enabled: false,
        mode: 'single',
        strategy: 'auto',
        price_weight: 40,
        speed_weight: 30,
        success_weight: 30,
        sticky_ttl_seconds: previousRouting.sticky_ttl_seconds,
        exploration_rate: previousRouting.exploration_rate,
        group_ids: []
      })
    }
    await keysAPI.update(key.id, { group_id: newGroupId })
    groupUpdateSucceeded = true
    appStore.showSuccess(t('keys.groupChangedSuccess'))
    invalidateRoutingListSummary(key.id)
    loadApiKeys()
  } catch (error: any) {
    // A timeout/error from the primary-group API is ambiguous: the database
    // write may have committed even when the client did not receive a
    // response. Restore the old routing profile only after a fresh key read
    // proves that the primary group is still unchanged. Otherwise keep the
    // profile disabled; re-enabling candidates validated against the old
    // primary could route requests to an incompatible group.
    let primaryGroupConfirmedUnchanged = false
    if (!groupUpdateSucceeded && safeRouting && previousRouting) {
      try {
        const currentKey = await keysAPI.getById(key.id)
        primaryGroupConfirmedUnchanged = currentKey.group_id === key.group_id
      } catch {
        // An unavailable read cannot establish that the write rolled back.
      }
      if (primaryGroupConfirmedUnchanged) {
        try {
          await updateSmartRouting(key.id, {
            version: safeRouting.version,
            enabled: previousRouting.enabled,
            mode: previousRouting.mode,
            strategy: previousRouting.strategy,
            price_weight: previousRouting.price_weight,
            speed_weight: previousRouting.speed_weight,
            success_weight: previousRouting.success_weight,
            sticky_ttl_seconds: previousRouting.sticky_ttl_seconds,
            exploration_rate: previousRouting.exploration_rate,
            group_ids: previousRouting.groups.filter(group => group.enabled).map(group => group.group_id)
          })
        } catch {
          // Keep the original error visible; single-group fail-closed is safer.
        }
      }
    }
    const errorMsg = extractApiErrorMessage(error, '') || t('keys.failedToChangeGroup')
    appStore.showError(errorMsg)
  }
}

const closeGroupSelector = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  // Check if click is inside the dropdown or the trigger button
  if (!target.closest('.group\\/dropdown') && !dropdownRef.value?.contains(target)) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  }
  if (columnDropdownRef.value && !columnDropdownRef.value.contains(target)) {
    showColumnDropdown.value = false
  }
}

const confirmDelete = (key: ApiKey) => {
  selectedKey.value = key
  showDeleteDialog.value = true
}

const handleSubmit = async () => {
  routingSubmitAttempted.value = true
  // Validate group_id is required
  if (formData.value.group_id === null) {
    appStore.showError(t('keys.groupRequired'))
    return
  }
  if (showEditModal.value && routingProfileLoadError.value) {
    appStore.showError(t('keys.routingLoadFailed'))
    return
  }
  const primaryGroupChanged = Boolean(
    showEditModal.value &&
    selectedKey.value &&
    selectedKey.value.group_id !== formData.value.group_id
  )
  if (formData.value.routing_mode === 'smart') {
    if (!routingWeightsValid.value && !primaryGroupChanged) {
      appStore.showError(t('keys.routingWeightInvalid'))
      return
    }
    if (routingCandidateValidationError.value && !primaryGroupChanged) {
      appStore.showError(routingCandidateValidationError.value)
      return
    }
  }

  // Validate custom key if enabled
  if (!showEditModal.value && formData.value.use_custom_key) {
    if (!formData.value.custom_key) {
      appStore.showError(t('keys.customKeyRequired'))
      return
    }
    if (customKeyError.value) {
      appStore.showError(customKeyError.value)
      return
    }
  }

  // Parse IP lists only if IP restriction is enabled
  const parseIPList = (text: string): string[] =>
    text.split('\n').map(ip => ip.trim()).filter(ip => ip.length > 0)
  const ipWhitelist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_whitelist) : []
  const ipBlacklist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_blacklist) : []

  // Calculate quota value (null/empty/0 = unlimited, stored as 0)
  const quota = formData.value.quota && formData.value.quota > 0 ? formData.value.quota : 0

  // Calculate expiration
  let expiresInDays: number | undefined
  let expiresAt: string | null | undefined
  if (formData.value.enable_expiration && formData.value.expiration_date) {
    if (!showEditModal.value) {
      // Create mode: calculate days from date
      const expDate = new Date(formData.value.expiration_date)
      const now = new Date()
      const diffDays = Math.ceil((expDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
      expiresInDays = diffDays > 0 ? diffDays : 1
    } else {
      // Edit mode: use custom date directly
      expiresAt = new Date(formData.value.expiration_date).toISOString()
    }
  } else if (showEditModal.value) {
    // Edit mode: if expiration disabled or date cleared, send empty string to clear
    expiresAt = ''
  }

  // Calculate rate limit values (send 0 when toggle is off)
  const rateLimitData = formData.value.enable_rate_limit ? {
    rate_limit_5h: formData.value.rate_limit_5h && formData.value.rate_limit_5h > 0 ? formData.value.rate_limit_5h : 0,
    rate_limit_1d: formData.value.rate_limit_1d && formData.value.rate_limit_1d > 0 ? formData.value.rate_limit_1d : 0,
    rate_limit_7d: formData.value.rate_limit_7d && formData.value.rate_limit_7d > 0 ? formData.value.rate_limit_7d : 0,
  } : { rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0 }

  submitting.value = true
  let createdKeyId: number | null = null
  let editedKeyGroupChanged = false
  let editedKeyUpdateSucceeded = false
  try {
    if (showEditModal.value && selectedKey.value) {
      const groupChanged = selectedKey.value.group_id !== formData.value.group_id
      if (formData.value.routing_mode === 'smart') {
        if (!routingWeightsValid.value && !groupChanged) throw new Error(t('keys.routingWeightInvalid'))
        if (routingCandidateValidationError.value && !groupChanged) throw new Error(routingCandidateValidationError.value)
      }
      const updates: UpdateApiKeyRequest = {
        name: formData.value.name,
        group_id: formData.value.group_id,
        ip_whitelist: ipWhitelist,
        ip_blacklist: ipBlacklist,
        quota: quota,
        expires_at: expiresAt,
        rate_limit_5h: rateLimitData.rate_limit_5h,
        rate_limit_1d: rateLimitData.rate_limit_1d,
        rate_limit_7d: rateLimitData.rate_limit_7d,
      }
      if (shouldSubmitEditStatus(selectedKey.value, formData.value.status)) {
        updates.status = formData.value.status
      }
      const routingFallbackToSingle = formData.value.routing_mode === 'smart' &&
        (!routingWeightsValid.value || Boolean(routingCandidateValidationError.value))
      editedKeyGroupChanged = groupChanged
      const previousRouting = routingProfile.value
      let safeRoutingProfile: SmartRoutingProfile | null = null
      if (groupChanged) {
        // Disable routing before changing the primary group. This prevents a
        // transient request from using candidates validated against the old
        // primary group while the two user-facing APIs are being updated.
        safeRoutingProfile = await updateSmartRouting(selectedKey.value.id, {
          version: previousRouting?.version ?? 0,
          enabled: false,
          mode: 'single',
          strategy: 'auto',
          price_weight: 40,
          speed_weight: 30,
          success_weight: 30,
          sticky_ttl_seconds: 0,
          exploration_rate: 0.05,
          group_ids: []
        })
      }
      try {
        await keysAPI.update(selectedKey.value.id, updates)
        editedKeyUpdateSucceeded = true
      } catch (error) {
        // A failed response is not proof that the primary-group update was
        // rolled back. Only restore the old routing profile after a fresh
        // read confirms the key still has its old primary group; otherwise
        // leave the already-disabled profile in place.
        let primaryGroupConfirmedUnchanged = false
        if (safeRoutingProfile && previousRouting) {
          try {
            const currentKey = await keysAPI.getById(selectedKey.value.id)
            primaryGroupConfirmedUnchanged = currentKey.group_id === selectedKey.value.group_id
          } catch {
            // An unavailable read cannot establish that the write rolled back.
          }
        }
        if (primaryGroupConfirmedUnchanged && safeRoutingProfile && previousRouting) {
          await updateSmartRouting(selectedKey.value.id, {
            version: safeRoutingProfile.version,
            enabled: previousRouting.enabled,
            mode: previousRouting.mode,
            strategy: previousRouting.strategy,
            price_weight: previousRouting.price_weight,
            speed_weight: previousRouting.speed_weight,
            success_weight: previousRouting.success_weight,
            sticky_ttl_seconds: previousRouting.sticky_ttl_seconds,
            exploration_rate: previousRouting.exploration_rate,
            group_ids: previousRouting.groups.filter(group => group.enabled).map(group => group.group_id)
          })
        }
        throw error
      }
      await updateSmartRouting(selectedKey.value.id, {
        version: (safeRoutingProfile || routingProfile.value)?.version ?? 0,
        enabled: formData.value.routing_mode === 'smart' && !routingFallbackToSingle,
        mode: formData.value.routing_mode === 'smart' && !routingFallbackToSingle ? 'smart' : 'single',
        strategy: formData.value.routing_mode === 'smart' && !routingFallbackToSingle ? formData.value.routing_strategy : 'auto',
        price_weight: formData.value.routing_mode === 'smart' && !routingFallbackToSingle ? formData.value.routing_price_weight : 40,
        speed_weight: formData.value.routing_mode === 'smart' && !routingFallbackToSingle ? formData.value.routing_speed_weight : 30,
        success_weight: formData.value.routing_mode === 'smart' && !routingFallbackToSingle ? formData.value.routing_success_weight : 30,
        sticky_ttl_seconds: 0,
        exploration_rate: routingProfile.value?.exploration_rate ?? 0.05,
        group_ids: formData.value.routing_mode === 'smart' && !routingFallbackToSingle ? formData.value.routing_group_ids : []
      })
      if (routingFallbackToSingle) appStore.showWarning(t('keys.routingResetAfterGroupChange'))
      appStore.showSuccess(t('keys.keyUpdatedSuccess'))
    } else {
      const customKey = formData.value.use_custom_key ? formData.value.custom_key : undefined
      const createdKey = await keysAPI.create(
        formData.value.name,
        formData.value.group_id,
        customKey,
        ipWhitelist,
        ipBlacklist,
        quota,
        expiresInDays,
        rateLimitData
      )
      createdKeyId = createdKey.id
      if (formData.value.routing_mode === 'smart') {
        if (!routingWeightsValid.value) throw new Error(t('keys.routingWeightInvalid'))
        if (routingCandidateValidationError.value) throw new Error(routingCandidateValidationError.value)
        await updateSmartRouting(createdKey.id, {
          version: 0,
          enabled: true,
          mode: 'smart',
          strategy: formData.value.routing_strategy,
          price_weight: formData.value.routing_price_weight,
          speed_weight: formData.value.routing_speed_weight,
          success_weight: formData.value.routing_success_weight,
          sticky_ttl_seconds: 0,
          exploration_rate: 0.05,
          group_ids: formData.value.routing_group_ids
        })
      }
      appStore.showSuccess(t('keys.keyCreatedSuccess'))
      // Only advance tour if active, on submit step, and creation succeeded
      if (onboardingStore.isCurrentStep('[data-tour="key-form-submit"]')) {
        onboardingStore.nextStep(500)
      }
    }
    invalidateRoutingListSummary(showEditModal.value ? selectedKey.value?.id : createdKeyId)
    closeModals()
    loadApiKeys()
  } catch (error: any) {
    // The key and routing profile are persisted by separate APIs. If the
    // primary group changed but the follow-up routing write failed (including
    // an indeterminate timeout), force a conservative single-group profile so
    // candidates validated against the old primary can never remain active.
    if (editedKeyGroupChanged && editedKeyUpdateSucceeded && selectedKey.value) {
      try {
        const rollbackProfile = await getSmartRouting(selectedKey.value.id).catch(() => null)
        await updateSmartRouting(selectedKey.value.id, {
          version: rollbackProfile?.version ?? 0,
          enabled: false,
          mode: 'single',
          strategy: 'auto',
          price_weight: 40,
          speed_weight: 30,
          success_weight: 30,
          sticky_ttl_seconds: 0,
          exploration_rate: 0.05,
          group_ids: []
        })
      } catch {
        // Keep the original error visible. The backend also fail-opens to the
        // primary group when routing configuration cannot be read.
      }
    }
    if (createdKeyId !== null) {
      try {
        await keysAPI.delete(createdKeyId)
      } catch {
        // Keep the original error visible; the key can be removed manually.
      }
    }
    const errorMsg = extractApiErrorMessage(error, '') || t('keys.failedToSave')
    appStore.showError(errorMsg)
    // Don't advance tour on error
  } finally {
    submitting.value = false
  }
}

/**
 * 处理删除 API Key 的操作
 * 优化：错误处理改进，优先显示后端返回的具体错误消息（如权限不足等），
 * 若后端未返回消息则显示默认的国际化文本
 */
const handleDelete = async () => {
  if (!selectedKey.value) return

  try {
    await keysAPI.delete(selectedKey.value.id)
    appStore.showSuccess(t('keys.keyDeletedSuccess'))
    showDeleteDialog.value = false
    loadApiKeys()
  } catch (error: any) {
    // 优先使用后端返回的错误消息，提供更具体的错误信息给用户
    const errorMsg = error?.message || t('keys.failedToDelete')
    appStore.showError(errorMsg)
  }
}

const closeModals = () => {
  routingRequestGeneration += 1
  routingLoading.value = false
  routingProfile.value = null
  routingProfileLoadError.value = false
  routingCandidateToAdd.value = null
  routingSubmitAttempted.value = false
  clearRoutingGroupDrag()
  showCreateModal.value = false
  showEditModal.value = false
  selectedKey.value = null
  formData.value = {
    name: '',
    group_id: null,
    status: 'active',
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: false,
    ip_whitelist: '',
    ip_blacklist: '',
    enable_quota: false,
    quota: null,
    enable_rate_limit: false,
    rate_limit_5h: null,
    rate_limit_1d: null,
    rate_limit_7d: null,
    enable_expiration: false,
    expiration_preset: '30',
    expiration_date: ''
    ,routing_mode: 'single'
    ,routing_enabled: false
    ,routing_strategy: 'auto'
    ,routing_price_weight: 40
    ,routing_speed_weight: 30
    ,routing_success_weight: 30
    ,routing_group_ids: []
  }
}

const openCreateModal = () => {
  closeModals()
  showCreateModal.value = true
}

// Show reset quota confirmation dialog
const confirmResetQuota = () => {
  showResetQuotaDialog.value = true
}

// Set expiration date based on quick select days
const setExpirationDays = (days: number) => {
  formData.value.expiration_preset = days.toString() as '7' | '30' | '90'
  const expDate = new Date()
  expDate.setDate(expDate.getDate() + days)
  formData.value.expiration_date = formatDateTimeLocal(expDate.toISOString())
}

// Reset quota used for an API key
const resetQuotaUsed = async () => {
  if (!selectedKey.value) return
  showResetQuotaDialog.value = false
  try {
    await keysAPI.update(selectedKey.value.id, { reset_quota: true })
    appStore.showSuccess(t('keys.quotaResetSuccess'))
    // Update local state
    if (selectedKey.value) {
      selectedKey.value.quota_used = 0
    }
  } catch (error: any) {
    const errorMsg = extractApiErrorMessage(error, t('keys.failedToResetQuota'))
    appStore.showError(errorMsg)
  }
}

// Show reset rate limit confirmation dialog (from edit modal)
const confirmResetRateLimit = () => {
  showResetRateLimitDialog.value = true
}

// Show reset rate limit confirmation dialog (from table row)
const confirmResetRateLimitFromTable = (row: ApiKey) => {
  selectedKey.value = row
  showResetRateLimitDialog.value = true
}

// Reset rate limit usage for an API key
const resetRateLimitUsage = async () => {
  if (!selectedKey.value) return
  showResetRateLimitDialog.value = false
  try {
    await keysAPI.update(selectedKey.value.id, { reset_rate_limit_usage: true })
    appStore.showSuccess(t('keys.rateLimitResetSuccess'))
    // Refresh key data
    await loadApiKeys()
    // Update the editing key with fresh data
    const refreshedKey = apiKeys.value.find(k => k.id === selectedKey.value!.id)
    if (refreshedKey) {
      selectedKey.value = refreshedKey
    }
  } catch (error: any) {
    const errorMsg = extractApiErrorMessage(error, t('keys.failedToResetRateLimit'))
    appStore.showError(errorMsg)
  }
}

const importToCcswitch = (row: ApiKey) => {
  const platform = row.group?.platform || 'anthropic'

  // For antigravity platform, show client selection dialog
  if (platform === 'antigravity') {
    pendingCcsRow.value = row
    showCcsClientSelect.value = true
    return
  }

  // For other platforms, execute directly
  executeCcsImport(row, platform === 'gemini' ? 'gemini' : 'claude')
}

const executeCcsImport = (row: ApiKey, clientType: CcSwitchClientType) => {
  const baseUrl = publicSettings.value?.api_base_url || window.location.origin
  const platform = row.group?.platform || 'anthropic'

  const usageScript = `({
    request: {
      url: "{{baseUrl}}/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
      const unit = response?.unit ?? response?.quota?.unit ?? "USD";
      return {
        isValid: response?.is_active ?? response?.isValid ?? true,
        remaining,
        unit
      };
    }
  })`
  const providerName = (publicSettings.value?.site_name || 'sub2api').trim() || 'sub2api'
  const deeplink = buildCcSwitchImportDeeplink({
    baseUrl,
    platform,
    clientType,
    providerName,
    apiKey: row.key,
    usageScript
  })

  try {
    window.open(deeplink, '_self')

    // Check if the protocol handler worked by detecting if we're still focused
    setTimeout(() => {
      if (document.hasFocus()) {
        // Still focused means the protocol handler likely failed
        appStore.showError(t('keys.ccSwitchNotInstalled'))
      }
    }, 100)
  } catch (error) {
    appStore.showError(t('keys.ccSwitchNotInstalled'))
  }
}

const handleCcsClientSelect = (clientType: CcSwitchClientType) => {
  if (pendingCcsRow.value) {
    executeCcsImport(pendingCcsRow.value, clientType)
  }
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

const closeCcsClientSelect = () => {
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

function formatResetTime(resetAt: string | null): string {
  if (!resetAt) return ''
  const diff = new Date(resetAt).getTime() - now.value.getTime()
  if (diff <= 0) return t('keys.resetNow')
  const days = Math.floor(diff / 86400000)
  const hours = Math.floor((diff % 86400000) / 3600000)
  const mins = Math.floor((diff % 3600000) / 60000)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

onMounted(() => {
  loadSavedColumns()
  loadApiKeys()
  loadGroups()
  loadUserGroupRates()
  loadPublicSettings()
  document.addEventListener('click', closeGroupSelector)
  resetTimer = setInterval(() => { now.value = new Date() }, 60000)
})

onUnmounted(() => {
  routingRequestGeneration += 1
  routingListGeneration.value += 1
  document.removeEventListener('click', closeGroupSelector)
  if (resetTimer) clearInterval(resetTimer)
})
</script>

<style scoped>
/* Legacy routing rules are kept below as a reference; the compact reference-aligned
 * layer that follows is the only active routing editor styling. */
/*
.routing-panel {
  color: #172033;
}

.routing-field-head {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.routing-segmented {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  width: 100%;
  gap: 3px;
  padding: 3px;
  border: 1px solid #e5e7eb;
  border-radius: 9px;
  background: #f8fafc;
}

.routing-segmented button {
  min-height: 35px;
  padding: 0 10px;
  border-radius: 6px;
  color: #667085;
  font-size: 12px;
  font-weight: 600;
  transition: color .16s ease, background .16s ease, box-shadow .16s ease;
}

.routing-segmented button.active {
  color: #0f766e;
  background: #fff;
  box-shadow: 0 1px 3px rgb(15 23 42 / 10%);
}

.routing-segmented button:disabled,
.routing-strategy-card:disabled {
  cursor: wait;
  opacity: .65;
}

.routing-segmented button:focus-visible,
.routing-strategy-card:focus-visible,
.routing-icon-button:focus-visible {
  outline: 2px solid rgb(20 184 166 / 55%);
  outline-offset: 2px;
}

.routing-mode-hint {
  margin: 0;
  color: #98a2b3;
  font-size: 11px;
  line-height: 1.5;
}

.routing-smart-content {
  margin-top: 24px;
}

.routing-loading-hint {
  margin: 0 0 10px;
  color: #667085;
  font-size: 11px;
}

.routing-section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.routing-section-head h4 {
  margin: 0;
  color: #344054;
  font-size: 13px;
  font-weight: 700;
}

.routing-section-head p {
  margin: 4px 0 0;
  color: #98a2b3;
  font-size: 11px;
  line-height: 1.45;
}

.routing-count,
.routing-weight-total {
  flex-shrink: 0;
  color: #667085;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.routing-weight-total.valid { color: #0f9f8d; }
.routing-weight-total.invalid { color: #dc5b62; }

.routing-candidates-section {
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid #edf0f4;
}

.routing-candidate-box {
  margin-top: 10px;
  padding: 9px;
  border: 1px solid #dfe4ec;
  border-radius: 9px;
  background: #fff;
}

.routing-selected-list {
  display: grid;
  gap: 6px;
}

.routing-selected-row {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  padding: 7px 5px 7px 7px;
  border: 1px solid #edf0f4;
  border-radius: 7px;
  background: #fbfcfe;
}

.routing-selected-row.invalid {
  border-color: #f2c5c8;
  background: #fff8f8;
}

.routing-selected-row.dragging {
  opacity: .55;
  border-color: #2cc4b4;
}

.routing-selected-row.drag-over-before {
  border-top-color: #16a394;
  box-shadow: inset 0 2px 0 #16a394;
}

.routing-selected-row.drag-over-after {
  border-bottom-color: #16a394;
  box-shadow: inset 0 -2px 0 #16a394;
}

.routing-order {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 22px;
  width: 22px;
  height: 22px;
  border-radius: 6px;
  background: #eef2f6;
  color: #667085;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.routing-drag-handle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 20px;
  width: 20px;
  height: 24px;
  padding: 0;
  border: 0;
  background: transparent;
  color: #98a2b3;
  cursor: grab;
}

.routing-drag-handle:active { cursor: grabbing; }
.routing-drag-handle:disabled { cursor: not-allowed; opacity: .45; }

.routing-selected-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 1px;
}

.routing-selected-copy strong {
  overflow: hidden;
  color: #344054;
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.routing-selected-copy small {
  overflow: hidden;
  color: #98a2b3;
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.routing-invalid-badge {
  flex: 0 0 auto;
  color: #c24d55;
  font-size: 10px;
}

.routing-icon-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 26px;
  width: 26px;
  height: 26px;
  border-radius: 6px;
  color: #667085;
  transition: color .16s ease, background .16s ease;
}

.routing-icon-button:hover:not(:disabled) { color: #0f766e; background: #effcf9; }
.routing-icon-button:disabled { cursor: not-allowed; color: #cbd5e1; }
.routing-icon-button.danger:hover:not(:disabled) { color: #b42318; background: #fff1f2; }

.routing-empty {
  padding: 7px 3px 9px;
  color: #98a2b3;
  font-size: 12px;
}

.routing-add-select {
  width: 100%;
  margin-top: 8px;
}

.routing-add-select :deep(.select-trigger) {
  min-height: 36px;
  border-style: dashed;
  background: #f8fafc;
}

.routing-add-select :deep(.select-trigger:focus-visible) {
  border-color: #14b8a6;
  box-shadow: 0 0 0 3px rgb(20 184 166 / 12%);
}

.routing-select-option {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 1px;
}

.routing-select-option strong {
  color: #344054;
  font-size: 12px;
}

.routing-select-option small {
  color: #98a2b3;
  font-size: 10px;
}

.routing-strategy-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 10px;
}

.routing-strategy-card {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 9px;
  min-width: 0;
  min-height: 84px;
  padding: 12px;
  border: 1px solid #e4e7ec;
  border-radius: 10px;
  background: #fff;
  text-align: left;
  transition: border-color .16s ease, box-shadow .16s ease, transform .16s ease;
}

.routing-strategy-card:hover { border-color: #a8ddd7; transform: translateY(-1px); }
.routing-strategy-card:disabled:hover { border-color: #e4e7ec; transform: none; }
.routing-strategy-card.selected { border-color: #2cc4b4; background: #f1fcfa; box-shadow: 0 0 0 1px #2cc4b4 inset; }
.routing-strategy-card.custom { grid-column: 1 / -1; }

.routing-strategy-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 30px;
  width: 30px;
  height: 30px;
  border-radius: 8px;
}
.routing-strategy-icon.teal { color: #0f9f8d; background: #dff8f4; }
.routing-strategy-icon.green { color: #17845a; background: #e3f8ee; }
.routing-strategy-icon.amber { color: #bc7614; background: #fff2dc; }
.routing-strategy-icon.rose { color: #c65767; background: #ffebef; }
.routing-strategy-icon.violet { color: #7657c9; background: #f0ebff; }

.routing-strategy-copy { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 3px; }
.routing-strategy-copy strong { color: #344054; font-size: 12px; line-height: 1.25; }
.routing-strategy-copy small { color: #667085; font-size: 11px; line-height: 1.4; }
.routing-strategy-copy em { color: #98a2b3; font-size: 10px; font-style: normal; font-variant-numeric: tabular-nums; }
.routing-strategy-check { flex: 0 0 auto; color: #0f9f8d; }

.routing-custom-weights {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid #edf0f4;
}

.routing-custom-weights.invalid { border-top-color: #f2c5c8; }

.routing-custom-weight-total {
  grid-column: 1 / -1;
  color: #667085;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.routing-custom-weight-total.valid { color: #0f9f8d; }
.routing-custom-weight-total.invalid { color: #dc5b62; }

.routing-preview-section {
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid #edf0f4;
}

.routing-preview-controls {
  display: flex;
  align-items: stretch;
  flex-wrap: wrap;
  gap: 7px;
  margin-top: 10px;
}

.routing-preview-input {
  min-width: 0;
  flex: 1;
  min-height: 36px;
  padding: 0 10px;
  border: 1px solid #dfe4ec;
  border-radius: 7px;
  background: #fff;
  color: #344054;
  font-size: 12px;
  outline: none;
}

.routing-preview-input:focus { border-color: #14b8a6; box-shadow: 0 0 0 3px rgb(20 184 166 / 10%); }

.routing-preview-endpoint {
  flex: 0 0 132px;
  min-width: 0;
}

.routing-preview-endpoint :deep(.select-trigger) { min-height: 36px; }

.routing-preview-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  min-height: 36px;
  padding: 0 11px;
  border-radius: 7px;
  background: #eafaf7;
  color: #0f766e;
  font-size: 12px;
  font-weight: 600;
}

.routing-preview-button:disabled { cursor: wait; opacity: .6; }

.routing-preview-list {
  display: grid;
  gap: 5px;
  margin-top: 10px;
}

.routing-preview-row {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
  padding: 7px;
  border: 1px solid #edf0f4;
  border-radius: 7px;
  background: #fbfcfe;
}

.routing-preview-row:not(.eligible) { background: #fff8f8; border-color: #f2c5c8; }
.routing-preview-rank { flex: 0 0 20px; color: #98a2b3; font-size: 11px; text-align: center; }
.routing-preview-copy { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 1px; }
.routing-preview-copy strong { overflow: hidden; color: #344054; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.routing-preview-copy small { overflow: hidden; color: #98a2b3; font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }
.routing-preview-metric { color: #0f9f8d; font-size: 11px; font-variant-numeric: tabular-nums; }

.routing-custom-weights label { color: #667085; font-size: 11px; }
.routing-number-input { display: flex; align-items: center; margin-top: 5px; border: 1px solid #dfe4ec; border-radius: 6px; background: #fff; }
.routing-number-input:focus-within { border-color: #14b8a6; box-shadow: 0 0 0 3px rgb(20 184 166 / 10%); }
.routing-number-input input { width: 100%; min-width: 0; padding: 7px 8px; border: 0; background: transparent; color: #344054; font-size: 12px; outline: none; }
.routing-number-input b { padding-right: 8px; color: #98a2b3; font-size: 11px; font-weight: 500; }

.routing-validation-error {
  margin: 7px 0 0;
  color: #c24d55;
  font-size: 11px;
  line-height: 1.45;
}

.routing-profile-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin: 0 0 10px;
  padding: 8px 10px;
  border: 1px solid #f2c5c8;
  border-radius: 7px;
  background: #fff8f8;
}

.routing-profile-error .routing-validation-error {
  margin: 0;
}

.routing-retry-button {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  gap: 4px;
  min-height: 28px;
  padding: 0 7px;
  border-radius: 5px;
  color: #b42318;
  font-size: 11px;
  font-weight: 600;
}

.routing-retry-button:hover:not(:disabled) { background: #ffeded; }
.routing-retry-button:disabled { cursor: wait; opacity: .55; }

.key-routing-cell {
  display: flex;
  min-width: 100px;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
}

.key-routing-badge {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 7px;
  border-radius: 5px;
  font-size: 11px;
  font-weight: 600;
  line-height: 1;
}

.key-routing-badge-smart { color: #0f766e; background: #eafaf7; }
.key-routing-badge-single { color: #667085; background: #f1f3f5; }
.key-routing-badge-muted { color: #98a2b3; background: #f6f7f9; }
.key-routing-count,
.key-routing-status { color: #98a2b3; font-size: 10px; line-height: 1.2; }
.key-routing-status.active { color: #0f9f8d; }
.key-routing-status.disabled { color: #98a2b3; }
.key-routing-status.warning { color: #bc7614; }

.routing-custom-weights .routing-validation-error {
  grid-column: 1 / -1;
}

.dark .routing-panel { color: #e5e7eb; }
.dark .routing-profile-error { border-color: #713d46; background: #2a2025; }
.dark .routing-retry-button { color: #fda4af; }
.dark .routing-retry-button:hover:not(:disabled) { background: #3b232a; }
.dark .key-routing-badge-smart { color: #5eead4; background: #153433; }
.dark .key-routing-badge-single { color: #aeb8c5; background: #303846; }
.dark .key-routing-badge-muted { color: #8994a5; background: #252d38; }
.dark .key-routing-status.active { color: #5eead4; }
.dark .key-routing-status.warning { color: #fbbf24; }
.dark .routing-segmented { border-color: #394150; background: #202733; }
.dark .routing-segmented button { color: #9aa4b2; }
.dark .routing-segmented button.active { color: #5eead4; background: #293241; }
.dark .routing-mode-hint,
.dark .routing-loading-hint,
.dark .routing-section-head p,
.dark .routing-selected-copy small,
.dark .routing-select-option small { color: #8994a5; }
.dark .routing-candidates-section { border-top-color: #303846; }
.dark .routing-candidate-box,
.dark .routing-selected-row,
.dark .routing-number-input { border-color: #3b4554; background: #202733; }
.dark .routing-selected-row.invalid { border-color: #713d46; background: #2a2025; }
.dark .routing-selected-copy strong,
.dark .routing-select-option strong,
.dark .routing-strategy-copy strong { color: #e5e7eb; }
.dark .routing-order { background: #303846; color: #aeb8c5; }
.dark .routing-add-select :deep(.select-trigger) { border-color: #4b5666; background: #202733; }
.dark .routing-strategy-card,
.dark .routing-number-input { border-color: #3b4554; background: #202733; }
.dark .routing-strategy-card.selected { border-color: #2cc4b4; background: #153433; }
.dark .routing-custom-weights { border-top-color: #303846; }
.dark .routing-custom-weights.invalid { border-top-color: #713d46; }
.dark .routing-number-input input { color: #e5e7eb; }
.dark .routing-preview-section { border-top-color: #303846; }
.dark .routing-preview-input,
.dark .routing-preview-row { border-color: #3b4554; background: #202733; }
.dark .routing-preview-row:not(.eligible) { border-color: #713d46; background: #2a2025; }
.dark .routing-preview-input { color: #e5e7eb; }
.dark .routing-preview-copy strong { color: #e5e7eb; }

*/

/*
Smart-routing editor visual hierarchy. Keep the controls dense, but make
safety state, strategy, candidates, and preview read as separate tools.
.routing-panel {
  margin-top: 2px;
  color: #172033;
}

.routing-panel-heading {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: start;
  gap: 10px;
  margin-bottom: 14px;
}

.routing-panel-heading-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: 1px solid #dce5f7;
  border-radius: 10px;
  color: #4663a8;
  background: #eff3ff;
}

.routing-panel-heading-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.routing-panel-kicker {
  color: #667085;
  font-size: 10px;
  font-weight: 700;
  line-height: 1.2;
  text-transform: uppercase;
}

.routing-panel-heading-copy strong {
  color: #1f2937;
  font-size: 15px;
  font-weight: 700;
  line-height: 1.25;
}

.routing-panel-heading-copy p {
  margin: 1px 0 0;
  color: #667085;
  font-size: 11px;
  line-height: 1.45;
}

.routing-panel-status {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 23px;
  padding: 0 8px;
  border: 1px solid #e3e7ee;
  border-radius: 999px;
  color: #667085;
  background: #f8fafc;
  font-size: 10px;
  font-weight: 700;
  white-space: nowrap;
}

.routing-panel-status.active {
  border-color: #b9dfd9;
  color: #0f766e;
  background: #eefbf8;
}

.routing-status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #98a2b3;
}

.routing-panel-status.active .routing-status-dot {
  background: #16a394;
  box-shadow: 0 0 0 3px rgb(22 163 148 / 12%);
}

.routing-segmented {
  gap: 5px;
  padding: 4px;
  border-color: #e1e6ef;
  border-radius: 11px;
  background: #f6f8fb;
}

.routing-segmented button {
  display: flex;
  align-items: flex-start;
  justify-content: flex-start;
  gap: 8px;
  min-height: 58px;
  padding: 9px 10px;
  border: 1px solid transparent;
  border-radius: 8px;
  color: #667085;
  text-align: left;
}

.routing-segmented button > svg {
  flex: 0 0 auto;
  margin-top: 1px;
  color: #98a2b3;
}

.routing-segmented button.active {
  border-color: #dbe3f4;
  color: #31477c;
  background: #fff;
  box-shadow: 0 2px 6px rgb(31 41 55 / 8%);
}

.routing-segmented button.active > svg {
  color: #4663a8;
}

.routing-mode-option-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.routing-mode-option-copy strong {
  color: inherit;
  font-size: 12px;
  font-weight: 700;
  line-height: 1.2;
}

.routing-mode-option-copy small {
  overflow: hidden;
  color: #98a2b3;
  font-size: 10px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.routing-mode-hint {
  display: flex;
  align-items: flex-start;
  gap: 5px;
  margin-top: 8px;
  color: #667085;
}

.routing-mode-hint svg {
  flex: 0 0 auto;
  margin-top: 1px;
  color: #8090ad;
}

.routing-error-copy {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  gap: 6px;
}

.routing-error-copy > svg {
  flex: 0 0 auto;
  margin-top: 1px;
  color: #c24d55;
}

.routing-smart-content {
  margin-top: 16px;
}

.routing-loading-hint {
  display: flex;
  align-items: center;
  min-height: 28px;
  margin-bottom: 10px;
  padding: 0 9px;
  border: 1px solid #e4e9f2;
  border-radius: 7px;
  color: #667085;
  background: #f8fafc;
}

.routing-safety-note {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px 11px;
  border: 1px solid #d8e4f5;
  border-radius: 9px;
  color: #435b82;
  background: #f3f7fd;
  font-size: 11px;
  line-height: 1.45;
}

.routing-safety-note > svg {
  flex: 0 0 auto;
  margin-top: 1px;
  color: #4663a8;
}

.routing-safety-note > span {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.routing-safety-note strong {
  color: #31477c;
  font-size: 11px;
  font-weight: 700;
}

.routing-section-head {
  margin-top: 18px;
}

.routing-section-head h4 {
  color: #26344f;
  font-size: 13px;
}

.routing-section-head p {
  max-width: 50ch;
  color: #7b879c;
}

.routing-strategy-grid {
  gap: 7px;
  margin-top: 9px;
}

.routing-strategy-card {
  min-height: 88px;
  padding: 11px;
  border-color: #e2e7ef;
  border-radius: 9px;
  background: #fff;
  box-shadow: 0 1px 1px rgb(16 24 40 / 2%);
}

.routing-strategy-card:hover {
  border-color: #aebcdf;
  background: #fbfcff;
  transform: translateY(-1px);
}

.routing-strategy-card.selected {
  border-color: #6f86cf;
  background: #f1f4ff;
  box-shadow: 0 0 0 1px #6f86cf inset, 0 2px 5px rgb(70 99 168 / 8%);
}

.routing-strategy-icon {
  width: 29px;
  height: 29px;
  border-radius: 8px;
}

.routing-strategy-icon.teal { color: #4663a8; background: #e9efff; }
.routing-strategy-icon.green { color: #17845a; background: #e5f7ee; }
.routing-strategy-icon.amber { color: #a76514; background: #fff2dd; }
.routing-strategy-icon.rose { color: #b54f66; background: #ffedf1; }
.routing-strategy-icon.violet { color: #6f58b7; background: #f0ecff; }

.routing-strategy-copy strong {
  color: #26344f;
  font-size: 12px;
}

.routing-strategy-copy small {
  color: #6f7c91;
  font-size: 10.5px;
}

.routing-strategy-copy em {
  color: #98a2b3;
  font-size: 9.5px;
}

.routing-strategy-check { color: #4663a8; }

.routing-candidates-section,
.routing-preview-section {
  margin-top: 20px;
  padding-top: 18px;
  border-top-color: #e7ebf2;
}

.routing-count {
  padding: 3px 7px;
  border: 1px solid #e2e7ef;
  border-radius: 999px;
  color: #667085;
  background: #f8fafc;
  font-size: 10px;
  font-weight: 700;
}

.routing-candidate-box {
  margin-top: 9px;
  padding: 8px;
  border-color: #dfe5ee;
  border-radius: 9px;
  background: #f8fafc;
}

.routing-selected-row {
  min-height: 43px;
  padding: 7px 5px 7px 7px;
  border-color: #e4e9f1;
  border-radius: 7px;
  background: #fff;
}

.routing-order {
  width: 21px;
  height: 21px;
  border: 1px solid #e0e6ef;
  border-radius: 6px;
  color: #667085;
  background: #f4f6fa;
  font-size: 10px;
  font-weight: 700;
}

.routing-selected-copy strong { color: #26344f; font-size: 11.5px; }
.routing-selected-copy small { color: #8490a3; font-size: 9.5px; }

.routing-group-status {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 3px;
  font-size: 9px;
  font-weight: 700;
  white-space: nowrap;
}

.routing-group-status.eligible { color: #17845a; }
.routing-group-status.invalid { color: #b54f66; }

.key-routing-badge {
  gap: 4px;
  border: 1px solid transparent;
}

.key-routing-badge-smart { border-color: #cfe7e3; color: #0f766e; background: #eefbf8; }
.key-routing-badge-single { border-color: #e1e5eb; color: #667085; background: #f6f7f9; }

.key-routing-status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.key-routing-status-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #aeb8c5;
}

.key-routing-status.active .key-routing-status-dot { background: #16a394; }
.key-routing-status.warning .key-routing-status-dot { background: #d4932f; }

.routing-icon-button:hover:not(:disabled) { color: #4663a8; background: #eef2ff; }
.routing-icon-button.danger:hover:not(:disabled) { color: #b42318; background: #fff1f2; }

.routing-add-select :deep(.select-trigger) {
  min-height: 38px;
  border-color: #cbd5e1;
  border-style: solid;
  background: #fff;
}

.routing-add-select :deep(.select-trigger)::before {
  content: '+';
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  margin-right: 6px;
  border-radius: 50%;
  color: #4663a8;
  background: #e9efff;
  font-size: 13px;
  font-weight: 700;
}

.routing-preview-controls { gap: 7px; margin-top: 9px; }

.routing-preview-input,
.routing-number-input {
  border-color: #d6dde8;
  border-radius: 7px;
  background: #fff;
}

.routing-preview-input:focus,
.routing-number-input:focus-within {
  border-color: #6f86cf;
  box-shadow: 0 0 0 3px rgb(70 99 168 / 11%);
}

.routing-preview-button {
  border: 1px solid #cbd6f1;
  border-radius: 7px;
  color: #3d5798;
  background: #eef2ff;
}

.routing-preview-button:hover:not(:disabled) { background: #e2e9ff; }

.routing-preview-list { gap: 5px; margin-top: 9px; }

.routing-preview-row {
  min-height: 44px;
  padding: 7px;
  border-color: #e4e9f1;
  border-radius: 7px;
  background: #fff;
}

.routing-preview-row:not(.eligible) {
  border-color: #f2c5c8;
  background: #fff8f8;
}

.routing-preview-copy strong { color: #26344f; font-size: 11px; }
.routing-preview-copy small { color: #8490a3; font-size: 9.5px; }

.routing-preview-metrics {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 4px;
}

.routing-preview-metrics span {
  min-width: 33px;
  padding: 3px 4px;
  border: 1px solid #e1e7f0;
  border-radius: 5px;
  color: #4663a8;
  background: #f4f6fd;
  font-size: 9px;
  font-variant-numeric: tabular-nums;
  text-align: center;
}

.routing-custom-weights {
  gap: 9px;
  margin-top: 9px;
  padding-top: 11px;
  border-top-color: #e7ebf2;
}

.routing-custom-weight-total.valid { color: #17845a; }
.routing-custom-weight-total.invalid { color: #b54f66; }

.routing-validation-error { color: #b54f66; }

.dark .routing-panel-heading-icon { border-color: #394768; color: #aebdf1; background: #25304b; }
.dark .routing-panel-heading-copy strong,
.dark .routing-section-head h4,
.dark .routing-strategy-copy strong,
.dark .routing-selected-copy strong,
.dark .routing-preview-copy strong { color: #e9edf5; }
.dark .routing-panel-heading-copy p,
.dark .routing-mode-hint,
.dark .routing-mode-option-copy small,
.dark .routing-section-head p,
.dark .routing-selected-copy small,
.dark .routing-preview-copy small { color: #9aa6ba; }
.dark .routing-panel-status { border-color: #3d4657; color: #9aa6ba; background: #202733; }
.dark .routing-panel-status.active { border-color: #2f6d68; color: #77d5c8; background: #1c3638; }
.dark .routing-segmented { border-color: #394150; background: #202733; }
.dark .routing-segmented button.active { border-color: #3d4a68; color: #b8c4ef; background: #293241; }
.dark .routing-segmented button.active > svg { color: #aebdf1; }
.dark .routing-safety-note { border-color: #3b4b70; color: #b9c5df; background: #222d45; }
.dark .routing-safety-note > svg,
.dark .routing-safety-note strong { color: #aebdf1; }
.dark .routing-strategy-card { border-color: #3b4554; background: #202733; }
.dark .routing-strategy-card:hover { border-color: #5a6fa8; background: #252d3b; }
.dark .routing-strategy-card.selected { border-color: #6f86cf; background: #283450; }
.dark .routing-strategy-copy small { color: #a5b0c0; }
.dark .routing-strategy-check { color: #aebdf1; }
.dark .routing-candidates-section,
.dark .routing-preview-section,
.dark .routing-custom-weights { border-top-color: #303846; }
.dark .routing-count { border-color: #3b4554; color: #aeb8c5; background: #202733; }
.dark .routing-candidate-box,
.dark .routing-selected-row,
.dark .routing-preview-row,
.dark .routing-preview-input,
.dark .routing-number-input { border-color: #3b4554; background: #202733; }
.dark .routing-selected-row.invalid,
.dark .routing-preview-row:not(.eligible) { border-color: #713d46; background: #2a2025; }
.dark .routing-order { border-color: #3b4554; color: #aeb8c5; background: #303846; }
.dark .routing-add-select :deep(.select-trigger) { border-color: #4b5666; background: #202733; }
.dark .routing-add-select :deep(.select-trigger)::before { color: #b8c4ef; background: #313d62; }
.dark .routing-preview-input,
.dark .routing-number-input input { color: #e5e7eb; }
.dark .routing-preview-button { border-color: #4d5d8c; color: #b8c4ef; background: #2a3654; }
.dark .routing-preview-button:hover:not(:disabled) { background: #344263; }
.dark .routing-preview-metrics span { border-color: #3b4b70; color: #b8c4ef; background: #283450; }
.dark .key-routing-badge-smart { border-color: #2f6d68; }
.dark .key-routing-badge-single { border-color: #3b4554; }
.dark .routing-group-status.eligible { color: #70d3a6; }
.dark .routing-group-status.invalid { color: #f2a4b0; }

@media (max-width: 520px) {
  .routing-panel { padding-top: 16px; }
  .routing-panel-heading { grid-template-columns: auto minmax(0, 1fr); }
  .routing-panel-status { grid-column: 2; justify-self: start; }
  .routing-segmented { grid-template-columns: 1fr; }
  .routing-segmented button { min-height: 51px; }
  .routing-mode-option-copy small { white-space: normal; }
  .routing-profile-error { align-items: flex-start; flex-direction: column; }
  .routing-retry-button { align-self: flex-end; }
  .routing-strategy-grid { grid-template-columns: 1fr; }
  .routing-strategy-card.custom { grid-column: auto; }
  .routing-custom-weights { grid-template-columns: 1fr; }
  .routing-invalid-badge { display: none; }
  .routing-group-status span { display: none; }
  .routing-group-status { margin-left: auto; }
  .routing-preview-controls { flex-direction: column; }
  .routing-preview-input,
  .routing-preview-endpoint { width: 100%; }
  .routing-preview-button { width: 100%; }
  .routing-preview-metrics { gap: 3px; }
  .routing-preview-metrics span { min-width: 28px; padding: 3px; }
}

*/

/* Compact reference-aligned routing editor. */
.key-routing-cell { display: flex; min-width: 100px; flex-direction: column; align-items: flex-start; gap: 4px; }
.key-routing-badge { display: inline-flex; align-items: center; gap: 4px; min-height: 22px; padding: 0 7px; border: 1px solid transparent; border-radius: 5px; font-size: 11px; font-weight: 600; line-height: 1; }
.key-routing-badge-smart { border-color: #cdebe4; color: #0f766e; background: #f1fbf8; }
.key-routing-badge-single { border-color: #e5e7eb; color: #6b7280; background: #f5f6f7; }
.key-routing-badge-muted { border-color: #edf0f4; color: #9ca3af; background: #f8fafc; }
.key-routing-count, .key-routing-status { color: #9ca3af; font-size: 10px; line-height: 1.2; }
.key-routing-status { display: inline-flex; align-items: center; gap: 4px; }
.key-routing-status.active { color: #0f9f8d; }
.key-routing-status.warning { color: #b7791f; }
.key-routing-status-dot { width: 5px; height: 5px; border-radius: 50%; background: #aeb8c5; }
.key-routing-status.active .key-routing-status-dot { background: #16a394; }
.key-routing-status.warning .key-routing-status-dot { background: #d4932f; }

.routing-panel {
  margin-top: 0;
  color: #111827;
}

.routing-panel-header {
  display: flex;
  align-items: stretch;
  flex-direction: column;
  gap: 8px;
}

.routing-segmented {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  width: 100%;
  gap: 3px;
  padding: 3px;
  border: 1px solid #edf0f3;
  border-radius: 9px;
  background: #f5f6f8;
}

.routing-segmented button {
  flex: 1;
  min-height: 36px;
  padding: 0 11px;
  border-radius: 7px;
  color: #667085;
  background: transparent;
  font-size: 12px;
  font-weight: 600;
  line-height: 1;
}

.routing-segmented button.active {
  color: #138f84;
  background: #fff;
  box-shadow: 0 1px 3px rgb(16 24 40 / 10%), inset 0 0 0 1px rgb(19 143 132 / 4%);
}

.routing-segmented button:disabled { cursor: wait; opacity: .65; }
.routing-segmented button:focus-visible,
.routing-drag-handle:focus-visible,
.routing-icon-button:focus-visible,
.routing-preview-toggle:focus-visible { outline: 2px solid rgb(15 159 141 / 45%); outline-offset: 2px; }

.routing-mode-hint,
.routing-field-hint {
  display: flex;
  align-items: flex-start;
  gap: 5px;
  margin: 9px 0 0;
  color: #667085;
  font-size: 10px;
  line-height: 1.45;
}

.routing-mode-hint svg { flex: 0 0 auto; margin-top: 1px; color: #9ca3af; }

.routing-profile-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 10px;
  padding: 8px 10px;
  border: 1px solid #fecaca;
  border-radius: 6px;
  color: #b91c1c;
  background: #fff7f7;
  font-size: 11px;
  line-height: 1.4;
}

.routing-error-copy { display: inline-flex; min-width: 0; align-items: flex-start; gap: 6px; }
.routing-error-copy svg { flex: 0 0 auto; margin-top: 1px; }
.routing-retry-button { display: inline-flex; flex: 0 0 auto; align-items: center; gap: 4px; color: #b91c1c; font-size: 11px; font-weight: 600; }
.routing-retry-button:hover:not(:disabled) { text-decoration: underline; }
.routing-retry-button:disabled { cursor: wait; opacity: .55; }

.routing-workspace { margin-top: 17px; }
.routing-loading-hint { margin: 0 0 8px; color: #6b7280; font-size: 11px; }

.routing-safety-note {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  margin-bottom: 10px;
  padding: 8px 10px;
  border: 1px solid #cdebe4;
  border-radius: 6px;
  color: #28766e;
  background: #f1fbf8;
  font-size: 10.5px;
  line-height: 1.45;
}

.routing-safety-note svg { flex: 0 0 auto; margin-top: 1px; color: #0f9f8d; }
.routing-safety-note strong { margin-right: 4px; color: #0f766e; font-weight: 700; }

.routing-workspace-box {
  padding: 12px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  background: #fff;
}

.routing-workspace-header { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.routing-count { flex: 0 0 auto; color: #9ca3af; font-size: 10px; font-variant-numeric: tabular-nums; }
.routing-workspace-box > .routing-field-hint { margin-top: 4px; }

.routing-selected-list { display: grid; gap: 4px; margin-top: 8px; max-height: 150px; overflow-y: auto; }
.routing-selected-row {
  display: flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
  min-height: 35px;
  padding: 4px 4px 4px 6px;
  border: 1px solid #edf0f4;
  border-radius: 5px;
  background: #fbfcfe;
}
.routing-selected-row.invalid { border-color: #fecaca; background: #fff8f8; }
.routing-selected-row.dragging { opacity: .55; }
.routing-selected-row.drag-over-before { border-top: 2px solid #0f9f8d; }
.routing-selected-row.drag-over-after { border-bottom: 2px solid #0f9f8d; }
.routing-order { display: inline-flex; flex: 0 0 18px; align-items: center; justify-content: center; width: 18px; height: 18px; border-radius: 4px; color: #6b7280; background: #f1f3f5; font-size: 10px; font-variant-numeric: tabular-nums; }
.routing-drag-handle { display: inline-flex; flex: 0 0 18px; align-items: center; justify-content: center; width: 18px; height: 22px; padding: 0; color: #9ca3af; cursor: grab; }
.routing-drag-handle:active { cursor: grabbing; }
.routing-drag-handle:disabled { cursor: not-allowed; opacity: .45; }
.routing-selected-copy { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 1px; }
.routing-selected-copy strong { overflow: hidden; color: #374151; font-size: 11px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.routing-selected-copy small { overflow: hidden; color: #9ca3af; font-size: 9px; text-overflow: ellipsis; white-space: nowrap; }
.routing-group-status { display: inline-flex; flex: 0 0 auto; align-items: center; gap: 2px; color: #0f9f8d; font-size: 9px; font-weight: 600; white-space: nowrap; }
.routing-group-status.invalid { color: #dc2626; }
.routing-icon-button { display: inline-flex; flex: 0 0 22px; align-items: center; justify-content: center; width: 22px; height: 22px; border-radius: 4px; color: #9ca3af; }
.routing-icon-button:hover:not(:disabled) { color: #0f766e; background: #effcf9; }
.routing-icon-button.danger:hover:not(:disabled) { color: #dc2626; background: #fff1f2; }
.routing-icon-button:disabled { cursor: not-allowed; opacity: .4; }
.routing-empty { margin-top: 8px; padding: 8px; border: 1px dashed #d1d5db; border-radius: 5px; color: #9ca3af; font-size: 11px; text-align: center; }

.routing-add-select { width: 100%; margin-top: 8px; }
.routing-add-select :deep(.select-trigger) { min-height: 34px; border-style: dashed; border-radius: 5px; background: #fff; }
.routing-select-option { display: flex; min-width: 0; flex-direction: column; gap: 1px; }
.routing-select-option strong { color: #374151; font-size: 11px; }
.routing-select-option small { color: #9ca3af; font-size: 9px; }

.routing-strategy-block { margin-top: 0; padding-top: 0; }
.routing-strategy-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; margin-top: 10px; }
.routing-strategy-card {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
  min-height: 112px;
  padding: 12px;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  color: #374151;
  background: #fff;
  text-align: left;
  cursor: pointer;
  appearance: none;
  transition: border-color .14s ease, background-color .14s ease, box-shadow .14s ease;
}
.routing-strategy-card:hover:not(:disabled) { border-color: #9fd9cf; background: #fbfefd; }
.routing-strategy-card:focus-visible { outline: 2px solid rgb(15 159 141 / 35%); outline-offset: 1px; }
.routing-strategy-card:disabled { cursor: wait; opacity: .6; }
.routing-strategy-card.selected { border-color: #67d8cf; background: #effcfb; box-shadow: 0 0 0 1px #a6eee8 inset; }
.routing-strategy-card.custom { grid-column: 1 / -1; }
.routing-strategy-icon { display: inline-flex; flex: 0 0 30px; align-items: center; justify-content: center; width: 30px; height: 30px; border-radius: 8px; }
.routing-strategy-icon.teal { color: #0f766e; background: #e4f7f2; }
.routing-strategy-icon.green { color: #17845a; background: #e6f7ee; }
.routing-strategy-icon.amber { color: #a76514; background: #fff2dd; }
.routing-strategy-icon.rose { color: #b54f66; background: #ffedf1; }
.routing-strategy-icon.violet { color: #6f58b7; background: #f0ecff; }
.routing-strategy-copy { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 2px; }
.routing-strategy-copy strong { color: #25324a; font-size: 13px; font-weight: 700; line-height: 1.25; }
.routing-strategy-copy small { display: -webkit-box; overflow: hidden; color: #667085; font-size: 11px; line-height: 1.4; white-space: normal; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.routing-strategy-copy em { color: #8b95a5; font-size: 10px; font-style: normal; font-variant-numeric: tabular-nums; line-height: 1.35; white-space: normal; }
.routing-strategy-check { flex: 0 0 auto; margin-top: 1px; color: #0f9f8d; }
.routing-strategy-card.custom { min-height: 76px; }
.routing-candidates-block { margin-top: 16px; padding-top: 12px; border-top: 1px solid #edf0f4; }
.routing-custom-weights { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 7px; margin-top: 8px; padding-top: 9px; border-top: 1px solid #edf0f4; }
.routing-custom-weights label { display: block; min-width: 0; color: #6b7280; font-size: 10px; }
.routing-custom-weight-total { grid-column: 1 / -1; color: #9ca3af; font-size: 10px; font-variant-numeric: tabular-nums; line-height: 1.25; }
.routing-custom-weight-total.valid { color: #0f9f8d; }
.routing-custom-weight-total.invalid { color: #dc2626; }
.routing-custom-weights.invalid { border-top-color: #fecaca; }
.routing-weight-total { flex: 0 0 auto; color: #9ca3af; font-size: 10px; font-variant-numeric: tabular-nums; }
.routing-weight-total.valid { color: #0f9f8d; }
.routing-weight-total.invalid { color: #dc2626; }
.routing-weight-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; margin-top: 8px; }
.routing-weight-grid label { color: #6b7280; font-size: 10px; }
.routing-number-input { display: flex; align-items: center; margin-top: 4px; border: 1px solid #dfe4ec; border-radius: 5px; background: #fff; }
.routing-number-input:focus-within { border-color: #0f9f8d; box-shadow: 0 0 0 2px rgb(15 159 141 / 10%); }
.routing-number-input input { width: 100%; min-width: 0; padding: 7px 8px; border: 0; background: transparent; color: #374151; font-size: 12px; outline: none; }
.routing-number-input b { padding-right: 7px; color: #9ca3af; font-size: 10px; font-weight: 500; }
.routing-strategy-select { margin-top: 9px; }
.routing-strategy-select :deep(.select-trigger) { min-height: 34px; border-radius: 5px; }
.routing-validation-error { margin: 6px 0 0; color: #dc2626; font-size: 10px; line-height: 1.4; }

.routing-preview-section { margin-top: 12px; padding-top: 11px; border-top: 1px solid #edf0f4; }
.routing-preview-toggle { display: flex; width: 100%; align-items: center; justify-content: space-between; color: #374151; font-size: 11px; font-weight: 600; }
.routing-preview-toggle > span { display: inline-flex; align-items: center; gap: 5px; }
.routing-preview-toggle:hover { color: #0f766e; }
.routing-preview-section > .routing-field-hint { margin-top: 4px; }
.routing-preview-content { margin-top: 8px; }
.routing-preview-controls { display: flex; align-items: stretch; gap: 6px; }
.routing-preview-input { min-width: 0; flex: 1; min-height: 34px; padding: 0 9px; border: 1px solid #dfe4ec; border-radius: 5px; color: #374151; font-size: 11px; outline: none; }
.routing-preview-input:focus { border-color: #0f9f8d; box-shadow: 0 0 0 2px rgb(15 159 141 / 10%); }
.routing-preview-endpoint { flex: 0 0 112px; min-width: 0; }
.routing-preview-endpoint :deep(.select-trigger) { min-height: 34px; border-radius: 5px; }
.routing-preview-button { display: inline-flex; flex: 0 0 auto; align-items: center; justify-content: center; gap: 4px; min-height: 34px; padding: 0 9px; border: 1px solid #cdebe4; border-radius: 5px; color: #0f766e; background: #f1fbf8; font-size: 11px; font-weight: 600; }
.routing-preview-button:hover:not(:disabled) { background: #e4f7f2; }
.routing-preview-button:disabled { cursor: wait; opacity: .6; }
.routing-preview-list { display: grid; gap: 4px; margin-top: 8px; }
.routing-preview-row { display: flex; align-items: center; gap: 6px; min-width: 0; min-height: 36px; padding: 5px 6px; border: 1px solid #edf0f4; border-radius: 5px; background: #fbfcfe; }
.routing-preview-row:not(.eligible) { border-color: #fecaca; background: #fff8f8; }
.routing-preview-rank { flex: 0 0 16px; color: #9ca3af; font-size: 10px; text-align: center; }
.routing-preview-copy { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 1px; }
.routing-preview-copy strong { overflow: hidden; color: #374151; font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }
.routing-preview-copy small { overflow: hidden; color: #9ca3af; font-size: 9px; text-overflow: ellipsis; white-space: nowrap; }
.routing-preview-metrics { display: inline-flex; flex: 0 0 auto; gap: 3px; }
.routing-preview-metrics span { min-width: 28px; padding: 2px 3px; border: 1px solid #e1e7f0; border-radius: 4px; color: #4663a8; background: #f4f6fd; font-size: 9px; font-variant-numeric: tabular-nums; text-align: center; }

.dark .routing-panel { color: #e5e7eb; }
.dark .routing-segmented { border-color: #374151; background: #202733; }
.dark .routing-segmented button { color: #aeb8c5; }
.dark .routing-segmented button.active { color: #77d5c8; background: #1c3638; box-shadow: 0 1px 3px rgb(0 0 0 / 22%), inset 0 0 0 1px rgb(119 213 200 / 12%); }
.dark .routing-mode-hint, .dark .routing-field-hint, .dark .routing-count, .dark .routing-selected-copy small, .dark .routing-select-option small { color: #9ca3af; }
.dark .routing-safety-note { border-color: #2f6d68; color: #a7d9d2; background: #1c3638; }
.dark .routing-safety-note strong { color: #77d5c8; }
.dark .routing-workspace-box, .dark .routing-selected-row, .dark .routing-number-input, .dark .routing-preview-input, .dark .routing-preview-row { border-color: #374151; background: #202733; }
.dark .routing-selected-row.invalid, .dark .routing-preview-row:not(.eligible) { border-color: #713d46; background: #2a2025; }
.dark .routing-selected-copy strong, .dark .routing-preview-copy strong, .dark .routing-preview-toggle { color: #e5e7eb; }
.dark .routing-number-input input, .dark .routing-preview-input { color: #e5e7eb; }
.dark .routing-add-select :deep(.select-trigger), .dark .routing-strategy-select :deep(.select-trigger), .dark .routing-preview-endpoint :deep(.select-trigger) { border-color: #4b5666; background: #202733; }
.dark .routing-number-input, .dark .routing-preview-input { border-color: #4b5666; }
.dark .routing-strategy-card { border-color: #374151; background: #202733; }
.dark .routing-strategy-card.selected { border-color: #57cfc6; background: #173b39; box-shadow: 0 0 0 1px #2f6d68 inset; }
.dark .routing-strategy-copy strong { color: #e9edf5; }
.dark .routing-strategy-copy small { color: #aeb8c5; }
.dark .routing-strategy-copy em { color: #8f9bad; }
.dark .routing-strategy-block, .dark .routing-candidates-block, .dark .routing-preview-section { border-top-color: #303846; }
.dark .routing-preview-button { border-color: #2f6d68; color: #77d5c8; background: #1c3638; }

@media (max-width: 520px) {
  .routing-panel-header { align-items: flex-start; flex-direction: column; }
  .routing-segmented { width: 100%; }
  .routing-segmented button { flex: 1; }
  .routing-profile-error { align-items: flex-start; flex-direction: column; }
  .routing-retry-button { align-self: flex-end; }
  .routing-group-status span { display: none; }
  .routing-group-status { margin-left: auto; }
  .routing-strategy-grid { grid-template-columns: 1fr; }
  .routing-strategy-card.custom { grid-column: auto; }
  .routing-custom-weights { grid-template-columns: 1fr; }
  .routing-custom-weight-total { grid-column: auto; }
  .routing-weight-grid { grid-template-columns: 1fr; }
  .routing-preview-controls { flex-direction: column; }
  .routing-preview-input, .routing-preview-endpoint, .routing-preview-button { width: 100%; }
}

/* Final layout guards for the reference-sized editor.  The form keeps its
 * existing single-group flow; only smart routing gets an explicit visual
 * order so the primary group remains available without dominating the panel. */
.routing-mobile-strategy-select,
.routing-mobile-weight-editor { display: none; }

.routing-smart-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.routing-smart-form > * { margin-top: 0 !important; }

.routing-workspace {
  display: flex;
  flex-direction: column;
}

.routing-workspace > .routing-loading-hint { order: 0; }
.routing-workspace > .routing-strategy-block { order: 1; }
.routing-workspace > .routing-candidates-block { order: 2; }
.routing-workspace > .routing-safety-note { order: 3; }
.routing-workspace > .routing-preview-section { order: 4; }

@media (max-width: 520px) {
  .routing-smart-form { gap: 16px; }

  /* On narrow screens the candidate pool is the primary action. */
  .routing-workspace > .routing-candidates-block { order: 1; }
  .routing-workspace > .routing-strategy-block { order: 2; }

  .routing-candidates-block {
    margin-top: 0;
    padding: 12px;
    border: 1px solid #e6e9ef;
    border-radius: 10px;
    background: #fff;
  }

  .routing-candidates-block .routing-field-hint { margin-top: 5px; }
  .routing-selected-list { max-height: 180px; }

  .routing-strategy-grid,
  .routing-custom-weights { display: none; }

  .routing-strategy-block {
    display: flex;
    flex-direction: column;
  }

  .routing-strategy-block > .routing-workspace-header,
  .routing-strategy-block > .routing-field-hint { display: none; }

  .routing-mobile-strategy-select {
    display: block;
    order: 2;
    margin-top: 10px;
  }

  .routing-mobile-strategy-select :deep(.select-trigger) {
    min-height: 44px;
    border: 2px solid #4dd4c7;
    border-radius: 10px;
    color: #0f766e;
    background: #fff;
    box-shadow: 0 0 0 2px rgb(77 212 199 / 12%);
    font-size: 13px;
    font-weight: 600;
  }

  .routing-mobile-weight-editor {
    display: grid;
    order: 1;
    grid-template-columns: 1fr;
    gap: 10px;
    margin-top: 0;
    padding: 0;
    border: 0;
  }

  .routing-mobile-weight-editor > label {
    display: block;
    min-width: 0;
    color: #667085;
    font-size: 12px;
  }

  .routing-mobile-weight-editor .routing-number-input {
    min-height: 48px;
    margin-top: 5px;
    border-radius: 10px;
  }

  .routing-mobile-weight-editor .routing-number-input input {
    min-height: 44px;
    padding: 9px 12px;
    font-size: 15px;
  }

  .routing-mobile-weight-editor .routing-number-input b {
    padding-right: 12px;
    font-size: 12px;
  }

  .routing-mobile-weight-editor .routing-custom-weight-total {
    order: 4;
    margin-top: 1px;
    color: #0f9f8d;
    font-size: 13px;
  }

  .routing-mobile-weight-editor > label:nth-of-type(1) { order: 1; }
  .routing-mobile-weight-editor > label:nth-of-type(2) { order: 2; }
  .routing-mobile-weight-editor > label:nth-of-type(3) { order: 3; }
  .routing-mobile-weight-editor .routing-validation-error { order: 5; }

  .routing-safety-note {
    margin-top: 0;
    margin-bottom: 0;
    padding: 9px 10px;
    font-size: 10px;
  }

  .routing-selected-row {
    gap: 4px;
    min-height: 42px;
    padding: 5px 4px 5px 6px;
  }

  .routing-selected-copy strong { font-size: 11px; }
  .routing-selected-copy small { font-size: 9px; }
  .routing-icon-button { flex-basis: 24px; width: 24px; height: 24px; }
  .routing-add-select :deep(.select-trigger) { min-height: 42px; }
}

.dark .routing-candidates-block { border-color: #374151; background: #202733; }
.dark .routing-mobile-strategy-select :deep(.select-trigger) {
  border-color: #57cfc6;
  color: #77d5c8;
  background: #202733;
  box-shadow: 0 0 0 2px rgb(87 207 198 / 12%);
}
.dark .routing-mobile-weight-editor > label { color: #aeb8c5; }
</style>
