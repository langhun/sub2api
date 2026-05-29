import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AccountTestModal from '../AccountTestModal.vue'

const { getAvailableModels, copyToClipboard } = vi.hoisted(() => ({
  getAvailableModels: vi.fn(),
  copyToClipboard: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getAvailableModels
    }
  }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.accounts.imagePromptDefault': 'Generate a cute orange cat astronaut sticker on a clean pastel background.'
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.accounts.imageReceived' && params?.count) {
          return `received-${params.count}`
        }
        return messages[key] || key
      }
    })
  }
})

function createStreamResponse(lines: string[]) {
  const encoder = new TextEncoder()
  const chunks = lines.map((line) => encoder.encode(line))
  let index = 0

  return {
    ok: true,
    body: {
      getReader: () => ({
        read: vi.fn().mockImplementation(async () => {
          if (index < chunks.length) {
            return { done: false, value: chunks[index++] }
          }
          return { done: true, value: undefined }
        })
      })
    }
  } as Response
}

function mountModal() {
  return mount(AccountTestModal, {
    props: {
      show: false,
      account: {
        id: 42,
        name: 'Gemini Image Test',
        platform: 'gemini',
        type: 'apikey',
        status: 'active'
      }
    } as any,
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Select: { template: '<div class="select-stub"></div>' },
        TextArea: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
        },
        Icon: true
      }
    }
  })
}

describe('AccountTestModal', () => {
  beforeEach(() => {
    getAvailableModels.mockResolvedValue([
      { id: 'gemini-2.0-flash', display_name: 'Gemini 2.0 Flash' },
      { id: 'gemini-2.5-flash-image', display_name: 'Gemini 2.5 Flash Image' },
      { id: 'gemini-3.1-flash-image', display_name: 'Gemini 3.1 Flash Image' }
    ])
    copyToClipboard.mockReset()
    Object.defineProperty(globalThis, 'localStorage', {
      value: {
        getItem: vi.fn((key: string) => (key === 'auth_token' ? 'test-token' : null)),
        setItem: vi.fn(),
        removeItem: vi.fn(),
        clear: vi.fn()
      },
      configurable: true
    })
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gemini-2.5-flash-image"}\n',
        'data: {"type":"image","image_url":"data:image/png;base64,QUJD","mime_type":"image/png"}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('首次以 show=true 挂载时会立即加载模型并选中默认项', async () => {
    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: {
          id: 42,
          name: 'Gemini Image Test',
          platform: 'gemini',
          type: 'apikey',
          status: 'active'
        }
      } as any,
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Select: { template: '<div class="select-stub"></div>' },
          TextArea: {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
          },
          Icon: true
        }
      }
    })

    await flushPromises()

    expect(getAvailableModels).toHaveBeenCalledWith(42)
    expect((wrapper.vm as any).availableModels).toHaveLength(3)
    expect((wrapper.vm as any).selectedModelId).toBe('gemini-3.1-flash-image')
  })

  it('gemini 图片模型测试会携带提示词并渲染图片预览', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const promptInput = wrapper.find('textarea.textarea-stub')
    expect(promptInput.exists()).toBe(true)
    await promptInput.setValue('draw a tiny orange cat astronaut')

    const buttons = wrapper.findAll('button')
    const startButton = buttons.find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toEqual({
      model_id: 'gemini-3.1-flash-image',
      prompt: 'draw a tiny orange cat astronaut',
      mode: 'default'
    })

    const preview = wrapper.find('img[alt="test-image-1"]')
    expect(preview.exists()).toBe(true)
    expect(preview.attributes('src')).toBe('data:image/png;base64,QUJD')
  })

  it('流结束时会解析未换行的最后一条 SSE 错误事件，而不是退化成 network error', async () => {
    getAvailableModels.mockResolvedValue([
      { id: 'gpt-image-2', display_name: 'GPT Image 2' }
    ])

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader: () => {
          const chunks = [
            new TextEncoder().encode('data: {"type":"test_start","model":"gpt-image-2"}\n'),
            new TextEncoder().encode('data: {"type":"content","text":"Calling Codex /responses image tool..."}\n'),
            new TextEncoder().encode('data: {"type":"error","error":"Upstream returned 403: Codex official clients required"}')
          ]
          let index = 0
          return {
            read: vi.fn().mockImplementation(async () => {
              if (index < chunks.length) {
                return { done: false, value: chunks[index++] }
              }
              return { done: true, value: undefined }
            })
          }
        }
      }
    } as Response) as any

    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: {
          id: 88,
          name: 'OpenAI Image Test',
          platform: 'openai',
          type: 'oauth',
          status: 'active'
        }
      } as any,
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Select: { template: '<div class="select-stub"></div>' },
          TextArea: {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
          },
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-image-2'

    const buttons = wrapper.findAll('button')
    const startButton = buttons.find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect((wrapper.vm as any).status).toBe('error')
    expect((wrapper.vm as any).errorMessage).toBe('Upstream returned 403: Codex official clients required')
    expect((wrapper.vm as any).outputLines.some((line: { text: string }) => line.text.includes('network error'))).toBe(false)
    expect((wrapper.vm as any).streamingContent).toBe('')
    expect((wrapper.vm as any).outputLines.some((line: { text: string }) => line.text === 'Calling Codex /responses image tool...')).toBe(true)
  })

  it('OpenAI OAuth 测试始终使用默认模式并随请求提交', async () => {
    getAvailableModels.mockResolvedValue([
      { id: 'gpt-5.4', display_name: 'GPT-5.4' }
    ])

    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gpt-5.4"}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any

    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: {
          id: 99,
          name: 'OpenAI OAuth Test',
          platform: 'openai',
          type: 'oauth',
          status: 'active'
        }
      } as any,
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Select: { template: '<div class="select-stub"></div>' },
          TextArea: {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
          },
          Icon: true
        }
      }
    })

    await flushPromises()

    expect((wrapper.vm as any).selectedModelId).toBe('gpt-5.4')

    const buttons = wrapper.findAll('button')
    const startButton = buttons.find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toEqual({
      model_id: 'gpt-5.4',
      prompt: '',
      mode: 'default'
    })
  })
})
