import { apiClient } from '@/api/client'

export type SupportContactChannel = 'wechat' | 'qq' | 'telegram' | 'email' | 'link'

export interface SupportContact {
  id: string
  channel: SupportContactChannel
  display_name: string
  contact: string
  link: string
  qr_image_url: string
  has_qr_image: boolean
  enabled: boolean
  sort_order: number
}

export interface SupportContactInput {
  channel: SupportContactChannel
  display_name: string
  contact: string
  link: string
  qr_image_url: string
  enabled: boolean
  sort_order: number
}

export const supportContactsAPI = {
  listPublic: () => apiClient.get<SupportContact[]>('/custom/support-contacts'),
  listAdmin: () => apiClient.get<SupportContact[]>('/custom/admin/support-contacts'),
  create: (input: SupportContactInput) => apiClient.post<SupportContact>('/custom/admin/support-contacts', input),
  update: (id: string, input: SupportContactInput) => apiClient.put<SupportContact>(`/custom/admin/support-contacts/${id}`, input),
  remove: (id: string) => apiClient.delete(`/custom/admin/support-contacts/${id}`),
  removeQR: (id: string) => apiClient.delete(`/custom/admin/support-contacts/${id}/qr`),
  uploadQR: (id: string, image: File) => {
    const form = new FormData()
    form.append('image', image)
    return apiClient.post<{ qr_image_url: string; has_qr_image: boolean }>(
      `/custom/admin/support-contacts/${id}/qr`,
      form,
      { headers: { 'Content-Type': 'multipart/form-data' } },
    )
  },
}
