import { Link } from 'react-router'
import { useLogout, useMe } from '../api/hooks'

export function AuthStatus() {
  const me = useMe()
  const logout = useLogout()

  if (me.isPending) {
    return null // don't flash anything while we check the session
  }

  if (!me.data) {
    return (
      <Link
        to="/login"
        className="rounded-lg bg-emerald-700 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-800"
      >
        Sign in
      </Link>
    )
  }

  return (
    <div className="flex items-center gap-3 text-sm">
      {me.data.role === 'admin' && (
        <Link to="/admin" className="font-medium text-emerald-700 hover:underline">
          Admin
        </Link>
      )}
      <span className="text-stone-500">{me.data.email}</span>
      <button
        type="button"
        onClick={() => logout.mutate()}
        className="text-stone-400 hover:text-stone-600"
      >
        Sign out
      </button>
    </div>
  )
}
