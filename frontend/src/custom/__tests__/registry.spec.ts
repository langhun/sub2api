import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import {
  customHeaderActions,
  customNavigation,
  customNotificationCenterExtensions,
  customAccountActions,
  customPaymentOrderDetailExtensions,
  customRechargeSummaryExtensions,
  customRoutes,
  customSettingsPanels,
  customPaymentDefinition,
} from '../registry'
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
const paymentViewSource = readFileSync(resolve(directory, '../../views/user/PaymentView.vue'), 'utf8')
const userOrdersViewSource = readFileSync(resolve(directory, '../../views/user/UserOrdersView.vue'), 'utf8')
const adminOrdersViewSource = readFileSync(resolve(directory, '../../views/admin/orders/AdminOrdersView.vue'), 'utf8')

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
      'activity',
      'recharge-promotions',
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

  it('mounts the OpenAI AT action through generic account adapters', () => {
    expect(customAccountActions.map((action) => action.id)).toContain('openai-rt')
    expect(accountActionMenuSource).toContain("import { customAccountActions } from '@/custom/registry'")
    expect(accountActionMenuSource).toContain("$emit('custom-action', action.id, account)")
    expect(accountsViewSource).toContain("import OpenAIRTModal from '@/custom/modules/openai-rt/OpenAIRTModal.vue'")
    expect(accountsViewSource).toContain('@custom-action="handleCustomAccountAction"')
  })

  it('mounts recharge overlays through generic checkout and order detail adapters', () => {
    expect(customRechargeSummaryExtensions.map((extension) => extension.id)).toContain('recharge-promotions')
    expect(customPaymentOrderDetailExtensions.map((extension) => extension.id)).toContain('recharge-promotions')
    expect(paymentViewSource).toContain("import { customRechargeSummaryExtensions } from '@/custom/registry'")
    expect(paymentViewSource).toContain('v-for="extension in customRechargeSummaryExtensions"')
    expect(paymentViewSource).not.toContain('@/custom/modules/recharge-promotions/RechargePromotionSummary.vue')
    expect(userOrdersViewSource).toContain("import { customPaymentOrderDetailExtensions } from '@/custom/registry'")
    expect(adminOrdersViewSource).toContain("import { customPaymentOrderDetailExtensions } from '@/custom/registry'")
  })

  it('registers custom payment business outside shared payment components', () => {
    expect(customPaymentDefinition('bepusdt')?.supportedTypes).toEqual(['bepusdt'])
    expect(customPaymentDefinition('bepusdt')?.callbackPaths?.notifyUrl).toBe('/api/v1/payment/webhook/bepusdt')
    expect(paymentViewSource).not.toContain("m === 'bepusdt'")
  })
})
