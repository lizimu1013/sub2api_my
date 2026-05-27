import { describe, expect, it } from 'vitest'
import {
  formatRegistrationEmailSuffixWhitelistForMessage,
  isRegistrationEmailSuffixAllowed,
  isRegistrationEmailSuffixDomainValid,
  normalizeRegistrationEmailBlacklist,
  normalizeRegistrationEmailBlacklistItem,
  normalizeRegistrationIdentityBlacklist,
  normalizeRegistrationIdentityBlacklistItem,
  normalizeRegistrationIPBlacklist,
  normalizeRegistrationIPBlacklistItem,
  normalizeRegistrationEmailSuffixDomain,
  normalizeRegistrationEmailSuffixDomains,
  normalizeRegistrationEmailSuffixWhitelist,
  parseRegistrationEmailBlacklistInput,
  parseRegistrationIdentityBlacklistInput,
  parseRegistrationIPBlacklistInput,
  parseRegistrationEmailSuffixWhitelistInput
} from '@/utils/registrationEmailPolicy'

describe('registrationEmailPolicy utils', () => {
  it('normalizeRegistrationEmailSuffixDomain lowercases, strips @, and ignores invalid chars', () => {
    expect(normalizeRegistrationEmailSuffixDomain(' @Exa!mple.COM ')).toBe('example.com')
    expect(normalizeRegistrationEmailSuffixDomain(' *.EDU!.CN ')).toBe('*.edu.cn')
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
        '@foo.bar',
        '*.EDU.CN',
        '*.edu.cn'
      ])
    ).toEqual(['example.com', 'foo.bar', '*.edu.cn'])
  })

  it('parseRegistrationEmailSuffixWhitelistInput supports separators and deduplicates', () => {
    const input = '\n  @example.com,example.com，@foo.bar\t@FOO.bar *.EDU.CN  '
    expect(parseRegistrationEmailSuffixWhitelistInput(input)).toEqual([
      'example.com',
      'foo.bar',
      '*.edu.cn'
    ])
  })

  it('parseRegistrationEmailSuffixWhitelistInput drops tokens containing invalid chars', () => {
    const input = '@exa!mple.com, @foo.bar, @bad#token.com, @ok-domain.com'
    expect(parseRegistrationEmailSuffixWhitelistInput(input)).toEqual(['foo.bar', 'ok-domain.com'])
  })

  it('parseRegistrationEmailSuffixWhitelistInput drops structurally invalid domains', () => {
    const input = '@-bad.com, @foo..bar.com, @foo.bar, @xn--ok.com, *., *, *.@, *.foo'
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
        ' @foo.bar ',
        '*.EDU.CN'
      ])
    ).toEqual(['@example.com', '@foo.bar', '*.edu.cn'])
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

  it('normalizeRegistrationIdentityBlacklist supports raw and provider-scoped IDs', () => {
    expect(
      normalizeRegistrationIdentityBlacklist([' 12345 ', 'LinuxDo:ABC', 'linuxdo:abc', 'bad value'])
    ).toEqual(['12345', 'linuxdo:ABC'])
  })

  it('normalizeRegistrationIdentityBlacklistItem returns canonical identity tokens', () => {
    expect(normalizeRegistrationIdentityBlacklistItem(' LinuxDo:ABC123 ')).toBe('linuxdo:ABC123')
    expect(normalizeRegistrationIdentityBlacklistItem(' abc-123 ')).toBe('abc-123')
    expect(normalizeRegistrationIdentityBlacklistItem('linux do:123')).toBe('')
    expect(normalizeRegistrationIdentityBlacklistItem('linuxdo:')).toBe('')
  })

  it('parseRegistrationIdentityBlacklistInput supports separators and deduplicates', () => {
    const input = ' 12345, linuxdo:ABC linuxdo:abc，oidc:user:with:colon invalid value '
    expect(parseRegistrationIdentityBlacklistInput(input)).toEqual([
      '12345',
      'linuxdo:ABC',
      'oidc:user:with:colon',
      'invalid',
      'value'
    ])
  })

  it('normalizeRegistrationIPBlacklist supports IPs and CIDR ranges', () => {
    expect(
      normalizeRegistrationIPBlacklist([
        ' 192.0.2.1 ',
        '198.51.100.0/24',
        '192.0.2.1',
        'bad'
      ])
    ).toEqual(['192.0.2.1', '198.51.100.0/24'])
  })

  it('normalizeRegistrationIPBlacklistItem returns canonical IP tokens', () => {
    expect(normalizeRegistrationIPBlacklistItem(' 192.000.002.001 ')).toBe('192.0.2.1')
    expect(normalizeRegistrationIPBlacklistItem('198.51.100.0/24')).toBe('198.51.100.0/24')
    expect(normalizeRegistrationIPBlacklistItem('2001:db8::1')).toBe('2001:db8::1')
    expect(normalizeRegistrationIPBlacklistItem('192.0.2.0/99')).toBe('')
    expect(normalizeRegistrationIPBlacklistItem('not-an-ip')).toBe('')
  })

  it('parseRegistrationIPBlacklistInput supports separators and deduplicates', () => {
    const input = '192.0.2.1, 198.51.100.0/24，192.0.2.1 bad'
    expect(parseRegistrationIPBlacklistInput(input)).toEqual(['192.0.2.1', '198.51.100.0/24'])
  })

  it('isRegistrationEmailSuffixDomainValid matches backend-compatible domain rules', () => {
    expect(isRegistrationEmailSuffixDomainValid('example.com')).toBe(true)
    expect(isRegistrationEmailSuffixDomainValid('foo-bar.example.com')).toBe(true)
    expect(isRegistrationEmailSuffixDomainValid('*.edu.cn')).toBe(true)
    expect(isRegistrationEmailSuffixDomainValid('-bad.com')).toBe(false)
    expect(isRegistrationEmailSuffixDomainValid('foo..bar.com')).toBe(false)
    expect(isRegistrationEmailSuffixDomainValid('localhost')).toBe(false)
    expect(isRegistrationEmailSuffixDomainValid('*.foo')).toBe(false)
    expect(isRegistrationEmailSuffixDomainValid('*')).toBe(false)
    expect(isRegistrationEmailSuffixDomainValid('*.@')).toBe(false)
  })

  it('isRegistrationEmailSuffixAllowed allows any email when whitelist is empty', () => {
    expect(isRegistrationEmailSuffixAllowed('user@example.com', [])).toBe(true)
  })

  it('isRegistrationEmailSuffixAllowed applies exact suffix matching', () => {
    expect(isRegistrationEmailSuffixAllowed('user@example.com', ['@example.com'])).toBe(true)
    expect(isRegistrationEmailSuffixAllowed('user@sub.example.com', ['@example.com'])).toBe(false)
    expect(isRegistrationEmailSuffixAllowed('user@qq.com', ['@qq.com'])).toBe(true)
    expect(isRegistrationEmailSuffixAllowed('user@sub.qq.com', ['@qq.com'])).toBe(false)
  })

  it('isRegistrationEmailSuffixAllowed applies wildcard suffix matching', () => {
    expect(isRegistrationEmailSuffixAllowed('student@cs.edu.cn', ['*.edu.cn'])).toBe(true)
    expect(isRegistrationEmailSuffixAllowed('student@edu.cn', ['*.edu.cn'])).toBe(true)
    expect(isRegistrationEmailSuffixAllowed('student@foo.cn', ['*.edu.cn'])).toBe(false)
  })

  it('isRegistrationEmailSuffixAllowed supports mixed exact and wildcard entries', () => {
    const whitelist = ['@a.com', '*.b.cn']
    expect(isRegistrationEmailSuffixAllowed('user@a.com', whitelist)).toBe(true)
    expect(isRegistrationEmailSuffixAllowed('user@school.b.cn', whitelist)).toBe(true)
    expect(isRegistrationEmailSuffixAllowed('user@b.cn', whitelist)).toBe(true)
    expect(isRegistrationEmailSuffixAllowed('user@c.cn', whitelist)).toBe(false)
  })

  it('formatRegistrationEmailSuffixWhitelistForMessage lists up to five entries', () => {
    expect(
      formatRegistrationEmailSuffixWhitelistForMessage(
        ['@a.com', '@b.com', '@c.com', '@d.com', '@e.com'],
        { separator: ', ', more: (count) => `and ${count} more` }
      )
    ).toBe('@a.com, @b.com, @c.com, @d.com, @e.com')
    expect(
      formatRegistrationEmailSuffixWhitelistForMessage(
        ['@a.com', '@b.com', '@c.com', '@d.com', '@e.com', '*.edu.cn', '@f.com'],
        { separator: ', ', more: (count) => `and ${count} more` }
      )
    ).toBe('@a.com, @b.com, @c.com, @d.com, @e.com, and 2 more')
  })
})
