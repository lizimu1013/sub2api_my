<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { opsAPI, type OpsRecentAccountStatusCategory, type OpsRecentAccountStatusResponse } from '@/api/admin/ops'
import { formatDateTime } from '@/utils/format'

interface Props {
  platformFilter?: string
  groupIdFilter?: number | null
  refreshToken: number
}

const props = withDefaults(defineProps<Props>(), {
  platformFilter: '',
  groupIdFilter: null
})

const { t } = useI18n()

const loading = ref(false)
const errorMessage = ref('')
const response = ref<OpsRecentAccountStatusResponse | null>(null)
const selectedDate = ref(toDateInputValue(new Date()))
const selectedRange = ref<'day' | '24h' | '7d'>('day')

const rangeTabs = computed(() => [
  { key: 'day' as const, label: t('admin.ops.recentAccounts.rangeTabs.today') },
  { key: '24h' as const, label: t('admin.ops.recentAccounts.rangeTabs.last24h') },
  { key: '7d' as const, label: t('admin.ops.recentAccounts.rangeTabs.last7d') }
])

const items = computed(() => response.value?.items ?? [])

const stats = computed(() => {
  const data = response.value
  return [
    { key: 'total', label: t('admin.ops.recentAccounts.stats.total'), value: data?.total_count ?? 0, tone: 'slate' },
    { key: 'normal', label: t('admin.ops.recentAccounts.stats.normal'), value: data?.normal_count ?? 0, tone: 'green' },
    { key: 'rateLimited', label: t('admin.ops.recentAccounts.stats.rateLimited'), value: data?.rate_limited_count ?? 0, tone: 'amber' },
    { key: 'error', label: t('admin.ops.recentAccounts.stats.error'), value: data?.error_count ?? 0, tone: 'red' },
    { key: 'overloaded', label: t('admin.ops.recentAccounts.stats.overloaded'), value: data?.overloaded_count ?? 0, tone: 'orange' },
    { key: 'paused', label: t('admin.ops.recentAccounts.stats.paused'), value: (data?.paused_count ?? 0) + (data?.temp_unschedulable_count ?? 0), tone: 'gray' },
    { key: 'disabled', label: t('admin.ops.recentAccounts.stats.disabled'), value: data?.disabled_count ?? 0, tone: 'gray' },
    { key: 'other', label: t('admin.ops.recentAccounts.stats.other'), value: data?.other_count ?? 0, tone: 'slate' }
  ]
})

function toDateInputValue(date: Date): string {
  const year = date.getFullYear()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  return `${year}-${month}-${day}`
}

function selectedDayRange() {
  const [year, month, day] = selectedDate.value.split('-').map((part) => Number.parseInt(part, 10))
  const start = Number.isFinite(year) && Number.isFinite(month) && Number.isFinite(day)
    ? new Date(year, month - 1, day, 0, 0, 0, 0)
    : new Date()
  if (!Number.isFinite(year) || !Number.isFinite(month) || !Number.isFinite(day)) {
    start.setHours(0, 0, 0, 0)
  }
  const end = new Date(start)
  end.setDate(end.getDate() + 1)
  return { startTime: start.toISOString(), endTime: end.toISOString() }
}

async function loadData() {
  loading.value = true
  errorMessage.value = ''
  try {
    const rangeParams = selectedRange.value === 'day'
      ? (() => {
          const { startTime, endTime } = selectedDayRange()
          return { start_time: startTime, end_time: endTime }
        })()
      : { time_range: selectedRange.value }
    response.value = await opsAPI.getRecentAccountStatus({
      ...rangeParams,
      platform: props.platformFilter || undefined,
      group_id: typeof props.groupIdFilter === 'number' && props.groupIdFilter > 0 ? props.groupIdFilter : undefined,
      limit: 20
    })
  } catch (err: any) {
    console.error('[OpsRecentAccountsStatusCard] Failed to load data', err)
    response.value = null
    errorMessage.value = err?.message || t('admin.ops.recentAccounts.failedToLoad')
  } finally {
    loading.value = false
  }
}

function statClass(tone: string): string {
  switch (tone) {
    case 'green':
      return 'border-green-100 bg-green-50 text-green-700 dark:border-green-900/50 dark:bg-green-950/30 dark:text-green-300'
    case 'amber':
      return 'border-amber-100 bg-amber-50 text-amber-700 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-300'
    case 'orange':
      return 'border-orange-100 bg-orange-50 text-orange-700 dark:border-orange-900/50 dark:bg-orange-950/30 dark:text-orange-300'
    case 'red':
      return 'border-red-100 bg-red-50 text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300'
    case 'gray':
      return 'border-gray-200 bg-gray-50 text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300'
    default:
      return 'border-slate-200 bg-slate-50 text-slate-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300'
  }
}

function statusLabel(category: OpsRecentAccountStatusCategory): string {
  return t(`admin.ops.recentAccounts.status.${category}`)
}

