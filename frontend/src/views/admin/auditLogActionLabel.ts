export type AuditActionTranslator = (key: string) => string
export type AuditActionTranslationExists = (key: string) => boolean

const ACTION_SEGMENT_PREFIX = 'admin.audit.actionSegments.'
const ACTOR_ROLE_PREFIX = 'admin.audit.roles.'

function translateSegment(
  segment: string,
  translate: AuditActionTranslator,
  hasTranslation: AuditActionTranslationExists
): string {
  const normalized = segment.trim().replace(/-/g, '_').toLowerCase()
  if (!normalized) return segment

  const exactKey = `${ACTION_SEGMENT_PREFIX}${normalized}`
  if (hasTranslation(exactKey)) return translate(exactKey)

  const tokens = normalized.split('_').filter(Boolean)
  if (tokens.length <= 1) return normalized

  const translatedTokens = tokens.map((token) => {
    const key = `${ACTION_SEGMENT_PREFIX}${token}`
    return hasTranslation(key)
      ? { translated: true, label: translate(key) }
      : { translated: false, label: token }
  })

  // Chinese token translations read naturally without separators. Preserve spaces
  // when a future route token is unknown so its backend action remains diagnosable.
  return translatedTokens.every((token) => token.translated)
    ? translatedTokens.map((token) => token.label).join('')
    : translatedTokens.map((token) => token.label).join(' ')
}

export function formatAuditAction(
  action: string,
  translate: AuditActionTranslator,
  hasTranslation: AuditActionTranslationExists
): string {
  const parts = action.split('.').map((part) => part.trim()).filter(Boolean)
  const visibleParts = parts[0]?.toLowerCase() === 'admin' ? parts.slice(1) : parts
  if (visibleParts.length === 0) return action

  return visibleParts
    .map((part) => translateSegment(part, translate, hasTranslation))
    .join(' · ')
}

export function formatAuditActorRole(
  role: string,
  translate: AuditActionTranslator,
  hasTranslation: AuditActionTranslationExists
): string {
  const normalized = role.trim().toLowerCase()
  if (!normalized) return role

  const key = `${ACTOR_ROLE_PREFIX}${normalized}`
  return hasTranslation(key) ? translate(key) : role
}
