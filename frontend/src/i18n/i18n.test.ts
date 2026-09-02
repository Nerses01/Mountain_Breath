import { beforeEach, describe, expect, it } from 'vitest'
import i18n from './index'

describe('i18n configuration', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en')
  })

  it('translates chrome strings in each language', async () => {
    expect(i18n.t('common:nav.shop')).toBe('Shop')

    await i18n.changeLanguage('hy')
    expect(i18n.t('common:nav.shop')).toBe('Խանութ')

    await i18n.changeLanguage('ru')
    expect(i18n.t('common:nav.shop')).toBe('Магазин')
  })

  it('falls back to English for a key a locale is missing', async () => {
    await i18n.changeLanguage('hy')

    // Deliberately not present in hy.ts. A gap must degrade to readable
    // English rather than rendering the raw key or an empty string, so a
    // half-translated page still works.
    expect(i18n.t('common:missing.key.for.test', 'Fallback text')).toBe(
      'Fallback text',
    )
  })

  it('applies English plural rules', () => {
    expect(i18n.t('common:itemCount', { count: 1 })).toBe('1 item')
    expect(i18n.t('common:itemCount', { count: 5 })).toBe('5 items')
  })

  it("applies Russian's three plural forms", async () => {
    await i18n.changeLanguage('ru')

    // The reason this project took an i18n dependency: Russian picks its
    // form from the last digit AND the tens, so 21 is singular-like, 22 is
    // "few", 25 is "many". No count === 1 ternary can express that.
    expect(i18n.t('common:itemCount', { count: 1 })).toBe('1 товар')
    expect(i18n.t('common:itemCount', { count: 2 })).toBe('2 товара')
    expect(i18n.t('common:itemCount', { count: 5 })).toBe('5 товаров')
    expect(i18n.t('common:itemCount', { count: 21 })).toBe('21 товар')
    expect(i18n.t('common:itemCount', { count: 22 })).toBe('22 товара')
  })

  it('interpolates values', () => {
    expect(i18n.t('footer:legal.rights', { year: 2026 })).toBe(
      '© 2026 Mountain Breath Apiary',
    )
  })
})
