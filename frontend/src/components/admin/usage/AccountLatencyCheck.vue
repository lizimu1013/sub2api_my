<template>
  <div class="card overflow-hidden">
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-600">
      <div class="flex items-center gap-2">
        <div class="rounded-lg bg-cyan-100 p-2 text-cyan-700 dark:bg-cyan-500/15 dark:text-cyan-300">
          <Icon name="bolt" size="sm" :stroke-width="2" />
        </div>
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-gray-100">{{ t('admin.usage.accountLatencyTitle') }}</h3>
          <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.usage.accountLatencyTarget') }}</div>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <div class="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-gray-700 dark:bg-dark-800">
          <button
            type="button"
            class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="metric === 'first_token'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            @click="metric = 'first_token'"
          >
            {{ t('admin.usage.metricFirstToken') }}
          </button>
          <button
            type="button"
            class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
            :class="metric === 'duration'
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            @click="metric = 'duration'"
          >
            {{ t('admin.usage.metricDuration') }}
          </button>
        </div>
        <button
          class="inline-flex h-8 w-8 items-center justify-center rounded-md text-gray-500 transition hover:bg-gray-100 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-dark-700 dark:hover:text-gray-100"
          :disabled="loading || trendLoading"
          :title="t('common.refresh')"
          @click="$emit('refresh')"
        >
          <Icon name="refresh" size="sm" :class="loading || trendLoading ? 'animate-spin' : ''" :stroke-width="2" />
        </button>
      </div>
    </div>

    <div class="border-b border-gray-100 p-4 dark:border-dark-600">
      <div v-if="trendLoading" class="flex h-56 items-center justify-center">
        <LoadingSpinner />
      </div>
      <div v-else-if="chartData" class="h-56">
        <Line :data="chartData" :options="lineOptions" />
      </div>
      <div v-else class="flex h-56 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.dashboard.noDataAvailable') }}
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-600">
        <thead class="bg-gray-50 dark:bg-dark-700/50">
          <tr>
            <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.usage.account') }}</th>
            <th class="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('usage.totalRequests') }}</th>
            <th class="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.usage.avgFirstToken') }}</th>
            <th class="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.usage.maxFirstToken') }}</th>
            <th class="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.usage.avgDuration') }}</th>
            <th class="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.usage.maxDuration') }}</th>
            <th class="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.usage.lastUsedAt') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-600 dark:bg-dark-800">
          <tr v-if="loading && items.length === 0">
            <td colspan="7" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
              <Icon name="refresh" size="md" class="mx-auto mb-2 animate-spin" />
              {{ t('common.loading') }}
            </td>
          </tr>
          <tr v-else-if="items.length === 0">
            <td colspan="7" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.usage.noAccountLatencyData') }}
            </td>
          </tr>
          <template v-else>
            <tr v-for="item in items" :key="item.account_id" class="hover:bg-gray-50 dark:hover:bg-dark-700/40">
              <td class="max-w-[220px] px-4 py-2">
                <div class="truncate text-sm font-medium text-gray-900 dark:text-gray-100">{{ item.account_name }}</div>
                <div class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
                  <span>#{{ item.account_id }}</span>
                  <span v-if="item.account_platform">· {{ item.account_platform }}</span>
                  <span v-if="item.account_status" class="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] uppercase dark:bg-dark-600">
                    {{ item.account_status }}
                  </span>
                </div>
              </td>
              <td class="px-4 py-2 text-right text-sm tabular-nums text-gray-700 dark:text-gray-300">{{ formatNumber(item.requests) }}</td>
              <td class="px-4 py-2 text-right text-sm tabular-nums text-gray-700 dark:text-gray-300">{{ formatDuration(item.avg_first_token_ms) }}</td>
              <td class="px-4 py-2 text-right text-sm tabular-nums text-gray-700 dark:text-gray-300">{{ formatDuration(item.max_first_token_ms) }}</td>
              <td class="px-4 py-2 text-right text-sm tabular-nums text-gray-700 dark:text-gray-300">{{ formatDuration(item.avg_duration_ms) }}</td>
              <td class="px-4 py-2 text-right text-sm tabular-nums text-gray-700 dark:text-gray-300">{{ formatDuration(item.max_duration_ms) }}</td>
              <td class="whitespace-nowrap px-4 py-2 text-right text-sm text-gray-600 dark:text-gray-400">{{ formatDateTime(item.last_used_at) }}</td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Title,
  Tooltip
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import type { AccountLatencyStat, AccountLatencyTrendSeries } from '@/api/admin/usage'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
)

const props = defineProps<{
  items: AccountLatencyStat[]
  loading: boolean
  trendItems: AccountLatencyTrendSeries[]
  trendLoading: boolean
}>()

defineEmits<{
  (e: 'refresh'): void
}>()

const { t } = useI18n()
const metric = ref<'first_token' | 'duration'>('first_token')

const isDarkMode = computed(() => document.documentElement.classList.contains('dark'))

const chartColors = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#374151',
  grid: isDarkMode.value ? '#374151' : '#e5e7eb',
  palette: [
    '#0891b2',
    '#2563eb',
    '#16a34a',
    '#f59e0b',
    '#dc2626',
    '#7c3aed',
    '#db2777',
    '#0f766e',
    '#ea580c',
    '#4f46e5',
    '#65a30d',
    '#9333ea',
    '#0284c7',
    '#be123c',
    '#ca8a04',
    '#059669',
    '#475569',
    '#c026d3',
    '#1d4ed8',
    '#b45309'
  ]
}))

const chartLabels = computed(() => {
  const labels = new Set<string>()
  props.trendItems.forEach((series) => {
    series.points.forEach((point) => labels.add(point.date))
  })
  return [...labels].sort()
})

const chartData = computed(() => {
  if (!props.trendItems.length || !chartLabels.value.length) return null
  const key = metric.value === 'first_token' ? 'avg_first_token_ms' : 'avg_duration_ms'
  return {
    labels: chartLabels.value,
    datasets: props.trendItems.map((series, index) => {
      const pointsByDate = new Map(series.points.map((point) => [point.date, point]))
      const color = chartColors.value.palette[index % chartColors.value.palette.length]
      return {
        label: series.account_name || `#${series.account_id}`,
        data: chartLabels.value.map((label) => pointsByDate.get(label)?.[key] ?? null),
        borderColor: color,
        backgroundColor: `${color}20`,
        borderWidth: index < 6 ? 2 : 1.5,
        pointRadius: 2,
        pointHoverRadius: 4,
        fill: false,
        tension: 0.3,
        spanGaps: true
      }
    })
  }
})

const lineOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 12,
        font: {
          size: 10
        }
      }
    },
    tooltip: {
      callbacks: {
        label: (context: any) => `${context.dataset.label}: ${formatDuration(context.raw)}`
      }
    }
  },
  scales: {
    x: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        }
      }
    },
    y: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        },
        callback: (value: string | number) => formatDuration(Number(value))
      }
    }
  }
}))

const formatDuration = (ms?: number | null) => {
  if (ms == null || !Number.isFinite(ms)) return '-'
  return ms < 1000 ? `${ms.toFixed(0)}ms` : `${(ms / 1000).toFixed(2)}s`
}

const formatNumber = (value: number) => value.toLocaleString()

const formatDateTime = (value?: string | null) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}
</script>
