<template>
  <div class="min-w-0 border-t border-gray-100 pt-4 dark:border-dark-600 sm:border-t-0 sm:pt-0 xl:border-l xl:pl-5" :aria-busy="loading" data-testid="oauth-group-availability">
    <p class="mb-2 flex items-center gap-2 text-sm text-gray-500 dark:text-dark-300">
      <Icon name="shield" size="sm" class="text-primary-500" />
      {{ t('admin.accounts.oauthAvailability.title') }}
    </p>
    <p v-if="!report && !error" class="text-sm text-gray-500">{{ loading ? t('common.loading') : '—' }}</p>
    <p v-if="error" role="alert" class="mb-2 text-xs text-red-600 dark:text-red-400">{{ t('admin.accounts.oauthAvailability.failed') }}</p>
    <template v-if="report">
      <dl v-if="report.groups.length" class="max-h-40 space-y-2 overflow-y-auto pr-2">
        <div v-for="group in report.groups" :key="group.group_id" class="flex items-baseline justify-between gap-3" :data-group-id="group.group_id">
          <dt class="min-w-0 break-all text-sm text-gray-700 dark:text-dark-200">{{ group.group_name }}</dt>
          <dd class="shrink-0 text-lg font-semibold tabular-nums" :class="group.available_accounts ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">{{ group.available_accounts }}<span class="ml-1 text-xs font-normal text-gray-500">{{ t('admin.accounts.oauthAvailability.unit') }}</span></dd>
        </div>
      </dl>
      <p v-else class="text-xs text-gray-500">{{ t('admin.accounts.oauthAvailability.empty') }}</p>
      <p class="mt-2 text-xs text-gray-500 dark:text-dark-300">{{ t('admin.accounts.oauthAvailability.queriedAt') }} {{ timestamp(report.queried_at) }} (UTC+8)</p>
    </template>
    <p class="mt-2 text-xs text-gray-500 dark:text-dark-300">{{ t('admin.accounts.oauthAvailability.hint') }}</p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { getOAuthGroupAvailability, type OAuthGroupAvailabilityReport } from '@/api/admin/oauthAvailability'

const props = defineProps<{ refreshKey: number }>()
const { t } = useI18n()
const report = ref<OAuthGroupAvailabilityReport | null>(null)
const loading = ref(false)
const error = ref(false)
let controller: AbortController | null = null
const timestamp = (value: string) => new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23' }).format(new Date(value))
async function load() {
  controller?.abort()
  const request = new AbortController()
  controller = request
  loading.value = true
  error.value = false
  try {
    const result = await getOAuthGroupAvailability(request.signal)
    if (!request.signal.aborted) report.value = result
  } catch {
    if (!request.signal.aborted) error.value = true
  } finally {
    if (controller === request) loading.value = false
  }
}
onMounted(load)
watch(() => props.refreshKey, load)
onUnmounted(() => controller?.abort())
</script>
