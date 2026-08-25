import { describe, expect, it } from 'vitest'

import { reorderRoutingGroups, reorderRoutingGroupsByKeyboard } from '../reorder'

describe('reorderRoutingGroups', () => {
  it('moves an item before the target', () => {
    expect(reorderRoutingGroups([1, 2, 3], 3, 1, 'before')).toEqual([3, 1, 2])
    expect(reorderRoutingGroups([1, 2, 3], 1, 3, 'before')).toEqual([2, 1, 3])
  })

  it('moves an item after the target, including to the end', () => {
    expect(reorderRoutingGroups([1, 2, 3], 1, 2, 'after')).toEqual([2, 1, 3])
    expect(reorderRoutingGroups([1, 2, 3], 1, 3, 'after')).toEqual([2, 3, 1])
  })

  it('is a no-op for stale or self-drop events', () => {
    expect(reorderRoutingGroups([1, 2, 3], 2, 2)).toEqual([1, 2, 3])
    expect(reorderRoutingGroups([1, 2, 3], 8, 2)).toEqual([1, 2, 3])
    expect(reorderRoutingGroups([1, 2, 3], 2, 8)).toEqual([1, 2, 3])
  })

  it('keeps keyboard movement direction consistent with drag placement', () => {
    expect(reorderRoutingGroupsByKeyboard([1, 2, 3], 2, 'up')).toEqual([2, 1, 3])
    expect(reorderRoutingGroupsByKeyboard([1, 2, 3], 2, 'down')).toEqual([1, 3, 2])
    expect(reorderRoutingGroupsByKeyboard([1, 2, 3], 1, 'end')).toEqual([2, 3, 1])
    expect(reorderRoutingGroupsByKeyboard([1, 2, 3], 3, 'home')).toEqual([3, 1, 2])
  })
})
