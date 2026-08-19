import { Link, NavLink } from 'react-router'

const tabs = [
  { to: '/admin', label: 'Categories', end: true },
  { to: '/admin/products', label: 'Products', end: false },
  { to: '/admin/orders', label: 'Orders', end: false },
  { to: '/admin/promos', label: 'Promos', end: false },
  { to: '/admin/reviews', label: 'Reviews', end: false },
  { to: '/admin/users', label: 'Users', end: false },
]

export function AdminNav() {
  return (
    <nav className="flex items-center gap-2">
      {tabs.map((t) => (
        <NavLink
          key={t.to}
          to={t.to}
          end={t.end}
          className={({ isActive }) =>
            isActive
              ? 'rounded-lg bg-stone-800 px-4 py-1.5 text-sm font-medium text-white'
              : 'rounded-lg px-4 py-1.5 text-sm font-medium text-stone-500 hover:bg-stone-200'
          }
        >
          {t.label}
        </NavLink>
      ))}
      {/* The way OUT — the back office has its own chrome with no site
          header, so without this the only return is editing the URL.
          English literal like every admin label: the back office is
          deliberately untranslated. */}
      <span aria-hidden className="mx-1 h-5 w-px bg-stone-300" />
      <Link
        to="/"
        className="rounded-lg px-4 py-1.5 text-sm font-medium text-stone-500 hover:bg-stone-200"
      >
        ← Back to shop
      </Link>
    </nav>
  )
}
