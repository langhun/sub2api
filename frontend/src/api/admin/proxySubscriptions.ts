import { apiClient } from '../client'
import type {
  PaginatedResponse,
  Proxy,
  ProxySubscriptionNode,
  ProxySubscriptionRefreshResult,
  ProxySubscriptionSource,
  CreateProxySubscriptionSourceRequest,
  UpdateProxySubscriptionSourceRequest
} from '@/types'

export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: { search?: string; enabled?: boolean }
): Promise<PaginatedResponse<ProxySubscriptionSource>> {
  const { data } = await apiClient.get<PaginatedResponse<ProxySubscriptionSource>>('/admin/proxies/subscriptions', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    }
  })
  return data
}

export async function getById(id: number): Promise<ProxySubscriptionSource> {
  const { data } = await apiClient.get<ProxySubscriptionSource>(`/admin/proxies/subscriptions/${id}`)
  return data
}

export async function create(payload: CreateProxySubscriptionSourceRequest): Promise<ProxySubscriptionSource> {
  const { data } = await apiClient.post<ProxySubscriptionSource>('/admin/proxies/subscriptions', payload)
  return data
}

export async function update(id: number, payload: UpdateProxySubscriptionSourceRequest): Promise<ProxySubscriptionSource> {
  const { data } = await apiClient.put<ProxySubscriptionSource>(`/admin/proxies/subscriptions/${id}`, payload)
  return data
}

export async function deleteSource(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/proxies/subscriptions/${id}`)
  return data
}

export async function refresh(id: number): Promise<ProxySubscriptionRefreshResult> {
  const { data } = await apiClient.post<ProxySubscriptionRefreshResult>(`/admin/proxies/subscriptions/${id}/refresh`)
  return data
}

export async function listNodes(id: number): Promise<ProxySubscriptionNode[]> {
  const { data } = await apiClient.get<ProxySubscriptionNode[]>(`/admin/proxies/subscriptions/${id}/nodes`)
  return data
}

export async function listProxies(id: number): Promise<Proxy[]> {
  const { data } = await apiClient.get<Proxy[]>(`/admin/proxies/subscriptions/${id}/proxies`)
  return data
}

export const proxySubscriptionsAPI = {
  list,
  getById,
  create,
  update,
  delete: deleteSource,
  refresh,
  listNodes,
  listProxies
}

export default proxySubscriptionsAPI
