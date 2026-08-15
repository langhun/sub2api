import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const moduleRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const catalogSource = readFileSync(join(moduleRoot, 'catalog.ts'), 'utf8')
const assetGeneratorSource = readFileSync(join(moduleRoot, 'scripts', 'generate-generic-assets.ps1'), 'utf8')
const baselineSlugs = [
  'quick-start', 'install-guide', 'api-keys', 'authentication', 'chat-completions', 'messages', 'embeddings', 'streaming',
  'sdk-openai', 'sdk-anthropic', 'sdk-gemini', 'new-api', 'billing', 'errors',
]
const slugs = [...baselineSlugs, 'grok', 'grok-cli', 'sillytavern', 'tavernai', 'models', 'troubleshooting', 'security']

const errors = []
const suspiciousKey = /\bsk-(?!demo-)[A-Za-z0-9_-]{12,}\b/g
const suspiciousCredential = /\b(?:xai-[A-Za-z0-9_-]{12,}|eyJ[A-Za-z0-9_-]{16,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,})\b/g
const exposedEmail = /\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b/gi
const dangerousMarkdown = /<\s*(?:script|iframe|object|embed|form|img|video|audio|source|picture)\b|\bon(?:error|load|click)\s*=|(?:javascript|vbscript|data:text\/html)\s*:/gi
const externalImage = /!\[[^\]]*\]\((?!asset:)[^)]+\)/gi
const assetReference = /asset:([A-Za-z0-9/_-]+\.(?:png|webp))/g
const documentedAsset = /!\[([^\]]+)\]\(asset:([A-Za-z0-9/_-]+\.(?:png|webp))\s+["']([^"']+)["']\)/g
const forbiddenSourceDomain = /(?:aiapipay|old\.aiapipay)\.com|api\.your-domain\.com/gi

const brandedApiAddresses = assetGeneratorSource.match(/https:\/\/api\.daigua\.icu(?:\/v1)?/g) || []
if (brandedApiAddresses.length !== 7) errors.push('generic illustrations must contain seven DaiGua public API addresses')
if (/api\.example\.com/i.test(assetGeneratorSource)) errors.push('generic illustrations still contain the placeholder API domain')

const verifiedAtMatch = catalogSource.match(/const verifiedAt = '(\d{4}-\d{2}-\d{2})'/)
if (!verifiedAtMatch) errors.push('catalog is missing a valid verification date')
if (baselineSlugs.length !== 14) errors.push('baseline topic list must contain fourteen slugs')
if (forbiddenSourceDomain.test(catalogSource)) errors.push('catalog contains a retired or placeholder source domain')

const sourceUrls = [
  ...catalogSource.matchAll(/\burl:\s*'(https?:\/\/[^']+)'/g),
  ...catalogSource.matchAll(/\bofficial\([^,]+,\s*'(https?:\/\/[^']+)'\)/g),
].map((match) => match[1])
if (!sourceUrls.length) errors.push('catalog is missing source URLs')
for (const url of sourceUrls) {
  if (!url.startsWith('https://')) errors.push(`source URL must use HTTPS: ${url}`)
  if (/\.(?:png|webp)(?:$|[?#])/i.test(url)) errors.push(`source URL points to an image: ${url}`)
}

const readmeSource = readFileSync(join(moduleRoot, 'README.md'), 'utf8')
if (forbiddenSourceDomain.test(readmeSource)) errors.push('README contains a retired or placeholder source domain')

for (const slug of slugs) {
  if (!catalogSource.includes(`slug: '${slug}'`)) errors.push(`catalog is missing ${slug}`)
  for (const locale of ['zh', 'en']) {
    const path = join(moduleRoot, 'content', locale, `${slug}.md`)
    if (!existsSync(path)) {
      errors.push(`missing ${locale}/${slug}.md`)
      continue
    }
    const content = readFileSync(path, 'utf8')
    if (!content.includes('<!-- beginner -->') || !content.includes('<!-- reference -->')) {
      errors.push(`${locale}/${slug}.md is missing view markers`)
    }
    if (externalImage.test(content)) errors.push(`${locale}/${slug}.md contains an external or unregistered image`)
    externalImage.lastIndex = 0
    if (suspiciousKey.test(content)) errors.push(`${locale}/${slug}.md contains a credential-like key`)
    suspiciousKey.lastIndex = 0
    if (suspiciousCredential.test(content)) errors.push(`${locale}/${slug}.md contains another credential-like value`)
    suspiciousCredential.lastIndex = 0
    if (exposedEmail.test(content)) errors.push(`${locale}/${slug}.md contains an email address`)
    exposedEmail.lastIndex = 0
    if (dangerousMarkdown.test(content)) errors.push(`${locale}/${slug}.md contains dangerous Markdown or HTML`)
    dangerousMarkdown.lastIndex = 0
    if (/api\.your-domain\.com|old\.aiapipay\.com|aiapipay\.com/i.test(content)) errors.push(`${locale}/${slug}.md contains a retired or placeholder domain`)
    const registeredAssets = new Set()
    for (const match of content.matchAll(assetReference)) {
      if (!existsSync(join(moduleRoot, 'assets', match[1]))) errors.push(`${locale}/${slug}.md references missing asset ${match[1]}`)
    }
    for (const match of content.matchAll(documentedAsset)) {
      if (!match[1].trim() || !match[3].trim()) errors.push(`${locale}/${slug}.md has an image without alt text or caption`)
      registeredAssets.add(match[2])
    }
    for (const match of content.matchAll(assetReference)) {
      if (!registeredAssets.has(match[1])) errors.push(`${locale}/${slug}.md image ${match[1]} needs alt text and a caption`)
    }
  }
}

for (const locale of ['zh', 'en']) {
  const actual = readdirSync(join(moduleRoot, 'content', locale)).filter((name) => name.endsWith('.md')).sort()
  const expected = slugs.map((slug) => `${slug}.md`).sort()
  if (JSON.stringify(actual) !== JSON.stringify(expected)) errors.push(`${locale} topic files do not match the catalog baseline`)
}

if (errors.length) {
  console.error(errors.map((error) => `- ${error}`).join('\n'))
  process.exit(1)
}

console.log(`Documentation check passed: ${slugs.length} bilingual topics.`)
