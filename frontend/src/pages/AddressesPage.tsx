import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAddresses, useCreateAddress, useDeleteAddress, useUpdateAddress } from '../api/hooks'
import type { AddressEntry, AddressInput } from '../api/types'
import { Button } from '../components/ui/Button'
import { Checkbox } from '../components/ui/Checkbox'
import { CheckIcon, PlusIcon } from '../components/ui/icons'
import { Input } from '../components/ui/Input'
import { cx } from '../lib/cx'
import { useFieldErrors } from '../i18n/useFieldErrors'

/**
 * /account/addresses — canvas 09: the E8 book re-hung as a card grid. The
 * CRUD, the validation-error plumbing (checkout's exact field keys) and
 * the default-flag logic all survive from E8; what changed is the room.
 *
 * The add/edit form is a PANEL below the grid rather than a card that
 * turns into a form — the canvas only draws the resting state, and a
 * panel keeps the nine inputs at full width instead of squeezed into half
 * a grid column.
 */
export function AddressesPage() {
  const { t } = useTranslation()
  const addresses = useAddresses(true)
  const create = useCreateAddress()
  const update = useUpdateAddress()
  const remove = useDeleteAddress()

  // null = closed, 0 = adding, >0 = editing that entry.
  const [editing, setEditing] = useState<number | null>(null)
  const [form, setForm] = useState<AddressInput>(EMPTY_INPUT)
  const formRef = useRef<HTMLFormElement>(null)

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
    // The panel sits below the grid; opening it from a card two rows up
    // must bring it into view or the click appears to do nothing.
    requestAnimationFrame(() =>
      formRef.current?.scrollIntoView({ behavior: 'smooth', block: 'nearest' }),
    )
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

  const entries = addresses.data ?? []

  return (
    <>
      <div className="flex flex-col gap-1.5">
        <h1 className="font-display text-display-md font-extrabold text-ink">
          {t('account:nav.addresses')}
        </h1>
        <p className="text-[0.9375rem] text-ink-soft">{t('account:addressesScreen.subtitle')}</p>
      </div>

      {addresses.isPending && (
        <p className="mt-6 text-sm text-ink-soft">{t('common:state.loading')}</p>
      )}

      <div className="mt-6 grid items-start gap-5 lg:grid-cols-2">
        {entries.map((entry) => (
          <AddressCard
            key={entry.id}
            entry={entry}
            onEdit={() => open(entry)}
            onMakeDefault={() => {
              const { id: _, ...input } = entry
              update.mutate({ id: entry.id, input: { ...input, is_default: true } })
            }}
            onRemove={() => remove.mutate(entry.id)}
          />
        ))}

        {/* The canvas's dashed card is the PRIMARY add affordance — an "add
            more" slot in the grid beats a link hiding in the header. */}
        <button
          type="button"
          onClick={() => open()}
          className="flex min-h-52 flex-col items-center justify-center gap-2.5 rounded-3xl border-2 border-dashed border-line-strong p-7 text-center transition hover:border-brand-ink"
        >
          <span
            aria-hidden
            className="flex size-11 items-center justify-center rounded-full bg-card text-ink-faint"
          >
            <PlusIcon size={20} />
          </span>
          <span className="font-display text-[0.9375rem] font-bold text-ink">
            {t('account:addresses.add')}
          </span>
          <span className="max-w-56 text-[0.8125rem] text-ink-muted">
            {t('account:addressesScreen.addBlurb')}
          </span>
        </button>

        {/* Canvas 09's "Pickup from the apiary" card — the honest stub
            (decision #6/#87 pattern): pickup is not a delivery method the
            checkout offers, so the card promises nothing bookable and says
            so. Deliberately WITHOUT the canvas's Saturday hours: inventing
            opening times in a real page is the placeholder-facts trap Era
            III's audit flagged. */}
        <section className="flex min-h-52 flex-col gap-2.5 rounded-3xl bg-bark p-6">
          <h2 className="font-display text-base font-bold text-ink-on-dark">
            {t('account:pickupStub.title')}
          </h2>
          <p className="text-sm leading-relaxed text-ink-on-dark-body">
            {t('account:pickupStub.blurb')}
          </p>
          <p className="mt-auto rounded-xl bg-ink-on-dark/8 px-4 py-3 text-[0.8125rem] text-ink-on-dark-soft">
            {t('account:pickupStub.comingSoon')}
          </p>
        </section>
      </div>

      {editing !== null && (
        <form
          ref={formRef}
          onSubmit={onSubmit}
          noValidate
          className="mt-6 flex flex-col gap-3.5 rounded-3xl bg-card p-6"
        >
          <h2 className="font-display text-sm font-bold uppercase tracking-label text-ink">
            {editing === 0 ? t('account:addresses.add') : t('account:addresses.edit')}
          </h2>
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
            <div className="flex flex-col justify-end gap-2 pb-1">
              <Checkbox
                checked={form.is_default}
                onChange={(e) => setForm((f) => ({ ...f, is_default: e.target.checked }))}
                label={t('account:addresses.default')}
              />
              {/* A4: the neighbour flag lives on the address (log #88) —
                  what is saved here prefills the checkout's checkbox. */}
              <Checkbox
                checked={form.leave_with_neighbour}
                onChange={(e) =>
                  setForm((f) => ({ ...f, leave_with_neighbour: e.target.checked }))
                }
                label={t('checkout:address.neighbour')}
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
    </>
  )
}

const EMPTY_INPUT: AddressInput = {
  label: '',
  is_default: false,
  leave_with_neighbour: false,
  first_name: '',
  last_name: '',
  phone: '',
  street: '',
  city: '',
  postal_code: '',
  country: 'AM',
}

/** One book entry as the canvas's card: label + Default badge, the address
 *  block, the neighbour note row, and the three text actions. */
function AddressCard({
  entry,
  onEdit,
  onMakeDefault,
  onRemove,
}: {
  entry: AddressEntry
  onEdit: () => void
  onMakeDefault: () => void
  onRemove: () => void
}) {
  const { t } = useTranslation()

  // Deleting is destructive and the canvas never draws a confirm (ours to
  // design): the SAME button asks again — "Remove?" — and arms for three
  // seconds. A second click within the window fires; letting it lapse
  // disarms. One control, keyboard-reachable, announced by its own text.
  const [arming, setArming] = useState(false)
  const disarm = useRef<ReturnType<typeof setTimeout>>(undefined)
  useEffect(() => () => clearTimeout(disarm.current), [])

  const removeClick = () => {
    if (arming) {
      clearTimeout(disarm.current)
      onRemove()
      return
    }
    setArming(true)
    disarm.current = setTimeout(() => setArming(false), 3000)
  }

  return (
    <article
      className={cx(
        'flex flex-col gap-3.5 rounded-3xl bg-card p-6',
        entry.is_default && 'border-2 border-brand',
      )}
    >
      <div className="flex items-center gap-2.5">
        <h2 className="font-display text-lg font-bold text-ink">
          {entry.label || t('account:addresses.unlabelled')}
        </h2>
        {entry.is_default && (
          <span className="rounded-full bg-honey px-2.5 py-1 font-display text-2xs font-bold uppercase tracking-label text-ink">
            {t('account:addresses.default')}
          </span>
        )}
      </div>

      <p className="text-[0.9375rem] leading-relaxed text-ink-strong">
        {entry.first_name} {entry.last_name}
        <br />
        {entry.street}
        <br />
        {entry.city} {entry.postal_code}, {entry.country}
        <br />
        {entry.phone}
      </p>

      {/* The note row states the flag either way — a quiet unchecked box
          says "you could turn this on" where hiding the row would not. */}
      <p
        className={cx(
          'flex items-center gap-2.5 rounded-xl bg-panel px-3.5 py-2.5 text-sm',
          entry.leave_with_neighbour ? 'text-ink-strong' : 'text-ink-muted',
        )}
      >
        <span
          aria-hidden
          className={cx(
            'flex size-5 shrink-0 items-center justify-center rounded-md',
            entry.leave_with_neighbour
              ? 'bg-brand-ink text-ink-on-dark'
              : 'border-[1.5px] border-line-strong',
          )}
        >
          {entry.leave_with_neighbour && <CheckIcon size={13} />}
        </span>
        {t('checkout:address.neighbour')}
      </p>

      <div className="mt-auto flex gap-4 pt-1 font-display text-sm font-semibold">
        <button type="button" onClick={onEdit} className="text-brand-ink hover:underline">
          {t('account:addresses.edit')}
        </button>
        {!entry.is_default && (
          <button type="button" onClick={onMakeDefault} className="text-brand-ink hover:underline">
            {t('account:addresses.makeDefault')}
          </button>
        )}
        <button
          type="button"
          onClick={removeClick}
          className={cx(
            'hover:underline',
            arming ? 'font-bold text-danger' : 'text-ink-faint hover:text-danger',
          )}
        >
          {arming ? t('account:addressesScreen.confirmRemove') : t('account:addresses.delete')}
        </button>
      </div>
    </article>
  )
}
