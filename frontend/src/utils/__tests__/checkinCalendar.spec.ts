import { describe, expect, it } from 'vitest'
import { formatCalendarDate, parseCalendarDate } from '@/utils/checkinCalendar'

describe('check-in calendar local dates', () => {
  it('round-trips a date without converting it through UTC', () => {
    expect(formatCalendarDate(parseCalendarDate('2026-07-01'))).toBe('2026-07-01')
  })

  it('formats the local calendar day', () => {
    expect(formatCalendarDate(new Date(2026, 6, 11, 0, 30))).toBe('2026-07-11')
  })
})
