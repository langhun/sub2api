export type DocsAudience = 'user' | 'admin'
export type DocsCategory = 'start' | 'api' | 'sdk' | 'clients' | 'account' | 'support'
export type DocsLocale = 'zh' | 'en'
export type DocsSourceKind = 'official' | 'github' | 'implementation'

export interface DocsSource {
  kind: DocsSourceKind
  label: string
  url?: string
  verifiedAt: string
}

export interface DocsScreenshot {
  asset: string
  client: string
  capture: 'reference-layout' | 'real-client'
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

const verifiedAt = '2026-08-27'
const repositorySource: DocsSource = {
  kind: 'implementation',
  label: 'DaiGua gateway and frontend implementation',
  verifiedAt,
}

const official = (label: string, url: string): DocsSource => ({ kind: 'official', label, url, verifiedAt })
const github = (label: string, url: string): DocsSource => ({ kind: 'github', label, url, verifiedAt })

function screenshot(
  slug: string,
  asset: string,
  client: string,
  version: string,
  zh: string,
  en: string,
  capture: DocsScreenshot['capture'] = 'reference-layout',
): DocsScreenshot {
  return {
    asset,
    client,
    capture,
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
    sources: [official('OpenAI Codex', 'https://developers.openai.com/codex/cli'), official('Claude Code setup', 'https://docs.anthropic.com/en/docs/claude-code/setup'), github('Gemini CLI repository', 'https://github.com/google-gemini/gemini-cli')],
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
    title: { zh: 'OpenAI SDK 与 curl', en: 'OpenAI SDK and curl' },
    summary: { zh: '使用 curl、OpenAI Python 和 Node.js SDK 调用 Responses 或 Chat Completions。', en: 'Use curl and OpenAI Python or Node.js SDKs with Responses or Chat Completions.' },
    updatedAt: verifiedAt, clientVersion: 'OpenAI SDK 2026-08', sources: [official('OpenAI Responses guide', 'https://developers.openai.com/api/docs/guides/responses-vs-chat-completions'), official('OpenAI API libraries', 'https://developers.openai.com/api/docs/libraries')],
    screenshots: [screenshot('sdk-openai', 'sdk/openai-curl.png', 'OpenAI SDK 与 curl 字段参考', '2026-08', 'OpenAI SDK 和 curl 的 DaiGua Base URL、Key、模型与 Responses 最小请求参考图', 'DaiGua Base URL, key, model, and minimal Responses request reference for OpenAI SDK and curl')],
    related: ['install-guide', 'chat-completions', 'grok-cli'],
  },
  {
    slug: 'sdk-anthropic', category: 'sdk', audience: 'user', order: 100,
    title: { zh: 'Anthropic SDK 与 Claude Code', en: 'Anthropic SDK and Claude Code' },
    summary: { zh: '配置 Anthropic SDK 和 Claude Code 网关。', en: 'Configure the Anthropic SDK and Claude Code gateway.' },
    updatedAt: verifiedAt, clientVersion: 'Anthropic SDK/Claude Code 2026-08',
    sources: [official('Anthropic SDKs', 'https://docs.anthropic.com/en/api/client-sdks'), official('Anthropic Claude Code LLM gateway', 'https://docs.anthropic.com/en/docs/claude-code/llm-gateway'), official('Anthropic Claude Code authentication', 'https://docs.anthropic.com/en/docs/claude-code/authentication'), repositorySource],
    screenshots: [],
    related: ['install-guide', 'messages', 'grok-cli'],
  },
  {
    slug: 'sdk-gemini', category: 'sdk', audience: 'user', order: 110,
    title: { zh: 'Gemini SDK 与 CLI', en: 'Gemini SDK and CLI' },
    summary: { zh: '配置 Google GenAI SDK 和兼容端点。', en: 'Configure the Google GenAI SDK and compatible endpoints.' },
    updatedAt: verifiedAt, clientVersion: 'google-genai 2.18.1 / Gemini CLI 2026-08', sources: [official('Gemini API', 'https://ai.google.dev/api/generate-content'), github('Google Gen AI Python SDK', 'https://github.com/googleapis/python-genai')], screenshots: [],
    related: ['install-guide', 'authentication', 'models'],
  },
  {
    slug: 'new-api', category: 'sdk', audience: 'admin', order: 120,
    title: { zh: 'New API 上游配置', en: 'New API upstream setup' },
    summary: { zh: '把本站作为 New API 的 OpenAI 兼容上游。', en: 'Use this site as an OpenAI-compatible upstream in New API.' },
    updatedAt: verifiedAt, clientVersion: 'New API 2026-08',
    sources: [official('New API documentation', 'https://docs.newapi.pro'), github('New API repository', 'https://github.com/QuantumNous/new-api'), repositorySource],
    screenshots: [],
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
    updatedAt: verifiedAt, clientVersion: 'xAI API 2026-08', sources: [
      official('xAI developer quickstart', 'https://docs.x.ai/developers/quickstart'),
      official('xAI image generation', 'https://docs.x.ai/developers/model-capabilities/images/generation'),
      official('xAI video generation', 'https://docs.x.ai/developers/model-capabilities/video/generation'),
      official('xAI text to speech', 'https://docs.x.ai/developers/model-capabilities/audio/text-to-speech'),
      official('xAI speech to text', 'https://docs.x.ai/developers/model-capabilities/audio/speech-to-text'),
      official('xAI voice and realtime', 'https://docs.x.ai/developers/model-capabilities/audio/voice'),
      repositorySource,
    ],
    screenshots: [screenshot('grok', 'grok/grok-api-fields.png', '本站 Grok 配置', '2026-08', 'Grok Base URL、API Key 和模型字段', 'Grok Base URL, API key, and model fields')],
    related: ['grok-cli', 'streaming', 'models'],
  },
  {
    slug: 'grok-cli', category: 'clients', audience: 'user', order: 160,
    title: { zh: '编程客户端与 CLI 接入', en: 'Coding clients and CLI integration' },
    summary: { zh: '配置 Grok CLI、Codex、Claude Code、OpenCode、OpenClaw、Hermes、WorkBuddy 与 ZCode。', en: 'Configure Grok CLI, Codex, Claude Code, OpenCode, OpenClaw, Hermes, WorkBuddy, and ZCode.' },
    updatedAt: verifiedAt, clientVersion: 'Codex CLI 0.147.0；其他客户端官方文档核验于 2026-08-27',
    sources: [
      official('xAI Grok Build settings', 'https://docs.x.ai/build/settings'),
      official('OpenAI Codex configuration', 'https://developers.openai.com/codex/config-file/config-reference'),
      official('Anthropic Claude Code LLM gateway', 'https://docs.anthropic.com/en/docs/claude-code/llm-gateway'),
      official('Anthropic Claude Code authentication', 'https://docs.anthropic.com/en/docs/claude-code/authentication'),
      official('OpenCode providers', 'https://opencode.ai/docs/providers/'),
      official('OpenClaw model providers', 'https://docs.openclaw.ai/concepts/model-providers'),
      github('OpenClaw repository', 'https://github.com/openclaw/openclaw'),
      official('Hermes Agent providers', 'https://hermes-agent.nousresearch.com/docs/zh-Hans/integrations/providers'),
      github('Hermes Agent repository', 'https://github.com/NousResearch/hermes-agent'),
      official('WorkBuddy model configuration', 'https://www.workbuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/Model'),
      official('ZCode model configuration', 'https://zcode.z.ai/cn/docs/configuration'),
      repositorySource,
    ],
    screenshots: [
      screenshot('grok-cli', 'grok/grok-cli-config.png', 'Grok Build CLI 配置参考', '2026-08', 'Grok Build CLI 的 DaiGua 脱敏 Responses 配置参考图', 'DaiGua redacted Responses configuration reference for Grok Build CLI'),
      screenshot('grok-cli', 'grok/codex-provider.png', 'Codex CLI 配置核验', '0.147.0', 'Codex 0.147.0 自定义 Responses Provider 的 DaiGua 配置核验图', 'DaiGua custom Responses provider verification for Codex 0.147.0'),
      screenshot('grok-cli', 'grok/claude-code-env.png', 'Claude Code 配置参考', '2026-08', 'Claude Code 网关环境变量和最小测试的 DaiGua 脱敏配置参考图', 'DaiGua redacted Claude Code gateway variables and minimal test reference'),
      screenshot('grok-cli', 'grok/opencode-provider.png', 'OpenCode 配置参考', '2026-08', 'OpenCode Chat 兼容 Provider 的 DaiGua 脱敏配置参考图', 'DaiGua redacted Chat-compatible provider reference for OpenCode'),
      screenshot('grok-cli', 'clients/openclaw-provider.png', 'OpenClaw Model Providers 字段参考', '官方文档 2026-08', 'OpenClaw 自定义 Provider、Base URL、Key、协议和模型列表的 DaiGua 中文参考图', 'Chinese DaiGua reference for OpenClaw custom provider, Base URL, key, protocol, and model catalog'),
      screenshot('grok-cli', 'clients/hermes-custom-provider.png', 'Hermes Agent Custom Provider 字段参考', '官方文档 2026-08', 'Hermes Agent Custom Provider 向导的 DaiGua 中文字段参考图', 'Chinese DaiGua field reference for the Hermes Agent Custom Provider wizard'),
      screenshot('grok-cli', 'clients/workbuddy-custom-api.png', 'WorkBuddy 自定义 API 字段参考', '官方文档 2026-08', 'WorkBuddy 自定义 API、自动端点和模型能力开关的 DaiGua 中文参考图', 'Chinese DaiGua reference for WorkBuddy Custom API, automatic endpoint, and model capability options'),
      screenshot('grok-cli', 'clients/zcode-custom-provider.png', 'ZCode 自定义供应商字段参考', '官方文档 2026-08', 'ZCode 自定义 OpenAI 供应商地址、Key 和模型登记的 DaiGua 中文参考图', 'Chinese DaiGua reference for ZCode custom OpenAI provider URL, key, and model registration'),
    ],
    related: ['grok', 'sdk-openai', 'troubleshooting'],
  },
  {
    slug: 'sillytavern', category: 'clients', audience: 'user', order: 170,
    title: { zh: 'SillyTavern 接入', en: 'SillyTavern integration' },
    summary: { zh: '通过 Custom OpenAI-compatible 连接酒馆。', en: 'Connect through Custom OpenAI-compatible.' },
    updatedAt: verifiedAt, clientVersion: 'SillyTavern 1.18.0',
    sources: [official('SillyTavern OpenAI-compatible guide', 'https://docs.sillytavern.app/usage/api-connections/openai/'), github('SillyTavern repository', 'https://github.com/SillyTavern/SillyTavern'), repositorySource],
    screenshots: [screenshot('sillytavern', 'sillytavern/custom-openai.png', 'SillyTavern 字段布局参考', '1.18.0', 'SillyTavern Custom OpenAI-compatible 的 DaiGua 脱敏字段参考图', 'DaiGua redacted field reference for SillyTavern Custom OpenAI-compatible')],
    related: ['tavernai', 'chat-completions', 'troubleshooting'],
  },
  {
    slug: 'tavernai', category: 'clients', audience: 'user', order: 180,
    title: { zh: '酒馆 AI / TavernAI 兼容接入', en: 'TavernAI-compatible clients' },
    summary: { zh: '用字段对照表适配不同酒馆客户端。', en: 'Use a field map for different TavernAI-compatible clients.' },
    updatedAt: verifiedAt, clientVersion: '通用 OpenAI-compatible 2026-08', sources: [github('TavernAI repository', 'https://github.com/TavernAI/TavernAI'), official('SillyTavern API connections', 'https://docs.sillytavern.app/usage/api-connections/'), repositorySource],
    screenshots: [screenshot('tavernai', 'tavernai/field-map.png', '通用酒馆客户端字段参考', '2026-08', '酒馆兼容客户端中的 DaiGua Base URL、API Key、模型和流式字段参考图', 'DaiGua Base URL, API key, model, and streaming field reference for compatible Tavern clients')],
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
