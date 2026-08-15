export type DocsAudience = 'user' | 'admin'
export type DocsCategory = 'start' | 'api' | 'sdk' | 'clients' | 'account' | 'support'
export type DocsLocale = 'zh' | 'en'

export interface DocsSource {
  label: string
  url?: string
  verifiedAt: string
}

export interface DocsScreenshot {
  asset: string
  client: string
  capture: 'generic-ui'
  version: string
  capturedAt: string
  slug: string
  alt: { zh: string; en: string }
  caption: { zh: string; en: string }
}

export interface DocsEntry {
  slug: string
  category: DocsCategory
  audience: DocsAudience
  title: { zh: string; en: string }
  summary: { zh: string; en: string }
  order: number
  updatedAt: string
  clientVersion: string
  sources: DocsSource[]
  screenshots: DocsScreenshot[]
  related: string[]
  body: { zh: string; en: string }
}

const markdownModules = import.meta.glob('./content/**/*.md', {
  eager: true,
  query: '?raw',
  import: 'default',
}) as Record<string, string>

const assetModules = import.meta.glob('./assets/**/*.{png,webp}', {
  eager: true,
  query: '?url',
  import: 'default',
}) as Record<string, string>

function readMarkdown(locale: DocsLocale, slug: string): string {
  return markdownModules[`./content/${locale}/${slug}.md`] || ''
}

