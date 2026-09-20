<template>
  <div v-if="tickets.length" class="mb-1 space-y-0.5" data-testid="account-state-tickets">
    <div v-for="ticket in tickets" :key="ticket.model" class="flex flex-wrap items-center gap-x-1 text-[10px] leading-4" :title="details(ticket)">
      <span class="font-medium text-gray-500 dark:text-gray-400">STATE · {{ shortModel(ticket.model) }}</span>
      <span :class="tone(ticket)" :data-state="state(ticket)">
        {{ t(`admin.accounts.stateKit.states.${state(ticket)}`) }}
        <span v-if="usable(ticket)" class="tabular-nums"> {{ remaining(ticket) }}</span>
      </span>
      <span v-if="usable(ticket) && ticket.lastError" class="text-amber-600 dark:text-amber-400">{{ t('admin.accounts.stateKit.renewalFailed') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { StateKitAvailability, StateKitTicket } from '@/composables/useStateKitStatus'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{
  tickets: StateKitTicket[]
  availability: StateKitAvailability
  elapsedSeconds: number
  queriedAt: number
}>()
const { t } = useI18n()
const shortModel = (model: string) => ({ 'gpt-6-astra': 'astra', 'gpt-5.6-sol': 'sol' }[model] ?? model)
const seconds = (ticket: StateKitTicket) => Math.max(0, ticket.remainingSeconds - props.elapsedSeconds)

function state(ticket: StateKitTicket) {
  if (props.availability !== 'healthy') return props.availability === 'disabled' ? 'disabled' : props.availability === 'stale' ? 'stale' : 'unavailable'
  if ((ticket.state === 'ready' || ticket.state === 'renewing') && seconds(ticket) <= 0) return ticket.state === 'renewing' ? 'harvesting' : 'expired'
  return ticket.state
}
function usable(ticket: StateKitTicket) {
  return ['ready', 'renewing'].includes(state(ticket)) && seconds(ticket) > 0
}
function remaining(ticket: StateKitTicket) {
  const value = seconds(ticket)
  return `${Math.floor(value / 60)}m${String(value % 60).padStart(2, '0')}s`
}
function tone(ticket: StateKitTicket) {
  if (usable(ticket) && !ticket.lastError) return 'text-emerald-600 dark:text-emerald-400'
  if (state(ticket) === 'disabled') return 'text-gray-500 dark:text-gray-400'
  return 'text-amber-600 dark:text-amber-400'
}
function details(ticket: StateKitTicket) {
  return [ticket.model, t('admin.accounts.stateKit.hint'),
    props.queriedAt ? t('admin.accounts.stateKit.queriedAt', { time: formatDateTime(new Date(props.queriedAt)) }) : '',
    ticket.expiresAt ? t('admin.accounts.stateKit.expiresAt', { time: formatDateTime(ticket.expiresAt) }) : '',
    ticket.lastError ? t(`admin.accounts.stateKit.errors.${ticket.lastError}`) : '',
    ticket.attempts ? t('admin.accounts.stateKit.attempts', { count: ticket.attempts }) : ''
  ].filter(Boolean).join('\n')
}
</script>
