import { useState } from 'react'
import { useAdminReviews, useModerateReview, useMe } from '../api/hooks'
import type { ReviewStatus } from '../api/types'
import { AdminNav } from '../components/AdminNav'

/**
 * The moderation queue. English only, like the rest of the back office.
 *
 * Opens on `pending`, because that is the only status that represents WORK —
 * the other two are archives. A queue that opens on "everything" makes the
 * moderator filter before they can start.
 */
const TABS: { status: ReviewStatus | undefined; label: string }[] = [
  { status: 'pending', label: 'Pending' },
  { status: 'published', label: 'Published' },
  { status: 'rejected', label: 'Rejected' },
  { status: undefined, label: 'All' },
]

export function AdminReviewsPage() {
  const me = useMe()
  const [status, setStatus] = useState<ReviewStatus | undefined>('pending')
  const reviews = useAdminReviews(status)
  const moderate = useModerateReview()

  if (me.isPending) return <Shell>Loading…</Shell>
  if (me.data?.role !== 'admin') return <Shell>Admins only.</Shell>

  return (
    <Shell>
      <AdminNav />
      <h2 className="mt-4 text-xl font-bold text-stone-800">Admin — Reviews</h2>

      <div className="mt-4 flex gap-2">
        {TABS.map((tab) => (
          <button
            key={tab.label}
            type="button"
            aria-pressed={status === tab.status}
            onClick={() => setStatus(tab.status)}
            className={
              status === tab.status
                ? 'rounded-full bg-stone-800 px-4 py-1.5 text-sm font-medium text-white'
                : 'rounded-full bg-white px-4 py-1.5 text-sm text-stone-600 ring-1 ring-stone-200 hover:bg-stone-50'
            }
          >
            {tab.label}
          </button>
        ))}
      </div>

      {reviews.isPending && <p className="mt-6 text-stone-400">Loading…</p>}
      {reviews.isError && <p className="mt-6 text-red-600">Failed to load reviews.</p>}
      {reviews.data?.items.length === 0 && (
        <p className="mt-6 text-stone-400">Nothing here.</p>
      )}

      <div className="mt-6 space-y-3">
        {reviews.data?.items.map((r) => (
          <article key={r.id} className="rounded-xl border border-stone-200 bg-white p-4">
            <div className="flex flex-wrap items-center gap-3">
              <span className="font-semibold text-stone-800">
                {'★'.repeat(r.rating)}
                <span className="text-stone-300">{'★'.repeat(5 - r.rating)}</span>
              </span>
              {/* The moderator sees the real address: judging whether a
                  review is genuine sometimes turns on who wrote it. The
                  public list only ever gets a display name. */}
              <span className="text-sm text-stone-500">{r.email}</span>
              <span className="text-xs text-stone-400">
                {new Date(r.created_at).toLocaleDateString()}
              </span>
              <span
                className={
                  r.status === 'published'
                    ? 'rounded-full bg-emerald-100 px-2 py-0.5 text-xs text-emerald-700'
                    : r.status === 'rejected'
                      ? 'rounded-full bg-red-100 px-2 py-0.5 text-xs text-red-700'
                      : 'rounded-full bg-amber-100 px-2 py-0.5 text-xs text-amber-700'
                }
              >
                {r.status}
              </span>
            </div>

            {r.title && <h3 className="mt-2 font-semibold text-stone-800">{r.title}</h3>}
            {r.body && <p className="mt-1 text-sm text-stone-600">{r.body}</p>}

            <div className="mt-3 flex gap-3">
              {/* Publishing or rejecting recomputes the product's stored
                  average inside the same transaction — "moderation changes
                  the public average immediately" is that one line of SQL. */}
              <button
                type="button"
                disabled={moderate.isPending || r.status === 'published'}
                onClick={() => moderate.mutate({ id: r.id, status: 'published' })}
                className="text-sm font-medium text-emerald-700 hover:underline disabled:opacity-40"
              >
                publish
              </button>
              <button
                type="button"
                disabled={moderate.isPending || r.status === 'rejected'}
                onClick={() => moderate.mutate({ id: r.id, status: 'rejected' })}
                className="text-sm font-medium text-red-600 hover:underline disabled:opacity-40"
              >
                reject
              </button>
            </div>
          </article>
        ))}
      </div>
    </Shell>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-4xl px-4 py-8">{children}</div>
}
