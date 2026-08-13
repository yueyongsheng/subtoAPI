import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAppStore } from '@/stores/app'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import type { PublicSettings } from '@/types'

describe('model plaza feature flag', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('defaults on until public settings explicitly disable it', () => {
    const appStore = useAppStore()

    expect(isFeatureFlagEnabled(FeatureFlags.modelPlaza)).toBe(true)

    appStore.cachedPublicSettings = {
      model_plaza_enabled: false,
    } as PublicSettings
    expect(isFeatureFlagEnabled(FeatureFlags.modelPlaza)).toBe(false)

    appStore.cachedPublicSettings = {
      model_plaza_enabled: true,
    } as PublicSettings
    expect(isFeatureFlagEnabled(FeatureFlags.modelPlaza)).toBe(true)
  })
})
