import { describe, expect, it } from 'vitest'

import { DEFAULT_INTERVAL_SECONDS } from '../channelMonitor'

describe('channel monitor constants', () => {
  it('defaults channel polling to 30 seconds', () => {
    expect(DEFAULT_INTERVAL_SECONDS).toBe(30)
  })
})
