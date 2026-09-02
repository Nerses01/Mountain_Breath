import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { QtyStepper } from './QtyStepper'

const decrease = () => screen.getByRole('button', { name: /decrease/i })
const increase = () => screen.getByRole('button', { name: /increase/i })

describe('QtyStepper', () => {
  it('steps up and down through onChange', () => {
    const onChange = vi.fn()
    render(<QtyStepper value={3} max={10} onChange={onChange} />)

    fireEvent.click(increase())
    expect(onChange).toHaveBeenCalledWith(4)

    fireEvent.click(decrease())
    expect(onChange).toHaveBeenCalledWith(2)
  })

  it('clamps at the minimum of 1', () => {
    const onChange = vi.fn()
    render(<QtyStepper value={1} max={10} onChange={onChange} />)

    // Disabled rather than firing onChange(0) — the cart has no meaning for
    // a zero-quantity line; removing an item is a separate action.
    expect(decrease()).toBeDisabled()
    fireEvent.click(decrease())
    expect(onChange).not.toHaveBeenCalled()
  })

  it('clamps at available stock', () => {
    const onChange = vi.fn()
    render(<QtyStepper value={3} max={3} onChange={onChange} />)

    expect(increase()).toBeDisabled()
    fireEvent.click(increase())
    expect(onChange).not.toHaveBeenCalled()
  })

  it('allows unbounded increase when stock is unknown', () => {
    const onChange = vi.fn()
    render(<QtyStepper value={99} onChange={onChange} />)

    // No `max` given: the control must not invent a ceiling of its own.
    expect(increase()).not.toBeDisabled()
    fireEvent.click(increase())
    expect(onChange).toHaveBeenCalledWith(100)
  })

  it('labels both controls for screen readers', () => {
    render(<QtyStepper value={1} onChange={vi.fn()} label="Jars" />)

    // The mock's − and + are bare glyphs; without these names a screen
    // reader announces only "button".
    expect(screen.getByRole('button', { name: 'Decrease jars' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Increase jars' })).toBeInTheDocument()
  })
})
