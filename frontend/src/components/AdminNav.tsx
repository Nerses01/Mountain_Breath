import { NavLink } from 'react-router'

const tabs = [
  { to: '/admin', label: 'Categories', end: true },
  { to: '/admin/products', label: 'Products', end: false },
  { to: '/admin/orders', label: 'Orders', end: false },
  { to: '/admin/reviews', label: 'Reviews', end: false },
]

export function AdminNav() {
  return (
    <nav className="flex gap-2">
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
    </nav>
  )
}
