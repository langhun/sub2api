export type RoutingDropPlacement = 'before' | 'after'

export type RoutingKeyboardMove = 'up' | 'down' | 'home' | 'end'

/**
 * Return a reordered copy of the candidate ids without mutating the form
 * state. Missing ids and self-drops are treated as no-ops so stale drag
 * events cannot corrupt the persisted order.
 */
export function reorderRoutingGroups(
  ids: readonly number[],
  draggedGroupId: number,
  targetGroupId: number,
  placement: RoutingDropPlacement = 'before'
): number[] {
  const fromIndex = ids.indexOf(draggedGroupId)
  const targetIndex = ids.indexOf(targetGroupId)
  if (fromIndex < 0 || targetIndex < 0 || draggedGroupId === targetGroupId) {
    return [...ids]
  }

  const nextIds = [...ids]
  nextIds.splice(fromIndex, 1)

  // Resolve the target after removal so moving an item from above the target
  // does not introduce an off-by-one error.
  const targetIndexAfterRemoval = nextIds.indexOf(targetGroupId)
  if (targetIndexAfterRemoval < 0) return [...ids]
  const insertionIndex = placement === 'after' ? targetIndexAfterRemoval + 1 : targetIndexAfterRemoval
  nextIds.splice(insertionIndex, 0, draggedGroupId)
  return nextIds
}

/**
 * Apply the same placement semantics used by the drag handle to keyboard
 * movement. Down/End insert after the target; Up/Home insert before it.
 */
export function reorderRoutingGroupsByKeyboard(
  ids: readonly number[],
  draggedGroupId: number,
  movement: RoutingKeyboardMove
): number[] {
  const index = ids.indexOf(draggedGroupId)
  if (index < 0 || ids.length < 2) return [...ids]

  const targetIndex = movement === 'home'
    ? 0
    : movement === 'end'
      ? ids.length - 1
      : Math.max(0, Math.min(ids.length - 1, index + (movement === 'up' ? -1 : 1)))
  if (targetIndex === index) return [...ids]

  const targetGroupId = ids[targetIndex]
  const placement: RoutingDropPlacement = movement === 'down' || movement === 'end' ? 'after' : 'before'
  return reorderRoutingGroups(ids, draggedGroupId, targetGroupId, placement)
}
