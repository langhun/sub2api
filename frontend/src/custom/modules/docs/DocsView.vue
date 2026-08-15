<template>
  <AppLayout>
    <div class="docs-shell">
      <header class="docs-header">
        <div class="min-w-0">
          <div class="flex items-center gap-2.5">
            <Icon name="book" size="lg" class="shrink-0 text-primary-600 dark:text-primary-400" />
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('docs.pageTitle') }}</h1>
          </div>
          <p class="mt-1.5 max-w-2xl text-sm text-gray-500 dark:text-dark-400">
            {{ selectedEntry.summary[locale] }}
          </p>
        </div>

        <div class="docs-header-actions">
          <label class="docs-search">
            <Icon name="search" size="sm" class="shrink-0 text-gray-400" />
            <span class="sr-only">{{ t('docs.searchPlaceholder') }}</span>
            <input v-model.trim="search" type="search" :placeholder="t('docs.searchPlaceholder')" />
          </label>
          <button class="docs-menu-button" type="button" :aria-expanded="mobileMenuOpen" @click="mobileMenuOpen = !mobileMenuOpen">
            <Icon name="menu" size="sm" />
            {{ t('docs.menu') }}
          </button>
        </div>
      </header>

      <div class="docs-workspace">
        <aside :class="['docs-navigation', { 'docs-navigation-open': mobileMenuOpen }]">
          <div v-if="filteredEntries.length" class="space-y-5">
            <section v-for="group in navigationGroups" :key="group.category">
              <h2 class="docs-category-title">{{ t(`docs.categories.${group.category}`) }}</h2>
              <nav class="mt-1.5 space-y-0.5" :aria-label="t(`docs.categories.${group.category}`)">
                <button
                  v-for="entry in group.entries"
                  :key="entry.slug"
                  type="button"
                  :class="['docs-topic-link', { 'docs-topic-link-active': entry.slug === selectedEntry.slug }]"
                  @click="selectTopic(entry.slug)"
                >
                  <span>{{ entry.title[locale] }}</span>
                  <Icon v-if="entry.slug === selectedEntry.slug" name="chevronRight" size="xs" class="shrink-0" />
                </button>
              </nav>
            </section>
          </div>
          <p v-else class="px-3 py-8 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('docs.noResults') }}</p>
        </aside>

        <main class="docs-main">
          <div class="docs-article-header">
            <div class="min-w-0">
              <h2 class="break-words text-2xl font-semibold text-gray-900 dark:text-white">{{ selectedEntry.title[locale] }}</h2>
              <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">
                {{ t('docs.updated') }} {{ selectedEntry.updatedAt }} · {{ selectedEntry.clientVersion }}
              </p>
            </div>
            <div class="docs-view-switch" role="tablist" :aria-label="t('docs.pageTitle')">
              <button type="button" role="tab" :aria-selected="viewMode === 'beginner'" :class="{ active: viewMode === 'beginner' }" @click="viewMode = 'beginner'">
                {{ t('docs.beginner') }}
              </button>
              <button type="button" role="tab" :aria-selected="viewMode === 'reference'" :class="{ active: viewMode === 'reference' }" @click="viewMode = 'reference'">
                {{ t('docs.reference') }}
              </button>
            </div>
          </div>

          <article
            ref="articleElement"
            class="docs-markdown"
            data-testid="docs-content"
            v-html="renderedDocument.html"
            @click="handleArticleClick"
            @keydown="handleArticleKeydown"
          />

          <section v-if="visibleSources.length" class="docs-sources" :aria-label="t('docs.sources')">
            <h2>{{ t('docs.sources') }}</h2>
            <ul>
              <li v-for="source in visibleSources" :key="`${source.label}-${source.url || ''}`">
                <a v-if="safeExternalUrl(source.url)" :href="safeExternalUrl(source.url)" target="_blank" rel="noopener noreferrer">{{ source.label }}</a>
                <span v-else>{{ source.label }}</span>
                <time :datetime="source.verifiedAt">{{ source.verifiedAt }}</time>
              </li>
            </ul>
          </section>

          <section class="docs-related" :aria-label="t('docs.related')">
            <h2>{{ t('docs.related') }}</h2>
            <div class="docs-related-links">
              <button v-for="entry in relatedEntries" :key="entry.slug" type="button" @click="selectTopic(entry.slug)">
                {{ entry.title[locale] }}
                <Icon name="arrowRight" size="xs" />
              </button>
            </div>
          </section>
        </main>

        <aside class="docs-toc">
          <h2>{{ t('docs.onThisPage') }}</h2>
          <nav class="mt-2 space-y-1">
            <button
              v-for="item in renderedDocument.toc"
              :key="item.id"
              type="button"
              :class="{ 'pl-4': item.level === 3 }"
              @click="scrollToHeading(item.id)"
            >
              {{ item.text }}
            </button>
          </nav>
        </aside>
      </div>

      <Teleport to="body">
        <div v-if="previewImage" class="docs-image-modal" role="dialog" aria-modal="true" :aria-label="previewImage.alt" @click.self="closePreview" @keydown="handlePreviewKeydown">
          <button ref="previewCloseButton" type="button" :title="t('docs.closePreview')" :aria-label="t('docs.closePreview')" @click="closePreview">
            <Icon name="x" size="lg" />
          </button>
          <figure>
            <img :src="previewImage.src" :alt="previewImage.alt" />
            <figcaption v-if="previewImage.caption">{{ previewImage.caption }}</figcaption>
          </figure>
        </div>
      </Teleport>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'

