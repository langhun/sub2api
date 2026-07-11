import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

type NavigationGuard = (
  to: Record<string, any>,
  from: Record<string, any>,
  next: ReturnType<typeof vi.fn>
) => Promise<void>

const routerHarness = vi.hoisted(() => ({
  guard: null as NavigationGuard | null,
  routes: [] as Array<{ path: string; meta?: Record<string, unknown> }>,
}))

const authStore = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  isAuthenticated: true,
  isAdmin: false,
  isSimpleMode: false,
  hasPendingAuthSession: false,
}))

const appStore = vi.hoisted(() => ({
  siteName: 'Sub2API',
  backendModeEnabled: false,
  publicSettingsLoaded: false,
  cachedPublicSettings: null as null | {
    payment_enabled?: boolean
    risk_control_enabled?: boolean
	usage_query_enabled?: boolean
	checkin_enabled?: boolean
	checkin_luck_enabled?: boolean
	transfer_enabled?: boolean
	redpacket_enabled?: boolean
	game_hall_enabled?: boolean
	leaderboard_enabled?: boolean
	leaderboard_balance_enabled?: boolean
	leaderboard_consumption_enabled?: boolean
	leaderboard_checkin_enabled?: boolean
	leaderboard_transfer_enabled?: boolean
    custom_menu_items?: []
  },
  fetchPublicSettings: vi.fn(),
}))

vi.mock('vue-router', () => ({
  createWebHistory: vi.fn(() => ({})),
  createRouter: vi.fn((options: { routes: Array<{ path: string; meta?: Record<string, unknown> }> }) => {
    routerHarness.routes = options.routes
    return {
    beforeEach: vi.fn((guard: NavigationGuard) => {
      routerHarness.guard = guard
    }),
    afterEach: vi.fn(),
    onError: vi.fn(),
    }
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] }),
}))

vi.mock('@/stores/adminCompliance', () => ({
  useAdminComplianceStore: () => ({
    initialized: true,
    fetchStatus: vi.fn(),
    requireAcknowledgement: vi.fn(),
  }),
}))

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false },
  }),
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  const promise = new Promise<T>((resolvePromise) => {
    resolve = resolvePromise
  })
  return { promise, resolve }
}

function runGuard(meta: Record<string, unknown>, path: string) {
  if (!routerHarness.guard) {
    throw new Error('router guard was not registered')
  }

  const next = vi.fn()
  const navigation = routerHarness.guard(
    {
      path,
      fullPath: path,
      name: 'FeatureRoute',
      params: {},
      meta: { requiresAuth: true, ...meta },
    },
    {},
    next
  )
  return { navigation, next }
}

