import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Stars } from './Stars'

/**
 * Two things are worth pinning here: that the star row announces a NUMBER
 * rather than ten glyph names, and that a fractional average renders as a
 * fraction rather than being rounded to the nearest half.
 */
describe('Stars', () => {
  it('announces the rating as one image, not ten glyphs', () => {
    render(<Stars rating={4.7} count={12} />)

    // role="img" with a name: the parts are hidden, the whole is described.
    expect(screen.getByRole('img', { name: '4.7 out of 5, 12 reviews' })).toBeInTheDocument()
  })

  it('uses the singular for one review', () => {
    render(<Stars rating={5} count={1} />)

    expect(screen.getByRole('img', { name: '5.0 out of 5, 1 review' })).toBeInTheDocument()
  })

  it('omits the count from the name when there is none', () => {
    render(<Stars rating={3} />)

    expect(screen.getByRole('img', { name: '3.0 out of 5' })).toBeInTheDocument()
  })

  it('fills exactly the fraction, without rounding to a half star', () => {
    const { container } = render(<Stars rating={4.67} count={3} />)

    // 4.67 / 5 = 93.4%. Rounding this to 4.5 stars would put a visible lie
    // on the page — the clip width is what keeps the picture honest.
    const fill = container.querySelector('[style*="width"]') as HTMLElement
    expect(fill.style.width).toBe('93.4%')
  })

  it('clamps a nonsense rating instead of overflowing its box', () => {
    const { container } = render(<Stars rating={9} count={1} />)

    const fill = container.querySelector('[style*="width"]') as HTMLElement
    expect(fill.style.width).toBe('100%')
  })

  it('says so plainly when nothing has been rated', () => {
    render(<Stars rating={0} count={0} />)

    // An empty row of grey stars says nothing; a sentence does.
    expect(screen.getByText('No reviews yet')).toBeInTheDocument()
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
  })

  it('can hide the written count while keeping it in the accessible name', () => {
    render(<Stars rating={4} count={7} showCount={false} />)

    // Inside a review row the number is noise on screen but still belongs to
    // the star row's name.
    expect(screen.queryByText('(7 reviews)')).not.toBeInTheDocument()
    expect(screen.getByRole('img', { name: '4.0 out of 5, 7 reviews' })).toBeInTheDocument()
  })
})