export function resolveDocsAsset(path: string): string {
  const normalized = path.replace(/^asset:/, '').replace(/^\.\//, '')
  return assetModules[`./assets/${normalized}`] || ''
}

const verifiedAt = '2026-08-15'
const repositorySource: DocsSource = {
  label: 'Sub2API gateway and frontend implementation',
  verifiedAt,
}

const official = (label: string, url: string): DocsSource => ({ label, url, verifiedAt })

function screenshot(
  slug: string,
  asset: string,
  client: string,
  version: string,
  zh: string,
  en: string,
): DocsScreenshot {
  return {
    asset,
    client,
    capture: 'generic-ui',
    version,
    capturedAt: verifiedAt,
    slug,
    alt: { zh, en },
    caption: { zh, en },
  }
}

const entries: Omit<DocsEntry, 'body'>[] = [
  {
    slug: 'quick-start', category: 'start', audience: 'user', order: 10,
    title: { zh: '第一次调用', en: 'Your first request' },
    summary: { zh: '从创建 API Key 到收到第一条响应。', en: 'Create a key and receive your first response.' },
    updatedAt: verifiedAt, clientVersion: '本站网页 2026-08', sources: [repositorySource],
    screenshots: [screenshot('quick-start', 'common/api-key-create.png', '本站 API 密钥页面', '2026-08', '创建 API Key 时需要填写的四个关键位置', 'The four key fields when creating an API key')],
    related: ['api-keys', 'authentication', 'chat-completions'],
  },
  {
    slug: 'install-guide', category: 'start', audience: 'user', order: 20,
    title: { zh: '客户端安装指南', en: 'Client installation' },
    summary: { zh: '安装常用命令行客户端并确认环境可用。', en: 'Install common command-line clients and verify the environment.' },
    updatedAt: verifiedAt, clientVersion: 'Codex/Claude Code/Gemini CLI 2026-08',
    sources: [official('OpenAI Codex', 'https://developers.openai.com/codex/cli'), official('Claude Code', 'https://code.claude.com/docs/en/setup'), official('Gemini CLI', 'https://github.com/google-gemini/gemini-cli')],
    screenshots: [], related: ['sdk-openai', 'sdk-anthropic', 'sdk-gemini'],
  },
  {
    slug: 'api-keys', category: 'start', audience: 'user', order: 30,
    title: { zh: 'API Key 与分组', en: 'API keys and groups' },
    summary: { zh: '配置分组、额度、有效期和 IP 限制。', en: 'Configure groups, quotas, expiration, and IP restrictions.' },
    updatedAt: verifiedAt, clientVersion: '本站网页 2026-08', sources: [repositorySource],
    screenshots: [screenshot('api-keys', 'common/api-key-create.png', '本站 API 密钥页面', '2026-08', 'API Key 创建与分组设置示意图', 'API key creation and group settings')],
    related: ['quick-start', 'authentication', 'models'],
  },
  {
    slug: 'authentication', category: 'start', audience: 'user', order: 40,
    title: { zh: '鉴权与 Base URL', en: 'Authentication and Base URL' },
    summary: { zh: '按 OpenAI、Anthropic 和 Gemini 协议正确传递密钥。', en: 'Send credentials correctly for OpenAI, Anthropic, and Gemini protocols.' },
    updatedAt: verifiedAt, clientVersion: 'Gateway 2026-08', sources: [repositorySource], screenshots: [],
    related: ['api-keys', 'errors', 'security'],
  },
  {
    slug: 'chat-completions', category: 'api', audience: 'user', order: 50,
    title: { zh: 'Chat Completions', en: 'Chat Completions' },
    summary: { zh: '调用 OpenAI 兼容的对话接口。', en: 'Call the OpenAI-compatible chat endpoint.' },
    updatedAt: verifiedAt, clientVersion: 'OpenAI-compatible 2026-08', sources: [repositorySource], screenshots: [],
    related: ['streaming', 'sdk-openai', 'grok'],
  },
  {
    slug: 'messages', category: 'api', audience: 'user', order: 60,
    title: { zh: 'Anthropic Messages', en: 'Anthropic Messages' },
    summary: { zh: '使用 /v1/messages 与 Anthropic SDK。', en: 'Use /v1/messages and the Anthropic SDK.' },
    updatedAt: verifiedAt, clientVersion: 'Anthropic API 2023-06-01', sources: [official('Anthropic Messages API', 'https://docs.anthropic.com/en/api/messages'), repositorySource], screenshots: [],
    related: ['sdk-anthropic', 'streaming', 'authentication'],
  },
  {
    slug: 'embeddings', category: 'api', audience: 'user', order: 70,
    title: { zh: 'Embeddings', en: 'Embeddings' },
    summary: { zh: '生成向量并了解上游相关限制。', en: 'Generate embeddings and understand upstream-specific limits.' },
    updatedAt: verifiedAt, clientVersion: 'OpenAI-compatible 2026-08', sources: [repositorySource], screenshots: [],
    related: ['authentication', 'models', 'errors'],
  },
  {
    slug: 'streaming', category: 'api', audience: 'user', order: 80,
    title: { zh: '流式响应', en: 'Streaming responses' },
    summary: { zh: '区分 Chat、Messages 与 Responses 的 SSE 事件。', en: 'Distinguish Chat, Messages, and Responses SSE events.' },
    updatedAt: verifiedAt, clientVersion: 'Gateway 2026-08', sources: [repositorySource], screenshots: [],
    related: ['chat-completions', 'messages', 'grok'],
  },
  {
    slug: 'sdk-openai', category: 'sdk', audience: 'user', order: 90,
    title: { zh: 'OpenAI SDK 与 Codex', en: 'OpenAI SDK and Codex' },
    summary: { zh: '配置 Python、Node.js 和 Codex。', en: 'Configure Python, Node.js, and Codex.' },
    updatedAt: verifiedAt, clientVersion: 'OpenAI SDK/Codex 2026-08', sources: [official('OpenAI Responses guide', 'https://developers.openai.com/api/docs/guides/responses-vs-chat-completions'), official('OpenAI Codex configuration', 'https://developers.openai.com/codex/config-file/config-reference')], screenshots: [],
    related: ['install-guide', 'chat-completions', 'grok-cli'],
  },
  {
    slug: 'sdk-anthropic', category: 'sdk', audience: 'user', order: 100,
    title: { zh: 'Anthropic SDK 与 Claude Code', en: 'Anthropic SDK and Claude Code' },
    summary: { zh: '配置 Anthropic SDK 和 Claude Code 网关。', en: 'Configure the Anthropic SDK and Claude Code gateway.' },
    updatedAt: verifiedAt, clientVersion: 'Anthropic SDK/Claude Code 2026-08', sources: [official('Claude Code LLM gateway', 'https://code.claude.com/docs/en/llm-gateway')], screenshots: [],
    related: ['install-guide', 'messages', 'grok-cli'],
  },
  {
    slug: 'sdk-gemini', category: 'sdk', audience: 'user', order: 110,
    title: { zh: 'Gemini SDK 与 CLI', en: 'Gemini SDK and CLI' },
    summary: { zh: '配置 Google GenAI SDK 和兼容端点。', en: 'Configure the Google GenAI SDK and compatible endpoints.' },
    updatedAt: verifiedAt, clientVersion: 'google-genai 2.18.1 / Gemini CLI 2026-08', sources: [official('Gemini API', 'https://ai.google.dev/api/generate-content'), official('Google Gen AI Python SDK', 'https://github.com/googleapis/python-genai')], screenshots: [],
    related: ['install-guide', 'authentication', 'models'],
  },
  {
    slug: 'new-api', category: 'sdk', audience: 'admin', order: 120,
    title: { zh: 'New API 上游配置', en: 'New API upstream setup' },
    summary: { zh: '把本站作为 New API 的 OpenAI 兼容上游。', en: 'Use this site as an OpenAI-compatible upstream in New API.' },
    updatedAt: verifiedAt, clientVersion: 'New API 2026-08', sources: [repositorySource], screenshots: [],
    related: ['authentication', 'chat-completions', 'grok'],
  },
  {
    slug: 'billing', category: 'account', audience: 'user', order: 130,
    title: { zh: '余额、额度与用量', en: 'Balance, quota, and usage' },
    summary: { zh: '区分账户余额、Key 额度、订阅窗口和用量记录。', en: 'Distinguish account balance, key quota, subscription windows, and usage records.' },
    updatedAt: verifiedAt, clientVersion: '本站网页 2026-08', sources: [repositorySource], screenshots: [],
    related: ['api-keys', 'errors', 'troubleshooting'],
  },
  {
    slug: 'errors', category: 'support', audience: 'user', order: 140,
    title: { zh: '错误码', en: 'Error codes' },
    summary: { zh: '按协议识别鉴权、额度、限流和上游错误。', en: 'Identify authentication, quota, rate-limit, and upstream errors by protocol.' },
    updatedAt: verifiedAt, clientVersion: 'Gateway 2026-08', sources: [repositorySource], screenshots: [],
    related: ['authentication', 'billing', 'troubleshooting'],
  },
  {
    slug: 'grok', category: 'api', audience: 'user', order: 150,
    title: { zh: 'Grok API 接入', en: 'Grok API integration' },
    summary: { zh: '使用本站 Key 调用 Grok 文本与媒体能力。', en: 'Call Grok text and media capabilities with a site key.' },
    updatedAt: verifiedAt, clientVersion: 'xAI API 2026-08', sources: [official('xAI developer documentation', 'https://docs.x.ai/developers/quickstart'), repositorySource],
    screenshots: [screenshot('grok', 'grok/grok-api-fields.png', '本站 Grok 配置', '2026-08', 'Grok Base URL、API Key 和模型字段', 'Grok Base URL, API key, and model fields')],
    related: ['grok-cli', 'streaming', 'models'],
  },
  {
    slug: 'grok-cli', category: 'clients', audience: 'user', order: 160,
    title: { zh: 'Grok CLI 与编程客户端', en: 'Grok CLI and coding clients' },
    summary: { zh: '配置 Grok CLI、Codex、Claude Code 与 OpenCode。', en: 'Configure Grok CLI, Codex, Claude Code, and OpenCode.' },
    updatedAt: verifiedAt, clientVersion: 'Grok Build/Codex/Claude Code/OpenCode 2026-08', sources: [official('Grok Build settings', 'https://docs.x.ai/build/settings'), official('OpenAI Codex configuration', 'https://developers.openai.com/codex/config-file/config-reference'), official('Claude Code LLM gateway', 'https://code.claude.com/docs/en/llm-gateway'), official('OpenCode providers', 'https://opencode.ai/docs/providers/'), repositorySource],
    screenshots: [
      screenshot('grok-cli', 'grok/grok-cli-config.png', 'Grok CLI 配置', '2026-08', 'Grok CLI responses 后端配置', 'Grok CLI responses backend configuration'),
      screenshot('grok-cli', 'grok/codex-provider.png', 'Codex 配置', '2026-08', 'Codex 自定义 Responses Provider 配置字段', 'Codex custom Responses provider fields'),
      screenshot('grok-cli', 'grok/claude-code-env.png', 'Claude Code 配置', '2026-08', 'Claude Code 网关环境变量与最小测试命令', 'Claude Code gateway variables and minimal test command'),
      screenshot('grok-cli', 'grok/opencode-provider.png', 'OpenCode 配置', '2026-08', 'OpenCode Responses Provider、环境变量和模型字段', 'OpenCode Responses provider, environment key, and model fields'),
    ],
    related: ['grok', 'sdk-openai', 'troubleshooting'],
  },
  {
    slug: 'sillytavern', category: 'clients', audience: 'user', order: 170,
    title: { zh: 'SillyTavern 接入', en: 'SillyTavern integration' },
    summary: { zh: '通过 Custom OpenAI-compatible 连接酒馆。', en: 'Connect through Custom OpenAI-compatible.' },
    updatedAt: verifiedAt, clientVersion: 'SillyTavern 1.18.0', sources: [official('SillyTavern OpenAI-compatible guide', 'https://docs.sillytavern.app/usage/api-connections/openai/'), repositorySource],
    screenshots: [screenshot('sillytavern', 'sillytavern/custom-openai.png', 'SillyTavern', '1.18.0', 'SillyTavern Custom OpenAI-compatible 字段和测试按钮', 'SillyTavern Custom OpenAI-compatible fields and test button')],
    related: ['tavernai', 'chat-completions', 'troubleshooting'],
  },
  {
    slug: 'tavernai', category: 'clients', audience: 'user', order: 180,
    title: { zh: '酒馆 AI / TavernAI 兼容接入', en: 'TavernAI-compatible clients' },
    summary: { zh: '用字段对照表适配不同酒馆客户端。', en: 'Use a field map for different TavernAI-compatible clients.' },
    updatedAt: verifiedAt, clientVersion: '通用 OpenAI-compatible 2026-08', sources: [official('TavernAI repository', 'https://github.com/TavernAI/TavernAI'), official('SillyTavern API connections', 'https://docs.sillytavern.app/usage/api-connections/'), repositorySource],
    screenshots: [screenshot('tavernai', 'tavernai/field-map.png', '通用酒馆客户端', '2026-08', '不同客户端中的 Base URL、API Key、模型和流式字段', 'Base URL, API key, model, and streaming fields in compatible clients')],
    related: ['sillytavern', 'authentication', 'troubleshooting'],
  },
  {
    slug: 'models', category: 'api', audience: 'user', order: 190,
    title: { zh: '模型与协议兼容性', en: 'Models and protocol compatibility' },
    summary: { zh: '从 /v1/models 和 Key 分组确认实际可用模型。', en: 'Confirm available models through /v1/models and key groups.' },
    updatedAt: verifiedAt, clientVersion: 'Gateway 2026-08', sources: [repositorySource], screenshots: [],
    related: ['api-keys', 'grok', 'troubleshooting'],
  },
  {
    slug: 'troubleshooting', category: 'support', audience: 'user', order: 200,
    title: { zh: '常见问题排查', en: 'Troubleshooting' },
    summary: { zh: '按 Key、分组、模型、端点和上游顺序定位问题。', en: 'Troubleshoot keys, groups, models, endpoints, and upstreams in order.' },
    updatedAt: verifiedAt, clientVersion: 'Gateway 2026-08', sources: [repositorySource], screenshots: [],
    related: ['errors', 'models', 'security'],
  },
  {
    slug: 'security', category: 'support', audience: 'user', order: 210,
    title: { zh: '密钥安全', en: 'Key security' },
    summary: { zh: '安全保存、轮换和吊销 API Key。', en: 'Store, rotate, and revoke API keys safely.' },
    updatedAt: verifiedAt, clientVersion: '本站网页 2026-08', sources: [repositorySource], screenshots: [],
    related: ['api-keys', 'authentication', 'troubleshooting'],
  },
]

export const baselineTopicSlugs = [
  'quick-start', 'api-keys', 'authentication', 'chat-completions', 'messages', 'embeddings', 'streaming',
  'sdk-openai', 'sdk-anthropic', 'sdk-gemini', 'new-api', 'billing', 'errors', 'install-guide',
] as const

export const docsCatalog: readonly DocsEntry[] = entries.map((entry) => ({
  ...entry,
  body: {
    zh: readMarkdown('zh', entry.slug),
    en: readMarkdown('en', entry.slug),
  },
}))

export function getDocsEntry(slug: string): DocsEntry | null {
  return docsCatalog.find((entry) => entry.slug === slug) || null
}

export function getDocsTitle(entry: DocsEntry, locale: DocsLocale): string {
  return entry.title[locale] || entry.title.en
}