import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  docsCatalog,
  getDocsEntry,
  resolveDocsAsset,
  type DocsCategory,
  type DocsEntry,
  type DocsLocale,
} from './catalog'

type DocsViewMode = 'beginner' | 'reference'
interface TocItem { id: string; text: string; level: number }
interface PreviewImage { src: string; alt: string; caption: string }

const categoryOrder: readonly DocsCategory[] = ['start', 'api', 'sdk', 'clients', 'account', 'support']
const route = useRoute()
const router = useRouter()
const { locale: i18nLocale, t } = useI18n()

const search = ref('')
const viewMode = ref<DocsViewMode>('beginner')
const mobileMenuOpen = ref(false)
const articleElement = ref<HTMLElement | null>(null)
const previewImage = ref<PreviewImage | null>(null)
const previewCloseButton = ref<HTMLButtonElement | null>(null)
let previewTrigger: HTMLElement | null = null
const allowedImageUrls = new Set(docsCatalog
  .flatMap((entry) => entry.screenshots.map((item) => resolveDocsAsset(item.asset)))
  .filter(Boolean))

const locale = computed<DocsLocale>(() => i18nLocale.value.toLowerCase().startsWith('zh') ? 'zh' : 'en')
const requestedSlug = computed(() => typeof route.params.topic === 'string' ? route.params.topic : 'quick-start')
const selectedEntry = computed(() => getDocsEntry(requestedSlug.value) || docsCatalog[0])

const filteredEntries = computed(() => {
  const query = search.value.toLocaleLowerCase()
  if (!query) return docsCatalog
  return docsCatalog.filter((entry) => [entry.title[locale.value], entry.summary[locale.value], entry.body[locale.value]]
    .some((value) => value.toLocaleLowerCase().includes(query)))
})

const navigationGroups = computed(() => categoryOrder
  .map((category) => ({ category, entries: filteredEntries.value.filter((entry) => entry.category === category) }))
  .filter((group) => group.entries.length))

const relatedEntries = computed(() => selectedEntry.value.related
  .map((slug) => getDocsEntry(slug))
  .filter((entry): entry is DocsEntry => entry !== null))

const visibleSources = computed(() => selectedEntry.value.sources.filter((source) => {
  if (!source.url) return true
  try {
    const hostname = new URL(source.url).hostname.toLowerCase()
    return hostname !== 'aiapipay.com' && !hostname.endsWith('.aiapipay.com')
  } catch {
    return false
  }
}))

const renderedDocument = computed(() => renderMarkdown(modeContent(selectedEntry.value.body[locale.value], viewMode.value)))

function safeExternalUrl(value?: string): string {
  if (!value) return ''
  try {
    return new URL(value).protocol === 'https:' ? value : ''
  } catch {
    return ''
  }
}

