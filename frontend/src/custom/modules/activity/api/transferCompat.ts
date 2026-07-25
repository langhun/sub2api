import { directTransferAPI } from '@/custom/modules/wallet-extension/api'
import {
  claimRedPacket,
  createRedPacket,
  getMyRedPackets,
  getRedPacketDetail,
} from './redpacket'
import { getTransferLeaderboard } from './transferLeaderboard'

// Preserve the legacy aggregate while each implementation stays in its module.
export const transferAPI = {
  ...directTransferAPI,
  getTransferLeaderboard,
  createRedPacket,
  claimRedPacket,
  getRedPacketDetail,
  getMyRedPackets,
}

export default transferAPI
