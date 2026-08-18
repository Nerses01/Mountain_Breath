import { createContext, useContext } from 'react'
import { DEFAULT_CURRENCY, isCurrency, type Currency } from './currencies'

/**
 * The market the visitor is shopping in.
 *
 * NOT in the URL, unlike the language — and the difference is worth stating,
 * because the two look like the same problem. A language is part of what a
 * page IS: /hy/shop and /ru/shop are different documents, they carry
 * different <html lang>, they are separately shareable and separately
 * indexable. A currency is a lens on the same document. Putting it in the
 * path would double the URL space for no reader benefit and split every
 * page's search ranking in half.
 *
 * So it lives in two places instead:
 *
 *   localStorage  — survives a reload, and is what the provider reads at
 *                   boot, before the first request goes out.
 *   a cookie      — so the SERVER sees the same choice. That matters for one
 *                   specific thing: POST /orders takes the currency from the
 *                   request rather than from a body field, so the cookie is
 *                   part of what decides the customer is billed in drams.
 *
 * The hook and the provider are split across two files only because a module
 * that exports both a component and a non-component breaks React Fast
 * Refresh — the same reason i18n/useLocale.ts holds no component either.
 */
export const CURRENCY_STORAGE_KEY = 'mb_currency'
export const CURRENCY_COOKIE_NAME = 'mb_currency'
export const CURRENCY_DISPLAY_STORAGE_KEY = 'mb_currency_display'

/**
 * A5 (decision log #89): whether prices draw the muted second-market line.
 * 'dual' is the design's default; 'single' is the settings screen's
 * "USD only" / "AMD only". Client-side ONLY — localStorage like the
 * currency itself, no users column until a two-devices complaint exists.
 * Pure display: orders always show their one charged currency regardless.
 */
export type CurrencyDisplay = 'dual' | 'single'

export interface CurrencyContextValue {
  currency: Currency
  setCurrency: (next: Currency) => void
  display: CurrencyDisplay
  setDisplay: (next: CurrencyDisplay) => void
}

export const CurrencyContext = createContext<CurrencyContextValue | null>(null)

export function readStoredCurrency(): Currency {
  // try/catch, not a feature test: localStorage EXISTS in a Safari private
  // window and throws on access, so `typeof localStorage` proves nothing.
  try {
    const stored = localStorage.getItem(CURRENCY_STORAGE_KEY)
    if (stored && isCurrency(stored)) return stored
  } catch {
    // storage unavailable — fall through to the default
  }
  return DEFAULT_CURRENCY
}

export function readStoredDisplay(): CurrencyDisplay {
  try {
    if (localStorage.getItem(CURRENCY_DISPLAY_STORAGE_KEY) === 'single') return 'single'
  } catch {
    // storage unavailable — the default it is
  }
  return 'dual'
}

export function writeCurrencyCookie(currency: Currency) {
  // Not HttpOnly (JavaScript sets it), SameSite=Lax so it rides along on
  // ordinary navigation, and a year's life because a shopper's market is not
  // a session-scoped fact. Nothing sensitive in it, so no Secure flag in dev
  // — the production origin is HTTPS and adds one at the proxy.
  document.cookie = `${CURRENCY_COOKIE_NAME}=${currency}; path=/; max-age=31536000; samesite=lax`
}

/**
 * The one way to ask "which market are we in?".
 *
 * Falls back to the default outside a provider rather than throwing, so a
 * component rendered in isolation by a test still shows a price. The
 * provider is mounted once, at the app root.
 */
export function useCurrency(): CurrencyContextValue {
  const ctx = useContext(CurrencyContext)
  if (ctx) return ctx
  return {
    currency: DEFAULT_CURRENCY,
    setCurrency: () => {},
    display: 'dual',
    setDisplay: () => {},
  }
}
