import { describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import App from './App'
import i18n from './i18n'

/**
 * Routing smoke tests.
 *
 * The locale prefixes mount the SAME route list three times, and the list is
 * built by a function returning an array of <Route> elements — a shape worth
 * proving at runtime, since a build that typechecks says nothing about
 * whether the router actually walks it.
 */
function renderAt(path: string) {
  // retry: false — the API is not running here, and the default retry policy
  // would leave queries in flight after the test ends.
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <MemoryRouter initialEntries={[path]}>
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('App routing', () => {
  it('renders the storefront shell at the unprefixed root', async () => {
    await i18n.changeLanguage('en')
    renderAt('/')

    expect(screen.getByRole('banner')).toBeInTheDocument()
    expect(screen.getByRole('contentinfo')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /shop/i })).toBeInTheDocument()
  })

  it('renders the same shell under a locale prefix', async () => {
    renderAt('/hy')

    const header = screen.getByRole('banner')
    // Scoped to the header on purpose: "Խանութ" is both the nav link and the
    // footer's Shop column heading, so an unscoped query is ambiguous.
    expect(
      await within(header).findByRole('link', { name: 'Խանութ' }),
    ).toHaveAttribute('href', '/hy/shop')

    // Proves useLocale drove i18next from the URL, not the reverse.
    expect(i18n.language).toBe('hy')

    await i18n.changeLanguage('en')
  })

  it('treats a non-locale first segment as a page, not a language', async () => {
    await i18n.changeLanguage('en')
    renderAt('/cart')

    // The reason the prefixes are enumerated instead of matched with a
    // `/:locale` param: a param would bind locale="cart" and render the
    // home page here.
    expect(screen.getByRole('banner')).toBeInTheDocument()
    expect(document.documentElement.lang).toBe('en')
  })

  it('keeps admin outside the storefront chrome', () => {
    renderAt('/admin')

    // Admin has its own navigation; the shopfront header would be noise.
    expect(screen.queryByRole('contentinfo')).not.toBeInTheDocument()
  })
})
