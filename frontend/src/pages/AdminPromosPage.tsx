import { useState } from 'react'
import { ApiError } from '../api/client'
import { useAdminPromos, useCreatePromo, useMe, useUpdatePromo } from '../api/hooks'
import type { AdminPromo, PromoInput, PromoKind, PromoValue } from '../api/types'
import { AdminNav } from '../components/AdminNav'
import type { Currency } from '../lib/currencies'
import { CURRENCIES } from '../lib/currencies'
import { formatMoney, inputToMinor, minorToInput } from '../lib/format'

/**
 * F2 (decision #94): the promo codes the shop runs — E7 shipped seed-only
 * codes and wrote "revisit when three codes stop being enough". One list,
 * one form for create and edit, and NO delete: redemption history hangs
 * off a code, so retiring one is `active: false`, not a vanished record.
 * Admin pages are pre-canvas utility UI — English-only by convention.
 */
export function AdminPromosPage() {
  const me = useMe()
  const promos = useAdminPromos()
  // null = form closed; 'new' = creating; an AdminPromo = editing it.
  const [editing, setEditing] = useState<AdminPromo | 'new' | null>(null)
  const update = useUpdatePromo()

  if (me.isPending) return <Shell>Checking access…</Shell>
  if (!me.data || me.data.role !== 'admin') {
    return (
      <Shell>
        <p className="rounded-lg bg-red-50 p-4 text-red-600">
          This area requires an admin account.
        </p>
      </Shell>
    )
  }

  return (
    <Shell>
      <div className="flex items-center gap-6">
        <h2 className="text-xl font-bold text-stone-800">Admin — Promo codes</h2>
        <AdminNav />
      </div>

      {promos.isPending && <p className="mt-4 text-stone-400">Loading…</p>}
      {promos.isError && <p className="mt-4 text-red-600">Failed to load promo codes.</p>}

      {promos.data && (
        <div className="mt-4 space-y-3">
          {promos.data.length === 0 && (
            <p className="text-stone-500">No promo codes yet.</p>
          )}
          {promos.data.map((p) => (
            <article key={p.id} className="rounded-xl border border-stone-200 bg-white p-4">
              <div className="flex flex-wrap items-center gap-3">
                <span className="font-mono text-base font-bold text-stone-800">{p.code}</span>
                <span className="text-sm text-stone-600">{kindSummary(p)}</span>
                <span
                  className={
                    p.active
                      ? 'rounded-full bg-emerald-100 px-2.5 py-0.5 text-xs font-medium text-emerald-800'
                      : 'rounded-full bg-stone-200 px-2.5 py-0.5 text-xs font-medium text-stone-500'
                  }
                >
                  {p.active ? 'active' : 'inactive'}
                </span>
                <span className="ml-auto text-xs text-stone-400">
                  used {p.redemptions} / {p.max_redemptions ?? '∞'}
                </span>
              </div>
              <p className="mt-1 text-xs text-stone-400">
                {windowSummary(p)}
                {floorSummary(p) && <> · {floorSummary(p)}</>}
              </p>
              <div className="mt-3 flex gap-2 border-t border-stone-100 pt-3">
                <button
                  type="button"
                  onClick={() => setEditing(p)}
                  className="rounded-lg bg-stone-100 px-4 py-1.5 text-sm font-medium text-stone-700 hover:bg-stone-200"
                >
                  edit
                </button>
                <button
                  type="button"
                  disabled={update.isPending}
                  onClick={() =>
                    update.mutate({ id: p.id, input: { ...promoToInput(p), active: !p.active } })
                  }
                  className="rounded-lg bg-stone-100 px-4 py-1.5 text-sm font-medium text-stone-700 hover:bg-stone-200 disabled:opacity-50"
                >
                  {p.active ? 'deactivate' : 'activate'}
                </button>
              </div>
            </article>
          ))}

          {editing === null ? (
            <button
              type="button"
              onClick={() => setEditing('new')}
              className="rounded-lg bg-emerald-700 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-800"
            >
              New code
            </button>
          ) : (
            <PromoForm
              key={editing === 'new' ? 'new' : editing.id}
              promo={editing === 'new' ? null : editing}
              onDone={() => setEditing(null)}
            />
          )}
        </div>
      )}
    </Shell>
  )
}

