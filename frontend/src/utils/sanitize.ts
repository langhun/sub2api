import DOMPurify from 'dompurify'

/** Sanitize admin-provided rich HTML (home content etc.) before v-html rendering. */
export function sanitizeHtml(html: string): string {
  if (!html) return ''
  return DOMPurify.sanitize(html, { ADD_ATTR: ['target', 'allowfullscreen'], ADD_TAGS: ['iframe'] })
}

export function sanitizeSvg(svg: string): string {
  if (!svg) return ''
  return DOMPurify.sanitize(svg, { USE_PROFILES: { svg: true, svgFilters: true } })
}
