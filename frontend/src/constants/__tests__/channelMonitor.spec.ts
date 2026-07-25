import { describe, expect, it } from 'vitest'

import {
  DEFAULT_MONITOR_INTERVAL_SECONDS,
  DEFAULT_STATUS_REFRESH_SECONDS,
} from '../channelMonitor'

describe('channel monitor constants', () => {
  it('keeps status refresh and active probing as separate defaults', () => {
    expect(DEFAULT_STATUS_REFRESH_SECONDS).toBe(30)
    expect(DEFAULT_MONITOR_INTERVAL_SECONDS).toBe(30)
  })
})
