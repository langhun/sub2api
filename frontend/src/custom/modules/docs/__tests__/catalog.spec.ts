import { describe, expect, it } from 'vitest'

import { aiApiPayBaselineSlugs, docsCatalog, resolveDocsAsset } from '../catalog'

describe('documentation catalog', () => {
  it('locks the fourteen AI API Pay baseline topics', () => {
    expect(aiApiPayBaselineSlugs).toHaveLength(14)
    expect(docsCatalog.map((entry) => entry.slug)).toEqual(expect.arrayContaining([...aiApiPayBaselineSlugs]))
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
      expect(entry.related.every((slug) => docsCatalog.some((related) => related.slug === slug))).toBe(true)
    }
  })

  it('resolves every registered local screenshot', () => {
    const screenshots = docsCatalog.flatMap((entry) => entry.screenshots)
    const uniqueAssets = new Set(screenshots.map((screenshot) => screenshot.asset))

    expect(screenshots.length).toBeGreaterThanOrEqual(8)
    expect(uniqueAssets.size).toBe(8)

    for (const screenshot of screenshots) {
      expect(resolveDocsAsset(screenshot.asset)).toBeTruthy()
      expect(screenshot.capture).toBe('generic-ui')
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

  it('contains no external images, old domains, or credential-like keys', () => {
    for (const entry of docsCatalog) {
      for (const body of Object.values(entry.body)) {
        expect(body).not.toMatch(/!\[[^\]]*\]\((?!asset:)[^)]+\)/i)
        expect(body).not.toMatch(/api\.your-domain\.com|old\.aiapipay\.com/i)
        expect(body).not.toMatch(/\bsk-(?!demo-)[A-Za-z0-9_-]{12,}\b/)
        expect(body).not.toMatch(/<\s*(?:script|iframe|object|embed|form|img|video|audio|source|picture)\b|\bon(?:error|load|click)\s*=|(?:javascript|vbscript|data:text\/html)\s*:/i)
        expect(body).not.toMatch(/\b(?:xai-[A-Za-z0-9_-]{12,}|eyJ[A-Za-z0-9_-]{16,}\.)\b/)
      }
    }
  })
})
