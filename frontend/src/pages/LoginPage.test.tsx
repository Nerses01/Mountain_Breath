import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { LoginPage } from './LoginPage'
import { ResetPasswordPage } from './ResetPasswordPage'
import { ForgotPasswordPage } from './ForgotPasswordPage'

/**
 * The E8 auth screens at the component boundary. What these pin:
 *
 *  - "Keep me signed in" actually TRAVELS: the login body carries
 *    remember=true only when the box is ticked;
 *  - the Google button is a NAVIGATION (a real <a> to the start endpoint),
 *    and the Apple stub is decorative — no dead click handlers;
 *  - the forgot page's success copy is the conditional sentence (the 204
 *    tells it nothing more), and the reset page turns invalid_token into
 *    the request-a-new-one path.
 */

let requests: { url: string; body: unknown }[] = []

function stubFetch(status = 200, payload: unknown = {}) {
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      requests.push({
        url: String(input),
        body: init?.body ? JSON.parse(String(init.body)) : undefined,
      })
      return Promise.resolve(
        new Response(status === 204 ? null : JSON.stringify(payload), {
          status,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    }),
  )
}

beforeEach(() => {
  requests = []
})
afterEach(() => vi.unstubAllGlobals())

function renderAt(path: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <MemoryRouter initialEntries={[path]}>
      <QueryClientProvider client={qc}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          <Route path="/reset-password/:token" element={<ResetPasswordPage />} />
          <Route path="/" element={<p>home</p>} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('LoginPage', () => {
  it('posts remember only when the box is ticked', async () => {
    stubFetch(200, {
      id: 1, email: 'a@x', role: 'customer',
      hive: { prior_orders: 0, member: false, member_discount_percent: 0, first_delivery_free: true },
    })
    renderAt('/login')

    fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'a@x' } })
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'password-123' } })
    fireEvent.click(screen.getByLabelText('Keep me signed in'))
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }))

    await screen.findByText('home')
    const login = requests.find((r) => r.url.includes('/auth/login'))
    expect(login?.body).toEqual({ email: 'a@x', password: 'password-123', remember: true })
  })

  it('show/hide really toggles the input type', () => {
    renderAt('/login')
    const password = screen.getByLabelText('Password')
    expect(password).toHaveAttribute('type', 'password')
    fireEvent.click(screen.getByRole('button', { name: 'Show' }))
    expect(password).toHaveAttribute('type', 'text')
    expect(screen.getByRole('button', { name: 'Hide' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('Google is a link to the start endpoint; Apple is decorative', () => {
    renderAt('/login')
    expect(screen.getByRole('link', { name: 'Continue with Google' })).toHaveAttribute(
      'href',
      '/api/v1/auth/oauth/google',
    )
    // aria-hidden: the stub is a picture of a button, not a button.
    expect(screen.queryByRole('button', { name: 'Continue with Apple' })).not.toBeInTheDocument()
  })

  it('a failed Google round-trip explains itself', () => {
    renderAt('/login?oauth_error=1')
    expect(screen.getByRole('alert')).toHaveTextContent(/Google sign-in/)
  })
})

describe('ForgotPasswordPage', () => {
  it('shows the conditional sent-copy after the 204', async () => {
    stubFetch(204)
    renderAt('/forgot-password')

    fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'a@x' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send the link' }))

    // "If that address is ours…" — the page cannot honestly claim more,
    // because the server answers 204 either way.
    expect(await screen.findByRole('status')).toHaveTextContent(/If that address is ours/)
  })
})

describe('ResetPasswordPage', () => {
  it('posts the URL token with the new password', async () => {
    stubFetch(204)
    renderAt('/reset-password/raw-token-from-mail')

    fireEvent.change(screen.getByLabelText('New password'), {
      target: { value: 'brand-new-password' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save the new password' }))

    await screen.findByRole('status')
    const reset = requests.find((r) => r.url.includes('/auth/reset-password'))
    expect(reset?.body).toEqual({ token: 'raw-token-from-mail', password: 'brand-new-password' })
  })

  it('a dead link offers the way out', async () => {
    stubFetch(400, {
      error: { code: 'invalid_token', message: 'this reset link is not valid' },
    })
    renderAt('/reset-password/spent-token')

    fireEvent.change(screen.getByLabelText('New password'), {
      target: { value: 'brand-new-password' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save the new password' }))

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent(/no longer works/)
    expect(screen.getByRole('link', { name: 'Request a new one.' })).toHaveAttribute(
      'href',
      '/forgot-password',
    )
  })
})
