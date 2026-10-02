import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import {
  customHeaderActions,
  customSupportContactEntries,
  customNavigation,
  customNotificationCenterExtensions,
  customAccountActions,
  customAdminUserActions,
  customAdminUserActionModals,
  customPaymentOrderDetailExtensions,
  customRechargeSummaryExtensions,
  customRoutes,
  customSettingsPanels,
  customPaymentDefinition,
} from '../registry'
import { customPaymentDisplayName } from '../paymentExtensions'
import { activityHeaderActions } from '../modules/activity/headerActions'
import { activityNavigation } from '../modules/activity/navigation'
import { activityRoutes } from '../modules/activity/routes'
import { brandHomeRoutes } from '../modules/brand-home/routes'
import { gameHallNavigation } from '../modules/game-hall/navigation'
import { gameHallRoutes } from '../modules/game-hall/routes'
import { supportTicketNotificationCenterExtensions } from '../modules/support-tickets/notificationCenter'

const directory = dirname(fileURLToPath(import.meta.url))
const routerSource = readFileSync(resolve(directory, '../../router/index.ts'), 'utf8')
const sidebarSource = readFileSync(resolve(directory, '../../components/layout/AppSidebar.vue'), 'utf8')
const headerSource = readFileSync(resolve(directory, '../../components/layout/AppHeader.vue'), 'utf8')
const announcementBellSource = readFileSync(resolve(directory, '../../components/common/AnnouncementBell.vue'), 'utf8')
const settingsSource = readFileSync(resolve(directory, '../../views/admin/SettingsView.vue'), 'utf8')
const accountActionMenuSource = readFileSync(resolve(directory, '../../components/admin/account/AccountActionMenu.vue'), 'utf8')
const accountsViewSource = readFileSync(resolve(directory, '../../views/admin/AccountsView.vue'), 'utf8')
const usersViewSource = readFileSync(resolve(directory, '../../views/admin/UsersView.vue'), 'utf8')
const paymentViewSource = readFileSync(resolve(directory, '../../views/user/PaymentView.vue'), 'utf8')
const userOrdersViewSource = readFileSync(resolve(directory, '../../views/user/UserOrdersView.vue'), 'utf8')
const adminOrdersViewSource = readFileSync(resolve(directory, '../../views/admin/orders/AdminOrdersView.vue'), 'utf8')
const profileSource = readFileSync(resolve(directory, '../../views/user/ProfileView.vue'), 'utf8')
const redeemSource = readFileSync(resolve(directory, '../../views/user/RedeemView.vue'), 'utf8')
const docsSource = readFileSync(resolve(directory, '../modules/docs/DocsView.vue'), 'utf8')
const auditLogSource = readFileSync(resolve(directory, '../../views/admin/AuditLogView.vue'), 'utf8')