function kindSummary(p: AdminPromo): string {
  switch (p.kind) {
    case 'percent':
      return `${p.percent}% off`
    case 'free_shipping':
      return 'free shipping (base rate)'
    case 'fixed': {
      const parts = CURRENCIES.flatMap((c) => {
        const m = p.values[c]?.amount_minor
        return m ? [formatMoney(m, c)] : []
      })
      return parts.length ? `${parts.join(' / ')} off` : 'fixed (no amounts!)'
    }
  }
}

function windowSummary(p: AdminPromo): string {
  const from = p.starts_at ? new Date(p.starts_at).toLocaleString() : null
  const to = p.ends_at ? new Date(p.ends_at).toLocaleString() : null
  if (!from && !to) return 'no time window'
  return `${from ?? 'always'} → ${to ?? 'no end'}`
}

function floorSummary(p: AdminPromo): string {
  const parts = CURRENCIES.flatMap((c) => {
    const m = p.values[c]?.min_subtotal_minor
    return m ? [`over ${formatMoney(m, c)}`] : []
  })
  return parts.join(' / ')
}

/** The list row → the whole-value write shape, for the quick active toggle. */
function promoToInput(p: AdminPromo): PromoInput {
  return {
    code: p.code,
    kind: p.kind,
    percent: p.percent,
    starts_at: p.starts_at,
    ends_at: p.ends_at,
    max_redemptions: p.max_redemptions,
    active: p.active,
    values: p.values,
  }
}

