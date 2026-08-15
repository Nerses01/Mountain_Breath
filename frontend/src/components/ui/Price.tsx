import { cx } from '../../lib/cx'
import { secondaryCurrency, type Currency } from '../../lib/currencies'
import { formatMoney } from '../../lib/format'
import { useCurrency } from '../../lib/useCurrency'

interface PriceProps {
  /** Every market this amount is available in, keyed by ISO code. */
  prices: Partial<Record<Currency, number>> | undefined
  /**
   * The amount in the currency the API already resolved. Preferred over
   * prices[currency] where a caller has it, so a card renders exactly what
   * the SERVER priced it at rather than re-picking from the map.
   */
  primaryMinor?: number
  /** The design uses 38 / 20 / 17 / 15 px in different places. */
  size?: 'sm' | 'md' | 'lg' | 'xl'
  /** Stacked (cards, cart totals) or side by side (product page). */
  layout?: 'stack' | 'inline'
  /** The design paints the cart total on bark in honey rather than orange. */
  tone?: 'default' | 'on-dark'
  /**
   * Wraps the formatted primary amount in a sentence — the card's "from
   * $14.00".
   *
   * A CALLBACK rather than a `prefix` string, because "from" is not a prefix
   * in every language: the Armenian catalogue writes `{{price}}-ից`, a
   * suffix agreeing with the number it follows. Handing the message the
   * formatted amount and letting it decide where the word goes is the only
   * version that survives translation.
   */
  format?: (amount: string) => string
  className?: string
}

const primarySize = {
  sm: 'text-[0.9375rem]', // 15px — cart line
  md: 'text-[1.0625rem]', // 17px — related cards
  lg: 'text-lg', // 20px — shop grid
  xl: 'text-display-md', // 38px — the product page's buy box
} as const

const secondarySize = {
  sm: 'text-2xs',
  md: 'text-xs',
  lg: 'text-xs',
  xl: 'text-[1.0625rem]', // 17px
} as const

/**
 * The design's price: a bold amount in the shopper's market, and the same
 * amount in the other one, muted, underneath — or beside it in the buy box.
 *
 * The second line is NOT a conversion. Both figures come from the API, which
 * read them from two rows of variant_prices; see migration 000016 for why a
 * shelf price per market beats one price and a rate.
 *
 * DEPARTURE FROM THE CANVAS (accessibility, standing exception 1). The mock
 * paints the secondary line #a9714b — --color-ink-faint — which fails AA as
 * text on every light surface. E5 kept it for the buy box's 17px believing
 * the large-text exemption applied; E10's axe run corrected the reading —
 * the 3:1 exemption starts at 18.66px BOLD or 24px regular, and this line
 * is 17px regular. Every size now uses --color-ink-muted: the design's own
 * colour two steps darker, still reading as a muted pair.
 *
 * It renders NOTHING when there is no price, rather than "$0.00": a variant
 * the shop cannot price in any market is not free, it is unbuyable, and the
 * store already declines to offer one. This branch is for the loading frame.
 */
export function Price({
  prices,
  primaryMinor,
  size = 'md',
  layout = 'stack',
  tone = 'default',
  format,
  className,
}: PriceProps) {
  const { currency } = useCurrency()

  const primary = primaryMinor ?? prices?.[currency]
  if (primary === undefined) return null

  const secondary = secondaryCurrency(currency, prices)
  const secondaryMinor = secondary ? prices?.[secondary] : undefined

  return (
    <span
      className={cx(
        'flex',
        layout === 'inline' ? 'items-baseline gap-3.5' : 'flex-col gap-0.5',
        className,
      )}
    >
      <span
        className={cx(
          'font-display font-extrabold',
          primarySize[size],
          tone === 'on-dark' ? 'text-honey' : 'text-brand-ink',
        )}
      >
        {format ? format(formatMoney(primary, currency)) : formatMoney(primary, currency)}
      </span>
      {secondary && secondaryMinor !== undefined && (
        <span
          className={cx(
            secondarySize[size],
            tone === 'on-dark' ? 'text-ink-on-dark-soft' : 'text-ink-muted',
          )}
        >
          {formatMoney(secondaryMinor, secondary)}
        </span>
      )}
    </span>
  )
}
