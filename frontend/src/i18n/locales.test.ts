import { describe, expect, it } from 'vitest'
import { isLocale, localePrefix, pathForLocale } from './locales'

describe('locale helpers', () => {
  it('recognises supported locales only', () => {
    expect(isLocale('en')).toBe(true)
    expect(isLocale('hy')).toBe(true)
    expect(isLocale('ru')).toBe(true)
    expect(isLocale('de')).toBe(false)
    expect(isLocale(undefined)).toBe(false)
  })

  it('gives English no prefix and the others one', () => {
    // The stated requirement: English is the default and bare `/` serves it,
    // so every link written elsewhere keeps working unprefixed.
    expect(localePrefix('en')).toBe('')
    expect(localePrefix('hy')).toBe('/hy')
    expect(localePrefix('ru')).toBe('/ru')
  })

  describe('pathForLocale', () => {
    it('adds a prefix when leaving English', () => {
      expect(pathForLocale('/products/honey', 'hy')).toBe('/hy/products/honey')
    })

    it('strips the prefix when returning to English', () => {
      expect(pathForLocale('/hy/products/honey', 'en')).toBe('/products/honey')
    })

    it('swaps one prefix for another, keeping the page', () => {
      // The point of the whole function: switching language must not throw
      // you back to the home page.
      expect(pathForLocale('/hy/products/honey', 'ru')).toBe('/ru/products/honey')
    })

    it('handles the root path in both directions', () => {
      expect(pathForLocale('/', 'hy')).toBe('/hy')
      expect(pathForLocale('/hy', 'en')).toBe('/')
      expect(pathForLocale('/ru', 'hy')).toBe('/hy')
    })

    it('leaves a non-locale first segment alone', () => {
      // `/de/...` is not a locale, so it is an ordinary path segment and
      // must survive the rewrite rather than being eaten as a prefix.
      expect(pathForLocale('/de/thing', 'hy')).toBe('/hy/de/thing')
    })
  })
})
