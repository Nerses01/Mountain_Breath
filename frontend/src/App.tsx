import { Link, Route, Routes } from 'react-router'
import { AuthStatus } from './components/AuthStatus'
import { CartLink } from './components/CartLink'
import { AdminOrdersPage } from './pages/AdminOrdersPage'
import { AdminPage } from './pages/AdminPage'
import { AdminProductsPage } from './pages/AdminProductsPage'
import { CartPage } from './pages/CartPage'
import { CatalogPage } from './pages/CatalogPage'
import { LoginPage } from './pages/LoginPage'
import { OrdersPage } from './pages/OrdersPage'
import { ProductPage } from './pages/ProductPage'

function App() {
  return (
    <div className="min-h-screen bg-stone-100">
      <header className="border-b border-stone-200 bg-white">
        <div className="mx-auto flex max-w-5xl items-center gap-3 px-4 py-4">
          <span className="text-2xl">🏔️</span>
          <Link to="/" className="flex-1">
            <h1 className="text-xl font-bold text-stone-800">Mountain Breath</h1>
            <p className="text-xs text-stone-400">
              tea · coffee · honey from the mountains
            </p>
          </Link>
          <CartLink />
          <AuthStatus />
        </div>
      </header>

      {/* The router swaps the page component based on the URL — no server
          round-trip, just React rendering a different component. */}
      <Routes>
        <Route path="/" element={<CatalogPage />} />
        <Route path="/products/:slug" element={<ProductPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/cart" element={<CartPage />} />
        <Route path="/orders" element={<OrdersPage />} />
        <Route path="/admin" element={<AdminPage />} />
        <Route path="/admin/products" element={<AdminProductsPage />} />
        <Route path="/admin/orders" element={<AdminOrdersPage />} />
      </Routes>
    </div>
  )
}

export default App
