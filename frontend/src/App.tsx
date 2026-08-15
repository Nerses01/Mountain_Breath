import { Route, Routes } from 'react-router'
import { Layout } from './components/layout/Layout'
import { PREFIXED_LOCALES } from './i18n/locales'
import { AdminOrdersPage } from './pages/AdminOrdersPage'
import { AdminPage } from './pages/AdminPage'
import { AdminProductsPage } from './pages/AdminProductsPage'
import { AdminReviewsPage } from './pages/AdminReviewsPage'
import { AccountPage } from './pages/AccountPage'
import { CartPage } from './pages/CartPage'
import { CheckoutPage } from './pages/CheckoutPage'
import { ForgotPasswordPage } from './pages/ForgotPasswordPage'
import { OrderDetailPage } from './pages/OrderDetailPage'
import { HomePage } from './pages/HomePage'
import { LoginPage } from './pages/LoginPage'
import { OrdersPage } from './pages/OrdersPage'
import { ProductPage } from './pages/ProductPage'
import { ResetPasswordPage } from './pages/ResetPasswordPage'
import { ShopPage } from './pages/ShopPage'
import { WishlistPage } from './pages/WishlistPage'

/**
 * The storefront pages, defined once and mounted under every locale prefix.
 * Paths are RELATIVE (no leading slash) so the same list works at `/` and
 * at `/hy`.
 */
function storefrontRoutes() {
  return [
    // E2 splits the two apart: `/` was the catalog because there was no home
    // page to put there. Now `/` is the designed Home and `/shop` is the
    // faceted listing — which is also why the filter state can live in the
    // query string, since `/shop` is the only page that has any.
    <Route key="index" index element={<HomePage />} />,
    <Route key="shop" path="shop" element={<ShopPage />} />,
    <Route key="product" path="products/:slug" element={<ProductPage />} />,
    <Route key="login" path="login" element={<LoginPage />} />,
    <Route key="cart" path="cart" element={<CartPage />} />,
    <Route key="orders" path="orders" element={<OrdersPage />} />,
    <Route key="order" path="orders/:id" element={<OrderDetailPage />} />,
    // E8: the account area and the reset flow. The reset route's token is a
    // URL param because that is what the EMAILED link carries — the page
    // just posts it back.
    <Route key="account" path="account" element={<AccountPage />} />,
    <Route key="wishlist" path="wishlist" element={<WishlistPage />} />,
    <Route key="forgot" path="forgot-password" element={<ForgotPasswordPage />} />,
    <Route key="reset" path="reset-password/:token" element={<ResetPasswordPage />} />,
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
      {/* Checkout sits OUTSIDE Layout: the design gives it its own minimal
          chrome (logo, steps, "Secure") and no site nav — the one page the
          shop wants no wandering from is the one with the money on it. */}
      <Route path="/checkout" element={<CheckoutPage />} />

      {/* The other languages are enumerated rather than matched with a
          `/:locale` param. A param would happily match `/cart` and treat
          "cart" as a language, silently rendering the home page there. */}
      {PREFIXED_LOCALES.map((code) => (
        <Route key={code} path={`/${code}`} element={<Layout />}>
          {storefrontRoutes()}
        </Route>
      ))}
      {PREFIXED_LOCALES.map((code) => (
        <Route key={`${code}-checkout`} path={`/${code}/checkout`} element={<CheckoutPage />} />
      ))}

      {/* Admin keeps its own chrome: no storefront header or footer, and no
          locale prefix — it is a back office, not a shopfront. */}
      <Route path="/admin" element={<AdminPage />} />
      <Route path="/admin/products" element={<AdminProductsPage />} />
      <Route path="/admin/orders" element={<AdminOrdersPage />} />
      <Route path="/admin/reviews" element={<AdminReviewsPage />} />
    </Routes>
  )
}

export default App
