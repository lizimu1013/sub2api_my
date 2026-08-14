import { describe, expect, it } from 'vitest'
import type { PromptAuditConfig } from '../types'
import {
  buildUpdateRequest,
  configToDraft,
  draftFingerprint,
  emptyEventFilters,
  eventFilterPayload,
  hasExplicitDeleteRange,
  SCANNER_CATALOG,
} from '../viewModel'

const config = (): PromptAuditConfig => ({
  enabled: true,
  blocking_enabled: false,
  blocking_latest_turn_only: false,
  store_pass_events: false,
  effective_mode: 'async_audit',
  strategy: 'priority',
  worker_count: 4,
  queue_capacity: 100,
  scanners: SCANNER_CATALOG.map((item) => item.id),
  all_groups: true,
  group_ids: [],
  endpoints: [{
    id: 'guard-1', name: 'Guard One', protocol: 'openai_compatible', base_url: 'http://127.0.0.1:8000',
    model: 'sileader/qwen3guard:0.6b', timeout_ms: 3000, input_limit: 4000, enabled: true,
    has_token: true, token_status: 'configured',
  }],
  config_version: 7,
  updated_at: '2026-07-16T00:00:00Z',
  updated_by: 1,
  change_summary: '{}',
})

describe('Prompt Audit view model', () => {
  it('normalizes legacy null collections from the public config', () => {
    const legacy = { ...config(), group_ids: null, scanners: null, endpoints: null } as unknown as PromptAuditConfig
	const draft = configToDraft(legacy)
	expect(draft).toMatchObject({
	  group_ids: [], scanners: [], endpoints: [], custom_prompt_max_tokens: 512,
	  custom_prompt_block_threshold: 0.7, custom_prompt_flag_threshold: 0.4,
	  block_http_status: 403, audit_previous_assistant_output: false,
	})
  })

  it('models all nine official input scanners', () => {
    expect(SCANNER_CATALOG).toHaveLength(9)
    expect(SCANNER_CATALOG.map((item) => item.id)).toContain('suicide_and_self_harm')
  })

  it('keeps, replaces, or explicitly clears a saved token without copying plaintext from the server', () => {
    const draft = configToDraft(config())
    expect(draft.endpoints[0].token).toBe('')
    expect(buildUpdateRequest(draft).endpoints[0]).toMatchObject({ token: undefined, clear_token: false })

    draft.endpoints[0].token = 'temporary-canary-token'
    expect(buildUpdateRequest(draft).endpoints[0]).toMatchObject({ token: 'temporary-canary-token', clear_token: false })

    draft.endpoints[0].token = ''
    draft.endpoints[0].clear_token = true
    expect(buildUpdateRequest(draft).endpoints[0]).toMatchObject({ token: undefined, clear_token: true })
  })

  it('includes the latest-input and previous-output scope in the update payload', () => {
    const draft = configToDraft(config())
	draft.audit_previous_assistant_output = true
	expect(buildUpdateRequest(draft)).toMatchObject({
	  blocking_latest_turn_only: true,
	  audit_previous_assistant_output: true,
	})
  })

  it('tracks dirty state from the full normalized save payload', () => {
    const original = configToDraft(config())
    const changed = configToDraft(config())
    expect(draftFingerprint(changed)).toBe(draftFingerprint(original))
    changed.queue_capacity += 1
    expect(draftFingerprint(changed)).not.toBe(draftFingerprint(original))

    changed.custom_prompt_max_tokens = 1024
    expect(buildUpdateRequest(changed).custom_prompt_max_tokens).toBe(1024)
	changed.custom_prompt_block_threshold = 0.8
	changed.custom_prompt_flag_threshold = 0.3
	changed.block_http_status = 422
	changed.block_message = 'blocked by test policy'
	expect(buildUpdateRequest(changed)).toMatchObject({
	  custom_prompt_block_threshold: 0.8,
	  custom_prompt_flag_threshold: 0.3,
	  block_http_status: 422,
	  block_message: 'blocked by test policy',
	})
  })

  it('requires a valid explicit range and sends canonical ISO timestamps for filter deletion', () => {
    const filters = emptyEventFilters()
    expect(hasExplicitDeleteRange(filters)).toBe(false)
    filters.start_at = '2026-07-15T10:00'
    filters.end_at = '2026-07-16T10:00'
    filters.group_id = '9'
    filters.execution_mode = 'async_audit'
    expect(hasExplicitDeleteRange(filters)).toBe(true)
    expect(eventFilterPayload(filters)).toMatchObject({
      group_id: 9,
      execution_mode: 'async_audit',
      start_at: new Date(filters.start_at).toISOString(),
      end_at: new Date(filters.end_at).toISOString(),
    })
  })
})
