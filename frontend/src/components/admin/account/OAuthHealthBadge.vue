<template>
  <span v-if="health" class="inline-flex max-w-full rounded px-1.5 py-0.5 text-xs" :class="stale ? oauthHealthTone('insufficient') : oauthHealthTone(health.status)" :title="tooltip">
    {{ stale ? t('admin.accounts.oauthHealth.historical') : '' }}{{ t(`admin.accounts.oauthHealth.status.${health.status}`) }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useNow } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { readOAuthHealth, oauthHealthTone } from '@/api/admin/oauthHealth'
const props = defineProps<{ value?: unknown; accountId?: number }>()
const { t } = useI18n()
const now = useNow({ interval: 60000 })
const health = computed(() => {
  const h = readOAuthHealth(props.value)
  return props.accountId != null && h?.account_id !== props.accountId ? null : h
})
const stale = computed(() => health.value && now.value.getTime() - Date.parse(health.value.checked_at) > 30 * 60000)
const tooltip = computed(() => health.value ? `${t('admin.accounts.oauthHealth.checkedAt')}: ${new Date(health.value.checked_at).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })} (UTC+8) · ${t(`admin.accounts.oauthHealth.reason.${health.value.reason}`)}` : '')
</script>
