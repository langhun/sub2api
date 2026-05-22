import { ADMIN_EMAIL, ADMIN_PASSWORD, buildAccounts, buildActiveSubscriptions, buildAdminApiKeyState, buildAdminApiKeyValue, buildAuthResponse, buildBatchTodayStats, buildCheckinStatus, buildDashboardSnapshot, buildGroupCapacitySummary, buildGroups, buildGroupUsageSummary, buildMihomoStatus, buildPaymentConfig, buildPaymentProviders, buildProxyList, buildProxySubscriptions, buildPublicSettings, buildSettings, buildSetupStatus, buildUserRanking, buildUserTrend, maskAdminApiKey } from './fixtures.js'

function jsonResponse(body, status = 200) {
  return {
    status,
    contentType: 'application/json; charset=utf-8',
    body: JSON.stringify(body),
  }
}

function apiSuccess(data) {
  return { code: 0, message: 'ok', data }
}

function matches(pathname, target) {
  return pathname === target || pathname.startsWith(`${target}/`)
}

export async function mockCommonAppRoutes(page, options = {}) {
  const initialSettings = buildSettings()
  const initialPaymentConfig = buildPaymentConfig({
    enabled: initialSettings.payment_enabled,
    min_amount: initialSettings.payment_min_amount,
    max_amount: initialSettings.payment_max_amount,
    daily_limit: initialSettings.payment_daily_limit,
    order_timeout_minutes: initialSettings.payment_order_timeout_minutes,
    max_pending_orders: initialSettings.payment_max_pending_orders,
    enabled_payment_types: [...initialSettings.payment_enabled_types],
    balance_disabled: initialSettings.payment_balance_disabled,
    balance_recharge_multiplier: initialSettings.payment_balance_recharge_multiplier,
    load_balance_strategy: initialSettings.payment_load_balance_strategy,
    product_name_prefix: initialSettings.payment_product_name_prefix,
    product_name_suffix: initialSettings.payment_product_name_suffix,
    help_image_url: initialSettings.payment_help_image_url,
    help_text: initialSettings.payment_help_text,
  })
  const state = {
    settings: initialSettings,
    paymentConfig: initialPaymentConfig,
    paymentProviders: buildPaymentProviders(),
    proxies: buildProxyList(),
    allProxies: buildProxyList().items,
    createdProxies: [],
    loginResponse: buildAuthResponse(),
    publicSettings: buildPublicSettings(options.publicSettings),
    setupStatus: buildSetupStatus(),
    dashboardSnapshot: buildDashboardSnapshot(),
    userTrend: buildUserTrend(),
    userRanking: buildUserRanking(),
    groups: buildGroups(),
    createdGroups: [],
    groupUsageSummary: buildGroupUsageSummary(),
    groupCapacitySummary: buildGroupCapacitySummary(),
    accounts: buildAccounts(),
    batchTodayStats: buildBatchTodayStats(),
    proxySubscriptions: buildProxySubscriptions(),
    mihomoStatus: buildMihomoStatus(),
    activeSubscriptions: buildActiveSubscriptions(),
    checkinStatus: buildCheckinStatus(),
    adminApiKey: buildAdminApiKeyState(options.adminApiKey),
  }

  function syncPaymentStateFromSettings(payload = {}) {
    state.paymentConfig = {
      ...state.paymentConfig,
      enabled: payload.payment_enabled ?? state.settings.payment_enabled,
      min_amount: payload.payment_min_amount ?? state.settings.payment_min_amount,
      max_amount: payload.payment_max_amount ?? state.settings.payment_max_amount,
      daily_limit: payload.payment_daily_limit ?? state.settings.payment_daily_limit,
      order_timeout_minutes: payload.payment_order_timeout_minutes ?? state.settings.payment_order_timeout_minutes,
      max_pending_orders: payload.payment_max_pending_orders ?? state.settings.payment_max_pending_orders,
      enabled_payment_types: [...(payload.payment_enabled_types ?? state.settings.payment_enabled_types ?? [])],
      balance_disabled: payload.payment_balance_disabled ?? state.settings.payment_balance_disabled,
      balance_recharge_multiplier: payload.payment_balance_recharge_multiplier ?? state.settings.payment_balance_recharge_multiplier,
      load_balance_strategy: payload.payment_load_balance_strategy ?? state.settings.payment_load_balance_strategy,
      product_name_prefix: payload.payment_product_name_prefix ?? state.settings.payment_product_name_prefix,
      product_name_suffix: payload.payment_product_name_suffix ?? state.settings.payment_product_name_suffix,
      help_image_url: payload.payment_help_image_url ?? state.settings.payment_help_image_url,
      help_text: payload.payment_help_text ?? state.settings.payment_help_text,
    }
  }

  await page.route('**/*', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const { pathname } = url
    const method = request.method()

    if (pathname.startsWith('/assets/') || pathname === '/' || pathname.endsWith('.js') || pathname.endsWith('.css') || pathname.endsWith('.svg') || pathname.endsWith('.png') || pathname.endsWith('.ico')) {
      await route.continue()
      return
    }

    if (pathname === '/setup/status' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.setupStatus)))
      return
    }

    if (pathname === '/api/v1/settings/public' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.publicSettings)))
      return
    }

    if (pathname === '/api/v1/auth/login' && method === 'POST') {
      const body = request.postDataJSON() || {}
      if (body.email === ADMIN_EMAIL && body.password === ADMIN_PASSWORD) {
        await route.fulfill(jsonResponse(apiSuccess(state.loginResponse)))
        return
      }
      await route.fulfill(jsonResponse({ code: 40101, message: 'invalid credentials', data: null }, 401))
      return
    }

    if (pathname === '/api/v1/auth/me' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.loginResponse.user)))
      return
    }

    if (pathname === '/api/v1/auth/logout' && method === 'POST') {
      await route.fulfill(jsonResponse(apiSuccess({ message: 'logged out' })))
      return
    }

    if (pathname === '/api/v1/subscriptions/active' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.activeSubscriptions)))
      return
    }

    if (pathname === '/api/v1/announcements' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess([])))
      return
    }

    if (matches(pathname, '/api/v1/announcements') && method === 'POST') {
      await route.fulfill(jsonResponse(apiSuccess({ message: 'read' })))
      return
    }

    if (pathname === '/api/v1/checkin/status' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.checkinStatus)))
      return
    }

    if (pathname === '/api/v1/admin/payment/config' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.paymentConfig)))
      return
    }

    if (pathname === '/api/v1/admin/payment/config' && method === 'PUT') {
      const payload = request.postDataJSON() || {}
      state.paymentConfig = {
        ...state.paymentConfig,
        ...payload,
      }
      await route.fulfill(jsonResponse(apiSuccess(state.paymentConfig)))
      return
    }

    if (pathname === '/api/v1/admin/payment/providers' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.paymentProviders)))
      return
    }

    if (pathname === '/api/v1/admin/payment/providers' && method === 'POST') {
      const payload = request.postDataJSON() || {}
      const created = {
        id: 4000 + state.paymentProviders.length + 1,
        provider_key: payload.provider_key || 'easypay',
        name: payload.name || `provider-${state.paymentProviders.length + 1}`,
        config: payload.config || {},
        supported_types: Array.isArray(payload.supported_types) ? payload.supported_types : [],
        enabled: payload.enabled === true,
        payment_mode: payload.payment_mode || '',
        refund_enabled: payload.refund_enabled === true,
        allow_user_refund: payload.allow_user_refund === true,
        limits: payload.limits || '',
        sort_order: payload.sort_order ?? state.paymentProviders.length,
      }
      state.paymentProviders = [...state.paymentProviders, created]
      await route.fulfill(jsonResponse(apiSuccess(created)))
      return
    }

    if (pathname === '/api/v1/admin/dashboard/snapshot-v2' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.dashboardSnapshot)))
      return
    }

    if (pathname === '/api/v1/admin/dashboard/users-trend' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.userTrend)))
      return
    }

    if (pathname === '/api/v1/admin/dashboard/users-ranking' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.userRanking)))
      return
    }

    if (pathname === '/api/v1/admin/settings' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.settings)))
      return
    }

    if (pathname === '/api/v1/admin/settings' && method === 'PUT') {
      const payload = request.postDataJSON() || {}
      state.settings = { ...state.settings, ...payload }
      syncPaymentStateFromSettings(payload)
      await route.fulfill(jsonResponse(apiSuccess(state.settings)))
      return
    }

    if (pathname === '/api/v1/admin/settings/admin-api-key' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess({
        exists: state.adminApiKey.exists,
        masked_key: state.adminApiKey.masked_key,
      })))
      return
    }

    if (pathname === '/api/v1/admin/settings/admin-api-key/regenerate' && method === 'POST') {
      const rotation = (state.adminApiKey.rotation || 0) + 1
      const key = buildAdminApiKeyValue(rotation)
      state.adminApiKey = buildAdminApiKeyState({
        currentKey: key,
        rotation,
        masked_key: maskAdminApiKey(key),
      })
      await route.fulfill(jsonResponse(apiSuccess({ key })))
      return
    }

    if (pathname === '/api/v1/admin/settings/admin-api-key' && method === 'DELETE') {
      state.adminApiKey = buildAdminApiKeyState()
      await route.fulfill(jsonResponse(apiSuccess({ message: 'deleted' })))
      return
    }

    if (matches(pathname, '/api/v1/admin/payment/providers') && method === 'PUT') {
      const payload = request.postDataJSON() || {}
      const id = Number(pathname.split('/').pop())
      const current = state.paymentProviders.find((provider) => provider.id === id)

      if (!current) {
        await route.fulfill(jsonResponse({ code: 404, message: 'provider not found', data: null }, 404))
        return
      }

      const updated = {
        ...current,
        ...payload,
        config: payload.config ? { ...current.config, ...payload.config } : current.config,
        supported_types: Array.isArray(payload.supported_types) ? payload.supported_types : current.supported_types,
      }
      state.paymentProviders = state.paymentProviders.map((provider) =>
        provider.id === id ? updated : provider,
      )
      await route.fulfill(jsonResponse(apiSuccess(updated)))
      return
    }

    if (matches(pathname, '/api/v1/admin/payment/providers') && method === 'DELETE') {
      const id = Number(pathname.split('/').pop())
      state.paymentProviders = state.paymentProviders.filter((provider) => provider.id !== id)
      await route.fulfill(jsonResponse(apiSuccess({ message: 'deleted' })))
      return
    }

    if (pathname === '/api/v1/admin/settings/overload-cooldown' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess({ enabled: false, cooldown_minutes: 10 })))
      return
    }

    if (pathname === '/api/v1/admin/settings/rate-limit-429-cooldown' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess({ enabled: false, cooldown_seconds: 300 })))
      return
    }

    if (pathname === '/api/v1/admin/settings/stream-timeout' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess({
        enabled: false,
        action: 'temp_unsched',
        temp_unsched_minutes: 5,
        threshold_count: 3,
        threshold_window_minutes: 10,
      })))
      return
    }

    if (pathname === '/api/v1/admin/settings/rectifier' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess({
        enabled: false,
        thinking_signature_enabled: false,
        thinking_budget_enabled: false,
        apikey_signature_enabled: false,
        apikey_signature_patterns: [],
      })))
      return
    }

    if (pathname === '/api/v1/admin/settings/beta-policy' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess({ rules: [] })))
      return
    }

    if (pathname === '/api/v1/admin/settings/web-search-emulation' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess({ enabled: false, providers: [] })))
      return
    }

    if (pathname === '/api/v1/admin/groups/all' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.groups)))
      return
    }

    if (pathname === '/api/v1/admin/groups' && method === 'GET') {
      const search = (url.searchParams.get('search') || '').trim().toLowerCase()
      const platform = (url.searchParams.get('platform') || '').trim()
      const status = (url.searchParams.get('status') || '').trim()
      const isExclusiveRaw = url.searchParams.get('is_exclusive')
      const pageNumber = Number(url.searchParams.get('page') || '1')
      const pageSize = Number(url.searchParams.get('page_size') || '20')

      let items = [...state.groups]

      if (search) {
        items = items.filter((group) =>
          group.name.toLowerCase().includes(search) ||
          String(group.description || '').toLowerCase().includes(search)
        )
      }

      if (platform) {
        items = items.filter((group) => group.platform === platform)
      }

      if (status) {
        items = items.filter((group) => group.status === status)
      }

      if (isExclusiveRaw === 'true' || isExclusiveRaw === 'false') {
        const isExclusive = isExclusiveRaw === 'true'
        items = items.filter((group) => group.is_exclusive === isExclusive)
      }

      const start = (pageNumber - 1) * pageSize
      const pagedItems = items.slice(start, start + pageSize)

      await route.fulfill(jsonResponse(apiSuccess({
        items: pagedItems,
        total: items.length,
        page: pageNumber,
        page_size: pageSize,
        pages: Math.max(1, Math.ceil(items.length / pageSize)),
      })))
      return
    }

    if (pathname === '/api/v1/admin/groups/usage-summary' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.groupUsageSummary)))
      return
    }

    if (pathname === '/api/v1/admin/groups/capacity-summary' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.groupCapacitySummary)))
      return
    }

    if (pathname === '/api/v1/admin/groups' && method === 'POST') {
      const payload = request.postDataJSON() || {}
      const created = {
        id: 1000 + state.createdGroups.length + 1,
        name: payload.name || '新分组',
        description: payload.description || '',
        platform: payload.platform || 'anthropic',
        rate_multiplier: payload.rate_multiplier ?? 1,
        rpm_limit: payload.rpm_limit ?? 0,
        is_exclusive: payload.is_exclusive === true,
        status: 'active',
        subscription_type: payload.subscription_type || 'standard',
        daily_limit_usd: payload.daily_limit_usd ?? null,
        weekly_limit_usd: payload.weekly_limit_usd ?? null,
        monthly_limit_usd: payload.monthly_limit_usd ?? null,
        allow_image_generation: payload.allow_image_generation === true,
        image_rate_independent: payload.image_rate_independent === true,
        image_rate_multiplier: payload.image_rate_multiplier ?? 1,
        image_price_1k: payload.image_price_1k ?? null,
        image_price_2k: payload.image_price_2k ?? null,
        image_price_4k: payload.image_price_4k ?? null,
        claude_code_only: payload.claude_code_only === true,
        fallback_group_id: payload.fallback_group_id ?? null,
        fallback_group_id_on_invalid_request: payload.fallback_group_id_on_invalid_request ?? null,
        allow_messages_dispatch: payload.allow_messages_dispatch === true,
        require_oauth_only: payload.require_oauth_only === true,
        require_privacy_set: payload.require_privacy_set === true,
        model_routing: payload.model_routing ?? null,
        model_routing_enabled: payload.model_routing_enabled === true,
        mcp_xml_inject: payload.mcp_xml_inject !== false,
        supported_model_scopes: payload.supported_model_scopes || ['claude', 'gemini_text', 'gemini_image'],
        account_count: 0,
        active_account_count: 0,
        rate_limited_account_count: 0,
        sort_order: state.groups.length + 1,
        created_at: '2026-05-22T00:00:00Z',
        updated_at: '2026-05-22T00:00:00Z',
      }
      state.createdGroups.push(created)
      state.groups = [created, ...state.groups]
      await route.fulfill(jsonResponse(apiSuccess(created)))
      return
    }

    if (pathname === '/api/v1/admin/proxies/all' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.allProxies)))
      return
    }

    if (pathname === '/api/v1/admin/accounts/today-stats/batch' && method === 'POST') {
      await route.fulfill(jsonResponse(apiSuccess(state.batchTodayStats)))
      return
    }

    if (pathname === '/api/v1/admin/accounts' && method === 'GET') {
      const search = (url.searchParams.get('search') || '').trim().toLowerCase()
      const platform = (url.searchParams.get('platform') || '').trim()
      const group = (url.searchParams.get('group') || '').trim()
      const mainStatus = (url.searchParams.get('main_status') || url.searchParams.get('status') || '').trim()
      const pageNumber = Number(url.searchParams.get('page') || '1')
      const pageSize = Number(url.searchParams.get('page_size') || '20')

      let items = [...state.accounts.items]

      if (search) {
        items = items.filter((account) =>
          account.name.toLowerCase().includes(search) ||
          String(account.notes || '').toLowerCase().includes(search)
        )
      }

      if (platform) {
        items = items.filter((account) => account.platform === platform)
      }

      if (group === 'ungrouped') {
        items = items.filter((account) => !Array.isArray(account.groups) || account.groups.length === 0)
      } else if (group) {
        items = items.filter((account) => Array.isArray(account.groups) && account.groups.some((item) => String(item.id) === group))
      }

      if (mainStatus) {
        items = items.filter((account) => account.status === mainStatus)
      }

      const start = (pageNumber - 1) * pageSize
      const pagedItems = items.slice(start, start + pageSize)

      await route.fulfill({
        ...jsonResponse(apiSuccess({
          items: pagedItems,
          total: items.length,
          page: pageNumber,
          page_size: pageSize,
          pages: Math.max(1, Math.ceil(items.length / pageSize)),
        })),
        headers: {
          ETag: '"playwright-accounts-etag"',
        },
      })
      return
    }

    if (pathname === '/api/v1/admin/proxy-subscriptions' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.proxySubscriptions)))
      return
    }

    if (pathname === '/api/v1/admin/proxies' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.proxies)))
      return
    }

    if (pathname === '/api/v1/admin/proxies' && method === 'POST') {
      const payload = request.postDataJSON() || {}
      const created = {
        id: 102 + state.createdProxies.length,
        name: payload.name || 'new-proxy',
        protocol: payload.protocol || 'http',
        host: payload.host || '127.0.0.1',
        port: payload.port || 8080,
        username: payload.username || '',
        password: payload.password || '',
        status: 'active',
        runtime_status: 'healthy',
        health_status: 'healthy',
        account_count: 0,
        auto_failover_pool_enabled: payload.auto_failover_pool_enabled === true,
        failover_switch_count: 0,
        managed_by_subscription: false,
        created_at: '2026-05-22T00:00:00Z',
        updated_at: '2026-05-22T00:00:00Z',
      }
      state.createdProxies.push(created)
      state.proxies = {
        ...state.proxies,
        items: [created, ...state.proxies.items],
        total: state.proxies.total + 1,
      }
      state.allProxies = [created, ...state.allProxies]
      await route.fulfill(jsonResponse(apiSuccess(created)))
      return
    }

    if (matches(pathname, '/api/v1/admin/proxies') && method === 'PUT') {
      const payload = request.postDataJSON() || {}
      const id = Number(pathname.split('/').pop())
      const current = state.proxies.items.find((proxy) => proxy.id === id)

      if (!current) {
        await route.fulfill(jsonResponse({ code: 404, message: 'proxy not found', data: null }, 404))
        return
      }

      const updated = {
        ...current,
        ...payload,
        updated_at: '2026-05-22T00:00:00Z',
      }

      state.proxies = {
        ...state.proxies,
        items: state.proxies.items.map((proxy) => (proxy.id === id ? updated : proxy)),
      }
      state.allProxies = state.allProxies.map((proxy) => (proxy.id === id ? updated : proxy))

      await route.fulfill(jsonResponse(apiSuccess(updated)))
      return
    }

    if (pathname === '/api/v1/admin/proxies/mihomo' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess(state.mihomoStatus)))
      return
    }

    if (pathname === '/api/v1/admin/proxies/data' && method === 'GET') {
      await route.fulfill(jsonResponse(apiSuccess({ version: '1', exported_at: '2026-05-22T00:00:00Z', proxies: state.proxies.items })))
      return
    }

    if (matches(pathname, '/api/v1/admin/proxies') && method === 'POST') {
      await route.fulfill(jsonResponse(apiSuccess({ success: true })))
      return
    }

    if (matches(pathname, '/api/v1/admin/proxies') && method === 'DELETE') {
      await route.fulfill(jsonResponse(apiSuccess({ message: 'deleted' })))
      return
    }

    await route.continue()
  })

  return state
}
