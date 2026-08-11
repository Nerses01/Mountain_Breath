import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { setApiCurrency } from '../api/client'
import type { Currency } from './currencies'
import {
  CURRENCY_STORAGE_KEY,
  CurrencyContext,
  readStoredCurrency,
  writeCurrencyCookie,
} from './useCurrency'

/** Mounted once at the app root; see useCurrency.ts for where the choice lives. */
export function CurrencyProvider({ children }: { children: ReactNode }) {
  const [currency, setCurrencyState] = useState<Currency>(readStoredCurrency)

  // Told to the API client SYNCHRONOUSLY during render, for exactly the
  // reason useLocale spells out: TanStack Query fires its request during this
  // render, and an effect would run too late — leaving the first request
  // after a reload asking for dollars when the visitor shops in drams.
  setApiCurrency(currency)

  // The cookie CAN wait for an effect, because nothing reads it before the
  // first checkout: every request also carries ?currency= explicitly, and
  // the cookie only covers what the frontend does not build a URL for.
  useEffect(() => {
    writeCurrencyCookie(currency)
  }, [currency])

  const setCurrency = useCallback((next: Currency) => {
    try {
      localStorage.setItem(CURRENCY_STORAGE_KEY, next)
    } catch {
      // storage unavailable — the cookie still carries the choice
    }
    setCurrencyState(next)
  }, [])

  const value = useMemo(() => ({ currency, setCurrency }), [currency, setCurrency])
  return <CurrencyContext value={value}>{children}</CurrencyContext>
}
