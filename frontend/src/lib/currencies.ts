/**
 * The markets the shop prices in — mirroring the `currencies` table from
 * migration 000016.
 *
 * WHY THIS IS A STATIC FILE AND NOT AN API CALL. Every price on every screen
 * goes through formatMoney, hundreds of times per render, inside components
 * that cannot await anything. Fetching the table would make formatting
 * asynchronous, put a loading state under every price tag, and buy nothing:
 * a currency's symbol and its number of decimals are ISO facts that change
 * approximately never. The same argument i18n/locales.ts already makes for
 * the locale set.
 *
 * The duplication is real, and the backend pins it: a store test compares
 * domain.Currencies against the table, and this file is checked against
 * domain.Currencies by the type of CURRENCIES below plus the api test that
 * asserts on the codes. What the database stays sole authority for is the
 * part only it uses — the ROUNDING of a converted price. No arithmetic here
 * ever produces a price; it only renders one the server computed.
 */
export const CURRENCIES = ['USD', 'AMD'] as const

export type Currency = (typeof CURRENCIES)[number]

export const DEFAULT_CURRENCY: Currency = 'USD'

interface CurrencyMeta {
  symbol: string
  /**
   * Where the symbol sits relative to the digits. It belongs to the
   * CURRENCY, not to the reader's language: the design writes "$14.00" and
   * "6,700 ֏" on the same line, and that pairing must not flip when the site
   * is read in Armenian.
   */
  position: 'prefix' | 'suffix'
  /**
   * How many decimal places the minor unit implies — the reason
   * `priceMinor / 100` is now a bug. 1400 USD-minor is $14.00; 6700 AMD-minor
   * is 6,700 ֏, because a dram has no subdivision in circulation.
   */
  minorExponent: number
}

export const CURRENCY_META: Record<Currency, CurrencyMeta> = {
  USD: { symbol: '$', position: 'prefix', minorExponent: 2 },
  AMD: { symbol: '֏', position: 'suffix', minorExponent: 0 },
}

export function isCurrency(value: string): value is Currency {
  return (CURRENCIES as readonly string[]).includes(value)
}

/**
 * The market to show in the muted second line — the one the visitor is NOT
 * currently shopping in.
 *
 * Returns undefined when there is nothing to show, which happens for real:
 * a variant priced only in dollars, with no rate on file, comes back from the
 * API with a single entry in `prices`.
 */
export function secondaryCurrency(
  primary: Currency,
  prices: Partial<Record<Currency, number>> | undefined,
): Currency | undefined {
  if (!prices) return undefined
  return CURRENCIES.find((c) => c !== primary && prices[c] !== undefined)
}
