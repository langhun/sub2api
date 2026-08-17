import { beforeEach, describe, expect, it, vi } from 'vitest'
import { checkin, luckCheckin } from '../api/checkin'

const apiClient = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient }))

describe('check-in captcha API payloads', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiClient.post.mockResolvedValue({ data: { reward_amount: 1 } })
  })

  it('sends the one-time proof for normal check-in', async () => {
    await checkin({ turnstile_token: 'proof-token' })

    expect(apiClient.post).toHaveBeenCalledWith('/checkin', {
      turnstile_token: 'proof-token',
    })
  })

  it('sends the proof together with lucky check-in inputs', async () => {
    await luckCheckin(5, true, {
      tencent_captcha_ticket: 'ticket',
      tencent_captcha_randstr: '@rand',
    })

    expect(apiClient.post).toHaveBeenCalledWith('/checkin/luck', {
      bet_amount: 5,
      use_max_balance: true,
      tencent_captcha_ticket: 'ticket',
      tencent_captcha_randstr: '@rand',
    })
  })
})
