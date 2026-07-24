import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import RechargePackageSelector from '../RechargePackageSelector.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

const packages = [
  { pay_amount: 36, credited_amount: 1000 },
  { pay_amount: 66, credited_amount: 2000, badge: 'recommended' },
  { pay_amount: 96, credited_amount: 3000 },
  { pay_amount: 156, credited_amount: 5000 },
  { pay_amount: 300, credited_amount: 10000, badge: 'best_value' },
]

describe('RechargePackageSelector', () => {
  it('renders all promotional packages and selects the exact CNY payment amount', async () => {
    const wrapper = mount(RechargePackageSelector, {
      props: {
        packages,
        modelValue: null,
      },
    })

    const options = wrapper.findAll('[data-test="recharge-package"]')
    expect(options).toHaveLength(5)
    expect(options[0].text()).toContain('1000')
    expect(options[0].text()).toContain('36.00')
    expect(options[1].text()).toContain('2000')

    await options[1].trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[66]])

    await wrapper.setProps({ modelValue: 66 })
    expect(options[1].attributes('aria-pressed')).toBe('true')
  })
})
