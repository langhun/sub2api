import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import BatchAccountTestModal from '../BatchAccountTestModal.vue'

const { getAvailableModels } = vi.hoisted(() => ({
  getAvailableModels: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getAvailableModels
    }
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.accounts.batchTest.successWithModelAndText') {
          return `passed ${params?.model}: ${params?.text}`
        }
        if (key === 'admin.accounts.batchTest.successWithModel') {
          return `passed ${params?.model}`
        }
        return key
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
  return mount(BatchAccountTestModal, {
    props: {
      show: false,
      targets: [
        { id: 1, name: 'ag-1', platform: 'antigravity', type: 'oauth' },
        { id: 2, name: 'ag-2', platform: 'antigravity', type: 'oauth' }
      ]
    },
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Select: {
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template: '<select class="select-stub" :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="option in options" :key="option.id" :value="option.id">{{ option.display_name }}</option></select>'
        },
        Icon: true
      }
    }
  })
}

describe('BatchAccountTestModal', () => {
  beforeEach(() => {
    getAvailableModels.mockImplementation(async (id: number) => {
      if (id === 1) {
        return [
          { id: 'gpt-5.4', display_name: 'GPT-5.4' },
          { id: 'claude-sonnet-4-5', display_name: 'Claude Sonnet 4.5' },
          { id: 'gemini-3-pro-preview', display_name: 'Gemini 3 Pro' }
        ]
      }
      return [
        { id: 'claude-sonnet-4-5', display_name: 'Claude Sonnet 4.5' },
        { id: 'gemini-3-pro-preview', display_name: 'Gemini 3 Pro' }
      ]
    })
    Object.defineProperty(globalThis, 'localStorage', {
      value: {
        getItem: vi.fn((key: string) => (key === 'auth_token' ? 'test-token' : null)),
        setItem: vi.fn(),
        removeItem: vi.fn(),
        clear: vi.fn()
      },
      configurable: true
    })
    global.fetch = vi.fn()
      .mockResolvedValueOnce(createStreamResponse([
        'data: {"type":"test_start","model":"claude-sonnet-4-5"}\n',
        'data: {"type":"content","text":"ok"}\n',
        'data: {"type":"test_complete","success":true}\n'
      ]))
      .mockResolvedValueOnce(createStreamResponse([
        'data: {"type":"test_start","model":"claude-sonnet-4-5"}\n',
        'data: {"type":"content","text":"ok"}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])) as any
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('只提供共同模型且不会自动选择默认模型', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    expect((wrapper.vm as any).availableModels.map((model: { id: string }) => model.id)).toEqual([
      'claude-sonnet-4-5',
      'gemini-3-pro-preview'
    ])
    expect((wrapper.vm as any).selectedModelId).toBe('')

    ;(wrapper.vm as any).selectedModelId = 'claude-sonnet-4-5'
    await (wrapper.vm as any).startBatch()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(2)
    for (const [, request] of (global.fetch as any).mock.calls) {
      expect(JSON.parse(request.body)).toEqual({ model_id: 'claude-sonnet-4-5' })
    }
  })
})
