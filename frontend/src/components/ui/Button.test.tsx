import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { Button, ButtonLink } from './Button'

describe('Button', () => {
  it('renders as a button defaulting to type="button"', () => {
    render(<Button>Add to cart</Button>)

    // Not type="submit" — an unqualified <button> in a form would submit it.
    expect(screen.getByRole('button', { name: 'Add to cart' })).toHaveAttribute(
      'type',
      'button',
    )
  })

  it('paints the primary variant in the AA-safe brand ink, not the raw brand orange', () => {
    render(<Button variant="primary">Buy</Button>)

    // Guards the contrast decision from E1: cream on the mock's #e4761f
    // measures 2.9:1, so `primary` must resolve to the darkened twin.
    const cls = screen.getByRole('button').className
    expect(cls).toContain('bg-brand-ink')
    expect(cls).not.toContain('bg-brand ')
  })

  it.each([
    ['dark', 'bg-bark'],
    ['honey', 'bg-honey'],
    ['outline', 'border-bark'],
  ] as const)('renders the %s variant', (variant, expected) => {
    render(<Button variant={variant}>Label</Button>)

    expect(screen.getByRole('button').className).toContain(expected)
  })

  it('drops the pill radius for the underlined ghost variant', () => {
    render(<Button variant="ghost">Meet the beekeepers</Button>)

    const cls = screen.getByRole('button').className
    expect(cls).toContain('border-honey')
    expect(cls).not.toContain('rounded-full')
  })

  it('forwards the disabled attribute', () => {
    render(<Button disabled>Buy</Button>)

    // Whether a browser suppresses clicks on a disabled control is the
    // platform's job to guarantee, not ours to re-test; what we own is
    // passing the attribute through at all.
    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('ButtonLink renders an anchor sharing the same styling', () => {
    render(
      <MemoryRouter>
        <ButtonLink to="/shop">Shop the hive</ButtonLink>
      </MemoryRouter>,
    )

    // An <a>, not a <button>: navigation, so middle-click and "open in new
    // tab" behave the way a link should.
    const link = screen.getByRole('link', { name: 'Shop the hive' })
    expect(link).toHaveAttribute('href', '/shop')
    expect(link.className).toContain('bg-brand-ink')
  })
})
