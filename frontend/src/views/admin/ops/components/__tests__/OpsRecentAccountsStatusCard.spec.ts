import { describe, it, expect, beforeEach, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import OpsRecentAccountsStatusCard from '../OpsRecentAccountsStatusCard.vue'

const mockGetRecentAccountStatus = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getRecentAccountStatus: (...args: any[]) => mockGetRecentAccountStatus(...args),
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, any>) => {
        if (key === 'admin.ops.recentAccounts.totalInRange' && params) {
          return `${params.count} accounts in range`
        }
        return key
      },
    }),
  }
})

const sampleResponse = {
  generated_at: '2026-05-17T00:00:00Z',
  start_time: '2026-05-17T00:00:00Z',
  end_time: '2026-05-18T00:00:00Z',
  platform: 'openai',
  group_id: 7,
  total_count: 1,
  normal_count: 1,
  rate_limited_count: 0,
  error_count: 0,
  overloaded_count: 0,
  temp_unschedulable_count: 0,
  paused_count: 0,
  disabled_count: 0,
  other_count: 0,
  items: [
    {
      account_id: 1,
      account_name: 'account-a',
      platform: 'openai',
      group_id: 7,
      group_name: 'Pro',
      created_at: '2026-05-17T01:00:00Z',
      status: 'active',
      status_category: 'normal',
      schedulable: true,
    },
  ],
}

describe('OpsRecentAccountsStatusCard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetRecentAccountStatus.mockResolvedValue(sampleResponse)
  })

  it('defaults to today and switches to 24h and 7d range tabs', async () => {
    const wrapper = mount(OpsRecentAccountsStatusCard, {
      props: {
        platformFilter: 'openai',
        groupIdFilter: 7,
        refreshToken: 0,
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    await flushPromises()
    expect(mockGetRecentAccountStatus).toHaveBeenCalledWith(
      expect.objectContaining({
        start_time: expect.any(String),
        end_time: expect.any(String),
        platform: 'openai',
        group_id: 7,
        limit: 20,
      })
    )
    expect(mockGetRecentAccountStatus.mock.calls[0][0]).not.toHaveProperty('time_range')
    expect(wrapper.find('input[type="date"]').exists()).toBe(true)

    const last24hButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.ops.recentAccounts.rangeTabs.last24h')
    )
    expect(last24hButton).toBeTruthy()
    await last24hButton!.trigger('click')
    await flushPromises()

    expect(mockGetRecentAccountStatus).toHaveBeenLastCalledWith(
      expect.objectContaining({
        time_range: '24h',
        platform: 'openai',
        group_id: 7,
        limit: 20,
      })
    )
    expect(wrapper.find('input[type="date"]').exists()).toBe(false)

    const last7dButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.ops.recentAccounts.rangeTabs.last7d')
    )
    expect(last7dButton).toBeTruthy()
    await last7dButton!.trigger('click')
    await flushPromises()

    expect(mockGetRecentAccountStatus).toHaveBeenLastCalledWith(
      expect.objectContaining({
        time_range: '7d',
        platform: 'openai',
        group_id: 7,
        limit: 20,
      })
    )
  })
})
