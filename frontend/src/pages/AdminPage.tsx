import { useState } from 'react'
import { Link } from 'react-router'
import { ApiError } from '../api/client'
import {
  useAdminCategories,
  useCreateCategory,
  useDeleteCategory,
  useMe,
  useReorderCategories,
  useUpdateCategory,
} from '../api/hooks'
import type { AdminCategory } from '../api/types'
import { AdminNav } from '../components/AdminNav'
import { useFieldErrors } from '../i18n/useFieldErrors'
import { PREFIXED_LOCALES } from '../i18n/locales'
import { localeLabel } from '../lib/translations'

// NOTE: this guard is user experience, not security. The backend's
// requireAdmin middleware is the real gate — anyone can bypass this page
// with devtools, and the API will still refuse them.
export function AdminPage() {
  const me = useMe()
  // null = closed, 'new' = creating, an AdminCategory = editing it (F2).
  const [editing, setEditing] = useState<AdminCategory | 'new' | null>('new')

  if (me.isPending) {
    return <Shell>Checking access…</Shell>
  }
  if (!me.data || me.data.role !== 'admin') {
    return (
      <Shell>
        <p className="rounded-lg bg-red-50 p-4 text-red-600">
          This area requires an admin account.
        </p>
        <Link to="/login" className="mt-4 inline-block text-emerald-700 underline">
          Sign in
        </Link>
      </Shell>
    )
  }

  return (
    <Shell>
      <div className="flex items-center gap-6">
        <h2 className="text-xl font-bold text-stone-800">Admin — Categories</h2>
        <AdminNav />
      </div>
      <CategoryForm
        key={editing === 'new' || editing === null ? 'new' : editing.id}
        category={editing === 'new' || editing === null ? null : editing}
        onDone={() => setEditing('new')}
      />
      <CategoryList onEdit={setEditing} />
    </Shell>
  )
}

// One form for create and edit (F2): same fields, same validation, same
// field-error attachment — only the verb differs.
function CategoryForm({
  category,
  onDone,
}: {
  category: AdminCategory | null
  onDone: () => void
}) {
  const [slug, setSlug] = useState(category?.slug ?? '')
  const [name, setName] = useState(category?.name ?? '')
  const [sortOrder, setSortOrder] = useState(category?.sort_order.toString() ?? '0')
  // One entry per translatable language, kept even while empty so the
  // inputs stay controlled. A category's translation is ONE string per
  // language (the backend's flat map — the nested {name} shape this form
  // used to send never decoded and 400'd; found during F2).
  const [translations, setTranslations] = useState<Record<string, string>>(() =>
    Object.fromEntries(
      PREFIXED_LOCALES.map((l) => [l, category?.translations[l] ?? '']),
    ),
  )
  const create = useCreateCategory()
  const update = useUpdateCategory()
  const mutation = category ? update : create

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    // Blank languages are dropped: absent means "fall back to English",
    // while a present-but-empty name would be a validation error.
    const filled = Object.fromEntries(
      Object.entries(translations).filter(([, v]) => v.trim() !== ''),
    )
    const input = {
      slug,
      name,
      sort_order: Number(sortOrder) || 0,
      translations: Object.keys(filled).length > 0 ? filled : undefined,
    }
    if (category) {
      update.mutate({ id: category.id, input }, { onSuccess: onDone })
    } else {
      create.mutate(input, {
        onSuccess: () => {
          setSlug('')
          setName('')
          setSortOrder('0')
          setTranslations(Object.fromEntries(PREFIXED_LOCALES.map((l) => [l, ''])))
        },
      })
    }
  }

  const { fieldError, formError } = useFieldErrors(mutation.error)

  return (
    <form
      onSubmit={onSubmit}
      className="mt-6 space-y-3 rounded-xl border border-stone-200 bg-white p-5"
    >
      <h3 className="font-medium text-stone-700">
        {category ? `Edit “${category.name}”` : 'New category'}
      </h3>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <Input placeholder="slug (e.g. herbal-tea)" value={slug} onChange={setSlug} error={fieldError('slug')} />
        <Input placeholder="Name (English)" value={name} onChange={setName} error={fieldError('name')} />
        <Input placeholder="Sort order" value={sortOrder} onChange={setSortOrder} />
      </div>

      <fieldset className="rounded-lg border border-stone-200 p-3">
        <legend className="px-1 text-xs font-medium text-stone-500">
          Translations — optional, blank falls back to English
        </legend>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {PREFIXED_LOCALES.map((locale) => (
            <Input
              key={locale}
              placeholder={`Name (${localeLabel(locale)})`}
              value={translations[locale] ?? ''}
              onChange={(v) => setTranslations({ ...translations, [locale]: v })}
              error={fieldError(`translations.${locale}.name`)}
            />
          ))}
        </div>
      </fieldset>

      {formError && (
        <p className="rounded-lg bg-red-50 p-3 text-sm text-red-600">{formError}</p>
      )}
      {mutation.isSuccess && !category && (
        <p className="rounded-lg bg-emerald-50 p-3 text-sm text-emerald-700">Category created.</p>
      )}

      <div className="flex gap-2">
        <button
          type="submit"
          disabled={mutation.isPending}
          className="rounded-lg bg-emerald-700 px-5 py-2 text-sm font-medium text-white hover:bg-emerald-800 disabled:opacity-50"
        >
          {mutation.isPending ? 'Saving…' : category ? 'Save changes' : 'Create'}
        </button>
        {category && (
          <button
            type="button"
            onClick={onDone}
            className="rounded-lg px-5 py-2 text-sm font-medium text-stone-500 hover:bg-stone-100"
          >
            Cancel
          </button>
        )}
      </div>
    </form>
  )
}

