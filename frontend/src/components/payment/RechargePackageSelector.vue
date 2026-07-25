<template>
  <div>
    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
      {{ t('payment.promotion.title') }}
    </label>
    <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <button
        v-for="pkg in packages"
        :key="pkg.pay_amount"
        type="button"
        data-test="recharge-package"
        :aria-pressed="modelValue === pkg.pay_amount"
        :class="[
          'relative min-h-40 rounded-lg border px-4 py-5 text-left transition-colors',
          modelValue === pkg.pay_amount
            ? 'border-primary-500 bg-primary-50 dark:border-primary-400 dark:bg-primary-900/30'
            : 'border-gray-200 bg-white hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800 dark:hover:border-primary-600',
        ]"
        @click="emit('update:modelValue', pkg.pay_amount)"
      >
        <span
          v-if="pkg.badge"
          class="mb-3 inline-flex rounded-md bg-primary-500 px-2 py-0.5 text-xs font-medium text-white"
        >
          {{ t(`payment.promotion.badges.${pkg.badge}`) }}
        </span>
        <span class="block text-sm font-semibold leading-5 text-gray-900 dark:text-white">
          {{ t('payment.promotion.package') }}
        </span>
        <span class="block text-base font-bold leading-6 text-gray-900 dark:text-white">
          {{ formatBalance(pkg.credited_amount) }} USD
        </span>
        <span class="mt-3 block text-xl font-bold text-emerald-600 dark:text-emerald-400">
          ¥{{ pkg.pay_amount.toFixed(2) }}
        </span>
        <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">
          {{ t('payment.promotion.creditPrefix') }} {{ formatBalance(pkg.credited_amount) }} USD {{ t('payment.promotion.balanceSuffix') }}
        </span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { RechargePackage } from '@/types/payment'

defineProps<{
  packages: RechargePackage[]
  modelValue: number | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: number]
}>()

const { t } = useI18n()

function formatBalance(value: number): string {
  return Number.isInteger(value) ? String(value) : value.toFixed(2)
}
</script>
