import { useState } from 'react'
import { Link } from 'react-router'
import { ApiError } from '../api/client'
import { useCategories, useCreateCategory, useMe } from '../api/hooks'
import { AdminNav } from '../components/AdminNav'

// NOTE: this guard is user experience, not security. The backend's
// requireAdmin middleware is the real gate — anyone can bypass this page
// with devtools, and the API will still refuse them.
export function AdminPage() {
  const me = useMe()

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
      <CategoryForm />
      <CategoryList />
    </Shell>
  )
}

function CategoryForm() {
  const [slug, setSlug] = useState('')
  const [name, setName] = useState('')
  const [sortOrder, setSortOrder] = useState('0')
  const create = useCreateCategory()

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    create.mutate(
      { slug, name, sort_order: Number(sortOrder) || 0 },
      {
        onSuccess: () => {
          setSlug('')
          setName('')
          setSortOrder('0')
        },
      },
    )
  }

  const err = create.error instanceof ApiError ? create.error : null

  return (
    <form
      onSubmit={onSubmit}
      className="mt-6 space-y-3 rounded-xl border border-stone-200 bg-white p-5"
    >
      <h3 className="font-medium text-stone-700">New category</h3>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <Input placeholder="slug (e.g. herbal-tea)" value={slug} onChange={setSlug} error={err?.fields?.slug} />
        <Input placeholder="Name" value={name} onChange={setName} error={err?.fields?.name} />
        <Input placeholder="Sort order" value={sortOrder} onChange={setSortOrder} />
      </div>

      {err && !err.fields && (
        <p className="rounded-lg bg-red-50 p-3 text-sm text-red-600">{err.message}</p>
      )}
      {create.isSuccess && (
        <p className="rounded-lg bg-emerald-50 p-3 text-sm text-emerald-700">Category created.</p>
      )}

      <button
        type="submit"
        disabled={create.isPending}
        className="rounded-lg bg-emerald-700 px-5 py-2 text-sm font-medium text-white hover:bg-emerald-800 disabled:opacity-50"
      >
        {create.isPending ? 'Creating…' : 'Create'}
      </button>
    </form>
  )
}

function CategoryList() {
  const categories = useCategories()

  return (
    <div className="mt-6 rounded-xl border border-stone-200 bg-white p-5">
      <h3 className="font-medium text-stone-700">Existing categories</h3>
      <ul className="mt-3 divide-y divide-stone-100 text-sm">
        {categories.data?.map((c) => (
          <li key={c.id} className="flex justify-between py-2">
            <span className="text-stone-700">{c.name}</span>
            <span className="text-stone-400">
              {c.slug} · order {c.sort_order}
            </span>
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