// datetime-local speaks local wall-clock, the API speaks RFC3339 — these
// two convert at the input's edge, the only place the local form exists.
function toLocalInput(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function fromLocalInput(v: string): string | undefined {
  return v ? new Date(v).toISOString() : undefined
}

function PromoForm({ promo, onDone }: { promo: AdminPromo | null; onDone: () => void }) {
  const create = useCreatePromo()
  const update = useUpdatePromo()
  const busy = create.isPending || update.isPending

  const [code, setCode] = useState(promo?.code ?? '')
  const [kind, setKind] = useState<PromoKind>(promo?.kind ?? 'percent')
  const [percent, setPercent] = useState(promo?.percent?.toString() ?? '')
  const [startsAt, setStartsAt] = useState(toLocalInput(promo?.starts_at))
  const [endsAt, setEndsAt] = useState(toLocalInput(promo?.ends_at))
  const [maxRedemptions, setMaxRedemptions] = useState(
    promo?.max_redemptions?.toString() ?? '',
  )
  const [active, setActive] = useState(promo?.active ?? true)
  // Money as the admin types it: MAJOR units per market, like the variant
  // editor — converted to minor at the submit edge and back on load.
  const [amounts, setAmounts] = useState<Record<Currency, string>>(() =>
    moneyState(promo, 'amount_minor'),
  )
  const [floors, setFloors] = useState<Record<Currency, string>>(() =>
    moneyState(promo, 'min_subtotal_minor'),
  )

  const error = create.error ?? update.error
  const fields = error instanceof ApiError ? (error.fields ?? {}) : {}
  const codeTaken = error instanceof ApiError && error.code === 'code_taken'

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    const values: Partial<Record<Currency, PromoValue>> = {}
    for (const c of CURRENCIES) {
      const v: PromoValue = {}
      if (kind === 'fixed' && amounts[c]) v.amount_minor = inputToMinor(amounts[c], c)
      if (floors[c]) v.min_subtotal_minor = inputToMinor(floors[c], c)
      if (v.amount_minor !== undefined || v.min_subtotal_minor !== undefined) values[c] = v
    }
    const input: PromoInput = {
      code,
      kind,
      percent: kind === 'percent' && percent !== '' ? Number(percent) : undefined,
      starts_at: fromLocalInput(startsAt),
      ends_at: fromLocalInput(endsAt),
      max_redemptions: maxRedemptions !== '' ? Number(maxRedemptions) : undefined,
      active,
      values,
    }
    if (promo) update.mutate({ id: promo.id, input }, { onSuccess: onDone })
    else create.mutate(input, { onSuccess: onDone })
  }

  return (
    <form onSubmit={submit} className="rounded-xl border border-stone-200 bg-white p-4">
      <h3 className="font-bold text-stone-800">{promo ? `Edit ${promo.code}` : 'New promo code'}</h3>

      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        <Field label="Code" error={fields.code}>
          <input
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder="HONEY10"
            className={inputCls}
          />
        </Field>
        <Field label="Kind" error={fields.kind}>
          <select value={kind} onChange={(e) => setKind(e.target.value as PromoKind)} className={inputCls}>
            <option value="percent">percent off</option>
            <option value="fixed">fixed amount off</option>
            <option value="free_shipping">free shipping (base rate)</option>
          </select>
        </Field>
        {kind === 'percent' && (
          <Field label="Percent (1–100)" error={fields.percent}>
            <input
              type="number"
              min={1}
              max={100}
              value={percent}
              onChange={(e) => setPercent(e.target.value)}
              className={inputCls}
            />
          </Field>
        )}
        <Field label="Max redemptions (blank = uncapped)" error={fields.max_redemptions}>
          <input
            type="number"
            min={1}
            value={maxRedemptions}
            onChange={(e) => setMaxRedemptions(e.target.value)}
            className={inputCls}
          />
        </Field>
        <Field label="Starts (blank = immediately)" error={fields.starts_at}>
          <input
            type="datetime-local"
            value={startsAt}
            onChange={(e) => setStartsAt(e.target.value)}
            className={inputCls}
          />
        </Field>
        <Field label="Ends (blank = never)" error={fields.ends_at}>
          <input
            type="datetime-local"
            value={endsAt}
            onChange={(e) => setEndsAt(e.target.value)}
            className={inputCls}
          />
        </Field>
      </div>

      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        {CURRENCIES.map((c) => (
          <div key={c} className="rounded-lg bg-stone-50 p-3">
            <p className="text-xs font-bold uppercase text-stone-500">{c}</p>
            {kind === 'fixed' && (
              <Field label={`Amount off (${c})`} error={fields[`values.${c}.amount_minor`]}>
                <input
                  value={amounts[c]}
                  onChange={(e) => setAmounts({ ...amounts, [c]: e.target.value })}
                  placeholder="5.00"
                  className={inputCls}
                />
              </Field>
            )}
            <Field
              label={`Min subtotal (${c}, blank = none)`}
              error={fields[`values.${c}.min_subtotal_minor`]}
            >
              <input
                value={floors[c]}
                onChange={(e) => setFloors({ ...floors, [c]: e.target.value })}
                className={inputCls}
              />
            </Field>
          </div>
        ))}
      </div>
      {fields.values && (
        <p className="mt-2 text-sm text-red-600">A fixed code needs an amount in at least one market.</p>
      )}

      <label className="mt-3 flex items-center gap-2 text-sm text-stone-700">
        <input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} />
        active
      </label>

      {codeTaken && (
        <p className="mt-2 text-sm text-red-600">
          A code with this text already exists (codes are case-insensitive).
        </p>
      )}
      {error && !codeTaken && !(error instanceof ApiError && error.fields) && (
        <p className="mt-2 text-sm text-red-600">Saving failed. Please try again.</p>
      )}

      <div className="mt-4 flex gap-2">
        <button
          type="submit"
          disabled={busy}
          className="rounded-lg bg-emerald-700 px-4 py-1.5 text-sm font-medium text-white hover:bg-emerald-800 disabled:opacity-50"
        >
          {promo ? 'Save changes' : 'Create code'}
        </button>
        <button
          type="button"
          onClick={onDone}
          className="rounded-lg px-4 py-1.5 text-sm font-medium text-stone-500 hover:bg-stone-100"
        >
          Cancel
        </button>
      </div>
    </form>
  )
}

function moneyState(promo: AdminPromo | null, field: keyof PromoValue): Record<Currency, string> {
  const out = {} as Record<Currency, string>
  for (const c of CURRENCIES) {
    const minor = promo?.values[c]?.[field]
    out[c] = minor !== undefined ? minorToInput(minor, c) : ''
  }
  return out
}

const inputCls =
  'w-full rounded-lg border border-stone-300 px-3 py-1.5 text-sm text-stone-800 focus:border-emerald-600 focus:outline-none'

function Field({
  label,
  error,
  children,
}: {
  label: string
  error?: string
  children: React.ReactNode
}) {
  return (
    <label className="block text-sm">
      <span className="mb-1 block text-xs font-medium text-stone-500">{label}</span>
      {children}
      {error && <span className="mt-1 block text-xs text-red-600">{error}</span>}
    </label>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-3xl px-4 py-8">{children}</div>
}
