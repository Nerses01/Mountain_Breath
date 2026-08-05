import { Route, Routes } from 'react-router'
import { Layout } from './components/layout/Layout'
import { PREFIXED_LOCALES } from './i18n/locales'
import { AdminOrdersPage } from './pages/AdminOrdersPage'
import { AdminPage } from './pages/AdminPage'
import { AdminProductsPage } from './pages/AdminProductsPage'
import { CartPage } from './pages/CartPage'
import { CatalogPage } from './pages/CatalogPage'
import { LoginPage } from './pages/LoginPage'
import { OrdersPage } from './pages/OrdersPage'
import { ProductPage } from './pages/ProductPage'

/**
 * The storefront pages, defined once and mounted under every locale prefix.
 * Paths are RELATIVE (no leading slash) so the same list works at `/` and
 * at `/hy`.
 */
function storefrontRoutes() {
  return [
    <Route key="index" index element={<CatalogPage />} />,
    <Route key="shop" path="shop" element={<CatalogPage />} />,
    <Route key="product" path="products/:slug" element={<ProductPage />} />,
    <Route key="login" path="login" element={<LoginPage />} />,
    <Route key="cart" path="cart" element={<CartPage />} />,
    <Route key="orders" path="orders" element={<OrdersPage />} />,
  ]
}

function App() {
  return (
    <Routes>
      {/* English lives at the root with no prefix — the stated default, so
          every link written elsewhere keeps working unprefixed. */}
      <Route path="/" element={<Layout />}>
        {storefrontRoutes()}
      </Route>

      {/* The other languages are enumerated rather than matched with a
          `/:locale` param. A param would happily match `/cart` and treat
          "cart" as a language, silently rendering the home page there. */}
      {PREFIXED_LOCALES.map((code) => (
        <Route key={code} path={`/${code}`} element={<Layout />}>
          {storefrontRoutes()}
        </Route>
      ))}

      {/* Admin keeps its own chrome: no storefront header or footer, and no
          locale prefix — it is a back office, not a shopfront. */}
      <Route path="/admin" element={<AdminPage />} />
      <Route path="/admin/products" element={<AdminProductsPage />} />
      <Route path="/admin/orders" element={<AdminOrdersPage />} />
    </Routes>
  )
}

export default App