function CategoryList({ onEdit }: { onEdit: (c: AdminCategory) => void }) {
  const categories = useAdminCategories()
  const reorder = useReorderCategories()
  const remove = useDeleteCategory()

  // Move = swap in the id list, then send the WHOLE order — the backend
  // rewrites sort_order by position, all-or-nothing.
  const move = (index: number, delta: -1 | 1) => {
    const list = categories.data
    if (!list) return
    const ids = list.map((c) => c.id)
    const target = index + delta
    if (target < 0 || target >= ids.length) return
    ;[ids[index], ids[target]] = [ids[target], ids[index]]
    reorder.mutate(ids)
  }

  const removeError =
    remove.error instanceof ApiError && remove.error.code === 'category_in_use'
      ? 'That category still has products — move or retire them first.'
      : remove.isError
        ? 'Deleting failed. Please try again.'
        : null

  return (
    <div className="mt-6 rounded-xl border border-stone-200 bg-white p-5">
      <h3 className="font-medium text-stone-700">Existing categories</h3>
      {removeError && (
        <p className="mt-2 rounded-lg bg-red-50 p-3 text-sm text-red-600">{removeError}</p>
      )}
      <ul className="mt-3 divide-y divide-stone-100 text-sm">
        {categories.data?.map((c, i) => (
          <li key={c.id} className="flex items-center gap-2 py-2">
            <div className="flex flex-col">
              <button
                type="button"
                aria-label={`Move ${c.name} up`}
                disabled={i === 0 || reorder.isPending}
                onClick={() => move(i, -1)}
                className="px-1 text-xs text-stone-400 hover:text-stone-700 disabled:opacity-30"
              >
                ▲
              </button>
              <button
                type="button"
                aria-label={`Move ${c.name} down`}
                disabled={i === (categories.data?.length ?? 0) - 1 || reorder.isPending}
                onClick={() => move(i, 1)}
                className="px-1 text-xs text-stone-400 hover:text-stone-700 disabled:opacity-30"
              >
                ▼
              </button>
            </div>
            <span className="text-stone-700">{c.name}</span>
            {PREFIXED_LOCALES.some((l) => c.translations[l]) && (
              <span className="text-xs text-stone-400">
                ({PREFIXED_LOCALES.flatMap((l) => c.translations[l] ?? []).join(' · ')})
              </span>
            )}
            <span className="ml-auto text-stone-400">{c.slug}</span>
            <button
              type="button"
              onClick={() => onEdit(c)}
              className="rounded-lg bg-stone-100 px-3 py-1 text-xs font-medium text-stone-700 hover:bg-stone-200"
            >
              edit
            </button>
            <button
              type="button"
              disabled={remove.isPending}
              onClick={() => remove.mutate(c.id)}
              className="rounded-lg bg-red-50 px-3 py-1 text-xs font-medium text-red-600 hover:bg-red-100 disabled:opacity-50"
            >
              delete
            </button>
          </li>
        ))}
      </ul>
    </div>
  )
}

function Input({
  placeholder,
  value,
  onChange,
  error,
}: {
  placeholder: string
  value: string
  onChange: (v: string) => void
  error?: string
}) {
  return (
    <div>
      <input
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-lg border border-stone-300 px-3 py-2 text-sm focus:border-emerald-600 focus:outline-none"
      />
      {error && <span className="mt-1 block text-xs text-red-600">{error}</span>}
    </div>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-3xl px-4 py-8">{children}</div>
}
