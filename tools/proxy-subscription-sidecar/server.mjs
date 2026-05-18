import http from 'node:http'
import { spawn } from 'node:child_process'
import { promises as fs } from 'node:fs'
import path from 'node:path'
import os from 'node:os'
import { execFileSync } from 'node:child_process'

const PORT = Number(process.env.PORT || '8080')
const LISTENER_HOST = process.env.LISTENER_HOST || '127.0.0.1'
const PORT_RANGE = process.env.LISTENER_PORT_RANGE || '21080-21180'
const DATA_DIR = process.env.DATA_DIR || path.join(os.tmpdir(), 'proxy-subscription-sidecar')
function resolveSingBoxBin() {
  if (process.env.SING_BOX_BIN) {
    return process.env.SING_BOX_BIN
  }
  try {
    const output = execFileSync('cmd.exe', ['/d', '/c', 'where sing-box'], {
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'ignore']
    }).trim()
    const first = output.split(/\r?\n/).find(Boolean)
    if (first) {
      return first.trim()
    }
  } catch {
    // ignore
  }
  return 'sing-box'
}

const SING_BOX_BIN = resolveSingBoxBin()

const runtimes = new Map()

function parseRange(raw) {
  const [startRaw, endRaw] = String(raw).split('-', 2)
  const start = Number(startRaw)
  const end = Number(endRaw || startRaw)
  if (!Number.isFinite(start) || !Number.isFinite(end) || start <= 0 || end < start) {
    return { start: 21080, end: 21180 }
  }
  return { start, end }
}

function json(res, status, payload) {
  const body = JSON.stringify(payload)
  res.writeHead(status, {
    'content-type': 'application/json; charset=utf-8',
    'content-length': Buffer.byteLength(body)
  })
  res.end(body)
}

function readBody(req) {
  return new Promise((resolve, reject) => {
    let chunks = ''
    req.setEncoding('utf8')
    req.on('data', chunk => {
      chunks += chunk
      if (chunks.length > 1024 * 1024) {
        reject(new Error('request body too large'))
      }
    })
    req.on('end', () => {
      try {
        resolve(chunks ? JSON.parse(chunks) : {})
      } catch (error) {
        reject(error)
      }
    })
    req.on('error', reject)
  })
}

function allocatePort() {
  const { start, end } = parseRange(PORT_RANGE)
  const used = new Set([...runtimes.values()].map(item => item.listener_port))
  for (let port = start; port <= end; port += 1) {
    if (!used.has(port)) {
      return port
    }
  }
  throw new Error('no free listener ports available')
}

function inboundTag(runtimeId) {
  return `in-${runtimeId}`
}

function outboundTag(runtimeId) {
  return `out-${runtimeId}`
}

function buildShadowsocksOutbound(tag, runtime) {
  const cfg = runtime.config || {}
  const username = cfg.username
  const password = cfg.password || cfg.password_base64 || ''
  return {
    type: 'shadowsocks',
    tag,
    server: runtime.server,
    server_port: runtime.port,
    method: cfg.method || 'aes-256-gcm',
    password,
    ...(username ? { plugin: '', plugin_opts: '', users: [{ name: username, password }] } : {})
  }
}

function buildTrojanOutbound(tag, runtime) {
  const cfg = runtime.config || {}
  return {
    type: 'trojan',
    tag,
    server: runtime.server,
    server_port: runtime.port,
    password: cfg.password || extractPasswordFromURI(cfg.uri) || '',
    tls: {
      enabled: true,
      server_name: cfg.sni || cfg.server_name || runtime.server,
      insecure: Boolean(cfg.allow_insecure || cfg.insecure)
    }
  }
}

