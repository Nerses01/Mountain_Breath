// Money arrives as integer minor units (e.g. 350000 = 3500.00) — format at
// the last moment, only for display. Locale is pinned so every machine
// (and every test) renders prices identically.
export function formatPrice(priceMinor: number): string {
  return (priceMinor / 100).toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
}

/**
 * The same number with its currency symbol — "$14.00".
 *
 * Separate from formatPrice rather than replacing it because the two have
 * different futures: E5 makes the shop dual-currency, at which point THIS is
 * the function that grows a currency argument and starts asking which market
 * the visitor is in, while formatPrice stays a plain number formatter for
 * the admin's price inputs.
 *
 * The symbol is hardcoded for now, which is honest: E2 seeds USD only
 * (decision #2 puts the AMD column in E5), so a currency parameter today
 * would be a parameter with one legal value.
 */
export function formatMoney(priceMinor: number): string {
  return `$${formatPrice(priceMinor)}`
}
