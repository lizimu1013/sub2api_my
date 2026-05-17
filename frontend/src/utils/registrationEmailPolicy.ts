const EMAIL_SUFFIX_TOKEN_SPLIT_RE = /[\s,，]+/
const EMAIL_SUFFIX_INVALID_CHAR_RE = /[^a-z0-9.-]/g
const EMAIL_SUFFIX_INVALID_CHAR_CHECK_RE = /[^a-z0-9.-]/
const EMAIL_SUFFIX_PREFIX_RE = /^@+/
const EMAIL_BLACKLIST_LOCAL_INVALID_RE = /[\s@]/
const EMAIL_SUFFIX_DOMAIN_PATTERN =
  /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$/

// normalizeRegistrationEmailSuffixDomain converts raw input into a canonical domain token.
// It removes leading "@", lowercases input, and strips all invalid characters.
export function normalizeRegistrationEmailSuffixDomain(raw: string): string {
  let value = String(raw || '').trim().toLowerCase()
  if (!value) {
    return ''
  }
  value = value.replace(EMAIL_SUFFIX_PREFIX_RE, '')
  value = value.replace(EMAIL_SUFFIX_INVALID_CHAR_RE, '')
  return value
}

export function normalizeRegistrationEmailSuffixDomains(
  items: string[] | null | undefined
): string[] {
  if (!items || items.length === 0) {
    return []
  }

  const seen = new Set<string>()
  const normalized: string[] = []
  for (const item of items) {
    const domain = normalizeRegistrationEmailSuffixDomain(item)
    if (!isRegistrationEmailSuffixDomainValid(domain) || seen.has(domain)) {
      continue
    }
    seen.add(domain)
    normalized.push(domain)
  }
  return normalized
}

export function parseRegistrationEmailSuffixWhitelistInput(input: string): string[] {
  if (!input || !input.trim()) {
    return []
  }

  const seen = new Set<string>()
  const normalized: string[] = []

  for (const token of input.split(EMAIL_SUFFIX_TOKEN_SPLIT_RE)) {
    const domain = normalizeRegistrationEmailSuffixDomainStrict(token)
    if (!isRegistrationEmailSuffixDomainValid(domain) || seen.has(domain)) {
      continue
    }
    seen.add(domain)
    normalized.push(domain)
  }

  return normalized
}

export function parseRegistrationEmailBlacklistInput(input: string): string[] {
  if (!input || !input.trim()) {
    return []
  }

  const seen = new Set<string>()
  const normalized: string[] = []

  for (const token of input.split(EMAIL_SUFFIX_TOKEN_SPLIT_RE)) {
    const item = normalizeRegistrationEmailBlacklistItemStrict(token)
    if (!item || seen.has(item)) {
      continue
    }
    seen.add(item)
    normalized.push(item)
  }

  return normalized
}

export function normalizeRegistrationEmailSuffixWhitelist(
  items: string[] | null | undefined
): string[] {
  return normalizeRegistrationEmailSuffixDomains(items).map((domain) => `@${domain}`)
}

export function normalizeRegistrationEmailBlacklist(
  items: string[] | null | undefined
): string[] {
  if (!items || items.length === 0) {
    return []
  }

  const seen = new Set<string>()
  const normalized: string[] = []
  for (const item of items) {
    const value = normalizeRegistrationEmailBlacklistItem(item)
    if (!value || seen.has(value)) {
      continue
    }
    seen.add(value)
    normalized.push(value)
  }
  return normalized
}

export function normalizeRegistrationEmailBlacklistItem(raw: string): string {
  const value = String(raw || '').trim().toLowerCase()
  if (!value) {
    return ''
  }
  if (!value.includes('@') || value.startsWith('@')) {
    const domain = normalizeRegistrationEmailSuffixDomain(value)
    return isRegistrationEmailSuffixDomainValid(domain) ? `@${domain}` : ''
  }
  return normalizeRegistrationEmailAddress(value)
}

function extractRegistrationEmailDomain(email: string): string {
  const raw = String(email || '').trim().toLowerCase()
  if (!raw) {
    return ''
  }
  const atIndex = raw.indexOf('@')
  if (atIndex <= 0 || atIndex >= raw.length - 1) {
    return ''
  }
  if (raw.indexOf('@', atIndex + 1) !== -1) {
    return ''
  }
  return raw.slice(atIndex + 1)
}

export function isRegistrationEmailSuffixAllowed(
  email: string,
  whitelist: string[] | null | undefined
): boolean {
  const normalizedWhitelist = normalizeRegistrationEmailSuffixWhitelist(whitelist)
  if (normalizedWhitelist.length === 0) {
    return true
  }
  const emailDomain = extractRegistrationEmailDomain(email)
  if (!emailDomain) {
    return false
  }
  const emailSuffix = `@${emailDomain}`
  return normalizedWhitelist.includes(emailSuffix)
}

// Pasted domains should be strict: any invalid character drops the whole token.
function normalizeRegistrationEmailSuffixDomainStrict(raw: string): string {
  let value = String(raw || '').trim().toLowerCase()
  if (!value) {
    return ''
  }
  value = value.replace(EMAIL_SUFFIX_PREFIX_RE, '')
  if (!value || EMAIL_SUFFIX_INVALID_CHAR_CHECK_RE.test(value)) {
    return ''
  }
  return value
}

function normalizeRegistrationEmailBlacklistItemStrict(raw: string): string {
  const value = String(raw || '').trim().toLowerCase()
  if (!value) {
    return ''
  }
  if (!value.includes('@') || value.startsWith('@')) {
    const domain = normalizeRegistrationEmailSuffixDomainStrict(value)
    return isRegistrationEmailSuffixDomainValid(domain) ? `@${domain}` : ''
  }
  return normalizeRegistrationEmailAddress(value)
}

function normalizeRegistrationEmailAddress(raw: string): string {
  const value = String(raw || '').trim().toLowerCase()
  const atIndex = value.indexOf('@')
  if (atIndex <= 0 || atIndex >= value.length - 1) {
    return ''
  }
  if (value.indexOf('@', atIndex + 1) !== -1) {
    return ''
  }
  const local = value.slice(0, atIndex)
  const domain = value.slice(atIndex + 1)
  if (
    !local ||
    EMAIL_BLACKLIST_LOCAL_INVALID_RE.test(local) ||
    !isRegistrationEmailSuffixDomainValid(domain)
  ) {
    return ''
  }
  return `${local}@${domain}`
}

export function isRegistrationEmailSuffixDomainValid(domain: string): boolean {
  if (!domain) {
    return false
  }
  return EMAIL_SUFFIX_DOMAIN_PATTERN.test(domain)
}
