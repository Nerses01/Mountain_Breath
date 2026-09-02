import { Outlet } from 'react-router'
import { SiteFooter } from './SiteFooter'
import { SiteHeader } from './SiteHeader'

/**
 * The public shell: header, page, footer.
 *
 * Rendered as a react-router layout route — the matched child page appears
 * at <Outlet />, so the header and footer mount once and survive navigation
 * instead of being re-created per page.
 *
 * Admin routes deliberately sit OUTSIDE this: they keep their own chrome
 * (AdminNav), and the storefront header would be noise in a back office.
 *
 * `min-h-screen` + `flex-col` with a growing <main> keeps the footer at the
 * bottom of short pages rather than floating up mid-screen.
 */
export function Layout() {
  return (
    <div className="flex min-h-screen flex-col bg-page">
      <SiteHeader />
      <main className="flex-1">
        <Outlet />
      </main>
      <SiteFooter />
    </div>
  )
}
