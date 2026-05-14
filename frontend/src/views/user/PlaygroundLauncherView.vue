<template>
  <AppLayout>
    <div class="playground-layout">
      <!-- 工具栏：选择 API Key -->
      <div class="playground-toolbar card">
        <div class="flex flex-1 items-center gap-3 min-w-0">
          <Icon name="key" size="sm" class="text-gray-500 flex-shrink-0" />
          <span class="text-sm font-medium text-gray-700 dark:text-dark-200 flex-shrink-0">
            {{ t('imagePlayground.useKey') }}
          </span>
          <select
            v-model="selectedKeyId"
            :disabled="loadingKeys || activeKeys.length === 0"
            class="form-select min-w-0 max-w-[360px] flex-1 truncate text-sm"
          >
            <option v-if="loadingKeys" :value="null">{{ t('imagePlayground.loadingKeys') }}</option>
            <option v-else-if="activeKeys.length === 0" :value="null">
              {{ t('imagePlayground.noKeys') }}
            </option>
            <option
              v-for="key in activeKeys"
              :key="key.id"
              :value="key.id"
            >
              {{ formatKeyOption(key) }}
            </option>
          </select>
          <button
            type="button"
            class="btn btn-secondary btn-sm flex-shrink-0"
            :disabled="loadingKeys"
            :title="t('imagePlayground.refresh')"
            @click="loadKeys"
          >
            <Icon name="refresh" size="sm" />
          </button>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <router-link to="/keys" class="btn btn-secondary btn-sm">
            <Icon name="cog" size="sm" class="mr-1.5" />
            {{ t('imagePlayground.manageKeys') }}
          </router-link>
          <a
            v-if="iframeSrc"
            :href="iframeSrc"
            target="_blank"
            rel="noopener noreferrer"
            class="btn btn-secondary btn-sm"
          >
            <Icon name="externalLink" size="sm" class="mr-1.5" />
            {{ t('imagePlayground.openInNewTab') }}
          </a>
        </div>
      </div>

      <!-- 主体：iframe 或空状态 -->
      <div class="card flex-1 min-h-0 overflow-hidden mt-3">
        <div v-if="loadingKeys" class="flex h-full items-center justify-center py-12">
          <div
            class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
          ></div>
        </div>

        <div
          v-else-if="activeKeys.length === 0"
          class="flex h-full flex-col items-center justify-center p-10 text-center"
        >
          <div
            class="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
          >
            <Icon name="key" size="lg" class="text-gray-400" />
          </div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('imagePlayground.emptyTitle') }}
          </h3>
          <p class="mt-2 max-w-md text-sm text-gray-500 dark:text-dark-400">
            {{ t('imagePlayground.emptyDesc') }}
          </p>
          <router-link to="/keys" class="btn btn-primary btn-sm mt-4">
            {{ t('imagePlayground.goCreateKey') }}
          </router-link>
        </div>

        <div v-else-if="loadError" class="flex h-full items-center justify-center p-10 text-center">
          <div class="max-w-md">
            <p class="text-sm text-red-500">{{ loadError }}</p>
            <button class="btn btn-secondary btn-sm mt-3" @click="loadKeys">
              {{ t('imagePlayground.retry') }}
            </button>
          </div>
        </div>

        <iframe
          v-else-if="iframeSrc"
          :src="iframeSrc"
          class="playground-frame"
          allow="clipboard-read; clipboard-write"
          allowfullscreen
        ></iframe>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { keysAPI } from '@/api/keys'
import type { ApiKey } from '@/types'

const { t } = useI18n()

const PLAYGROUND_BASE_PATH = '/playground/'
// 通过 sub2api 自身的 /v1 网关代理，让 playground 直接消费用户在主站创建的额度。
// 必须传绝对 URL（含 origin），因为 playground 的 normalizeBaseUrl 会把
// 相对路径误判成裸主机名（'/v1' → 'https://v1'）。
const LAST_KEY_STORAGE = 'playground:last_key_id'

const playgroundApiBase = computed(() => {
  if (typeof window === 'undefined') return '/v1'
  return `${window.location.origin}/v1`
})

const loadingKeys = ref(false)
const loadError = ref<string | null>(null)
const allKeys = ref<ApiKey[]>([])
const selectedKeyId = ref<number | null>(null)

const activeKeys = computed(() =>
  allKeys.value.filter((k) => k.status === 'active')
)

const selectedKey = computed(() =>
  activeKeys.value.find((k) => k.id === selectedKeyId.value) ?? null,
)

const iframeSrc = computed(() => {
  const key = selectedKey.value
  if (!key) return ''
  const params = new URLSearchParams()
  params.set('apiUrl', playgroundApiBase.value)
  params.set('apiKey', key.key)
  params.set('apiMode', 'images')
  params.set('ui_mode', 'embedded')
  return `${PLAYGROUND_BASE_PATH}?${params.toString()}`
})

function formatKeyOption(key: ApiKey): string {
  const name = key.name?.trim() || t('imagePlayground.unnamedKey')
  const suffix = key.key.slice(-6)
  return `${name} (••••${suffix})`
}

async function loadKeys() {
  loadingKeys.value = true
  loadError.value = null
  try {
    const resp = await keysAPI.list(1, 100, { status: 'active' })
    allKeys.value = resp.items ?? []

    // 选择优先级：上次选过的 > 第一个 active
    const remembered = readRememberedKeyId()
    const candidates = activeKeys.value
    if (remembered && candidates.some((k) => k.id === remembered)) {
      selectedKeyId.value = remembered
    } else if (candidates.length > 0) {
      selectedKeyId.value = candidates[0].id
    } else {
      selectedKeyId.value = null
    }
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : t('imagePlayground.loadFailed')
    allKeys.value = []
    selectedKeyId.value = null
  } finally {
    loadingKeys.value = false
  }
}

function readRememberedKeyId(): number | null {
  try {
    const raw = localStorage.getItem(LAST_KEY_STORAGE)
    if (!raw) return null
    const parsed = Number(raw)
    return Number.isFinite(parsed) ? parsed : null
  } catch {
    return null
  }
}

watch(selectedKeyId, (id) => {
  if (id == null) return
  try {
    localStorage.setItem(LAST_KEY_STORAGE, String(id))
  } catch {
    // ignore
  }
})

onMounted(() => {
  loadKeys()
})
</script>

<style scoped>
.playground-layout {
  @apply flex flex-col;
  height: calc(100vh - 64px - 4rem);
}

.playground-toolbar {
  @apply flex items-center justify-between gap-3 p-3;
}

.playground-frame {
  display: block;
  width: 100%;
  height: 100%;
  border: 0;
  background: transparent;
}

/* 工具栏在窄屏下换行 */
@media (max-width: 640px) {
  .playground-toolbar {
    @apply flex-col items-stretch;
  }
}
</style>
