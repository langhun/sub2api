import { describe, expect, it } from 'vitest'

import { baselineTopicSlugs, docsCatalog, resolveDocsAsset, type DocsSourceKind } from '../catalog'

describe('documentation catalog', () => {
  it('locks the fourteen core baseline topics', () => {
    expect(baselineTopicSlugs).toHaveLength(14)
    expect(docsCatalog.map((entry) => entry.slug)).toEqual(expect.arrayContaining([...baselineTopicSlugs]))
  })

  it('contains exactly twenty-one complete bilingual topics', () => {
    expect(docsCatalog).toHaveLength(21)
    expect(new Set(docsCatalog.map((entry) => entry.slug)).size).toBe(21)
    expect(new Set(docsCatalog.map((entry) => entry.order)).size).toBe(21)

    for (const entry of docsCatalog) {
      expect(entry.body.zh).toContain('<!-- beginner -->')
      expect(entry.body.zh).toContain('<!-- reference -->')
      expect(entry.body.en).toContain('<!-- beginner -->')
      expect(entry.body.en).toContain('<!-- reference -->')
      expect(entry.updatedAt).toMatch(/^\d{4}-\d{2}-\d{2}$/)
      expect(entry.clientVersion).not.toHaveLength(0)
      expect(entry.sources.length).toBeGreaterThan(0)
      expect(entry.sources.every((source) => /^\d{4}-\d{2}-\d{2}$/.test(source.verifiedAt))).toBe(true)
      expect(entry.sources.every((source) => source.url === undefined || source.url.startsWith('https://'))).toBe(true)
      expect(entry.sources.every((source) => ['official', 'github', 'implementation'].includes(source.kind))).toBe(true)
      expect(entry.sources.every((source) => source.kind === 'implementation' || Boolean(source.url))).toBe(true)
      expect(entry.related.every((slug) => docsCatalog.some((related) => related.slug === slug))).toBe(true)
    }
  })

  it('uses typed, recently verified sources for third-party integrations', () => {
    const expectedSources: Record<string, Array<{ kind: DocsSourceKind; url: string }>> = {
      grok: [{ kind: 'official', url: 'https://docs.x.ai/developers/quickstart' }],
      'grok-cli': [
        { kind: 'official', url: 'https://docs.x.ai/build/settings' },
        { kind: 'official', url: 'https://docs.anthropic.com/en/docs/claude-code/authentication' },
        { kind: 'official', url: 'https://opencode.ai/docs/providers/' },
        { kind: 'official', url: 'https://docs.openclaw.ai/concepts/model-providers' },
        { kind: 'github', url: 'https://github.com/openclaw/openclaw' },
        { kind: 'official', url: 'https://hermes-agent.nousresearch.com/docs/zh-Hans/integrations/providers' },
        { kind: 'github', url: 'https://github.com/NousResearch/hermes-agent' },
        { kind: 'official', url: 'https://www.workbuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/Model' },
        { kind: 'official', url: 'https://zcode.z.ai/cn/docs/configuration' },
      ],
      'sdk-anthropic': [
        { kind: 'official', url: 'https://docs.anthropic.com/en/api/client-sdks' },
        { kind: 'official', url: 'https://docs.anthropic.com/en/docs/claude-code/llm-gateway' },
      ],
      sillytavern: [
        { kind: 'official', url: 'https://docs.sillytavern.app/usage/api-connections/openai/' },
        { kind: 'github', url: 'https://github.com/SillyTavern/SillyTavern' },
      ],
      tavernai: [{ kind: 'github', url: 'https://github.com/TavernAI/TavernAI' }],
      'new-api': [
        { kind: 'official', url: 'https://docs.newapi.pro' },
        { kind: 'github', url: 'https://github.com/QuantumNous/new-api' },
      ],
    }

    for (const [slug, sources] of Object.entries(expectedSources)) {
      const entry = docsCatalog.find((item) => item.slug === slug)
      expect(entry).toBeTruthy()
      expect(entry!.updatedAt).toBe('2026-08-27')

      for (const expected of sources) {
        expect(entry!.sources).toContainEqual(expect.objectContaining({ ...expected, verifiedAt: '2026-08-27' }))
      }
    }
  })

  it('resolves every registered local screenshot', () => {
    const screenshots = docsCatalog.flatMap((entry) => entry.screenshots)
    const uniqueAssets = new Set(screenshots.map((screenshot) => screenshot.asset))

    expect(screenshots.length).toBeGreaterThanOrEqual(13)
    expect(uniqueAssets.size).toBe(13)

    for (const screenshot of screenshots) {
      expect(resolveDocsAsset(screenshot.asset)).toBeTruthy()
      expect(['reference-layout', 'real-client']).toContain(screenshot.capture)
      expect(screenshot.slug).toBe(docsCatalog.find((entry) => entry.screenshots.includes(screenshot))?.slug)
      expect(screenshot.capturedAt).toMatch(/^\d{4}-\d{2}-\d{2}$/)
      expect(screenshot.version).not.toHaveLength(0)
      expect(screenshot.client).not.toHaveLength(0)
      expect(screenshot.alt.zh).toBeTruthy()
      expect(screenshot.alt.en).toBeTruthy()
      expect(screenshot.caption.zh).toBeTruthy()
      expect(screenshot.caption.en).toBeTruthy()
    }
  })

  it('keeps every client integration as a complete ten-step beginner flow', () => {
    for (const slug of ['grok', 'grok-cli', 'sillytavern', 'tavernai']) {
      const entry = docsCatalog.find((item) => item.slug === slug)!
      expect(entry.body.zh.match(/^## 步骤 \d+：/gm)).toHaveLength(10)
      expect(entry.body.en.match(/^## Step \d+:/gm)).toHaveLength(10)
      expect(entry.body.zh).toContain('失败时先检查三项')
      expect(entry.body.en).toContain('Check these three items first')
      expect(entry.body.zh).toContain('X-Request-ID')
      expect(entry.body.en).toContain('X-Request-ID')
    }
  })

  it('covers the expanded coding clients and neutral service FAQs', () => {
    const clients = docsCatalog.find((entry) => entry.slug === 'grok-cli')!
    for (const name of ['OpenClaw', 'Hermes Agent', 'WorkBuddy', 'ZCode']) {
      expect(clients.body.zh).toContain(name)
      expect(clients.body.en).toContain(name)
    }

    const billing = docsCatalog.find((entry) => entry.slug === 'billing')!
    expect(billing.body.zh).toContain('分组默认倍率')
    expect(billing.body.zh).toContain('为什么没有退款按钮')
    expect(billing.body.en).toContain('Why is there no refund button?')

    const models = docsCatalog.find((entry) => entry.slug === 'models')!
    expect(models.body.zh).toContain('模型 ID 是调用契约')
    expect(models.body.en).toContain('A model ID is a request contract')
  })

  it('contains no external images, old domains, or credential-like keys', () => {
    for (const entry of docsCatalog) {
      for (const body of Object.values(entry.body)) {
        expect(body).not.toMatch(/!\[[^\]]*\]\((?!asset:)[^)]+\)/i)
        expect(body).not.toMatch(/api\.your-domain\.com|gateway\.example|old\.aiapipay\.com|aiapipay\.com/i)
        expect(body).not.toMatch(/\bsk-(?!demo-)[A-Za-z0-9_-]{12,}\b/)
        expect(body).not.toMatch(/<\s*(?:script|iframe|object|embed|form|img|video|audio|source|picture)\b|\bon(?:error|load|click)\s*=|(?:javascript|vbscript|data:text\/html)\s*:/i)
        expect(body).not.toMatch(/\b(?:xai-[A-Za-z0-9_-]{12,}|eyJ[A-Za-z0-9_-]{16,}\.)\b/)
        expect(body).not.toMatch(/\bkrill\b|api-slb\.krill-ai\.net|gpt-5\.6-sol|纯\s*pro\s*池|绝不降级|pure\s+pro\s+pool|no\s+downgrade/i)
      }
    }
  })
})
