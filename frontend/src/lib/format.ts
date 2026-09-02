import { CURRENCY_META, type Currency } from './currencies'

/**
 * Money, as a string, from integer minor units.
 *
 * E5 replaced `formatPrice(minor)` with this. The old function divided by
 * 100 and that was fine while every price was a dollar price; with drams in
 * the catalogue it is wrong by two orders of magnitude, because a dram has
 * no subdivision and its minor unit IS the dram. The scale now comes from
 * the currency, never from a constant.
 *
 * Built on Intl.NumberFormat for the hard parts — digit grouping and decimal
 * places — but NOT with `style: 'currency'`, deliberately. That mode decides
 * where the symbol goes from the DISPLAY LOCALE, so the same 6,700 drams
 * renders "֏6,700" for an English reader and "6 700 ֏" for an Armenian one:
 * the price tag would change shape with the site language. The design does
 * not do that, and neither does a shelf label. So Intl formats the number
 * and the symbol is placed from the currency's own metadata.
 *
 * The locale handed to Intl is pinned to 'en-US' for the same reason it was
 * before: every machine, and every test, must render a price identically.
 */
export function formatMoney(minor: number, currency: Currency): string {
  const meta = CURRENCY_META[currency]
  const digits = new Intl.NumberFormat('en-US', {
    minimumFractionDigits: meta.minorExponent,
    maximumFractionDigits: meta.minorExponent,
  }).format(minor / 10 ** meta.minorExponent)

  // \u00A0 is a NON-BREAKING space, written as an escape because the two
  // kinds of space are indistinguishable in an editor and this one is load
  // bearing: "15,300" and "֏" must never wrap onto separate lines.
  return meta.position === 'suffix'
    ? `${digits}\u00A0${meta.symbol}`
    : `${meta.symbol}${digits}`
}

/**
 * Minor units → the plain decimal string an admin edits: 1400 USD → "14.00",
 * 6700 AMD → "6700". No symbol, no grouping — a grouped "6,700" pasted back
 * into a number input is not a number.
 */
export function minorToInput(minor: number, currency: Currency): string {
  const { minorExponent } = CURRENCY_META[currency]
  return (minor / 10 ** minorExponent).toFixed(minorExponent)
}

/**
 * The inverse, for the admin's price boxes. Rounds rather than truncating,
 * so "14.999" entered in a dollar field becomes 1500 rather than 1499 — and
 * returns 0 for anything unparseable, which the caller's validation rejects.
 *
 * Math.round on a float is not the money crime it looks like: the float only
 * exists between the input element and this line, and the value that leaves
 * here — and everything after it — is an integer.
 */
export function inputToMinor(value: string, currency: Currency): number {
  const parsed = Number(value.replace(',', '.'))
  if (!Number.isFinite(parsed)) return 0
  return Math.round(parsed * 10 ** CURRENCY_META[currency].minorExponent)
}
