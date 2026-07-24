import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import AmountInput from '../AmountInput.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

describe('AmountInput', () => {
  it('offers the fixed provider amount when no standard preset is in range', async () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: null,
        amounts: [10, 20, 50, 100],
        min: 35,
        max: 35,
      },
    })

    const fixedAmountButton = wrapper.findAll('button').find(button => button.text() === '35')
    expect(fixedAmountButton).toBeDefined()
    await fixedAmountButton?.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[35]])
  })
})
