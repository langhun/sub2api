import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import DocsView from '../DocsView.vue'
import { docsCatalog } from '../catalog'

const docsApiMock = vi.hoisted(() => ({
  listRevisions: vi.fn(),
  createRevision: vi.fn(),
  revertRevision: vi.fn(),
}))
const authStoreMock = vi.hoisted(() => ({ isAuthenticated: true, isAdmin: true }))

vi.mock('../api', () => ({ docsAPI: docsApiMock }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => authStoreMock }))

vi.mock('vue-i18n', async (importOriginal) => {
  const original = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...original,
    useI18n: () => ({
      locale: { value: 'zh' },
      t: (key: string) => ({
        'docs.copyCode': '复制代码',
        'docs.copied': '已复制',
        'docs.closePreview': '关闭图片预览',
        'docs.edit': '编辑',
        'docs.history': '历史',
        'docs.editTitle': '编辑文档',
        'docs.historyTitle': '修订历史',
        'docs.saved': '修订已发布',
        'docs.conflict': '文档在编辑期间发生了变化，请重新加载最新修订后再保存。',
        'docs.saveFailed': '修订发布失败，请检查网络后重试。',
        'docs.loadFailed': '暂时无法读取用户修订',
        'docs.editSummary': '编辑摘要',
        'docs.editSummaryPlaceholder': '简要说明本次修改内容',
        'docs.editorPlaceholder': '在这里编写 Markdown',
        'docs.preview': '预览',
        'docs.saveRevision': '发布修订',
        'docs.cancel': '取消',
        'docs.saving': '正在发布',
        'docs.markdownHint': '仅支持 Markdown',
        'docs.revisionBy': '作者：{author}',
        'docs.current': '当前版本',
        'docs.viewRevision': '查看修订内容',
        'docs.revert': '回滚',
        'docs.revertConfirm': '确定将文档回滚到修订 #{id} 吗？',
      })[key] || key,
    }),
  }
})

async function mountView(path = '/docs/sillytavern') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/docs', component: DocsView },
      { path: '/docs/:topic', component: DocsView },
    ],
  })
  vi.stubGlobal('scrollTo', vi.fn())
  await router.push(path)
  await router.isReady()
  const wrapper = mount(DocsView, {
    attachTo: document.body,
    global: {
      plugins: [router],
      stubs: { AppLayout: { template: '<div><slot /></div>' }, Teleport: true },
    },
  })
  await flushPromises()
  return { wrapper, router }
}

afterEach(() => {
  vi.restoreAllMocks()
  docsApiMock.listRevisions.mockReset()
  docsApiMock.createRevision.mockReset()
  docsApiMock.revertRevision.mockReset()
  docsApiMock.listRevisions.mockResolvedValue({ current: null, history: [] })
  authStoreMock.isAuthenticated = true
  authStoreMock.isAdmin = true
  Object.defineProperty(navigator, 'clipboard', { configurable: true, value: undefined })
  document.body.innerHTML = ''
})

beforeEach(() => {
  docsApiMock.listRevisions.mockResolvedValue({ current: null, history: [] })
})

