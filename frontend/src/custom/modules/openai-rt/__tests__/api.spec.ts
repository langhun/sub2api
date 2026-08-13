import { beforeEach, describe, expect, it, vi } from 'vitest'

const { postMock } = vi.hoisted(() => ({
  postMock: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post: postMock,
  },
}))

import { replaceOpenAIRT } from '../api'

describe('OpenAI RT API', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    postMock.mockResolvedValue({ data: { account: { id: 7 }, masked_refresh_token: 'rt-***' } })
  })

  it('allows token rotation to outlive the shared 30 second client timeout', async () => {
    await replaceOpenAIRT(7, 'rt-new-secret')

    expect(postMock).toHaveBeenCalledWith(
      '/custom/admin/openai/accounts/7/refresh-token',
      { refresh_token: 'rt-new-secret' },
      { timeout: 90_000 },
    )
  })
})
