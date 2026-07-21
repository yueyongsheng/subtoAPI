import { describe, expect, it } from 'vitest'

import {
  CONFIG_DOCS_API_KEY,
  CONFIG_DOCS_BASE_URL,
  CONFIG_DOCS_MODEL,
  CONFIG_DOCS_RESPONSES_URL,
  configGuides,
  getConfigGuide,
} from '../configDocs'

describe('configDocs', () => {
  it('uses the public Yuexiang API connection values', () => {
    expect(CONFIG_DOCS_BASE_URL).toBe('https://api-yue88.xyz/v1')
    expect(CONFIG_DOCS_RESPONSES_URL).toBe('https://api-yue88.xyz/v1/responses')
    expect(CONFIG_DOCS_MODEL).toBe('gpt-5.6-sol')
    expect(CONFIG_DOCS_API_KEY).toBe('YOUR_YUEXIANG_API_KEY')
  })

  it('provides unique, complete guides', () => {
    expect(configGuides.length).toBeGreaterThanOrEqual(6)
    expect(new Set(configGuides.map((guide) => guide.id)).size).toBe(configGuides.length)

    for (const guide of configGuides) {
      expect(guide.title).toBeTruthy()
      expect(guide.summary).toBeTruthy()
      expect(guide.sections.length).toBeGreaterThan(0)
      expect(new Set(guide.sections.map((section) => section.id)).size).toBe(guide.sections.length)
    }
  })

  it('resolves a guide by route id', () => {
    expect(getConfigGuide('codex-app')?.title).toBe('Codex App 配置')
    expect(getConfigGuide('missing-guide')).toBeUndefined()
  })

  it('never embeds a key that looks like a real secret', () => {
    const serialized = JSON.stringify(configGuides)
    expect(serialized).toContain(CONFIG_DOCS_API_KEY)
    expect(serialized).not.toMatch(/sk-[A-Za-z0-9_-]{20,}/)
  })
})
