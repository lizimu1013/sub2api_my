import { describe, expect, it } from 'vitest'
import {
  isRegistrationEmailSuffixAllowed,
  isRegistrationEmailSuffixDomainValid,
  normalizeRegistrationEmailBlacklist,
  normalizeRegistrationEmailBlacklistItem,
  normalizeRegistrationEmailSuffixDomain,
  normalizeRegistrationEmailSuffixDomains,
  normalizeRegistrationEmailSuffixWhitelist,
  parseRegistrationEmailBlacklistInput,
  parseRegistrationEmailSuffixWhitelistInput
} from '@/utils/registrationEmailPolicy'

describe('registrationEmailPolicy utils', () => {
  it('normalizeRegistrationEmailSuffixDomain lowercases, strips @, and ignores invalid chars', () => {
    expect(normalizeRegistrationEmailSuffixDomain(' @Exa!mple.COM ')).toBe('example.com')
  })

  it('normalizeRegistrationEmailSuffixDomains deduplicates normalized domains', () => {
    expect(
      normalizeRegistrationEmailSuffixDomains([
        '@example.com',
        'Example.com',
        '',
        '-invalid.com',
        'foo..bar.com',
        ' @foo.bar ',
        '@foo.bar'
      ])
    ).toEqual(['example.com', 'foo.bar'])
  })

  it('parseRegistrationEmailSuffixWhitelistInput supports separators and deduplicates', () => {
    const input = '\n  @example.com,example.com，@foo.bar\t@FOO.bar  '
    expect(parseRegistrationEmailSuffixWhitelistInput(input)).toEqual(['example.com', 'foo.bar'])
  })

  it('parseRegistrationEmailSuffixWhitelistInput drops tokens containing invalid chars', () => {
    const input = '@exa!mple.com, @foo.bar, @bad#token.com, @ok-domain.com'
    expect(parseRegistrationEmailSuffixWhitelistInput(input)).toEqual(['foo.bar', 'ok-domain.com'])
  })

  it('parseRegistrationEmailSuffixWhitelistInput drops structurally invalid domains', () => {
    const input = '@-bad.com, @foo..bar.com, @foo.bar, @xn--ok.com'
    expect(parseRegistrationEmailSuffixWhitelistInput(input)).toEqual(['foo.bar', 'xn--ok.com'])
  })

  it('parseRegistrationEmailSuffixWhitelistInput returns empty list for blank input', () => {
    expect(parseRegistrationEmailSuffixWhitelistInput('   \n \n')).toEqual([])
  })

  it('normalizeRegistrationEmailSuffixWhitelist returns canonical @domain list', () => {
    expect(
      normalizeRegistrationEmailSuffixWhitelist([
        '@Example.com',
        'foo.bar',
        '',
        '-invalid.com',
        ' @foo.bar '
      ])
    ).toEqual(['@example.com', '@foo.bar'])
  })

  it('normalizeRegistrationEmailBlacklist supports exact emails and domains', () => {
    expect(
      normalizeRegistrationEmailBlacklist([
        'Blocked@Example.com',
        'example.com',
        '@EXAMPLE.com',
        'bad@@example.com'
      ])
    ).toEqual(['blocked@example.com', '@example.com'])
  })

  it('normalizeRegistrationEmailBlacklistItem returns canonical blacklist tokens', () => {
    expect(normalizeRegistrationEmailBlacklistItem(' User+tag@Example.COM ')).toBe(
      'user+tag@example.com'
    )
    expect(normalizeRegistrationEmailBlacklistItem(' example.com ')).toBe('@example.com')
    expect(normalizeRegistrationEmailBlacklistItem('bad@@example.com')).toBe('')
  })

  it('parseRegistrationEmailBlacklistInput supports separators and deduplicates', () => {
    const input = 'Blocked@Example.com, @evil.com evil.com bad@@example.com'
    expect(parseRegistrationEmailBlacklistInput(input)).toEqual([
      'blocked@example.com',
      '@evil.com'
    ])
  })

  it('isRegistrationEmailSuffixDomainValid matches backend-compatible domain rules', () => {
    expect(isRegistrationEmailSuffixDomainValid('example.com')).toBe(true)
    expect(isRegistrationEmailSuffixDomainValid('foo-bar.example.com')).toBe(true)
    expect(isRegistrationEmailSuffixDomainValid('-bad.com')).toBe(false)
    expect(isRegistrationEmailSuffixDomainValid('foo..bar.com')).toBe(false)
    expect(isRegistrationEmailSuffixDomainValid('localhost')).toBe(false)
  })

  it('isRegistrationEmailSuffixAllowed allows any email when whitelist is empty', () => {
    expect(isRegistrationEmailSuffixAllowed('user@example.com', [])).toBe(true)
  })

  it('isRegistrationEmailSuffixAllowed applies exact suffix matching', () => {
    expect(isRegistrationEmailSuffixAllowed('user@example.com', ['@example.com'])).toBe(true)
    expect(isRegistrationEmailSuffixAllowed('user@sub.example.com', ['@example.com'])).toBe(false)
  })
})
