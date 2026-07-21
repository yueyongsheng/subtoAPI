<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl">
      <div class="grid items-start gap-8 lg:grid-cols-[minmax(0,1fr)_340px] lg:gap-12">
        <section class="min-w-0">
          <div class="border-b border-gray-200 pb-6 dark:border-dark-700">
            <div
              class="mb-4 inline-flex h-11 w-11 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-900/25 dark:text-primary-300"
            >
              <Icon name="chat" size="lg" />
            </div>
            <h2 class="text-2xl font-semibold text-gray-900 dark:text-white">
              {{ t('support.heading') }}
            </h2>
            <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ t('support.description') }}
            </p>
          </div>

          <dl class="divide-y divide-gray-200 dark:divide-dark-700">
            <div
              v-for="item in contactItems"
              :key="item.key"
              class="grid min-h-[76px] grid-cols-[42px_minmax(0,1fr)_40px] items-center gap-3 py-4"
            >
              <div
                class="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-300"
              >
                <Icon :name="item.icon" size="md" />
              </div>
              <div class="min-w-0">
                <dt class="text-xs font-medium text-gray-500 dark:text-dark-400">
                  {{ item.label }}
                </dt>
                <dd class="mt-1 break-all text-base font-semibold text-gray-900 dark:text-white">
                  {{ item.value }}
                </dd>
              </div>
              <button
                type="button"
                class="btn-ghost btn-icon h-9 w-9"
                :title="t('support.copyValue', { label: item.label })"
                :aria-label="t('support.copyValue', { label: item.label })"
                @click="copyContact(item.value, item.label)"
              >
                <Icon name="copy" size="sm" />
              </button>
            </div>
          </dl>

          <div
            class="mt-6 flex items-start gap-3 border-l-4 border-emerald-500 bg-emerald-50 px-4 py-3 dark:bg-emerald-900/15"
          >
            <Icon name="clock" size="md" class="mt-0.5 flex-shrink-0 text-emerald-600 dark:text-emerald-400" />
            <div>
              <p class="text-sm font-semibold text-emerald-900 dark:text-emerald-200">
                {{ t('support.serviceHours') }} {{ contact.serviceHours }}
              </p>
              <p class="mt-0.5 text-xs leading-5 text-emerald-700 dark:text-emerald-300">
                {{ t('support.replyNotice') }}
              </p>
            </div>
          </div>

          <div class="mt-8 border-t border-gray-200 pt-6 dark:border-dark-700">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('support.prepareTitle') }}
            </h3>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ t('support.prepareDescription') }}
            </p>
          </div>
        </section>

        <aside class="card overflow-hidden" data-testid="support-qq-group">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <div class="flex items-center gap-3">
              <div
                class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-50 text-blue-600 dark:bg-blue-900/25 dark:text-blue-300"
              >
                <Icon name="users" size="md" />
              </div>
              <div class="min-w-0">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('support.qqGroupTitle') }}
                </h3>
                <button
                  type="button"
                  class="mt-0.5 inline-flex items-center gap-1.5 text-xs text-gray-500 hover:text-primary-600 dark:text-dark-400 dark:hover:text-primary-300"
                  :title="t('support.copyValue', { label: t('support.qqGroup') })"
                  @click="copyContact(contact.qqGroup, t('support.qqGroup'))"
                >
                  <span>{{ t('support.groupNumber') }} {{ contact.qqGroup }}</span>
                  <Icon name="copy" size="xs" />
                </button>
              </div>
            </div>
          </div>
          <div class="bg-gray-50 p-5 dark:bg-dark-800/60">
            <img
              :src="contact.qqGroupQrImage"
              :alt="t('support.qqGroupQrAlt')"
              class="mx-auto aspect-[294/365] w-full max-w-[294px] object-contain"
              width="294"
              height="365"
            >
          </div>
          <p class="px-5 py-4 text-center text-xs leading-5 text-gray-500 dark:text-dark-400">
            {{ t('support.qqGroupHint') }}
          </p>
        </aside>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { SUPPORT_CONTACT } from '@/config/support'

type SupportIcon = 'chat' | 'users'

const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const contact = SUPPORT_CONTACT

const contactItems = computed<Array<{
  key: string
  label: string
  value: string
  icon: SupportIcon
}>>(() => [
  {
    key: 'qq',
    label: t('support.qq'),
    value: contact.qq,
    icon: 'chat'
  },
  {
    key: 'wechat',
    label: t('support.wechat'),
    value: contact.wechat,
    icon: 'chat'
  },
  {
    key: 'qq-group',
    label: t('support.qqGroup'),
    value: contact.qqGroup,
    icon: 'users'
  }
])

async function copyContact(value: string, label: string): Promise<void> {
  await copyToClipboard(value, t('support.copied', { label }))
}
</script>
