import { describe, expect, it } from 'vitest'
import { formatPrice } from './format'

describe('formatPrice', () => {
  it('converts minor units to a two-decimal string', () => {
    expect(formatPrice(180000)).toBe('1,800.00')
    expect(formatPrice(950000)).toBe('9,500.00')
  })

  it('keeps cents', () => {
    expect(formatPrice(199)).toBe('1.99')
    expect(formatPrice(1)).toBe('0.01')
  })

  it('handles zero', () => {
    expect(formatPrice(0)).toBe('0.00')
  })
})
