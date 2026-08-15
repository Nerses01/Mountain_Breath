import { useEffect, useState, type ReactNode } from 'react'
import { Link, useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { useCart, useCheckout, useDefaultAddress, useMe, usePreview } from '../api/hooks'
import { ApiError } from '../api/client'
import type { Address, PaymentMethod } from '../api/types'
import { OrderSummary } from '../components/checkout/OrderSummary'
import { PromoBox } from '../components/checkout/PromoBox'
import { Button } from '../components/ui/Button'
import { Checkbox } from '../components/ui/Checkbox'
import { Input } from '../components/ui/Input'
import { useFieldErrors } from '../i18n/useFieldErrors'
import { useLocale } from '../i18n/useLocale'
import { useCurrency } from '../lib/useCurrency'
import { cx } from '../lib/cx'

/**
 * Screen 05 — the checkout. One page, two numbered steps (the design fills
 * both "1 Details" and "2 Payment" at once; "3 Done" is the confirmation).
 *
 * It renders under its OWN minimal chrome — logo, step indicator, "Secure" —
 * rather than inside Layout, because that is what the mock draws and the
 * reasoning is sound: the site nav is an invitation to wander off, and the
 * one page where the shop wants no wandering is the one with the money on it.
 *
 * VALIDATION IS HAND-ROLLED, mirroring the backend's field keys
 * (`address.postal_code`), per the plan's own instruction — this project has
 * avoided form libraries so far and nothing here needs one: the client
 * checks only presence (so an empty form fails without a round trip), and
 * every richer rule (cash-is-AMD-only) arrives from the server through the
 * same `fields` envelope and lands on the same inputs. Two sources, one
 * rendering path. Revisit if E8's forms make this hurt; note the decision
 * either way (this comment is the note).
 */

const EMPTY_ADDRESS: Address = {
  first_name: '',
  last_name: '',
  phone: '',
  street: '',
  city: '',
  postal_code: '',
  country: 'AM',
}

// The keys the form posts, in the order the inputs appear — used to focus
// the first invalid field after a failed submit.
const ADDRESS_FIELDS: (keyof Address)[] = [
  'first_name', 'last_name', 'phone', 'street', 'city', 'postal_code', 'country',
]

export function CheckoutPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const { currency } = useCurrency()
  const navigate = useNavigate()

  const qc = useQueryClient()
  const me = useMe()
  const cart = useCart(!!me.data)
  // E7: every figure in the sidebar comes from the one calculator — the
  // same domain.Price call that will price the order itself.
  const preview = usePreview(!!me.data)
  const saved = useDefaultAddress(!!me.data)
  const checkout = useCheckout()

  const [address, setAddress] = useState<Address>(EMPTY_ADDRESS)
  const [method, setMethod] = useState<PaymentMethod>('card')
  const [note, setNote] = useState('')
  const [neighbour, setNeighbour] = useState(false)
  // Presence errors found client-side, keyed exactly like the server's.
  const [localErrors, setLocalErrors] = useState<Record<string, string>>({})

  // Pre-fill from the address book once it arrives. An effect rather than
  // initial state because the query resolves after the first render — and it
  // must not overwrite anything the customer already typed, hence the guard
  // on a pristine form.
  useEffect(() => {
    if (saved.data && address === EMPTY_ADDRESS) {
      setAddress(saved.data)
    }
  }, [saved.data, address])

  const serverErrors = useFieldErrors(checkout.error)
  const fieldError = (key: string): string | undefined =>
    localErrors[key] ?? serverErrors.fieldError(key)

  function setField(key: keyof Address, value: string) {
    setAddress((a) => ({ ...a, [key]: value }))
    // An error clears the moment its field is edited — feedback tied to the
    // input, not to the last submit attempt.
    setLocalErrors(({ [`address.${key}`]: _, ...rest }) => rest)
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const missing: Record<string, string> = {}
    for (const key of ADDRESS_FIELDS) {
      if (!address[key].trim()) {
        missing[`address.${key}`] = t('validation:required')
      }
    }
    setLocalErrors(missing)
    if (Object.keys(missing).length > 0) {
      const first = ADDRESS_FIELDS.find((k) => missing[`address.${k}`])
      document.getElementById(`checkout-${first}`)?.focus()
      return
    }

    checkout.mutate(
      {
        address,
        payment_method: method,
        delivery_note: note,
        leave_with_neighbour: neighbour,
      },
      {
        onSuccess: (order) =>
          navigate(localePath(`/orders/${order.id}`), { state: { placed: true } }),
        onError: (err) => {
          // 409 promo_invalid: the cart's code died between apply and
          // "Place the order" (expired, sold out, basket shrank). The
          // server refused rather than silently charging a different
          // total; refetching the preview makes the promo box name the
          // reason inline, next to the code it is about.
          if (err instanceof ApiError && err.code === 'promo_invalid') {
            void qc.invalidateQueries({ queryKey: ['preview'] })
          }
        },
      },
    )
  }

  // ── Guard states ────────────────────────────────────────────────────────
  if (me.isPending || cart.isPending || (me.data && preview.isPending)) {
    return <CheckoutShell step={1}>{t('common:state.loading')}</CheckoutShell>
  }
  if (!me.data) {
    return (
      <CheckoutShell step={1}>
        <p className="text-ink-body">
          {t('checkout:signInFirst')}{' '}
          <Link to={localePath('/login')} className="font-semibold text-brand-ink hover:underline">
            {t('common:actions.signIn')}
          </Link>
        </p>
      </CheckoutShell>
    )
  }
  if (!cart.data || !preview.data || cart.data.items.length === 0) {
    return (
      <CheckoutShell step={1}>
        <p className="text-ink-body">
          {t('cart:empty')}{' '}
          <Link to={localePath('/shop')} className="font-semibold text-brand-ink hover:underline">
            {t('cart:browse')}
          </Link>
        </p>
      </CheckoutShell>
    )
  }

  const cashDisabled = currency !== 'AMD'

  return (
    <CheckoutShell step={1}>
      <form
        onSubmit={onSubmit}
        noValidate
        className="grid gap-9 lg:grid-cols-[1fr_400px]"
      >
        <div className="flex flex-col gap-6">
          <h1 className="font-display text-display-md font-extrabold text-ink">
            {t('checkout:title')}
          </h1>

          {serverErrors.formError && (
            <p role="alert" className="rounded-xl bg-danger/10 p-4 text-sm text-danger">
              {serverErrors.formError}
            </p>
          )}

          {/* ── Contact ─────────────────────────────────────────────── */}
          <CheckoutSection title={t('checkout:contact.title')}>
            <div className="grid gap-3.5 sm:grid-cols-2">
              <Input
                id="checkout-first_name"
                label={t('checkout:contact.firstName')}
                autoComplete="given-name"
                value={address.first_name}
                onChange={(e) => setField('first_name', e.target.value)}
                error={fieldError('address.first_name')}
              />
              <Input
                id="checkout-last_name"
                label={t('checkout:contact.lastName')}
                autoComplete="family-name"
                value={address.last_name}
                onChange={(e) => setField('last_name', e.target.value)}
                error={fieldError('address.last_name')}
              />
              {/* The account's email, shown but not editable: the order is
                  tied to the signed-in account, and a second email field
                  would imply otherwise. */}
              <Input
                id="checkout-email"
                label={t('checkout:contact.email')}
                value={me.data.email}
                readOnly
                className="opacity-70"
              />
              <Input
                id="checkout-phone"
                label={t('checkout:contact.phone')}
                type="tel"
                autoComplete="tel"
                placeholder="+374"
                value={address.phone}
                onChange={(e) => setField('phone', e.target.value)}
                error={fieldError('address.phone')}
              />
            </div>
          </CheckoutSection>

          {/* ── Delivery address ────────────────────────────────────── */}
          <CheckoutSection title={t('checkout:address.title')}>
            <Input
              id="checkout-street"
              label={t('checkout:address.street')}
              autoComplete="street-address"
              value={address.street}
              onChange={(e) => setField('street', e.target.value)}
              error={fieldError('address.street')}
            />
            <div className="grid gap-3.5 sm:grid-cols-[1.4fr_1fr_1fr]">
              <Input
                id="checkout-city"
                label={t('checkout:address.city')}
                autoComplete="address-level2"
                value={address.city}
                onChange={(e) => setField('city', e.target.value)}
                error={fieldError('address.city')}
              />
              <Input
                id="checkout-postal_code"
                label={t('checkout:address.postalCode')}
                autoComplete="postal-code"
                value={address.postal_code}
                onChange={(e) => setField('postal_code', e.target.value)}
                error={fieldError('address.postal_code')}
              />
              <Input
                id="checkout-country"
                label={t('checkout:address.country')}
                autoComplete="country-name"
                value={address.country}
                onChange={(e) => setField('country', e.target.value)}
                error={fieldError('address.country')}
              />
            </div>
            <Checkbox
              checked={neighbour}
              onChange={(e) => setNeighbour(e.target.checked)}
              label={t('checkout:address.neighbour')}
            />
            <Input
              id="checkout-note"
              label={t('checkout:address.note')}
              placeholder={t('checkout:address.notePlaceholder')}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              error={fieldError('delivery_note')}
            />
          </CheckoutSection>

          {/* ── Payment ─────────────────────────────────────────────── */}
          <CheckoutSection title={t('checkout:payment.title')}>
            <div
              role="radiogroup"
              aria-label={t('checkout:payment.title')}
              className="flex flex-col gap-3 sm:flex-row"
            >
              <PaymentCard
                selected={method === 'card'}
                onSelect={() => setMethod('card')}
                title={t('checkout:payment.card')}
                blurb={t('checkout:payment.cardBlurb')}
              />
              <PaymentCard
                selected={method === 'bank_transfer'}
                onSelect={() => setMethod('bank_transfer')}
                title={t('checkout:payment.bank')}
                blurb={t('checkout:payment.bankBlurb')}
              />
              <PaymentCard
                selected={method === 'cash_on_delivery'}
                onSelect={() => setMethod('cash_on_delivery')}
                title={t('checkout:payment.cash')}
                blurb={t('checkout:payment.cashBlurb')}
                // The design's own words made a rule: "Cash — on delivery,
                // AMD only". Disabled with the reason visible, rather than
                // clickable and rejected a round trip later.
                disabled={cashDisabled}
              />
            </div>
            {fieldError('payment_method') && (
              <p role="alert" className="text-xs text-danger">
                {fieldError('payment_method')}
              </p>
            )}
            {method === 'card' && (
              // The mock draws card fields; the API deliberately never
              // accepts card data (the provider integration is Phase 11, and
              // card numbers belong on the provider's servers, not this
              // one). So the fields are decorative-disabled with the truth
              // underneath — a state the mock never drew, ours to design.
              <div aria-hidden="true" className="grid gap-3.5 opacity-50 sm:grid-cols-[2fr_1fr_1fr]">
                <DisabledStub label={t('checkout:payment.cardNumber')} value="•••• •••• •••• ••••" />
                <DisabledStub label={t('checkout:payment.expiry')} value="MM / YY" />
                <DisabledStub label={t('checkout:payment.cvc')} value="•••" />
              </div>
            )}
            {method === 'card' && (
              <p className="text-xs text-ink-muted">{t('checkout:payment.cardStubNote')}</p>
            )}
          </CheckoutSection>
        </div>

        {/* ── Summary sidebar ───────────────────────────────────────── */}
        <div className="flex flex-col gap-4 self-start">
          <OrderSummary
            cart={cart.data}
            preview={preview.data}
            action={
              <Button type="submit" size="lg" disabled={checkout.isPending}>
                {checkout.isPending ? t('checkout:placing') : t('checkout:placeOrder')}
              </Button>
            }
          />
          {/* The promo box rides on the checkout too — the mock's cart owns
              it, but a code remembered here saves a trip back. Same
              component, same preview, so the figures cannot disagree. */}
          <PromoBox preview={preview.data} />
          <div className="flex flex-col gap-3 rounded-2xl bg-card p-6">
            {(['packed', 'replaced', 'labReport'] as const).map((key) => (
              <div key={key} className="flex items-start gap-2.5 text-sm text-ink-strong">
                <span
                  aria-hidden="true"
                  className="flex size-5 shrink-0 items-center justify-center rounded-full bg-honey text-[0.6875rem] font-bold text-ink"
                >
                  ✓
                </span>
                {t(`checkout:promises.${key}`)}
              </div>
            ))}
          </div>
        </div>
      </form>
    </CheckoutShell>
  )
}

