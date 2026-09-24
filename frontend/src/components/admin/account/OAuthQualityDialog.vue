<template>
  <BaseDialog :show="show" :title="t(k + 'title')" width="extra-wide" :close-on-escape="!busy" :show-close-button="!busy" @close="$emit('close')">
    <div class="space-y-5">
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t(k + 'description') }}</p>
      <p v-if="preview" class="rounded-lg bg-blue-50 p-3 text-sm text-blue-700 dark:bg-blue-900/20 dark:text-blue-200">{{ t(k + 'previewNotice') }}</p>
      <p v-if="error" role="alert" class="rounded-lg bg-red-50 p-3 text-sm text-red-700">{{ error }}</p>

      <section class="space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
        <h4 class="font-semibold">{{ t(k + 'scopeTitle') }}</h4>
        <div class="grid gap-4 md:grid-cols-2">
          <label class="text-sm"><span class="mb-1 block font-medium">{{ t(k + 'group') }}</span>
            <select v-model="selectedGroup" data-testid="quality-group" class="input w-full" :disabled="busy">
              <option value="">{{ t(k + 'allAccounts') }}</option>
              <option value="-1">{{ t('admin.accounts.ungroupedGroup') }}</option>
              <option v-for="group in groupOptions" :key="group.id" :value="String(group.id)">{{ group.name }}</option>
            </select>
          </label>
          <label class="text-sm"><span class="mb-1 block font-medium">{{ t(k + 'search') }}</span>
            <input v-model.trim="search" data-testid="quality-search" class="input w-full" :disabled="busy" :placeholder="t(k + 'searchPlaceholder')" />
          </label>
        </div>
        <fieldset :disabled="busy" class="flex flex-wrap items-center gap-2">
          <legend class="mb-2 text-sm font-medium">{{ t(k + 'types') }}</legend>
          <label v-for="kind in qualityAccountTypes" :key="kind" class="flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-2 text-sm" :class="accountTypes.includes(kind) ? 'border-primary-400 bg-primary-50 dark:bg-primary-900/20' : 'border-gray-200 dark:border-dark-700'">
            <input v-model="accountTypes" type="checkbox" :value="kind" :data-testid="'type-' + kind" />{{ typeLabel(kind) }}
          </label>
        </fieldset>
        <p class="text-xs text-gray-500">{{ t(k + 'scopeHint') }}</p>
        <div class="flex flex-wrap items-center gap-2 text-sm">
          <button class="btn btn-secondary" data-testid="select-visible" :disabled="busy || loadingAccounts || !visibleAccounts.length" @click="toggleAll">{{ allSelected ? t(k + 'clearSelection') : t(k + 'selectAll') }}</button>
          <button v-if="selectedIDs.length" class="btn btn-secondary" :disabled="busy" @click="selectedIDs = []">{{ t(k + 'clearAll') }}</button>
          <span>{{ t(k + 'selected', { count: selectedIDs.length, total: accounts.length }) }}</span>
          <span v-if="hiddenSelected" class="text-amber-600">{{ t(k + 'hiddenSelected', { count: hiddenSelected }) }}</span>
        </div>
        <p v-if="loadingAccounts" class="py-4 text-center text-gray-500">{{ t(k + 'loadingAccounts') }}</p>
        <div v-else class="max-h-56 space-y-1 overflow-auto">
          <p v-if="!visibleAccounts.length" class="p-5 text-center text-sm text-gray-500">{{ t(k + 'empty') }}</p>
          <label v-for="account in visibleAccounts" :key="account.id" class="flex cursor-pointer items-center gap-3 rounded-lg border border-gray-100 px-3 py-2 dark:border-dark-700">
            <input v-model="selectedIDs" type="checkbox" :value="account.id" :disabled="busy" :data-testid="'account-' + account.id" :aria-label="t(k + 'selectAccount', { id: account.id })" />
            <span class="min-w-0 flex-1 truncate text-sm font-medium">{{ account.name }} <span class="font-mono text-xs font-normal text-gray-500">#{{ account.id }}</span></span>
            <span class="text-xs text-gray-500">{{ account.platform }}</span>
            <span class="rounded bg-gray-100 px-2 py-1 text-xs dark:bg-dark-700">{{ typeLabel(account.type) }}</span>
            <span v-if="!account.schedulable" class="text-xs text-gray-500">{{ t(k + 'notSchedulable') }}</span>
          </label>
        </div>
      </section>

      <section class="space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
        <h4 class="font-semibold">{{ t(k + 'methodsTitle') }}</h4>
        <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <label v-for="key in qualityProbeKeys" :key="key" class="flex cursor-pointer items-start gap-3 rounded-xl border p-3" :class="probeKeys.includes(key) ? 'border-primary-400 bg-primary-50 dark:bg-primary-900/20' : 'border-gray-200 dark:border-dark-700'">
            <input v-model="probeKeys" type="checkbox" :value="key" :disabled="busy" :data-testid="'probe-' + key" class="mt-1" />
            <span><span class="block text-sm font-semibold">{{ probeLabel(key) }}</span><span class="mt-1 block text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t(k + 'methods.' + key + '.hint') }}</span></span>
          </label>
        </div>
        <div v-if="probeKeys.includes('custom')" class="space-y-3 rounded-xl bg-gray-50 p-3 dark:bg-dark-900">
          <label class="block text-sm"><span class="mb-1 block font-medium">{{ t(k + 'customPrompt') }}</span>
            <textarea v-model="customPrompt" data-testid="custom-prompt" rows="4" maxlength="8000" class="input w-full" :disabled="busy" :placeholder="t(k + 'customPlaceholder')" />
            <span class="text-xs text-gray-500">{{ customPrompt.length }} / 8000</span>
          </label>
          <label class="block text-sm"><span class="mb-1 block font-medium">{{ t(k + 'customExpected') }}</span>
            <textarea v-model="customExpected" data-testid="custom-expected" rows="2" maxlength="2000" class="input w-full" :disabled="busy" :placeholder="t(k + 'expectedPlaceholder')" />
          </label>
          <p class="text-xs text-gray-500">{{ t(k + 'expectedHint') }}</p>
        </div>
        <label class="block text-sm"><span class="mb-1 block font-medium">{{ t(k + 'model') }}</span>
          <input v-model.trim="modelID" data-testid="quality-model" class="input w-full" :disabled="busy" :placeholder="t(k + 'modelPlaceholder')" />
        </label>
        <p class="text-xs text-gray-500">{{ t(k + 'rules') }}</p>
      </section>

      <section v-if="report" ref="resultsElement" class="space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
          <h4 class="font-semibold">{{ t(k + 'resultTitle') }}</h4>
          <span>{{ t(k + 'progress', { done: report.accounts.length, total: plannedTotal }) }} · {{ report.model_id }}</span>
        </div>
        <p class="text-xs text-gray-500">{{ reportScope }} · {{ formatTime(report.checked_at) }} (UTC+8)</p>
        <p v-if="report.custom_probe" class="whitespace-pre-wrap break-words rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-900">{{ t(k + 'customPrompt') }}：{{ report.custom_probe.prompt }}<br />{{ t(k + 'customExpected') }}：{{ report.custom_probe.expected || t(k + 'manualReview') }}</p>
        <div class="overflow-x-auto rounded-xl border border-gray-200 dark:border-dark-700">
          <table class="w-full text-left text-sm">
            <thead class="bg-gray-50 text-gray-500 dark:bg-dark-900"><tr>
              <th class="p-3">{{ t(k + 'account') }}</th><th class="p-3">{{ t(k + 'types') }}</th>
              <th v-for="key in report.probe_keys" :key="key" class="whitespace-nowrap p-3">{{ probeLabel(key) }}</th>
              <th class="p-3">{{ t(k + 'result') }}</th>
            </tr></thead>
            <tbody><tr v-for="account in report.accounts" :key="account.id" class="border-t border-gray-100 dark:border-dark-700">
              <td class="p-3"><button class="text-left font-medium text-primary-600" @click="expandedID = expandedID === account.id ? null : account.id">{{ account.name }} <span class="text-xs">#{{ account.id }}</span></button></td>
              <td class="whitespace-nowrap p-3">{{ typeLabel(account.type) }}</td>
              <td v-for="probe in account.probes" :key="probe.key" class="whitespace-nowrap p-3"><span :class="statusClass(probe.status)">{{ statusLabel(probe.status) }}</span></td>
              <td class="whitespace-nowrap p-3"><button class="text-primary-600" @click="expandedID = expandedID === account.id ? null : account.id">{{ t(k + 'details') }} · {{ account.passed }}/{{ account.total }}</button></td>
            </tr></tbody>
          </table>
        </div>
        <div v-if="expandedAccount" class="space-y-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
          <p class="font-medium">{{ expandedAccount.name }} · {{ expandedAccount.summary }}</p>
          <div v-for="probe in expandedAccount.probes" :key="probe.key" class="rounded-lg bg-white p-3 text-sm dark:bg-dark-800">
            <div class="flex items-center gap-2"><span class="font-medium">{{ probeLabel(probe.key) }}</span><span :class="statusClass(probe.status)">{{ statusLabel(probe.status) }}</span><span class="text-xs text-gray-500">{{ probe.latency_ms }} ms</span></div>
            <p class="mt-2 text-gray-500">{{ probe.summary }}</p>
            <pre v-if="probe.output_preview" class="mt-2 max-h-80 overflow-auto whitespace-pre-wrap break-all text-xs">{{ probe.output_preview }}</pre>
          </div>
        </div>
        <p v-if="message" role="status" class="text-sm text-primary-600">{{ message }}</p>
      </section>
    </div>
    <template #footer>
      <div class="flex w-full flex-wrap items-center justify-between gap-3">
        <p class="text-sm text-gray-500">{{ t(k + 'estimate', { accounts: selectedIDs.length, methods: probeKeys.length, requests: selectedIDs.length * probeKeys.length }) }}</p>
        <div class="flex gap-2">
          <button v-if="busy" class="btn btn-secondary" :disabled="stopRequested" @click="stopRequested = true">{{ t(k + (stopRequested ? 'stopping' : 'stop')) }}</button>
          <button v-else class="btn btn-secondary" @click="$emit('close')">{{ t('common.close') }}</button>
          <button class="btn btn-primary" data-testid="run-quality" :disabled="!canRun" @click="run">{{ busy ? t(k + 'running') : t(k + 'run', { count: selectedIDs.length }) }}</button>
        </div>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { listOAuthQualityAccounts, runOAuthQuality, qualityAccountTypes, qualityProbeKeys,
  type QualityAccountType, type QualityProbeKey, type QualityCustomProbe,
  type OAuthQualityAccount, type OAuthQualityAccountResult, type OAuthQualityReport
} from '@/api/admin/oauthQuality'
import { previewQualityAccounts, previewQualityResult } from '@/components/admin/account/qualityPreview'

