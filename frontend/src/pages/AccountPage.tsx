import { useState } from 'react'
import { Link } from 'react-router'
import { Trans, useTranslation } from 'react-i18next'
import {
  useAddresses,
  useCreateAddress,
  useDeleteAddress,
  useLogout,
  useMe,
  useUpdateAddress,
} from '../api/hooks'
import type { AddressEntry, AddressInput } from '../api/types'
import { Button } from '../components/ui/Button'
import { Checkbox } from '../components/ui/Checkbox'
import { Input } from '../components/ui/Input'
import { useFieldErrors } from '../i18n/useFieldErrors'
import { useLocale } from '../i18n/useLocale'

/**
 * /account — the account area (E8): who you are (with your hive standing),
 * the address book behind the checkout's prefill, and the doors to orders
 * and the wishlist. Order history deliberately STAYS at /orders — it
 * already works there, and the account page is its front door, not its new
 * home.
 */
export function AccountPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const me = useMe()
  const logout = useLogout()

  if (me.isPending) {
    return <Shell>{t('common:state.loading')}</Shell>
  }
  if (!me.data) {
    return (
      <Shell>
        <p className="text-ink-body">
          <Trans
            i18nKey="account:signInRequired"
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

  const user = me.data

  return (
    <Shell>
      <h1 className="font-display text-display-md font-extrabold text-ink">
        {t('account:title')}
      </h1>

      <div className="mt-7 grid items-start gap-6 lg:grid-cols-[360px_1fr]">
        {/* ── Profile ────────────────────────────────────────────────── */}
        <div className="flex flex-col gap-4">
          <section className="flex flex-col gap-3 rounded-2xl bg-card p-6">
            <h2 className="font-display text-sm font-bold uppercase tracking-label text-ink">
              {t('account:profile.title')}
            </h2>
            <p className="text-[0.9375rem] text-ink-strong">{user.email}</p>
            {user.hive.member ? (
              <p className="flex items-center gap-2 text-sm text-ink-body">
                <span className="rounded-full bg-honey px-3 py-1 font-display text-xs font-bold text-ink">
                  {t('common:hive.badge')}
                </span>
                {t('account:profile.memberLine', {
                  percent: user.hive.member_discount_percent,
                })}
              </p>
            ) : (
              <p className="text-sm text-ink-body">{t('account:profile.firstOrderLine')}</p>
            )}
            <button
              type="button"
              onClick={() => logout.mutate()}
              className="self-start text-sm font-semibold text-brand-ink hover:underline"
            >
              {t('account:signOut')}
            </button>
          </section>

          <nav className="flex flex-col gap-1 rounded-2xl bg-card p-3">
            <AccountLink to={localePath('/orders')} label={t('account:ordersTitle')} />
            <AccountLink to={localePath('/wishlist')} label={t('account:wishlist.title')} />
          </nav>
        </div>

        {/* ── Address book ───────────────────────────────────────────── */}
        <AddressBook />
      </div>
    </Shell>
  )
}

function AccountLink({ to, label }: { to: string; label: string }) {
  return (
    <Link
      to={to}
      className="rounded-xl px-4 py-3 font-display text-[0.9375rem] font-semibold text-ink transition hover:bg-panel"
    >
      {label} →
    </Link>
  )
}

const EMPTY_INPUT: AddressInput = {
  label: '',
  is_default: false,
  first_name: '',
  last_name: '',
  phone: '',
  street: '',
  city: '',
  postal_code: '',
  country: 'AM',
}

/**
 * The book itself: list, add, edit, delete, set-default. One form serves
 * add AND edit (the id decides which mutation fires), with the checkout's
 * exact field keys so server errors land on the right inputs through the
 * same catalogue.
 */
function AddressBook() {
  const { t } = useTranslation()
  const me = useMe()
  const addresses = useAddresses(!!me.data)
  const create = useCreateAddress()
  const update = useUpdateAddress()
  const remove = useDeleteAddress()

  // null = closed, 0 = adding, >0 = editing that entry.
  const [editing, setEditing] = useState<number | null>(null)
  const [form, setForm] = useState<AddressInput>(EMPTY_INPUT)

  const active = editing === 0 ? create : update
  const errors = useFieldErrors(active.error)

  function open(entry?: AddressEntry) {
    create.reset()
    update.reset()
    if (entry) {
      const { id: _, ...input } = entry
      setForm(input)
      setEditing(entry.id)
    } else {
      setForm(EMPTY_INPUT)
      setEditing(0)
    }
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const close = { onSuccess: () => setEditing(null) }
    if (editing === 0) {
      create.mutate(form, close)
    } else if (editing) {
      update.mutate({ id: editing, input: form }, close)
    }
  }

  const set = (key: keyof AddressInput) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({ ...f, [key]: e.target.value }))

  return (
    <section className="flex flex-col gap-4 rounded-2xl bg-card p-6">
      <div className="flex items-center justify-between">
        <h2 className="font-display text-sm font-bold uppercase tracking-label text-ink">
          {t('account:addresses.title')}
        </h2>
        {editing === null && (
          <button
            type="button"
            onClick={() => open()}
            className="text-sm font-semibold text-brand-ink hover:underline"
          >
            {t('account:addresses.add')}
          </button>
        )}
      </div>

      {addresses.isPending && <p className="text-sm text-ink-soft">{t('common:state.loading')}</p>}

      {(addresses.data ?? []).length === 0 && !addresses.isPending && editing === null && (
        <p className="text-sm text-ink-body">{t('account:addresses.empty')}</p>
      )}

      <ul className="flex flex-col divide-y divide-line-soft">
        {(addresses.data ?? []).map((entry) => (
          <li key={entry.id} className="flex flex-wrap items-start justify-between gap-3 py-4 first:pt-0">
            <div className="text-sm leading-relaxed text-ink-body">
              <p className="flex items-center gap-2 font-semibold text-ink">
                {entry.label || t('account:addresses.unlabelled')}
                {entry.is_default && (
                  <span className="rounded-full bg-panel px-2.5 py-0.5 text-2xs font-semibold uppercase tracking-label text-ink-muted">
                    {t('account:addresses.default')}
                  </span>
                )}
              </p>
              {entry.first_name} {entry.last_name} · {entry.phone}
              <br />
              {entry.street}, {entry.city} {entry.postal_code}, {entry.country}
            </div>
            <div className="flex gap-4 text-sm font-semibold">
              {!entry.is_default && (
                <button
                  type="button"
                  onClick={() => {
                    const { id: _, ...input } = entry
                    update.mutate({ id: entry.id, input: { ...input, is_default: true } })
                  }}
                  className="text-brand-ink hover:underline"
                >
                  {t('account:addresses.makeDefault')}
                </button>
              )}
              <button
                type="button"
                onClick={() => open(entry)}
                className="text-brand-ink hover:underline"
              >
                {t('account:addresses.edit')}
              </button>
              <button
                type="button"
                onClick={() => remove.mutate(entry.id)}
                className="text-ink-faint hover:text-danger"
              >
                {t('account:addresses.delete')}
              </button>
            </div>
          </li>
        ))}
      </ul>

      {editing !== null && (
        <form onSubmit={onSubmit} noValidate className="flex flex-col gap-3.5 border-t border-line pt-4">
          <h3 className="font-display text-sm font-bold text-ink">
            {editing === 0 ? t('account:addresses.add') : t('account:addresses.edit')}
          </h3>
          {errors.formError && (
            <p role="alert" className="rounded-xl bg-danger/10 p-3 text-sm text-danger">
              {errors.formError}
            </p>
          )}
          <div className="grid gap-3.5 sm:grid-cols-2">
            <Input
              id="addr-label"
              label={t('account:addresses.label')}
              placeholder={t('account:addresses.labelPlaceholder')}
              value={form.label}
              onChange={set('label')}
              error={errors.fieldError('label')}
            />
            <div className="flex items-end pb-3">
              <Checkbox
                checked={form.is_default}
                onChange={(e) => setForm((f) => ({ ...f, is_default: e.target.checked }))}
                label={t('account:addresses.default')}
              />
            </div>
            <Input id="addr-first" label={t('checkout:contact.firstName')} autoComplete="given-name"
              value={form.first_name} onChange={set('first_name')} error={errors.fieldError('address.first_name')} />
            <Input id="addr-last" label={t('checkout:contact.lastName')} autoComplete="family-name"
              value={form.last_name} onChange={set('last_name')} error={errors.fieldError('address.last_name')} />
            <Input id="addr-phone" label={t('checkout:contact.phone')} type="tel" autoComplete="tel"
              value={form.phone} onChange={set('phone')} error={errors.fieldError('address.phone')} />
            <Input id="addr-street" label={t('checkout:address.street')} autoComplete="street-address"
              value={form.street} onChange={set('street')} error={errors.fieldError('address.street')} />
            <Input id="addr-city" label={t('checkout:address.city')} autoComplete="address-level2"
              value={form.city} onChange={set('city')} error={errors.fieldError('address.city')} />
            <Input id="addr-postal" label={t('checkout:address.postalCode')} autoComplete="postal-code"
              value={form.postal_code} onChange={set('postal_code')} error={errors.fieldError('address.postal_code')} />
            <Input id="addr-country" label={t('checkout:address.country')} autoComplete="country-name"
              value={form.country} onChange={set('country')} error={errors.fieldError('address.country')} />
          </div>
          <div className="flex gap-3">
            <Button type="submit" disabled={active.isPending}>
              {active.isPending ? t('account:working') : t('account:addresses.save')}
            </Button>
            <button
              type="button"
              onClick={() => setEditing(null)}
              className="text-sm font-semibold text-ink-muted hover:text-ink"
            >
              {t('account:addresses.cancel')}
            </button>
          </div>
        </form>
      )}
    </section>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-360 px-6 py-10 lg:px-14">{children}</div>
}
