// Money arrives as integer minor units (e.g. 350000 = 3500.00) — format at
// the last moment, only for display. Locale is pinned so every machine
// (and every test) renders prices identically.
export function formatPrice(priceMinor: number): string {
  return (priceMinor / 100).toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
}