describe('custom overlay registry', () => {
  it('aggregates routes and navigation from custom modules', () => {
    expect(customRoutes).toEqual(expect.arrayContaining(brandHomeRoutes))
    expect(customRoutes).toEqual(expect.arrayContaining(activityRoutes))
    expect(customRoutes).toEqual(expect.arrayContaining(gameHallRoutes))
    expect(customNavigation).toEqual(expect.arrayContaining(activityNavigation))
    expect(customNavigation).toEqual(expect.arrayContaining(gameHallNavigation))
    expect(customHeaderActions).toEqual(expect.arrayContaining(activityHeaderActions))
    expect(customNotificationCenterExtensions).toEqual(expect.arrayContaining(supportTicketNotificationCenterExtensions))
  })

  it('mounts custom routes before the catch-all route', () => {
    expect(routerSource.indexOf('...customRoutes')).toBeGreaterThan(-1)
    expect(routerSource.indexOf('...customRoutes')).toBeLessThan(routerSource.indexOf("path: '/:pathMatch(.*)*'"))
  })

	it('does not register removed transfer routes or navigation', () => {
		expect(routerSource).not.toContain("path: '/transfer'")
		expect(sidebarSource).not.toContain("{ path: '/transfer'")
		expect(customRoutes.filter((route) => route.path === '/transfer')).toHaveLength(0)
		expect(customNavigation.filter((item) => item.path === '/transfer')).toHaveLength(0)
  })

  it('keeps upstream usage routes and sidebar entry outside custom modules', () => {
    expect(routerSource).toContain("path: '/usage'")
    expect(routerSource).toContain("path: '/key-usage'")
    expect(sidebarSource).toContain("{ path: '/usage', label: t('nav.usage'), icon: ChartIcon, hideInSimpleMode: true }")
    expect(customRoutes.filter((route) => route.path === '/usage')).toHaveLength(0)
    expect(customRoutes.filter((route) => route.path === '/key-usage')).toHaveLength(0)
    expect(customNavigation.filter((item) => item.path === '/usage')).toHaveLength(0)
  })

  it('mounts custom navigation through the sidebar self-navigation builder', () => {
    expect(sidebarSource).toContain('function buildCustomNavigationItems(')
    expect(sidebarSource).toContain("...buildCustomNavigationItems('after-affiliate')")
    expect(sidebarSource).toContain('label: item.labelKey ? t(item.labelKey) : item.label')
  })

  it('mounts header actions through the registry without importing activity UI directly', () => {
    expect(headerSource).toContain("import { customHeaderActions } from '@/custom/registry'")
    expect(headerSource).toContain('v-for="action in customHeaderActions"')
    expect(headerSource).not.toContain("@/custom/modules/activity/components/CheckinHeaderActions.vue")
  })

  it('mounts notification center extensions through a generic upstream adapter', () => {
    expect(announcementBellSource).toContain("import { customNotificationCenterExtensions } from '@/custom/registry'")
    expect(announcementBellSource).toContain('v-for="extension in customNotificationCenterExtensions"')
    expect(announcementBellSource).not.toContain('@/custom/modules/support-tickets')
    expect(sidebarSource).toContain('item.onActivate')
    expect(sidebarSource).not.toContain('requestSupportTicketNotificationPermission')
    expect(announcementBellSource.indexOf('await nextTick()')).toBeLessThan(
      announcementBellSource.indexOf('extensionsOpen.value = true'),
    )
  })

  it('owns custom setting panels through the registry rather than the shared form', () => {
    expect(customSettingsPanels.map((panel) => panel.id)).toEqual([
      'brand-home',
      'game-hall',
      'activity-entry-switches',
      'apple-theme',
      'activity',
      'registration-protection',
      'support-contacts',
    ])
    expect(settingsSource).toContain("customSettingsPanelsFor('site')")
    expect(settingsSource).toContain("customSettingsPanelsFor('entry-switches')")
    expect(settingsSource).toContain("customSettingsPanelsFor('features')")
    expect(settingsSource).toContain('panel.toPayload(customSettingsForms[panel.id] ?? {})')
    expect(settingsSource).not.toContain('form.game_hall_enabled')
    expect(settingsSource).not.toContain('form.checkin_enabled')
    expect(settingsSource).not.toContain('form.transfer_enabled')
    expect(settingsSource).not.toContain('form.default_homepage')
  })

  it('mounts customer support contacts through generic entries without importing the module into pages', () => {
    expect(customSupportContactEntries.map((entry) => entry.id)).toContain('support-contacts')
    for (const source of [headerSource, profileSource, redeemSource, docsSource]) {
      expect(source).toContain("import { customSupportContactEntries } from '@/custom/registry'")
      expect(source).toContain('v-for="entry in customSupportContactEntries"')
      expect(source).not.toContain('@/custom/modules/support-contacts')
    }
  })

  it('mounts the OpenAI AT action through generic account adapters', () => {
    expect(customAccountActions.map((action) => action.id)).toContain('openai-rt')
    expect(accountActionMenuSource).toContain("import { customAccountActions } from '@/custom/registry'")
    expect(accountActionMenuSource).toContain("$emit('custom-action', action.id, account)")
    expect(accountsViewSource).toContain("import OpenAIRTModal from '@/custom/modules/openai-rt/OpenAIRTModal.vue'")
    expect(accountsViewSource).toContain('@custom-action="handleCustomAccountAction"')
  })

  it('mounts per-user check-in access through generic user action adapters', () => {
    expect(customAdminUserActions.map((action) => action.id)).toContain('checkin-user-access')
    expect(customAdminUserActionModals.map((modal) => modal.id)).toContain('checkin-user-access')
    expect(usersViewSource).toContain("import { customAdminUserActions, customAdminUserActionModals } from '@/custom/registry'")
    expect(usersViewSource).toContain('v-for="action in visibleCustomUserActions(user)"')
    expect(usersViewSource).toContain('v-for="modal in customAdminUserActionModals"')
    expect(usersViewSource).not.toContain('@/custom/modules/checkin-user-access')
  })

  it('leaves recharge bonus rendering and order data to the upstream implementation', () => {
    expect(customRechargeSummaryExtensions).toHaveLength(0)
    expect(customPaymentOrderDetailExtensions).toHaveLength(0)
    expect(paymentViewSource).not.toContain("import { customRechargeSummaryExtensions } from '@/custom/registry'")
    expect(paymentViewSource).not.toContain('v-for="extension in customRechargeSummaryExtensions"')
    expect(paymentViewSource).toContain('rechargeBonusTiers')
    expect(userOrdersViewSource).toContain("import { customPaymentOrderDetailExtensions } from '@/custom/registry'")
    expect(adminOrdersViewSource).toContain("import { customPaymentOrderDetailExtensions } from '@/custom/registry'")
  })

  it('mounts audit action labels through the registry without embedding translations in the page', () => {
    expect(auditLogSource).toContain("import { customAuditLogDisplay } from '@/custom/registry'")
    expect(auditLogSource).toContain('auditActionLabel(row.action)')
    expect(auditLogSource).toContain('auditActionLabel(detail.action)')
    expect(auditLogSource).not.toContain('批量创建账号今日统计')
  })

  it('registers custom payment business outside shared payment components', () => {
    expect(customPaymentDefinition('bepusdt')?.supportedTypes).toEqual(['bepusdt'])
    expect(customPaymentDisplayName('bepusdt', 'bepusdt', { displayName: ' 数字货币 ' })).toBe('数字货币')
    expect(customPaymentDisplayName('bepusdt', 'bepusdt', {})).toBe('USDT')
    expect(customPaymentDefinition('bepusdt')?.callbackPaths?.notifyUrl).toBe('/api/v1/payment/webhook/bepusdt')
    expect(paymentViewSource).not.toContain("m === 'bepusdt'")
    expect(customPaymentDefinition('gmpay_edge')?.supportedTypes).toEqual(['gmpay_edge'])
    expect(customPaymentDisplayName('gmpay_edge', 'gmpay_edge', {})).toBe('USDT / USDC')
    expect(customPaymentDefinition('gmpay_edge')?.callbackPaths?.notifyUrl).toBe('/api/v1/payment/webhook/gmpay_edge')
  })
})
