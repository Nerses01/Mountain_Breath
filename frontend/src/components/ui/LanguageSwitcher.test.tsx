import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { LanguageSwitcher } from './LanguageSwitcher'
import i18n from '../../i18n'

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <LanguageSwitcher />
    </MemoryRouter>,
  )
}

describe('LanguageSwitcher', () => {
  it('lists every language in its own script', () => {
    renderAt('/')

    // "Armenian" is no use to someone who only reads Armenian.
    expect(screen.getByRole('link', { name: 'English' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Հայերեն' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Русский' })).toBeInTheDocument()
  })

  it('adds a prefix when leaving the default language', () => {
    renderAt('/cart')

    expect(screen.getByRole('link', { name: 'English' })).toHaveAttribute(
      'href',
      '/cart',
    )
    expect(screen.getByRole('link', { name: 'Հայերեն' })).toHaveAttribute(
      'href',
      '/hy/cart',
    )
  })

  it('keeps you on the same page when switching', () => {
    renderAt('/hy/products/honey')

    // The whole point: changing language must not bounce you to the home
    // page and lose the product you were looking at.
    expect(screen.getByRole('link', { name: 'Русский' })).toHaveAttribute(
      'href',
      '/ru/products/honey',
    )
    expect(screen.getByRole('link', { name: 'English' })).toHaveAttribute(
      'href',
      '/products/honey',
    )
  })

  it('marks the active language for assistive tech', () => {
    renderAt('/ru/cart')

    expect(screen.getByRole('link', { name: 'Русский' })).toHaveAttribute(
      'aria-current',
      'true',
    )
    expect(screen.getByRole('link', { name: 'English' })).not.toHaveAttribute(
      'aria-current',
    )
  })

  it('tags each link with its own lang for correct pronunciation', () => {
    renderAt('/')

    // Without this a screen reader reads "Русский" with an English voice.
    expect(screen.getByRole('link', { name: 'Հայերեն' })).toHaveAttribute(
      'lang',
      'hy',
    )
  })

  it('switches i18next when the route locale changes', async () => {
    await i18n.changeLanguage('en')
    renderAt('/hy')

    // useLocale syncs i18next FROM the url, so the rendered page language
    // can never disagree with the address bar.
    await new Promise((r) => setTimeout(r, 0))
    expect(i18n.language).toBe('hy')

    await i18n.changeLanguage('en')
  })
})
