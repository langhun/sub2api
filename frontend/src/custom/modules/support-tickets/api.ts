import { apiClient } from '@/api/client'

export type SupportTicketStatus = 'open' | 'waiting_user' | 'resolved' | 'closed'
export type SupportTicketCategory = 'account' | 'billing' | 'api' | 'model' | 'feature' | 'other'
export type SupportTicketPriority = 'low' | 'normal' | 'high' | 'urgent'
export type SupportTicketAuthorType = 'user' | 'admin' | 'system'
export type SupportTicketSenderType = SupportTicketAuthorType
export interface SupportTicketMessage { id: number; author_type: SupportTicketAuthorType; message_type: 'reply' | 'internal_note' | 'system'; body: string; created_at: string }
export interface SupportTicket { public_id: string; user_id: number; user_name?: string; category: SupportTicketCategory; subject: string; status: SupportTicketStatus; priority: SupportTicketPriority; last_reply_by: SupportTicketAuthorType; last_activity_at: string; created_at: string; updated_at: string; resolved_at?: string | null; closed_at?: string | null; messages?: SupportTicketMessage[] }
export interface SupportTicketList { items: SupportTicket[]; total: number; page: number; page_size: number }
export interface SupportTicketListParams { page?: number; page_size?: number; status?: SupportTicketStatus; priority?: SupportTicketPriority }
export interface CreateSupportTicketPayload { subject: string; category: SupportTicketCategory; body: string }
export interface CreateAdminSupportTicketPayload extends CreateSupportTicketPayload { user_id: number }
export interface AddSupportTicketMessagePayload { body: string; internal?: boolean }
export interface UpdateSupportTicketPayload { status?: SupportTicketStatus; priority?: SupportTicketPriority }

function params(value: SupportTicketListParams = {}) { return { page: value.page ?? 1, page_size: value.page_size ?? 20, ...(value.status ? { status: value.status } : {}), ...(value.priority ? { priority: value.priority } : {}) } }
export async function listSupportTickets(value?: SupportTicketListParams) { const { data } = await apiClient.get<SupportTicketList>('/support/tickets', { params: params(value) }); return data }
export async function createSupportTicket(payload: CreateSupportTicketPayload) { const { data } = await apiClient.post<SupportTicket>('/support/tickets', payload); return data }
export async function getSupportTicket(publicID: string) { const { data } = await apiClient.get<SupportTicket>(`/support/tickets/${publicID}`); return data }
export async function addSupportTicketMessage(publicID: string, payload: AddSupportTicketMessagePayload) { const { data } = await apiClient.post<SupportTicket>(`/support/tickets/${publicID}/messages`, payload); return data }
export async function reopenSupportTicket(publicID: string) { const { data } = await apiClient.post<SupportTicket>(`/support/tickets/${publicID}/reopen`); return data }
export async function listAdminSupportTickets(value?: SupportTicketListParams) { const { data } = await apiClient.get<SupportTicketList>('/admin/support/tickets', { params: params(value) }); return data }
export async function createAdminSupportTicket(payload: CreateAdminSupportTicketPayload) { const { data } = await apiClient.post<SupportTicket>('/admin/support/tickets', payload); return data }
export async function getAdminSupportTicket(publicID: string) { const { data } = await apiClient.get<SupportTicket>(`/admin/support/tickets/${publicID}`); return data }
export async function addAdminSupportTicketMessage(publicID: string, payload: AddSupportTicketMessagePayload) { const { data } = await apiClient.post<SupportTicket>(`/admin/support/tickets/${publicID}/messages`, payload); return data }
export async function updateAdminSupportTicket(publicID: string, payload: UpdateSupportTicketPayload) { const { data } = await apiClient.patch<SupportTicket>(`/admin/support/tickets/${publicID}`, payload); return data }
