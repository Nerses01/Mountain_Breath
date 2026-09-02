import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { PriceRange } from './PriceRange'

function renderRange(
  overrides: Partial<React.ComponentProps<typeof PriceRange>> = {},
) {
  const onCommit = vi.fn()
  render(
    <PriceRange
      label="Price"
      min={900}
      max={10500}
      value={{ min: 900, max: 10500 }}
      onCommit={onCommit}
      {...overrides}
    />,
  )
  return {
    onCommit,
    low: screen.getByLabelText('Price — minimum') as HTMLInputElement,
    high: screen.getByLabelText('Price — maximum') as HTMLInputElement,
  }
}

describe('PriceRange', () => {
  it('renders both bounds as money', () => {
    renderRange({ value: { min: 1400, max: 3200 } })

    expect(screen.getByText('$14.00')).toBeInTheDocument()
    expect(screen.getByText('$32.00')).toBeInTheDocument()
  })

  it('clamps the low thumb so it cannot pass the high one', () => {
    const { low, high, onCommit } = renderRange({ value: { min: 900, max: 3200 } })

    fireEvent.change(low, { target: { value: '9000' } })
    fireEvent.pointerUp(low)

    // Dragged past the ceiling, so it stops AT it — the pair can meet but
    // never swap, which would otherwise produce min > max and an empty grid.
    expect(onCommit).toHaveBeenCalledWith({ min: 3200, max: 3200 })
    expect(high.value).toBe('3200')
  })

  it('clamps the high thumb so it cannot pass the low one', () => {
    const { high, onCommit } = renderRange({ value: { min: 3200, max: 10500 } })

    fireEvent.change(high, { target: { value: '100' } })
    fireEvent.pointerUp(high)

    expect(onCommit).toHaveBeenCalledWith({ min: 3200, max: 3200 })
  })

  it('does not commit while dragging, only on release', () => {
    const { low, onCommit } = renderRange()

    // Three steps of a drag: the label follows immediately...
    fireEvent.change(low, { target: { value: '1000' } })
    fireEvent.change(low, { target: { value: '1200' } })
    fireEvent.change(low, { target: { value: '1400' } })
    expect(onCommit).not.toHaveBeenCalled()
    expect(screen.getByText('$14.00')).toBeInTheDocument()

    // ...and one request goes out when the thumb is let go.
    fireEvent.pointerUp(low)
    expect(onCommit).toHaveBeenCalledTimes(1)
    expect(onCommit).toHaveBeenCalledWith({ min: 1400, max: 10500 })
  })

  it('commits on key release, so the keyboard works too', () => {
    const { low, onCommit } = renderRange()

    fireEvent.change(low, { target: { value: '1000' } })
    fireEvent.keyUp(low, { key: 'ArrowRight' })

    expect(onCommit).toHaveBeenCalledWith({ min: 1000, max: 10500 })
  })

  it('follows the value when the URL changes underneath it', () => {
    const { rerender } = render(
      <PriceRange
        label="Price"
        min={900}
        max={10500}
        value={{ min: 900, max: 10500 }}
        onCommit={vi.fn()}
      />,
    )
    expect(screen.getByText('$105.00')).toBeInTheDocument()

    // The back button, or a filter cleared elsewhere. Without the sync
    // effect the thumbs would keep showing the old selection.
    rerender(
      <PriceRange
        label="Price"
        min={900}
        max={10500}
        value={{ min: 1400, max: 3200 }}
        onCommit={vi.fn()}
      />,
    )
    expect(screen.getByText('$14.00')).toBeInTheDocument()
    expect(screen.getByText('$32.00')).toBeInTheDocument()
  })

  it('disables itself when every product costs the same', () => {
    // One product in the filtered catalog means min === max, which would
    // otherwise divide by zero when positioning the thumbs.
    const { low, high } = renderRange({
      min: 1400,
      max: 1400,
      value: { min: 1400, max: 1400 },
    })

    expect(low).toBeDisabled()
    expect(high).toBeDisabled()
  })
})