function buildVmessOutbound(tag, runtime) {
  const cfg = runtime.config || {}
  return {
    type: 'vmess',
    tag,
    server: runtime.server,
    server_port: runtime.port,
    uuid: cfg.uuid || extractUUIDFromURI(cfg.uri) || '',
    security: cfg.security || 'auto',
    alter_id: Number(cfg.alter_id || 0),
    tls: cfg.tls === false ? undefined : {
      enabled: true,
      server_name: cfg.sni || cfg.server_name || runtime.server,
      insecure: Boolean(cfg.allow_insecure || cfg.insecure)
    },
    transport: cfg.ws_path || cfg.transport === 'ws'
      ? {
          type: 'ws',
          path: cfg.ws_path || '/',
          headers: cfg.headers || {}
        }
      : undefined
  }
}

function buildVlessOutbound(tag, runtime) {
  const cfg = runtime.config || {}
  return {
    type: 'vless',
    tag,
    server: runtime.server,
    server_port: runtime.port,
    uuid: cfg.uuid || extractUUIDFromURI(cfg.uri) || '',
    flow: cfg.flow || '',
    tls: cfg.tls === false ? undefined : {
      enabled: true,
      server_name: cfg.sni || cfg.server_name || runtime.server,
      insecure: Boolean(cfg.allow_insecure || cfg.insecure)
    },
    transport: cfg.ws_path || cfg.transport === 'ws'
      ? {
          type: 'ws',
          path: cfg.ws_path || '/',
          headers: cfg.headers || {}
        }
      : undefined
  }
}

function buildHysteria2Outbound(tag, runtime) {
  const cfg = runtime.config || {}
  return {
    type: 'hysteria2',
    tag,
    server: runtime.server,
    server_port: runtime.port,
    password: cfg.password || '',
    obfs: cfg.obfs_password
      ? {
          type: cfg.obfs_type || 'salamander',
          password: cfg.obfs_password
        }
      : undefined,
    tls: {
      enabled: true,
      server_name: cfg.sni || cfg.server_name || runtime.server,
      insecure: Boolean(cfg.allow_insecure || cfg.insecure)
    }
  }
}

function buildOutbound(runtime) {
  const tag = outboundTag(runtime.runtime_id)
  switch (runtime.node_type) {
    case 'ss':
      return buildShadowsocksOutbound(tag, runtime)
    case 'trojan':
      return buildTrojanOutbound(tag, runtime)
    case 'vmess':
      return buildVmessOutbound(tag, runtime)
    case 'vless':
      return buildVlessOutbound(tag, runtime)
    case 'hysteria2':
    case 'hysteria':
      return buildHysteria2Outbound(tag, runtime)
    default:
      throw new Error(`unsupported runtime node_type: ${runtime.node_type}`)
  }
}

function buildConfig(runtime) {
  return {
    log: {
      level: 'warn'
    },
    inbounds: [
      {
        type: 'socks',
        tag: inboundTag(runtime.runtime_id),
        listen: runtime.listener_host,
        listen_port: runtime.listener_port
      }
    ],
    outbounds: [
      buildOutbound(runtime),
      {
        type: 'direct',
        tag: 'direct'
      }
    ],
    route: {
      final: outboundTag(runtime.runtime_id)
    }
  }
}

async function ensureDataDir() {
  await fs.mkdir(DATA_DIR, { recursive: true })
}

function runtimeConfigPath(runtimeId) {
  return path.join(DATA_DIR, `${runtimeId}.json`)
}

async function stopRuntime(runtimeId) {
  const existing = runtimes.get(runtimeId)
  if (!existing) return
  if (existing.process && !existing.process.killed) {
    existing.process.kill('SIGTERM')
  }
  runtimes.delete(runtimeId)
}

async function startRuntime(runtime) {
  await ensureDataDir()
  const configPath = runtimeConfigPath(runtime.runtime_id)
  const config = buildConfig(runtime)
  await fs.writeFile(configPath, JSON.stringify(config, null, 2), 'utf8')

  let process = null
  let launch_error = ''
  try {
    process = spawn(SING_BOX_BIN, ['run', '-c', configPath], {
      stdio: 'ignore',
      windowsHide: true
    })
    process.on('error', error => {
      const current = runtimes.get(runtime.runtime_id)
      if (current) {
        current.running = false
        current.launch_error = error instanceof Error ? error.message : 'failed to start sing-box'
      }
    })
    process.on('exit', () => {
      const current = runtimes.get(runtime.runtime_id)
      if (current && current.process === process) {
        current.running = false
      }
    })
  } catch (error) {
    launch_error = error instanceof Error ? error.message : 'failed to start sing-box'
  }

  return {
    ...runtime,
    config_path: configPath,
    process,
    running: Boolean(process),
    launch_error
  }
}