function modeContent(content: string, mode: DocsViewMode): string {
  const beginnerMarker = '<!-- beginner -->'
  const referenceMarker = '<!-- reference -->'
  const beginnerIndex = content.indexOf(beginnerMarker)
  const referenceIndex = content.indexOf(referenceMarker)
  if (beginnerIndex === -1 || referenceIndex === -1) return content
  return mode === 'beginner'
    ? content.slice(beginnerIndex + beginnerMarker.length, referenceIndex)
    : content.slice(referenceIndex + referenceMarker.length)
}

function replaceAssetReferences(content: string): string {
  return content.replace(/asset:([A-Za-z0-9/_-]+\.(?:png|webp))/g, (match, assetPath: string) => resolveDocsAsset(assetPath) || match)
}

function renderMarkdown(content: string): { html: string; toc: TocItem[] } {
  if (typeof DOMParser === 'undefined') return { html: '', toc: [] }
  const parsed = marked.parse(replaceAssetReferences(content), { gfm: true, breaks: false }) as string
  const sanitized = DOMPurify.sanitize(parsed, {
    FORBID_TAGS: ['script', 'style', 'iframe', 'object', 'embed', 'form', 'video', 'audio', 'source', 'picture'],
    FORBID_ATTR: ['style', 'srcset'],
  })
  const documentNode = new DOMParser().parseFromString(`<main>${sanitized}</main>`, 'text/html')
  const root = documentNode.body.firstElementChild as HTMLElement
  const toc: TocItem[] = []

  root.querySelectorAll<HTMLHeadingElement>('h2, h3').forEach((heading, index) => {
    const id = `docs-heading-${index}`
    heading.id = id
    toc.push({ id, text: heading.textContent?.trim() || '', level: Number(heading.tagName.slice(1)) })
  })

  root.querySelectorAll<HTMLPreElement>('pre').forEach((pre) => {
    const button = documentNode.createElement('button')
    button.type = 'button'
    button.className = 'docs-copy-button'
    button.dataset.docsCopy = 'true'
    button.textContent = t('docs.copyCode')
    pre.prepend(button)
  })

  root.querySelectorAll<HTMLImageElement>('img').forEach((img) => {
    const src = img.getAttribute('src') || ''
    if (!allowedImageUrls.has(src)) {
      img.remove()
      return
    }
    img.loading = 'lazy'
    img.decoding = 'async'
    img.dataset.docsImage = 'true'
    img.tabIndex = 0
    img.setAttribute('role', 'button')
    const captionText = img.getAttribute('title') || img.getAttribute('alt') || ''
    img.removeAttribute('title')
    const figure = documentNode.createElement('figure')
    figure.className = 'docs-figure'
    const parent = img.parentElement
    if (parent?.tagName === 'P' && parent.childElementCount === 1) parent.replaceWith(figure)
    else img.replaceWith(figure)
    figure.append(img)
    if (captionText) {
      const caption = documentNode.createElement('figcaption')
      caption.textContent = captionText
      figure.append(caption)
    }
  })

  root.querySelectorAll<HTMLAnchorElement>('a').forEach((link) => {
    if (/^https?:\/\//i.test(link.href)) {
      link.target = '_blank'
      link.rel = 'noopener noreferrer'
    }
  })

  return { html: root.innerHTML, toc }
}

async function selectTopic(slug: string) {
  mobileMenuOpen.value = false
  await router.push(`/docs/${slug}`)
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

async function copyText(value: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value)
    return
  }
  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.append(textarea)
  textarea.select()
  const copied = document.execCommand('copy')
  textarea.remove()
  if (!copied) throw new Error('Clipboard copy failed')
}

function openPreview(image: HTMLImageElement) {
  const figure = image.closest('figure')
  previewTrigger = image
  previewImage.value = {
    src: image.currentSrc || image.src,
    alt: image.alt,
    caption: figure?.querySelector('figcaption')?.textContent || '',
  }
  void nextTick(() => previewCloseButton.value?.focus())
}

async function handleArticleClick(event: MouseEvent) {
  const target = event.target as HTMLElement
  const copyButton = target.closest<HTMLButtonElement>('[data-docs-copy="true"]')
  if (copyButton) {
    const code = copyButton.parentElement?.querySelector('code')?.textContent || ''
    try {
      await copyText(code)
      copyButton.textContent = t('docs.copied')
      window.setTimeout(() => { copyButton.textContent = t('docs.copyCode') }, 1200)
    } catch {
      copyButton.textContent = t('docs.copyCode')
    }
    return
  }

  const image = target.closest<HTMLImageElement>('img[data-docs-image="true"]')
  if (image) openPreview(image)
}

function handleArticleKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' && event.key !== ' ') return
  const image = (event.target as HTMLElement).closest<HTMLImageElement>('img[data-docs-image="true"]')
  if (!image) return
  event.preventDefault()
  openPreview(image)
}

function scrollToHeading(id: string) {
  articleElement.value?.querySelector<HTMLElement>(`#${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function closePreview() {
  previewImage.value = null
  const trigger = previewTrigger
  previewTrigger = null
  void nextTick(() => trigger?.focus())
}

function handlePreviewKeydown(event: KeyboardEvent) {
  if (event.key !== 'Tab') return
  event.preventDefault()
  previewCloseButton.value?.focus()
}

function handleEscape(event: KeyboardEvent) {
  if (event.key === 'Escape') closePreview()
}

watch(requestedSlug, async (slug) => {
  if (!getDocsEntry(slug)) await router.replace('/docs/quick-start')
})

onMounted(async () => {
  window.addEventListener('keydown', handleEscape)
  if (!route.params.topic || !getDocsEntry(requestedSlug.value)) await router.replace('/docs/quick-start')
})

onUnmounted(() => window.removeEventListener('keydown', handleEscape))
</script>

<style scoped>
.docs-shell {
  @apply mx-auto max-w-[1600px] overflow-hidden border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900;
  border-radius: 8px;
}

.docs-header {
  @apply flex flex-col gap-4 border-b border-gray-200 px-4 py-4 sm:px-6 lg:flex-row lg:items-center lg:justify-between dark:border-dark-700;
}

.docs-header-actions { @apply flex w-full items-center gap-2 lg:w-auto; }
.docs-search { @apply flex min-w-0 flex-1 items-center gap-2 border border-gray-300 bg-white px-3 py-2 text-sm focus-within:border-primary-500 focus-within:ring-1 focus-within:ring-primary-500 lg:w-72 dark:border-dark-600 dark:bg-dark-800; border-radius: 6px; }
.docs-search input { @apply min-w-0 flex-1 bg-transparent text-gray-900 outline-none placeholder:text-gray-400 dark:text-white; }
.docs-menu-button { @apply inline-flex h-10 shrink-0 items-center gap-2 border border-gray-300 px-3 text-sm font-medium text-gray-700 lg:hidden dark:border-dark-600 dark:text-dark-200; border-radius: 6px; }

.docs-workspace { @apply grid min-h-[calc(100vh-13rem)] grid-cols-1 lg:grid-cols-[250px_minmax(0,1fr)] 2xl:grid-cols-[250px_minmax(0,1fr)_220px]; }
.docs-navigation { @apply hidden border-r border-gray-200 bg-gray-50/70 p-3 lg:block dark:border-dark-700 dark:bg-dark-950/40; }
.docs-navigation-open { @apply block border-b lg:border-b-0; }
.docs-category-title { @apply px-3 text-xs font-semibold uppercase text-gray-500 dark:text-dark-400; letter-spacing: 0; }
.docs-topic-link { @apply flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-sm text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white; border-radius: 6px; }
.docs-topic-link-active { @apply bg-primary-50 font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300; }

.docs-main { @apply min-w-0 px-4 py-6 sm:px-7 lg:px-9; }
.docs-article-header { @apply mb-6 flex flex-col gap-4 border-b border-gray-200 pb-5 sm:flex-row sm:items-start sm:justify-between dark:border-dark-700; }
.docs-view-switch { @apply inline-flex shrink-0 border border-gray-300 bg-gray-50 p-0.5 dark:border-dark-600 dark:bg-dark-800; border-radius: 6px; }
.docs-view-switch button { @apply px-3 py-1.5 text-sm font-medium text-gray-600 dark:text-dark-300; border-radius: 4px; }
.docs-view-switch button.active { @apply bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white; }

.docs-toc { @apply hidden border-l border-gray-200 px-5 py-7 2xl:block dark:border-dark-700; }
.docs-toc > h2 { @apply text-xs font-semibold uppercase text-gray-500 dark:text-dark-400; letter-spacing: 0; }
.docs-toc button { @apply block w-full py-1 text-left text-xs leading-5 text-gray-500 hover:text-primary-600 dark:text-dark-400 dark:hover:text-primary-400; }

.docs-sources, .docs-related { @apply mt-10 border-t border-gray-200 pt-6 dark:border-dark-700; }
.docs-sources h2, .docs-related h2 { @apply text-sm font-semibold text-gray-900 dark:text-white; }
.docs-sources ul { @apply mt-3 space-y-2 text-xs text-gray-500 dark:text-dark-400; }
.docs-sources li { @apply flex flex-wrap items-center justify-between gap-2; }
.docs-sources a { @apply break-all text-primary-600 hover:underline dark:text-primary-400; }
.docs-related-links { @apply mt-3 flex flex-wrap gap-2; }
.docs-related-links button { @apply inline-flex items-center gap-1 border border-gray-300 px-3 py-2 text-sm text-gray-700 hover:border-primary-400 hover:text-primary-600 dark:border-dark-600 dark:text-dark-200 dark:hover:text-primary-400; border-radius: 6px; }

.docs-markdown { @apply min-w-0 max-w-none text-[15px] leading-7 text-gray-700 dark:text-dark-200; }
.docs-markdown :deep(h2) { @apply mb-3 mt-8 scroll-mt-24 text-lg font-semibold text-gray-900 first:mt-0 dark:text-white; }
.docs-markdown :deep(h3) { @apply mb-2 mt-6 scroll-mt-24 text-base font-semibold text-gray-900 dark:text-white; }
.docs-markdown :deep(p) { @apply my-3 break-words; }
.docs-markdown :deep(ul), .docs-markdown :deep(ol) { @apply my-3 ml-5 space-y-1; }
.docs-markdown :deep(ul) { @apply list-disc; }
.docs-markdown :deep(ol) { @apply list-decimal; }
.docs-markdown :deep(a) { @apply break-all text-primary-600 underline-offset-2 hover:underline dark:text-primary-400; }
.docs-markdown :deep(code:not(pre code)) { @apply break-words bg-gray-100 px-1.5 py-0.5 font-mono text-[13px] text-gray-800 dark:bg-dark-800 dark:text-dark-100; border-radius: 4px; }
.docs-markdown :deep(pre) { @apply relative my-4 max-w-full overflow-x-auto border border-gray-200 bg-gray-950 p-4 pt-11 font-mono text-[13px] leading-6 text-gray-100 dark:border-dark-600; border-radius: 6px; }
.docs-markdown :deep(.docs-copy-button) { @apply absolute right-2 top-2 border border-gray-600 bg-gray-800 px-2.5 py-1 text-xs text-gray-200 hover:bg-gray-700; border-radius: 4px; }
.docs-markdown :deep(table) { @apply my-5 block w-full overflow-x-auto border-collapse text-sm; }
.docs-markdown :deep(th), .docs-markdown :deep(td) { @apply min-w-32 border border-gray-200 px-3 py-2 text-left align-top dark:border-dark-600; }
.docs-markdown :deep(th) { @apply bg-gray-50 font-semibold text-gray-900 dark:bg-dark-800 dark:text-white; }
.docs-markdown :deep(.docs-figure) { @apply my-5; }
.docs-markdown :deep(.docs-figure img) { @apply block max-h-[540px] w-auto max-w-full cursor-zoom-in border border-gray-200 object-contain dark:border-dark-600; border-radius: 6px; }
.docs-markdown :deep(.docs-figure figcaption) { @apply mt-2 text-xs text-gray-500 dark:text-dark-400; }

.docs-image-modal { @apply fixed inset-0 z-[100] flex items-center justify-center bg-black/80 p-4; }
.docs-image-modal > button { @apply absolute right-4 top-4 flex h-10 w-10 items-center justify-center bg-black/60 text-white hover:bg-black/80; border-radius: 6px; }
.docs-image-modal figure { @apply flex max-h-full max-w-6xl flex-col items-center gap-3; }
.docs-image-modal img { @apply max-h-[85vh] max-w-full object-contain; }
.docs-image-modal figcaption { @apply max-w-3xl text-center text-sm text-gray-200; }

@media (max-width: 1023px) {
  .docs-workspace { min-height: auto; }
  .docs-navigation { max-height: 60vh; overflow-y: auto; }
}
</style>