describe('feature route guard', () => {
  beforeAll(async () => {
    await import('@/router')
  })

  beforeEach(() => {
    authStore.isAuthenticated = true
    authStore.isAdmin = false
    authStore.isSimpleMode = false
    appStore.publicSettingsLoaded = false
    appStore.cachedPublicSettings = null
    appStore.fetchPublicSettings.mockReset()
  })

  it('waits for the first public-settings request before deciding payment access', async () => {
    const deferred = createDeferred<{ payment_enabled: boolean }>()
    appStore.fetchPublicSettings.mockImplementation(async () => {
      const settings = await deferred.promise
      appStore.cachedPublicSettings = settings
      appStore.publicSettingsLoaded = true
      return settings
    })

    const { navigation, next } = runGuard({ requiresPayment: true }, '/purchase')

    await vi.waitFor(() => expect(appStore.fetchPublicSettings).toHaveBeenCalledTimes(1))
    expect(next).not.toHaveBeenCalled()

    deferred.resolve({ payment_enabled: true })
    await navigation
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()
  })

  it.each([
    ['payment', { requiresPayment: true }, '/purchase'],
    ['risk control', { requiresRiskControl: true }, '/admin/risk-control'],
  ])('does not treat a failed %s settings load as explicitly disabled', async (_name, meta, path) => {
    authStore.isAdmin = meta.requiresRiskControl === true
    appStore.fetchPublicSettings.mockResolvedValue(null)

    const { navigation, next } = runGuard(meta, path)
    await navigation

    expect(appStore.publicSettingsLoaded).toBe(false)
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()
  })

  it.each([
    ['payment', { requiresPayment: true }, { payment_enabled: false }, '/dashboard'],
    [
      'risk control',
      { requiresRiskControl: true },
      { risk_control_enabled: false },
      '/admin/settings',
    ],
  ])('redirects when loaded settings explicitly disable %s', async (_name, meta, settings, target) => {
    authStore.isAdmin = meta.requiresRiskControl === true
    appStore.cachedPublicSettings = settings
    appStore.publicSettingsLoaded = true

    const { navigation, next } = runGuard(meta, '/feature')
    await navigation

    expect(appStore.fetchPublicSettings).not.toHaveBeenCalled()
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith(target)
  })

	it.each([
		['usage query', 'usage_query_enabled'],
		['check-in', 'checkin_enabled'],
		['transfer', 'transfer_enabled'],
		['red packet', 'redpacket_enabled'],
		['game hall', 'game_hall_enabled'],
		['leaderboard', 'leaderboard_enabled'],
	])('redirects when the %s route feature is disabled', async (_name, key) => {
		appStore.cachedPublicSettings = { [key]: false }
		appStore.publicSettingsLoaded = true

		const { navigation, next } = runGuard({ requiresFeature: key }, '/feature')
		await navigation

		expect(next).toHaveBeenCalledOnce()
		expect(next).toHaveBeenCalledWith('/dashboard')
	})

	it('allows check-in route when only lucky check-in is enabled', async () => {
		appStore.cachedPublicSettings = { checkin_enabled: false, checkin_luck_enabled: true }
		appStore.publicSettingsLoaded = true
		const { navigation, next } = runGuard({ requiresAnyFeature: ['checkin_enabled', 'checkin_luck_enabled'] }, '/checkin')
		await navigation
		expect(next).toHaveBeenCalledWith()
	})

	it('redirects check-in route when all check-in modes are disabled', async () => {
		appStore.cachedPublicSettings = { checkin_enabled: false, checkin_luck_enabled: false }
		appStore.publicSettingsLoaded = true
		const { navigation, next } = runGuard({ requiresAnyFeature: ['checkin_enabled', 'checkin_luck_enabled'] }, '/checkin')
		await navigation
		expect(next).toHaveBeenCalledWith('/dashboard')
	})

	it('treats a missing opt-in route flag as disabled after settings load', async () => {
		appStore.cachedPublicSettings = {}
		appStore.publicSettingsLoaded = true

		const { navigation, next } = runGuard({ requiresFeature: 'game_hall_enabled' }, '/game-hall')
		await navigation

		expect(next).toHaveBeenCalledWith('/dashboard')
	})

	it('treats a missing opt-out route flag as enabled after settings load', async () => {
		appStore.cachedPublicSettings = {}
		appStore.publicSettingsLoaded = true

		const { navigation, next } = runGuard({ requiresFeature: 'leaderboard_enabled' }, '/leaderboard')
		await navigation

		expect(next).toHaveBeenCalledWith()
	})

	it.each([
		['transfer disabled', { transfer_enabled: false, leaderboard_enabled: true, leaderboard_transfer_enabled: true }],
		['leaderboard disabled', { transfer_enabled: true, leaderboard_enabled: false, leaderboard_transfer_enabled: true }],
		['transfer leaderboard disabled', { transfer_enabled: true, leaderboard_enabled: true, leaderboard_transfer_enabled: false }],
		['transfer leaderboard missing', { transfer_enabled: true, leaderboard_enabled: true }],
	])('blocks a direct transfer leaderboard route when %s', async (_name, settings) => {
		appStore.cachedPublicSettings = settings
		appStore.publicSettingsLoaded = true

		const { navigation, next } = runGuard({
			requiresAllFeatures: ['transfer_enabled', 'leaderboard_enabled', 'leaderboard_transfer_enabled'],
		}, '/transfer/leaderboard')
		await navigation

		expect(next).toHaveBeenCalledWith('/dashboard')
	})

	it('allows the transfer leaderboard route only when all switches are enabled', async () => {
		appStore.cachedPublicSettings = {
			transfer_enabled: true,
			leaderboard_enabled: true,
			leaderboard_transfer_enabled: true,
		}
		appStore.publicSettingsLoaded = true

		const { navigation, next } = runGuard({
			requiresAllFeatures: ['transfer_enabled', 'leaderboard_enabled', 'leaderboard_transfer_enabled'],
		}, '/transfer/leaderboard')
		await navigation

		expect(next).toHaveBeenCalledWith()
	})

	it('wires the real leaderboard routes to the effective switch groups', () => {
		const leaderboard = routerHarness.routes.find((route) => route.path === '/leaderboard')
		const transferLeaderboard = routerHarness.routes.find((route) => route.path === '/transfer/leaderboard')

		expect(leaderboard?.meta?.requiresFeature).toBe('leaderboard_enabled')
		expect(leaderboard?.meta?.requiresAnyFeatureGroups).toEqual([
			['leaderboard_balance_enabled'],
			['leaderboard_consumption_enabled'],
			['leaderboard_checkin_enabled'],
			['leaderboard_transfer_enabled', 'transfer_enabled'],
		])
		expect(transferLeaderboard?.meta?.requiresAllFeatures).toEqual([
			'transfer_enabled',
			'leaderboard_enabled',
			'leaderboard_transfer_enabled',
		])
	})

	it('blocks the leaderboard when no effective board group is enabled', async () => {
		appStore.cachedPublicSettings = {
			leaderboard_enabled: true,
			leaderboard_balance_enabled: false,
			leaderboard_consumption_enabled: false,
			leaderboard_checkin_enabled: false,
			leaderboard_transfer_enabled: true,
			transfer_enabled: false,
		}
		appStore.publicSettingsLoaded = true

		const { navigation, next } = runGuard({
			requiresFeature: 'leaderboard_enabled',
			requiresAnyFeatureGroups: [
				['leaderboard_balance_enabled'],
				['leaderboard_consumption_enabled'],
				['leaderboard_checkin_enabled'],
				['leaderboard_transfer_enabled', 'transfer_enabled'],
			],
		}, '/leaderboard')
		await navigation

		expect(next).toHaveBeenCalledWith('/dashboard')
	})
})
