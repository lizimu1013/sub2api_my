import { describe, expect, it } from 'vitest'

import enLanding from '../locales/en/landing'
import zhLanding from '../locales/zh/landing'

const requiredHomeKeys = [
  'navLinks.pain',
  'navLinks.features',
  'navLinks.compare',
  'navLinks.providers',
  'heroBadge',
  'heroTitleLine1',
  'heroTitleLine2',
  'heroTitleAccent',
  'stats.models',
  'stats.availability',
  'stats.billingValue',
  'stats.billingLabel',
  'terminalRouting',
  'tags.loadBalancing',
  'painPoints.kicker',
  'painPoints.description'
] as const

function getValue(source: Record<string, unknown>, path: string): unknown {
  return path.split('.').reduce<unknown>((value, segment) => {
    if (!value || typeof value !== 'object') return undefined
    return (value as Record<string, unknown>)[segment]
  }, source)
}

describe('home locale keys', () => {
  it.each([
    ['en', enLanding.home],
    ['zh', zhLanding.home]
  ])('%s includes every custom home key', (_locale, home) => {
    for (const key of requiredHomeKeys) {
      expect(getValue(home, key), key).toEqual(expect.any(String))
    }
  })
})