const props = defineProps<{
  show: boolean; groupId: number | null; scopeLabel: string; preview?: boolean
  groups?: { id: number; name: string }[]; initialTypes?: string[]; initialAccountIds?: number[]
}>()
defineEmits<{ close: [] }>()
const { t } = useI18n()
const k = 'admin.accounts.oauthQuality.'
const accounts = ref<OAuthQualityAccount[]>([])
const selectedIDs = ref<number[]>([])
const selectedGroup = ref('')
const accountTypes = ref<QualityAccountType[]>(['oauth', 'apikey'])
const probeKeys = ref<QualityProbeKey[]>(['svg_html'])
const customPrompt = ref('')
const customExpected = ref('')
const search = ref('')
const modelID = ref('gpt-6-astra')
const report = ref<OAuthQualityReport | null>(null)
const reportScope = ref('')
const loadingAccounts = ref(false)
const busy = ref(false)
const error = ref('')
const message = ref('')
const stopRequested = ref(false)
const plannedTotal = ref(0)
const expandedID = ref<number | null>(null)
const resultsElement = ref<HTMLElement | null>(null)
let loadVersion = 0
let alive = true
let resettingScope = false
const groupID = computed(() => selectedGroup.value === '' ? null : Number(selectedGroup.value))
const groupOptions = computed(() => {
  const options = [...(props.groups ?? [])]
  if (props.groupId != null && props.groupId > 0 && !options.some(g => g.id === props.groupId)) options.push({ id: props.groupId, name: props.scopeLabel })
  return options
})
const visibleAccounts = computed(() => accounts.value.filter(a => !search.value || (a.name + ' #' + a.id).toLowerCase().includes(search.value.toLowerCase())))
const allSelected = computed(() => visibleAccounts.value.length > 0 && visibleAccounts.value.every(a => selectedIDs.value.includes(a.id)))
const hiddenSelected = computed(() => selectedIDs.value.filter(id => !visibleAccounts.value.some(a => a.id === id)).length)
const expandedAccount = computed(() => report.value?.accounts.find(a => a.id === expandedID.value))
const canRun = computed(() => !busy.value && !loadingAccounts.value && selectedIDs.value.length > 0 && accountTypes.value.length > 0 && probeKeys.value.length > 0 && !!modelID.value && (!probeKeys.value.includes('custom') || !!customPrompt.value.trim()))
function typeLabel(type: string) { return ({ oauth: 'OAuth', apikey: 'API Key', 'setup-token': 'Setup Token', bedrock: 'AWS Bedrock' } as Record<string, string>)[type] ?? type }
function probeLabel(key: string) { return t(k + 'methods.' + key + '.label') }
function statusLabel(status: string) { return t(k + (status === 'passed' ? 'passed' : status === 'failed' ? 'failed' : 'review')) }
function statusClass(status: string) {
  return 'rounded px-2 py-1 text-xs ' + (status === 'passed' ? 'bg-emerald-100 text-emerald-700' : status === 'failed' ? 'bg-red-100 text-red-700' : 'bg-amber-100 text-amber-700')
}
function formatTime(value: string) { return new Date(value).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai', hour12: false }) }
function toggleAll() {
  const visible = new Set(visibleAccounts.value.map(a => a.id))
  selectedIDs.value = allSelected.value ? selectedIDs.value.filter(id => !visible.has(id)) : [...new Set([...selectedIDs.value, ...visible])]
}
async function loadAccounts(initial = false) {
  const version = ++loadVersion
  loadingAccounts.value = true
  error.value = ''
  accounts.value = []
  selectedIDs.value = []
  const scope = groupID.value
  const types = [...accountTypes.value]
  try {
    const result = !types.length ? [] : props.preview
      ? previewQualityAccounts.filter(a => types.includes(a.type) && (scope == null || (scope === -1 ? !a.group_ids.length : a.group_ids.includes(scope))))
      : await listOAuthQualityAccounts(scope, types)
    if (version !== loadVersion || !props.show || !alive) return
    accounts.value = result
    if (initial) selectedIDs.value = result.filter(a => props.initialAccountIds?.includes(a.id)).map(a => a.id)
  } catch {
    if (version === loadVersion && alive) error.value = t(k + 'loadFailed')
  } finally { if (version === loadVersion && alive) loadingAccounts.value = false }
}
async function run() {
  if (!canRun.value) return
  busy.value = true
  stopRequested.value = false
  error.value = ''; message.value = ''; expandedID.value = null
  const selected = accounts.value.filter(a => selectedIDs.value.includes(a.id))
  const keys = [...probeKeys.value]
  const types = [...accountTypes.value]
  const model = modelID.value
  const scope = groupID.value
  const custom: QualityCustomProbe | undefined = keys.includes('custom') ? { prompt: customPrompt.value.trim(), expected: customExpected.value.trim() } : undefined
  plannedTotal.value = selected.length
  reportScope.value = scope == null ? t(k + 'allAccounts') : scope === -1 ? t('admin.accounts.ungroupedGroup') : groupOptions.value.find(g => g.id === scope)?.name ?? String(scope)
  report.value = { checked_at: new Date().toISOString(), group_id: scope, model_id: model, account_types: types, probe_keys: keys, custom_probe: custom, accounts: [] }
  await nextTick()
  resultsElement.value?.scrollIntoView?.({ behavior: 'smooth', block: 'start' })
  try {
    // One probe per request stays within the edge proxy's response timeout.
    // The UI still groups all selected probes under their account.
    for (const account of selected) {
      if (stopRequested.value || !alive) break
      let result: OAuthQualityAccountResult
      try {
        if (props.preview) {
          await new Promise(resolve => window.setTimeout(resolve, 250))
          result = previewQualityResult(account, keys, custom)
        } else {
          result = { ...account, status: 'review', summary: '', passed: 0, total: keys.length, probes: [] }
          for (const key of keys) {
            if (!alive) break
            const batch = await runOAuthQuality(scope, [account.id], model, types, [key], key === 'custom' ? custom : undefined)
            const found = batch.accounts.find(item => item.id === account.id)
            const probe = found?.probes.find(item => item.key === key)
            if (!probe) throw new Error('Missing probe result')
            result.probes.push(probe)
          }
          result.passed = result.probes.filter(probe => probe.status === 'passed').length
          result.status = result.probes.every(probe => probe.status === 'failed') ? 'failed' : result.passed === result.total ? 'passed' : 'review'
          result.summary = t(k + 'probeSummary', { passed: result.passed, total: result.total })
        }
      } catch {
        // A rejected batch or a network timeout is not evidence of model quality.
        // Stop to avoid starting more requests when the prior outcome is unknown.
        error.value = t(k + 'runFailed')
        break
      }
      if (!alive) break
      report.value.accounts.push(result)
      if (expandedID.value == null) {
        expandedID.value = result.id
        await nextTick()
        resultsElement.value?.scrollIntoView?.({ behavior: 'smooth', block: 'start' })
      }
    }
    if (alive && !error.value) message.value = t(k + (stopRequested.value ? 'stopped' : 'completed'))
  } finally { if (alive) busy.value = false }
}
watch(() => props.show, async show => {
  if (!show) { ++loadVersion; stopRequested.value = true; return }
  resettingScope = true
  selectedGroup.value = props.groupId == null ? '' : String(props.groupId)
  const initial = (props.initialTypes ?? []).filter((kind): kind is QualityAccountType => qualityAccountTypes.some(t => t === kind))
  accountTypes.value = initial.length ? initial : ['oauth', 'apikey']
  search.value = ''
  report.value = null
  await nextTick()
  resettingScope = false
  if (props.show && alive) void loadAccounts(true)
}, { immediate: true })
watch(() => [selectedGroup.value, [...accountTypes.value].sort().join(',')] as const, () => {
  if (props.show && !busy.value && !resettingScope) void loadAccounts()
})
onBeforeUnmount(() => { alive = false; ++loadVersion; stopRequested.value = true })
</script>

