import { ApiError } from '../api/client'
import { useAdminUsers, useMe, useUpdateUserRole } from '../api/hooks'
import { AdminNav } from '../components/AdminNav'

/**
 * F2 (decision #96): promotion stops being a psql incantation. Roles only —
 * no creation (registration owns that), no deletion (the privacy item's
 * job), no editing anyone's profile (that belongs to its owner). The
 * backend refuses to demote the last admin (409 last_admin), self-demotion
 * included; this page just renders that refusal honestly.
 */
export function AdminUsersPage() {
  const me = useMe()
  const users = useAdminUsers()
  const update = useUpdateUserRole()

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

  const updateError =
    update.error instanceof ApiError && update.error.code === 'last_admin'
      ? 'The shop must keep at least one admin — promote someone else first.'
      : update.isError
        ? 'Changing the role failed. Please try again.'
        : null

  return (
    <Shell>
      <div className="flex items-center gap-6">
        <h2 className="text-xl font-bold text-stone-800">Admin — Users</h2>
        <AdminNav />
      </div>

      {users.isPending && <p className="mt-4 text-stone-400">Loading…</p>}
      {users.isError && <p className="mt-4 text-red-600">Failed to load users.</p>}
      {updateError && (
        <p className="mt-4 rounded-lg bg-red-50 p-3 text-sm text-red-600">{updateError}</p>
      )}

      {users.data && (
        <ul className="mt-4 divide-y divide-stone-100 rounded-xl border border-stone-200 bg-white p-5 text-sm">
          {users.data.map((u) => (
            <li key={u.id} className="flex flex-wrap items-center gap-3 py-2.5">
              <div className="flex min-w-0 flex-col">
                <span className="font-medium text-stone-800">
                  {u.email}
                  {u.id === me.data!.id && (
                    <span className="ml-1.5 text-xs font-normal text-stone-400">(you)</span>
                  )}
                </span>
                {u.full_name && <span className="text-xs text-stone-500">{u.full_name}</span>}
              </div>
              <span
                className={
                  u.role === 'admin'
                    ? 'rounded-full bg-emerald-100 px-2.5 py-0.5 text-xs font-medium text-emerald-800'
                    : 'rounded-full bg-stone-100 px-2.5 py-0.5 text-xs font-medium text-stone-600'
                }
              >
                {u.role}
              </span>
              <span className="ml-auto text-xs text-stone-400">
                {u.orders} {u.orders === 1 ? 'order' : 'orders'} · joined{' '}
                {new Date(u.created_at).toLocaleDateString()}
              </span>
              <button
                type="button"
                disabled={update.isPending}
                onClick={() =>
                  update.mutate({
                    id: u.id,
                    role: u.role === 'admin' ? 'customer' : 'admin',
                  })
                }
                className="rounded-lg bg-stone-100 px-3 py-1 text-xs font-medium text-stone-700 hover:bg-stone-200 disabled:opacity-50"
              >
                {u.role === 'admin' ? 'make customer' : 'make admin'}
              </button>
            </li>
          ))}
        </ul>
      )}
    </Shell>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-3xl px-4 py-8">{children}</div>
}
