import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, describe, expect, it, vi } from 'vitest'

import DocsView from '../DocsView.vue'
import { docsCatalog } from '../catalog'

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
  Object.defineProperty(navigator, 'clipboard', { configurable: true, value: undefined })
  document.body.innerHTML = ''
})

describe('DocsView', () => {
  it('opens beginner steps by default and switches to reference', async () => {
    const { wrapper } = await mountView()
    expect(wrapper.text()).toContain('步骤 1：准备 SillyTavern 和本站账号')
    expect(wrapper.text()).not.toContain('部分旧版本出现')
    expect(wrapper.text()).not.toContain('AI API Pay documentation baseline')
    expect(wrapper.text()).toContain('Sub2API gateway and frontend implementation')

    const tabs = wrapper.findAll('[role="tab"]')
    await tabs[1].trigger('click')
    expect(wrapper.text()).toContain('部分旧版本出现')
  })

  it('keeps official sources while hiding the AI API Pay baseline URL', async () => {
    const { wrapper } = await mountView('/docs/grok')
    expect(wrapper.text()).toContain('xAI developer documentation')
    expect(wrapper.text()).not.toContain('AI API Pay documentation baseline')
    expect(wrapper.find('a[href*="aiapipay.com"]').exists()).toBe(false)
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
})