/**
 * The checkout's own chrome: logo home-link, the three-step indicator,
 * "Secure". Steps 1 and 2 are filled together because this checkout is one
 * page — the indicator communicates progress through the PURCHASE (details
 * and payment now, done after), not through a wizard.
 */
function CheckoutShell({ step, children }: { step: 1 | 3; children: ReactNode }) {
  const { t } = useTranslation()
  const { localePath } = useLocale()

  return (
    <div className="min-h-screen bg-panel">
      <header className="flex flex-wrap items-center justify-between gap-4 border-b border-line-soft bg-panel-soft px-6 py-5 lg:px-14">
        <Link to={localePath('/')} className="flex items-center gap-3">
          <span className="flex size-9 items-center justify-center rounded-xl bg-honey font-display font-extrabold text-ink">
            M
          </span>
          <span className="font-display text-[1.0625rem] font-extrabold tracking-wide text-ink">
            {t('common:brand').toUpperCase()}
          </span>
        </Link>

        <ol className="flex items-center gap-4 font-display text-sm">
          <StepDot n={1} label={t('checkout:steps.details')} active={step >= 1} />
          <StepSeparator />
          <StepDot n={2} label={t('checkout:steps.payment')} active={step >= 1} />
          <StepSeparator />
          <StepDot n={3} label={t('checkout:steps.done')} active={step >= 3} />
        </ol>

        <span className="text-sm text-ink-faint" aria-hidden="true">
          {t('checkout:secure')}
        </span>
      </header>

      <main className="mx-auto max-w-360 px-6 py-11 lg:px-14">{children}</main>
    </div>
  )
}

