<template>
  <BaseDialog :show="show" :title="t('admin.accounts.oauthHealth.title')" width="extra-wide" :close-on-escape="!busy" :show-close-button="!busy" @close="$emit('close')">
    <div class="space-y-5">
      <div class="rounded-xl bg-slate-50 p-4 text-sm dark:bg-dark-900">
        <p class="font-semibold text-gray-900 dark:text-white">{{ t('admin.accounts.oauthHealth.scope', { scope: scopeLabel }) }}</p>
        <p class="mt-2 text-gray-600 dark:text-gray-300">{{ t('admin.accounts.oauthHealth.description') }}</p>
        <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.oauthHealth.rules') }}</p>
      </div>
      <p v-if="error" role="alert" class="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">{{ error }}</p>
      <p v-if="message" role="status" class="rounded-lg bg-blue-50 p-3 text-sm text-blue-700 dark:bg-blue-900/20 dark:text-blue-300">{{ message }}</p>
      <div v-if="busy && !report" class="py-10 text-center text-gray-500">{{ t('admin.accounts.oauthHealth.checking') }}</div>
      <template v-if="report">
        <div class="flex flex-wrap items-center justify-between gap-3 text-sm">
          <span>{{ t('admin.accounts.oauthHealth.summary', { total: report.accounts.length, recommended: recommendations.length }) }}</span>
          <span class="text-xs text-gray-500">{{ formatTime(report.checked_at) }} (UTC+8)</span>
        </div>
        <p v-if="!report.accounts.length" class="py-8 text-center text-gray-500">{{ t('admin.accounts.oauthHealth.empty') }}</p>
        <div v-for="account in report.accounts" :key="account.id" class="rounded-xl border border-gray-200 p-4 dark:border-dark-700" :data-account-id="account.id">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <input v-if="canAdjust(account)" v-model="selected" type="checkbox" :value="account.id" :disabled="busy" :aria-label="t('admin.accounts.oauthHealth.selectAccount', { id: account.id })" />
                <span class="break-all font-medium">{{ account.name }}</span><span class="text-xs text-gray-500">#{{ account.id }} · {{ account.platform }}</span>
                <OAuthHealthBadge :value="account.health" />
              </div>
              <p v-if="account.group_ids?.length > 1" class="mt-1 text-xs text-amber-700 dark:text-amber-300">{{ t('admin.accounts.oauthHealth.sharedGroups', { count: account.group_ids.length }) }}</p>
            </div>
            <span class="whitespace-nowrap text-sm">{{ t('admin.accounts.oauthHealth.concurrency') }} <strong>{{ account.concurrency }}</strong><span v-if="canAdjust(account)" class="text-amber-700 dark:text-amber-300"> → {{ account.health?.action === 'cooldown' ? t('admin.accounts.oauthHealth.cooldownAction') : account.health?.recommended_concurrency }}</span></span>
          </div>
          <template v-if="account.health">
            <div class="mt-3 grid grid-cols-2 gap-x-5 gap-y-2 text-xs sm:grid-cols-4">
              <span>{{ t('admin.accounts.oauthHealth.observed') }} <b>{{ account.health.stats.observed_requests }}</b></span>
              <span>429 <b>{{ account.health.stats.rate_limited_requests + account.health.stats.quota_requests }}</b></span>
              <span>503 <b>{{ account.health.stats.overloaded_requests }}</b></span>
              <span>401 / 403 <b>{{ account.health.stats.auth_requests }}</b></span>
              <span>{{ t('admin.accounts.oauthHealth.outputs') }} <b>{{ account.health.stats.output_requests }}</b></span>
              <span>500 / 502 / 504 <b>{{ account.health.stats.other_error_requests }}</b></span>
              <span class="col-span-2">{{ t('admin.accounts.oauthHealth.p95') }} <b>{{ account.health.stats.p95_first_token_ms == null ? '—' : `${(account.health.stats.p95_first_token_ms / 1000).toFixed(2)}s` }}</b></span>
            </div>
            <p class="mt-3 text-xs text-gray-500">{{ t('admin.accounts.oauthHealth.pressureSummary', { count: account.health.stats.pressure_requests, total: account.health.stats.observed_requests, rate: pressureRate(account), last: account.health.stats.latest_pressure_at ? formatTime(account.health.stats.latest_pressure_at) : '—' }) }}</p>
            <p v-if="account.health.stats.current_concurrency != null" class="mt-1 text-xs text-gray-500">{{ t('admin.accounts.oauthHealth.currentLoad', { value: account.health.stats.current_concurrency, limit: account.concurrency }) }}</p>
            <p v-if="account.health.required_models?.length" class="mt-1 break-all text-xs text-gray-500">{{ t('admin.accounts.oauthHealth.recoveryModels', { models: account.health.required_models.join(', ') }) }}</p>
            <p class="mt-3 text-sm text-gray-600 dark:text-gray-300">{{ account.outcome === 'conflict' ? t('admin.accounts.oauthHealth.conflict') : t(`admin.accounts.oauthHealth.reason.${account.health.reason}`) }}</p>
            <p v-if="account.health.hold_until" class="mt-1 text-xs text-gray-500">{{ t('admin.accounts.oauthHealth.holdUntil', { time: formatTime(account.health.hold_until) }) }}</p>
            <p v-if="account.health.cooldown_until" class="mt-1 text-xs text-gray-500">{{ t('admin.accounts.oauthHealth.cooldownUntil', { time: formatTime(account.health.cooldown_until) }) }}</p>
            <div v-if="canAdjust(account)" class="mt-3"><button class="btn btn-secondary text-xs" :disabled="busy" @click="change([account], false)">{{ t(`admin.accounts.oauthHealth.action.${account.health.action}`, { value: account.health.recommended_concurrency }) }}</button></div>
            <p class="mt-1 text-xs text-gray-500">{{ formatTime(account.health.window_start) }} — {{ formatTime(account.health.window_end) }} (UTC+8)</p>
            <div v-if="account.health.last_change" class="mt-3 flex flex-wrap items-center justify-between gap-2 border-t border-gray-100 pt-3 text-xs dark:border-dark-700">
              <span>{{ t(`admin.accounts.oauthHealth.change.${account.health.last_change.action}`) }} {{ account.health.last_change.before }} → {{ account.health.last_change.after }} · {{ formatTime(account.health.last_change.at) }}</span>
            </div>
          </template>
        </div>
      </template>
    </div>
    <template #footer>
      <div class="flex w-full flex-wrap items-center justify-end gap-3">
        <button class="btn btn-secondary" :disabled="busy" @click="check">{{ busy ? t('common.loading') : t('admin.accounts.oauthHealth.recheck') }}</button>
        <button class="btn btn-primary" :disabled="busy || selectedAccounts.length === 0" @click="change(selectedAccounts, false)">{{ t('admin.accounts.oauthHealth.apply', { count: selectedAccounts.length }) }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import OAuthHealthBadge from './OAuthHealthBadge.vue'
import { checkOAuthHealth, adjustOAuthHealth, type OAuthHealthReport, type OAuthHealthAccount } from '@/api/admin/oauthHealth'

const props = defineProps<{ show: boolean; groupId: number | null; scopeLabel: string }>()
const emit = defineEmits<{ close: []; updated: [] }>()
const { t } = useI18n()
const report = ref<OAuthHealthReport | null>(null)
const busy = ref(false)
const error = ref('')
const message = ref('')
const selected = ref<number[]>([])
const canAdjust = (a: OAuthHealthAccount) => a.outcome !== 'conflict' && a.health?.policy_version === 2 && ['reduce', 'increase', 'rollback', 'cooldown'].includes(a.health.action ?? '')
const pressureRate = (a: OAuthHealthAccount) => a.health?.stats.observed_requests ? (a.health.stats.pressure_requests / a.health.stats.observed_requests * 100).toFixed(2) : '0.00'
const recommendations = computed(() => report.value?.accounts.filter(canAdjust) ?? [])
const selectedAccounts = computed(() => recommendations.value.filter(a => selected.value.includes(a.id)))
const formatTime = (value: string) => new Date(value).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai', hour12: false })

async function check() {
  if (busy.value) return
  busy.value = true; error.value = ''; message.value = ''; selected.value = []
  try {
    report.value = await checkOAuthHealth(props.groupId)
    selected.value = recommendations.value.map(a => a.id)
    emit('updated')
  } catch { report.value = null; error.value = t('admin.accounts.oauthHealth.failed') }
  finally { busy.value = false }
}
async function change(accounts: OAuthHealthAccount[], restore: boolean) {
  if (busy.value || !accounts.length) return
  busy.value = true; error.value = ''; message.value = ''
  try {
    const result = await adjustOAuthHealth(props.groupId, accounts, restore)
    const updates = new Map(result.accounts.map(a => [a.id, a]))
    if (report.value) report.value.accounts = report.value.accounts.map(a => updates.get(a.id) ?? a)
    const done = result.accounts.filter(a => ['reduce', 'increase', 'rollback', 'cooldown'].includes(a.outcome ?? '')).length
    message.value = t('admin.accounts.oauthHealth.applied', { done, skipped: result.accounts.length - done })
    selected.value = []; emit('updated')
  } catch { error.value = t('admin.accounts.oauthHealth.failed'); selected.value = []; emit('updated') }
  finally { busy.value = false }
}
watch(() => props.show, show => { if (show) { report.value = null; void check() } }, { immediate: true })
</script>
