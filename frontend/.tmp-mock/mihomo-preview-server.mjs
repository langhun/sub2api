import http from 'node:http'
import { buildAuthResponse, buildPublicSettings, buildProxyList, buildProxySubscriptions, buildMihomoStatus, buildGroups, ADMIN_EMAIL, ADMIN_PASSWORD } from '../e2e/support/fixtures.js'

const auth = buildAuthResponse()
const publicSettings = buildPublicSettings()
const groups = buildGroups()
const proxies = buildProxyList()
proxies.items = proxies.items.map((proxy, index) => ({
  ...proxy,
  protocol: index === 0 ? 'socks5h' : proxy.protocol,
  name: index === 0 ? 'mihomo-41003-sg' : proxy.name,
  host: index === 0 ? '127.0.0.1' : proxy.host,
  port: index === 0 ? 41003 : proxy.port,
  username: index === 0 ? 'sub2api' : proxy.username,
  password: index === 0 ? 'preview-pass' : proxy.password,
  country: index === 0 ? '新加坡' : proxy.country,
  country_code: index === 0 ? 'SG' : proxy.country_code,
  region: index === 0 ? '新加坡' : proxy.region,
  city: index === 0 ? '新加坡' : proxy.city,
  managed_by_subscription: index === 0 ? true : proxy.managed_by_subscription,
  subscription_source_name: index === 0 ? 'sub' : proxy.subscription_source_name,
  subscription_node_type: index === 0 ? 'vmess' : proxy.subscription_node_type,
}))
const proxySubscriptions = {
  items: [
    {
      id: 1,
      name: 'sub',
      url: 'https://sub.example.com/api/file/chatgpt',
      source_format: 'clash_yaml',
      enabled: true,
      refresh_interval_hours: 6,
      target_entry_count: 6,
      auto_add_to_pool: true,
      last_refreshed_at: '2026-05-28T02:09:11.897746+08:00',
      last_success_at: '2026-05-28T02:09:11.897746+08:00',
      last_error: '',
      last_node_count: 26,
      last_materialized_proxy_count: 6,
      created_at: '2026-05-28T01:00:00+08:00',
      updated_at: '2026-05-28T02:09:11.897746+08:00',
    },
  ],
  total: 1,
  page: 1,
  page_size: 100,
  pages: 1,
}
const mihomoStatus = buildMihomoStatus()
mihomoStatus.config_path = '/app/data/mihomo/config.yaml'
mihomoStatus.available_regions = ['香港', '日本', '新加坡', '美国']
mihomoStatus.settings.target_host = 'mihomo-sub2api'
mihomoStatus.settings.controller_url = 'http://mihomo-sub2api:9097'
mihomoStatus.settings.listener_regions = ['香港', '日本', '新加坡', '美国']

const subscriptionNodes = [
  { id: 1, source_id: 1, node_key: 'hk-01', display_name: '香港节点 01', node_type: 'vmess', server: '1.1.1.1', port: 443, config_json: {}, landing_status: 'active', last_error: '', last_seen_at: '2026-05-28T02:09:11.897746+08:00', created_at: '2026-05-28T01:00:00+08:00', updated_at: '2026-05-28T02:09:11.897746+08:00' },
  { id: 2, source_id: 1, node_key: 'jp-01', display_name: '日本节点 01', node_type: 'vmess', server: '2.2.2.2', port: 443, config_json: {}, landing_status: 'active', last_error: '', last_seen_at: '2026-05-28T02:09:11.897746+08:00', created_at: '2026-05-28T01:00:00+08:00', updated_at: '2026-05-28T02:09:11.897746+08:00' },
]

function send(res, body, status = 200) {
  res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' })
  res.end(JSON.stringify({ code: 0, message: 'ok', data: body }))
}
function sendRaw(res, body, status = 200) {
  res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' })
  res.end(JSON.stringify(body))
}

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, 'http://127.0.0.1:18080')
  const path = url.pathname
  const method = req.method || 'GET'
  let raw = ''
  for await (const chunk of req) raw += chunk
  let json = {}
  try { json = raw ? JSON.parse(raw) : {} } catch {}

  if (path === '/setup/status' && method === 'GET') return send(res, { needs_setup: false })
  if (path === '/api/v1/settings/public' && method === 'GET') return send(res, publicSettings)
  if (path === '/api/v1/auth/login' && method === 'POST') {
    if (json.email === ADMIN_EMAIL && json.password === ADMIN_PASSWORD) return send(res, auth)
    return sendRaw(res, { code: 40101, message: 'invalid credentials', data: null }, 401)
  }
  if (path === '/api/v1/auth/me' && method === 'GET') return send(res, auth.user)
  if (path === '/api/v1/auth/logout' && method === 'POST') return send(res, { message: 'logged out' })
  if (path === '/api/v1/subscriptions/active' && method === 'GET') return send(res, [])
  if (path === '/api/v1/announcements' && method === 'GET') return send(res, [])
  if (path === '/api/v1/checkin/status' && method === 'GET') return send(res, { enabled: false, can_checkin: false })
  if (path === '/api/v1/admin/groups/all' && method === 'GET') return send(res, groups)
  if (path === '/api/v1/admin/proxies' && method === 'GET') return send(res, proxies)
  if (path === '/api/v1/admin/proxies/mihomo' && method === 'GET') return send(res, mihomoStatus)
  if ((path === '/api/v1/admin/proxy-subscriptions' || path === '/api/v1/admin/proxies/subscriptions') && method === 'GET') return send(res, proxySubscriptions)
  if ((path === '/api/v1/admin/proxy-subscriptions/1/nodes' || path === '/api/v1/admin/proxies/subscriptions/1/nodes') && method === 'GET') return send(res, subscriptionNodes)
  if ((path === '/api/v1/admin/proxy-subscriptions' || path === '/api/v1/admin/proxies/subscriptions') && method === 'POST') return send(res, { ...proxySubscriptions.items[0], id: 2, ...json })
  if (path === '/api/v1/admin/proxies/mihomo/sync' && method === 'POST') return send(res, { config_path: mihomoStatus.config_path, proxies: [], created: 1, reused: 5, reloaded: true })
  if ((path.startsWith('/api/v1/admin/proxy-subscriptions/') || path.startsWith('/api/v1/admin/proxies/subscriptions/')) && method === 'POST') return send(res, { source_id: 1, node_count: 26, materialized_proxy_count: 6, created_proxy_count: 1, updated_proxy_count: 5, disabled_proxy_count: 0, deleted_proxy_count: 0, skipped_node_count: 0, conflict_node_count: 0, unsupported_node_count: 0, errors: [] })
  if ((path.startsWith('/api/v1/admin/proxy-subscriptions/') || path.startsWith('/api/v1/admin/proxies/subscriptions/')) && method === 'PUT') return send(res, { ...proxySubscriptions.items[0], ...json })
  if ((path.startsWith('/api/v1/admin/proxy-subscriptions/') || path.startsWith('/api/v1/admin/proxies/subscriptions/')) && method === 'DELETE') return send(res, { message: 'deleted' })
  if (path === '/api/v1/admin/proxies/mihomo' && method === 'PUT') return send(res, json)
  return sendRaw(res, { code: 404, message: `unhandled ${method} ${path}`, data: null }, 404)
})

server.listen(18080, '127.0.0.1', () => {
  console.log('mock-server:18080')
})
