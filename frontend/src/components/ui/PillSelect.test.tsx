import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { PillSelect } from './PillSelect'

/**
 * The select-only combobox contract: one tab stop, arrows move a pointed-at
 * option (aria-activedescendant, never DOM focus), Enter commits, Escape
 * abandons, outside presses close. The rendering details are free to
 * change; this behaviour is what a keyboard or screen-reader user relies
 * on.
 */
const options = [
  { value: 'popular', label: 'Most loved' },
  { value: 'rating', label: 'Best rated' },
  { value: 'newest', label: 'Newest' },
] as const

function renderSelect(onChange = vi.fn()) {
  render(
    <PillSelect
      ariaLabel="Sort products"
      prefix="Sort:"
      value="popular"
      onChange={onChange}
      options={[...options]}
    />,
  )
  return onChange
}

describe('PillSelect', () => {
  it('shows the prefix and selected label on the closed pill', () => {
    renderSelect()

    const trigger = screen.getByRole('combobox', { name: 'Sort products' })
    expect(trigger).toHaveTextContent('Sort: Most loved')
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
  })

  it('opens on click with the CURRENT value selected, prefix-free rows', () => {
    renderSelect()

    fireEvent.click(screen.getByRole('combobox'))

    expect(screen.getByRole('listbox')).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Most loved' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
    expect(screen.getByRole('option', { name: 'Best rated' })).toHaveAttribute(
      'aria-selected',
      'false',
    )
  })

  it('arrows point at options and Enter commits the pointed-at one', () => {
    const onChange = renderSelect()
    const trigger = screen.getByRole('combobox')

    fireEvent.keyDown(trigger, { key: 'ArrowDown' }) // opens on the current value
    fireEvent.keyDown(trigger, { key: 'ArrowDown' }) // → Best rated
    expect(trigger).toHaveAttribute(
      'aria-activedescendant',
      screen.getByRole('option', { name: 'Best rated' }).id,
    )

    fireEvent.keyDown(trigger, { key: 'Enter' })

    expect(onChange).toHaveBeenCalledWith('rating')
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
  })

  it('Escape closes without committing', () => {
    const onChange = renderSelect()
    const trigger = screen.getByRole('combobox')

    fireEvent.keyDown(trigger, { key: 'ArrowDown' })
    fireEvent.keyDown(trigger, { key: 'ArrowDown' })
    fireEvent.keyDown(trigger, { key: 'Escape' })

    expect(onChange).not.toHaveBeenCalled()
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
  })

  it('clicking an option commits it', () => {
    const onChange = renderSelect()

    fireEvent.click(screen.getByRole('combobox'))
    fireEvent.click(screen.getByRole('option', { name: 'Newest' }))

    expect(onChange).toHaveBeenCalledWith('newest')
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
  })

  it('re-selecting the current value closes without firing onChange', () => {
    const onChange = renderSelect()

    fireEvent.click(screen.getByRole('combobox'))
    fireEvent.click(screen.getByRole('option', { name: 'Most loved' }))

    // No-op selections must not refetch the product list.
    expect(onChange).not.toHaveBeenCalled()
  })

  it('a press outside closes the list', () => {
    renderSelect()

    fireEvent.click(screen.getByRole('combobox'))
    fireEvent.pointerDown(document.body)

    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
  })
})