function extractPasswordFromURI(uri) {
  if (!uri) return ''
  const parts = String(uri).split('://', 2)
  if (parts.length !== 2) return ''
  const body = parts[1]
  const atIndex = body.indexOf('@')
  if (atIndex === -1) return ''
  const creds = body.slice(0, atIndex)
  const decoded = tryBase64Decode(creds)
  if (decoded.includes(':')) {
    return decoded.split(':').slice(1).join(':')
  }
  return ''
}

function extractUUIDFromURI(uri) {
  if (!uri) return ''
  const parsed = String(uri)
  const match = parsed.match(/[0-9a-fA-F-]{32,36}/)
  return match ? match[0] : ''
}

function tryBase64Decode(value) {
  try {
    return Buffer.from(value, 'base64').toString('utf8')
  } catch {
    return ''
  }
}

const server = http.createServer(async (req, res) => {
  try {
    if (req.method === 'GET' && req.url === '/healthz') {
      return json(res, 200, {
        ok: true,
        runtimes: runtimes.size,
        data_dir: DATA_DIR,
        sing_box_bin: SING_BOX_BIN
      })
    }

    if (req.method === 'POST' && req.url === '/v1/runtimes/upsert') {
      const body = await readBody(req)
      const runtimeId = String(body.runtime_id || '').trim()
      if (!runtimeId) {
        return json(res, 400, { error: 'runtime_id is required' })
      }

      await stopRuntime(runtimeId)
      const existing = runtimes.get(runtimeId)
      const listenerPort = existing?.listener_port ?? allocatePort()
      const runtime = {
        runtime_id: runtimeId,
        node_type: body.node_type || '',
        display_name: body.display_name || '',
        server: body.server || '',
        port: Number(body.port || 0),
        config: body.config || {},
        listener_host: body.listener_host || LISTENER_HOST,
        listener_port: listenerPort,
        protocol: 'socks5h',
        updated_at: new Date().toISOString()
      }
      const started = await startRuntime(runtime)
      runtimes.set(runtimeId, started)
      return json(res, 200, {
        runtime_id: started.runtime_id,
        node_type: started.node_type,
        display_name: started.display_name,
        server: started.server,
        port: started.port,
        listener_host: started.listener_host,
        listener_port: started.listener_port,
        protocol: started.protocol,
        running: started.running,
        launch_error: started.launch_error,
        updated_at: started.updated_at
      })
    }

    if (req.method === 'DELETE' && req.url?.startsWith('/v1/runtimes/')) {
      const parts = req.url.split('/')
      const runtimeId = decodeURIComponent(parts[3] || '')
      await stopRuntime(runtimeId)
      return json(res, 200, { deleted: true, runtime_id: runtimeId })
    }

    if (req.method === 'POST' && req.url?.startsWith('/v1/runtimes/') && req.url.endsWith('/check')) {
      const parts = req.url.split('/')
      const runtimeId = decodeURIComponent(parts[3] || '')
      const runtime = runtimes.get(runtimeId)
      if (!runtime) {
        return json(res, 404, { error: 'runtime not found' })
      }
      return json(res, 200, {
        ok: true,
        runtime_id: runtimeId,
        listener_port: runtime.listener_port,
        running: runtime.running,
        launch_error: runtime.launch_error || ''
      })
    }

    json(res, 404, { error: 'not found' })
  } catch (error) {
    json(res, 500, { error: error instanceof Error ? error.message : 'internal error' })
  }
})

server.listen(PORT, '0.0.0.0', async () => {
  await ensureDataDir()
  console.log(`proxy-subscription-sidecar listening on :${PORT}`)
})
