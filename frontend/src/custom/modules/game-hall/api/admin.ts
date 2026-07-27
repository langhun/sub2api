import { apiClient } from '@/api/client'

export interface GameHallUserAccess {
  user_id: number
  disabled: boolean
  updated_at: string
}

export async function getGameHallUserAccess(userID: number): Promise<GameHallUserAccess> {
  const { data } = await apiClient.get<GameHallUserAccess>(`/admin/game-hall/users/${userID}/access`)
  return data
}

export async function updateGameHallUserAccess(
  userID: number,
  disabled: boolean,
): Promise<GameHallUserAccess> {
  const { data } = await apiClient.put<GameHallUserAccess>(
    `/admin/game-hall/users/${userID}/access`,
    { disabled },
  )
  return data
}

export const gameHallAdminAPI = {
  getUserAccess: getGameHallUserAccess,
  updateUserAccess: updateGameHallUserAccess,
}