function statusBadgeClass(category: OpsRecentAccountStatusCategory): string {
  switch (category) {
    case 'normal':
      return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
    case 'rate_limited':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    case 'error':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
    case 'overloaded':
      return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
}

watch(
  () => [selectedDate.value, selectedRange.value, props.platformFilter, props.groupIdFilter, props.refreshToken] as const,
  () => {
    void loadData()
  },
  { immediate: true }
)
</script>

<template>
  <section class="card p-4 md:p-5">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <div>
        <h3 class="flex items-center gap-2 text-sm font-bold text-gray-900 dark:text-white">
          <Icon name="userPlus" size="sm" class="text-primary-500" />
          {{ t('admin.ops.recentAccounts.title') }}
        </h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.ops.recentAccounts.subtitle') }}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <label v-if="selectedRange === 'day'" class="sr-only" for="ops-recent-accounts-date">
          {{ t('admin.ops.recentAccounts.date') }}
        </label>
        <div
          v-if="selectedRange === 'day'"
          class="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-2 py-1.5 dark:border-dark-600 dark:bg-dark-800"
        >
          <Icon name="calendar" size="xs" class="text-gray-400" />
          <input
            id="ops-recent-accounts-date"
            v-model="selectedDate"
            type="date"
            class="bg-transparent text-xs font-semibold text-gray-700 outline-none dark:text-gray-200"
          />
        </div>
        <button
          class="flex items-center gap-1 rounded-lg bg-gray-100 px-2 py-1.5 text-xs font-semibold text-gray-700 transition-colors hover:bg-gray-200 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600"
          :disabled="loading"
          :title="t('common.refresh')"
          @click="loadData"
        >
          <Icon name="refresh" size="xs" :class="{ 'animate-spin': loading }" />
        </button>
      </div>
    </div>

    <div class="mb-4 flex flex-wrap items-center gap-2">
      <button
        v-for="tab in rangeTabs"
        :key="tab.key"
        type="button"
        :class="[
          'rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors',
          selectedRange === tab.key
            ? 'bg-primary-600 text-white shadow-sm'
            : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'
        ]"
        @click="selectedRange = tab.key"
      >
        {{ tab.label }}
      </button>
    </div>

    <div v-if="errorMessage" class="mb-4 rounded-lg bg-red-50 px-3 py-2 text-xs text-red-600 dark:bg-red-900/20 dark:text-red-400">
      {{ errorMessage }}
    </div>

    <div class="grid grid-cols-2 gap-2 md:grid-cols-4 xl:grid-cols-8">
      <div
        v-for="item in stats"
        :key="item.key"
        :class="['rounded-lg border px-3 py-2', statClass(item.tone)]"
      >
        <div class="text-[11px] font-semibold">
          {{ item.label }}
        </div>
        <div class="mt-1 text-xl font-bold tabular-nums">
          {{ item.value }}
        </div>
      </div>
    </div>

    <div class="mt-4 overflow-hidden rounded-xl border border-gray-200 dark:border-dark-700">
      <div class="flex items-center justify-between border-b border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900">
        <span class="text-[11px] font-bold uppercase tracking-wider text-gray-500 dark:text-gray-400">
          {{ t('admin.ops.recentAccounts.latest') }}
        </span>
        <span class="text-[11px] text-gray-500 dark:text-gray-400">
          {{ t('admin.ops.recentAccounts.totalInRange', { count: response?.total_count ?? 0 }) }}
        </span>
      </div>

      <div v-if="loading" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.ops.loadingText') }}
      </div>
      <div v-else-if="items.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.ops.recentAccounts.empty') }}
      </div>
      <div v-else class="max-h-[360px] overflow-auto">
        <table class="min-w-full text-left text-xs md:text-sm">
          <thead class="sticky top-0 z-10 bg-white dark:bg-dark-800">
            <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-gray-400">
              <th class="px-3 py-2 font-semibold">{{ t('admin.ops.recentAccounts.table.account') }}</th>
              <th class="px-3 py-2 font-semibold">{{ t('admin.ops.recentAccounts.table.platform') }}</th>
              <th class="px-3 py-2 font-semibold">{{ t('admin.ops.recentAccounts.table.group') }}</th>
              <th class="px-3 py-2 font-semibold">{{ t('admin.ops.recentAccounts.table.status') }}</th>
              <th class="px-3 py-2 font-semibold">{{ t('admin.ops.recentAccounts.table.createdAt') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="account in items"
              :key="account.account_id"
              class="border-b border-gray-100 last:border-0 dark:border-dark-700"
            >
              <td class="max-w-[220px] truncate px-3 py-2 font-semibold text-gray-900 dark:text-white" :title="account.account_name">
                {{ account.account_name }}
              </td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-300">
                {{ account.platform || '-' }}
              </td>
              <td class="max-w-[180px] truncate px-3 py-2 text-gray-600 dark:text-gray-300" :title="account.group_name">
                {{ account.group_name || '-' }}
              </td>
              <td class="px-3 py-2">
                <span :class="['inline-flex rounded-full px-2 py-0.5 text-[11px] font-bold', statusBadgeClass(account.status_category)]">
                  {{ statusLabel(account.status_category) }}
                </span>
              </td>
              <td class="whitespace-nowrap px-3 py-2 font-mono text-[11px] text-gray-500 dark:text-gray-400">
                {{ formatDateTime(account.created_at) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>
