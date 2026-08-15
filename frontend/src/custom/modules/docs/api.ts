import { apiClient } from '@/api/client'
import type { DocsLocale } from './catalog'

export interface DocsRevision {
  id: number
  slug: string
  locale: DocsLocale
  content: string
  edit_summary: string
  author_name: string
  created_at: string
}

export interface DocsRevisionState {
  current: DocsRevision | null
  history: DocsRevision[]
}

export const docsAPI = {
  async listRevisions(slug: string, locale: DocsLocale): Promise<DocsRevisionState> {
    const { data } = await apiClient.get<DocsRevisionState>(`/docs/${encodeURIComponent(slug)}`, { params: { locale } })
    return { current: data.current || null, history: data.history || [] }
  },

  async createRevision(input: {
    slug: string
    locale: DocsLocale
    content: string
    editSummary: string
    baseRevisionId: number | null
  }): Promise<DocsRevision> {
    const { data } = await apiClient.post<DocsRevision>(`/docs/${encodeURIComponent(input.slug)}/revisions`, {
      locale: input.locale,
      content: input.content,
      edit_summary: input.editSummary,
      base_revision_id: input.baseRevisionId,
    })
    return data
  },

  async revertRevision(slug: string, locale: DocsLocale, revisionId: number): Promise<DocsRevision> {
    const { data } = await apiClient.post<DocsRevision>(
      `/admin/docs/${encodeURIComponent(slug)}/revisions/${revisionId}/revert`,
      null,
      { params: { locale } },
    )
    return data
  },
}
