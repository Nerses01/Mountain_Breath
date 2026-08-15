import { Link } from 'react-router'
import { Trans, useTranslation } from 'react-i18next'
import {
  useCart,
  useMe,
  usePreview,
  useRemoveCartItem,
  useSaveForLater,
  useSetCartItem,
} from '../api/hooks'
import type { CartItem } from '../api/types'
import type { Currency } from '../lib/currencies'
import { useLocale } from '../i18n/useLocale'
import { formatMoney } from '../lib/format'
import { FreeShippingBanner } from '../components/checkout/FreeShippingBanner'
import { OrderSummary } from '../components/checkout/OrderSummary'
import { PromoBox } from '../components/checkout/PromoBox'
import { Price } from '../components/ui/Price'
import { QtyStepper } from '../components/ui/QtyStepper'

/**
 * Screen 04 — the designed cart (E7; the plan parked the cart's design work
 * as "E6/E7" and this is the E7 half). Left column: the lines, the
 * keep-shopping row, the free-shipping banner. Right column: the dark
 * summary card and the promo box, exactly as the mock stacks them.
 *
 * Every money figure on this page comes from /checkout/preview — the same
 * calculator the checkout renders and the order is priced by. The cart
 * response only contributes the LINES (and their per-market prices for the
 * muted second line); nothing here sums anything.
 */
export function CartPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const me = useMe()
  const cart = useCart(!!me.data)
  const preview = usePreview(!!me.data)
  const setItem = useSetCartItem()
  const removeItem = useRemoveCartItem()
  const saveForLater = useSaveForLater()

  if (me.isPending || cart.isPending || (me.data && preview.isPending)) {
    return <Shell>{t('common:state.loading')}</Shell>
  }
  if (!me.data) {
    return (
      <Shell>
        <p className="text-ink-body">
          {/* <Trans> renders a sentence that CONTAINS a link — split into
              string + link + string it would be untranslatable, since word
              order differs per language and Armenian puts the verb last. */}
          <Trans
            i18nKey="cart:signInRequired"
            components={[
              <span key="0" />,
              <Link
                key="1"
                to={localePath('/login')}
                className="font-semibold text-brand-ink hover:underline"
              />,
            ]}
          />
        </p>
      </Shell>
    )
  }
  if (cart.isError || preview.isError) {
    return (
      <Shell>
        <p className="text-danger">{t('common:state.loadFailed')}</p>
      </Shell>
    )
  }

  const items = cart.data?.items ?? []

  return (
    <Shell>
      <h1 className="font-display text-display-md font-extrabold text-ink">
        {t('cart:title')}
      </h1>

      {items.length === 0 && (
        <p className="mt-4 text-ink-body">
          {t('cart:empty')}{' '}
          <Link to={localePath('/shop')} className="font-semibold text-brand-ink hover:underline">
            {t('cart:browse')}
          </Link>
        </p>
      )}

      {items.length > 0 && cart.data && preview.data && (
        <div className="mt-7 grid items-start gap-9 lg:grid-cols-[1fr_400px]">
          <div className="flex flex-col gap-5">
            <ul className="flex flex-col rounded-2xl bg-card px-7">
              {items.map((it) => (
                <CartLine
                  key={it.variant_id}
                  item={it}
                  currency={cart.data.currency}
                  onQty={(qty) =>
                    qty < 1
                      ? removeItem.mutate(it.variant_id)
                      : setItem.mutate({ variantId: it.variant_id, qty })
                  }
                  onRemove={() => removeItem.mutate(it.variant_id)}
                  onSaveForLater={() => saveForLater.mutate(it.variant_id)}
                  savingForLater={saveForLater.isPending}
                />
              ))}
              <li className="flex items-center justify-between py-5">
                <Link
                  to={localePath('/shop')}
                  className="font-display text-[0.9375rem] font-semibold text-brand transition hover:text-brand-ink"
                >
                  {t('cart:keepShopping')}
                </Link>
                <span className="text-sm text-ink-faint">{t('cart:pricesIncludeVat')}</span>
              </li>
            </ul>

            <FreeShippingBanner preview={preview.data} cart={cart.data} />
          </div>

          <div className="flex flex-col gap-4.5">
            <OrderSummary
              cart={cart.data}
              preview={preview.data}
              action={
                <Link
                  to={localePath('/checkout')}
                  className="rounded-full bg-brand px-8 py-4 text-center font-display font-bold text-ink-on-dark transition hover:bg-brand-ink"
                >
                  {t('cart:goToCheckout')}
                </Link>
              }
            />
            <PromoBox preview={preview.data} />
          </div>
        </div>
      )}
    </Shell>
  )
}

/**
 * One designed cart row: image slot, name linking to the product, size and
 * unit price, the qty stepper, the dual-currency line total, remove. Not
 * one big link — the row holds three other controls, and nesting
 * interactive elements inside an anchor is invalid HTML (the ProductCard
 * lesson, reapplied).
 */
function CartLine({
  item,
  currency,
  onQty,
  onRemove,
  onSaveForLater,
  savingForLater,
}: {
  item: CartItem
  currency: Currency
  onQty: (qty: number) => void
  onRemove: () => void
  onSaveForLater: () => void
  savingForLater: boolean
}) {
  const { t } = useTranslation()
  const { localePath } = useLocale()

  return (
    <li className="flex flex-wrap items-center gap-5 border-b border-line-soft py-5">
      <div
        aria-hidden="true"
        className="size-16 shrink-0 rounded-xl bg-[repeating-linear-gradient(135deg,rgba(70,40,28,0.12)_0_8px,rgba(70,40,28,0.05)_8px_16px)]"
      />
      <div className="flex min-w-36 flex-1 flex-col gap-0.5">
        <Link
          to={localePath(`/products/${item.product_slug}`)}
          className="font-display text-[0.9375rem] font-semibold text-ink hover:text-brand-ink"
        >
          {item.product_name}
        </Link>
        <span className="text-xs text-ink-soft">
          {/* "6,700 ֏ each" — the unit price, in the cart's own resolved
              currency (not the switcher's, so an in-flight response is
              labelled with what it was actually priced in). */}
          {item.label} · {formatMoney(item.price_minor, currency)} {t('cart:each')}
        </span>
      </div>

      <div className="flex flex-col items-center gap-1.5">
        <QtyStepper
          value={item.qty}
          onChange={onQty}
          min={0}
          max={item.stock_qty}
          label={t('common:actions.cart')}
          decreaseLabel={t('cart:decrease')}
          increaseLabel={t('cart:increase')}
          className="px-4 py-2"
        />
        {/* E8: the design's "Save for later" — a MOVE to the wishlist, so
            the line leaves the cart and a heart appears in its place. */}
        <button
          type="button"
          onClick={onSaveForLater}
          disabled={savingForLater}
          className="text-xs font-semibold text-ink-muted transition hover:text-brand-ink disabled:opacity-50"
        >
          {t('cart:saveForLater')}
        </button>
      </div>

      <Price prices={item.line_totals} primaryMinor={item.line_total_minor} className="w-24 items-end" />

      <button
        type="button"
        onClick={onRemove}
        aria-label={t('cart:remove')}
        title={t('cart:remove')}
        className="text-ink-faint transition hover:text-danger"
      >
        ✕
      </button>
    </li>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-360 px-6 py-10 lg:px-14">{children}</div>
}
