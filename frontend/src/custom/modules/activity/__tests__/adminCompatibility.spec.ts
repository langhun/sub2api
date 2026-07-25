import { describe, expect, it } from 'vitest'

import { adminAPI } from '@/api/admin'
import BlindboxPrizePoolCard from '@/custom/modules/activity/admin/components/BlindboxPrizePoolCard.vue'
import RewardDeliveryOpsPanel from '@/custom/modules/activity/admin/components/RewardDeliveryOpsPanel.vue'
import { blindboxAPI } from '@/custom/modules/activity/api/admin/blindbox'
import { rewardDeliveriesAPI } from '@/custom/modules/activity/api/admin/rewardDeliveries'

describe('activity admin integration surface', () => {
  it('keeps the core admin shell and API barrel pointed at the activity module', () => {
    expect(adminAPI.blindbox).toBe(blindboxAPI)
    expect(adminAPI.rewardDeliveries).toBe(rewardDeliveriesAPI)
    expect(BlindboxPrizePoolCard).toBeTruthy()
    expect(RewardDeliveryOpsPanel).toBeTruthy()
  })
})