describe('DocsView', () => {
  it('opens beginner steps by default and switches to reference', async () => {
    const { wrapper } = await mountView()
    expect(wrapper.text()).toContain('步骤 1：准备 SillyTavern 和本站账号')
    expect(wrapper.text()).not.toContain('部分旧版本出现')
    expect(wrapper.text()).toContain('Sub2API gateway and frontend implementation')

    const tabs = wrapper.findAll('[role="tab"]')
    await tabs[1].trigger('click')
    expect(wrapper.text()).toContain('部分旧版本出现')
  })

  it('keeps official sources without exposing retired source metadata', async () => {
    const { wrapper } = await mountView('/docs/grok')
    expect(wrapper.text()).toContain('xAI developer documentation')
    expect(wrapper.text()).not.toMatch(/aiapipay\.com/i)
  })

  it('searches topic bodies and navigates to a result', async () => {
    const { wrapper, router } = await mountView()
    await wrapper.get('input[type="search"]').setValue('OpenCode')
    expect(wrapper.text()).toContain('Grok CLI 与编程客户端')
    const topic = wrapper.findAll('.docs-topic-link').find((button) => button.text().includes('Grok CLI'))
    expect(topic).toBeTruthy()
    await topic!.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.params.topic).toBe('grok-cli')
  })

  it('copies code and opens local images in the preview', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })
    const { wrapper } = await mountView('/docs/quick-start')
    await wrapper.get('[data-docs-copy="true"]').trigger('click')
    expect(writeText).toHaveBeenCalled()

    await wrapper.get('img[data-docs-image="true"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="dialog"]').exists()).toBe(true)
    expect(document.activeElement?.getAttribute('aria-label')).toBe('关闭图片预览')

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(document.activeElement?.matches('img[data-docs-image="true"]')).toBe(true)
  })

  it('reports clipboard fallback failure and removes unregistered media', async () => {
    Object.defineProperty(document, 'execCommand', { configurable: true, value: vi.fn(() => false) })
    const entry = docsCatalog.find((item) => item.slug === 'quick-start')!
    const originalBody = entry.body.zh
    entry.body.zh += '\n<img src="//external.example/track.png" alt="external"><video src="https://external.example/video.mp4"></video>'

    try {
      const { wrapper } = await mountView('/docs/quick-start')
      await wrapper.get('[data-docs-copy="true"]').trigger('click')
      await flushPromises()
      expect(wrapper.get('[data-docs-copy="true"]').text()).toBe('复制代码')
      expect(wrapper.find('img[src*="external.example"]').exists()).toBe(false)
      expect(wrapper.find('video').exists()).toBe(false)
    } finally {
      entry.body.zh = originalBody
    }
  })

  it('redirects an unknown slug to quick start', async () => {
    const { router } = await mountView('/docs/not-a-topic')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/docs/quick-start')
  })

  it('publishes a revision with the current revision as the concurrency base', async () => {
    docsApiMock.listRevisions.mockResolvedValue({
      current: { id: 7, slug: 'quick-start', locale: 'zh', content: '# 旧版', edit_summary: '旧', author_name: 'Alice', created_at: '2026-08-15T00:00:00Z' },
      history: [{ id: 7, slug: 'quick-start', locale: 'zh', content: '# 旧版', edit_summary: '旧', author_name: 'Alice', created_at: '2026-08-15T00:00:00Z' }],
    })
    docsApiMock.createRevision.mockResolvedValue({ id: 8, slug: 'quick-start', locale: 'zh', content: '# 新版', edit_summary: '修正步骤', author_name: 'Bob', created_at: '2026-08-15T01:00:00Z' })
    const { wrapper } = await mountView('/docs/quick-start')
    await wrapper.get('button[title="编辑"]').trigger('click')
    await flushPromises()
    await wrapper.get('#docs-edit-summary').setValue('修正步骤')
    await wrapper.get('#docs-editor-content').setValue('# 新版')
    await wrapper.find('.docs-dialog-button-primary').trigger('click')
    await flushPromises()
    expect(docsApiMock.createRevision).toHaveBeenCalledWith(expect.objectContaining({ baseRevisionId: 7, content: '# 新版', editSummary: '修正步骤' }))
    expect(wrapper.text()).toContain('修订已发布')
    expect(wrapper.find('[data-testid="docs-content"]').text()).toContain('新版')
  })

  it('reloads the latest revision after an optimistic concurrency conflict', async () => {
    docsApiMock.createRevision.mockRejectedValue({ status: 409 })
    docsApiMock.listRevisions
      .mockResolvedValueOnce({ current: null, history: [] })
      .mockResolvedValueOnce({ current: { id: 9, slug: 'quick-start', locale: 'zh', content: '# 最新版', edit_summary: '', author_name: 'Alice', created_at: '2026-08-15T00:00:00Z' }, history: [] })
    const { wrapper } = await mountView('/docs/quick-start')
    await wrapper.get('button[title="编辑"]').trigger('click')
    await wrapper.get('#docs-editor-content').setValue('# 本地修改')
    await wrapper.find('.docs-dialog-button-primary').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('文档在编辑期间发生了变化')
    expect(wrapper.find('[data-testid="docs-content"]').text()).toContain('最新版')
  })
})
