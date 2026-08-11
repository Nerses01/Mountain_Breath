import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import './index.css'
// Imported for its side effect: this is where i18next.init() runs, and it
// must happen before any component calls useTranslation().
import './i18n'
import App from './App.tsx'
import { CurrencyProvider } from './lib/CurrencyProvider'

// One QueryClient for the whole app: it owns the request cache.
const queryClient = new QueryClient()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <QueryClientProvider client={queryClient}>
        {/* INSIDE QueryClientProvider: changing currency changes query keys,
            so the provider that owns the currency has to sit where the query
            client can already be reached. */}
        <CurrencyProvider>
          <App />
        </CurrencyProvider>
      </QueryClientProvider>
    </BrowserRouter>
  </StrictMode>,
)
