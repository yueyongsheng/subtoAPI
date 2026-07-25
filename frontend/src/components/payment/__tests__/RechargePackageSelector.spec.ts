import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import RechargePackageSelector from '../RechargePackageSelector.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

const packages = [
  { pay_amount: 38, credited_amount: 1000 },
  { pay_amount: 72, credited_amount: 2000, badge: 'recommended' },
  { pay_amount: 105, credited_amount: 3000 },
  { pay_amount: 170, credited_amount: 5000, badge: 'best_value' },
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
    expect(options).toHaveLength(4)
    expect(options[0].text()).toContain('1000')
    expect(options[0].text()).toContain('38.00')
    expect(options[1].text()).toContain('2000')
    expect(options[1].text()).toContain('payment.promotion.badges.recommended')
    expect(options[3].text()).toContain('5000')
    expect(options[3].text()).toContain('170.00')
    expect(options[3].text()).toContain('payment.promotion.badges.best_value')

    await options[1].trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[72]])

    await wrapper.setProps({ modelValue: 72 })
    expect(options[1].attributes('aria-pressed')).toBe('true')
  })
})