function StepDot({ n, label, active }: { n: number; label: string; active: boolean }) {
  return (
    <li
      aria-current={active ? 'step' : undefined}
      className={cx('flex items-center gap-2', active ? 'font-semibold text-ink' : 'text-ink-faint')}
    >
      <span
        className={cx(
          'flex size-6 items-center justify-center rounded-full text-xs',
          active ? 'bg-brand text-ink-on-dark' : 'border-[1.5px] border-line-strong',
        )}
      >
        {n}
      </span>
      {label}
    </li>
  )
}

function StepSeparator() {
  return (
    <li aria-hidden="true" className="text-line-strong">
      —
    </li>
  )
}

function CheckoutSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-4 rounded-2xl bg-card p-7">
      <h2 className="font-display text-base font-bold uppercase tracking-label text-ink">
        {title}
      </h2>
      {children}
    </section>
  )
}

function PaymentCard({
  selected,
  onSelect,
  title,
  blurb,
  disabled,
}: {
  selected: boolean
  onSelect: () => void
  title: string
  blurb: string
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      disabled={disabled}
      onClick={onSelect}
      className={cx(
        'flex flex-1 flex-col gap-0.5 rounded-2xl border p-4 text-left transition',
        selected
          ? 'border-2 border-brand bg-panel'
          : 'border-[1.5px] border-line hover:border-line-strong',
        disabled && 'cursor-not-allowed opacity-50',
      )}
    >
      <span className="font-display text-[0.9375rem] font-bold text-ink">{title}</span>
      <span className="text-xs text-ink-soft">{blurb}</span>
    </button>
  )
}

// A decorative, inert "input" — the card stub. A real disabled <input> would
// be announced to screen readers as a form control they cannot use; a styled
// box inside an aria-hidden container is honestly just a picture.
function DisabledStub({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-2">
      <span className="text-xs font-semibold text-ink-soft">{label}</span>
      <span className="rounded-xl border-[1.5px] border-line bg-panel px-4 py-3.5 text-[0.9375rem] text-ink-faint">
        {value}
      </span>
    </div>
  )
}
