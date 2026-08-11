import { describe, expect, it } from 'vitest'
import { formatMoney, inputToMinor, minorToInput } from './format'

// \u00A0 is a non-breaking space. Written as an escape throughout, because
// it is indistinguishable from an ordinary space in an editor and the
// difference is what half of these assertions are about.
const NBSP = '\u00A0'

describe('formatMoney', () => {
  it('puts the dollar symbol before two decimal places', () => {
    expect(formatMoney(1400, 'USD')).toBe('$14.00')
    expect(formatMoney(180000, 'USD')).toBe('$1,800.00')
    expect(formatMoney(199, 'USD')).toBe('$1.99')
    expect(formatMoney(1, 'USD')).toBe('$0.01')
    expect(formatMoney(0, 'USD')).toBe('$0.00')
  })

  // THE POINT OF E5. The old formatPrice divided by 100 unconditionally, so
  // 6700 drams would have rendered "67.00" — the shop would have looked a
  // hundred times cheaper in Armenia.
  it('renders drams whole, with the symbol after the number', () => {
    expect(formatMoney(6700, 'AMD')).toBe(`6,700${NBSP}֏`)
    expect(formatMoney(15300, 'AMD')).toBe(`15,300${NBSP}֏`)
    expect(formatMoney(0, 'AMD')).toBe(`0${NBSP}֏`)
  })

  it('separates a suffixed symbol with a NON-BREAKING space', () => {
    // An ordinary space would let a narrow column wrap the symbol onto its
    // own line, which reads as a different price.
    expect(formatMoney(15300, 'AMD')).toContain(NBSP)
    expect(formatMoney(15300, 'AMD')).not.toContain(' ')
  })

  // Symbol placement belongs to the CURRENCY, not to the reader's language.
  // Intl's own `style: 'currency'` would render "֏6,700" for an en-US reader
  // and "6 700 ֏" for an hy-AM one, so a price tag would change shape with
  // the site language. Pinning the number formatting to one locale and
  // placing the symbol ourselves is what keeps it stable.
  it('does not depend on the ambient locale', () => {
    expect(formatMoney(1400, 'USD')).toBe('$14.00')
    expect(formatMoney(6700, 'AMD')).toBe(`6,700${NBSP}֏`)
  })
})

describe('admin price boxes', () => {
  it('round-trips a dollar price through the input', () => {
    expect(minorToInput(1400, 'USD')).toBe('14.00')
    expect(inputToMinor('14.00', 'USD')).toBe(1400)
    expect(inputToMinor('14', 'USD')).toBe(1400)
    // A European keyboard's decimal comma is a person typing, not a bug.
    expect(inputToMinor('14,50', 'USD')).toBe(1450)
  })

  it('round-trips a dram price with no decimals at all', () => {
    expect(minorToInput(6700, 'AMD')).toBe('6700')
    expect(inputToMinor('6700', 'AMD')).toBe(6700)
    // Someone typing cents into a dram field means the nearest dram.
    expect(inputToMinor('6700.4', 'AMD')).toBe(6700)
    expect(inputToMinor('6700.6', 'AMD')).toBe(6701)
  })

  it('reads unparseable input as zero, which validation then rejects', () => {
    expect(inputToMinor('', 'USD')).toBe(0)
    expect(inputToMinor('free', 'USD')).toBe(0)
  })
})
