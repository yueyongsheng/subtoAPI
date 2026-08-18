import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SupportView from '@/views/user/SupportView.vue'

const copyToClipboard = vi.hoisted(() => vi.fn())

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string>) => {
        if (key === 'support.copied') return `${params?.label} copied`
        return key
      }
    })
  }
})

describe('SupportView', () => {
  beforeEach(() => {
    copyToClipboard.mockReset()
    copyToClipboard.mockResolvedValue(true)
  })

  it('renders the confirmed support channels and both group QR codes', () => {
    const wrapper = mount(SupportView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true
        }
      }
    })

    expect(wrapper.text()).toContain('504002280')
    expect(wrapper.text()).toContain('yys504002280')
    expect(wrapper.text()).toContain('09:00-23:00')
    expect(wrapper.text()).toContain('964308879')
    expect(wrapper.findAll('img').map((img) => img.attributes('src'))).toEqual([
      '/support-qq-group.png',
      '/support-wechat-group.png'
    ])
    expect(wrapper.get('[data-testid="support-wechat-group"]').text()).toContain('support.wechatGroupTitle')
  })

  it('copies a support value from its icon button', async () => {
    const wrapper = mount(SupportView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true
        }
      }
    })

    await wrapper.get('button.btn-icon').trigger('click')

    expect(copyToClipboard).toHaveBeenCalledWith('504002280', 'support.qq copied')
  })
})
