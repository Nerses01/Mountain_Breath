import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PaymentInstructions } from './PaymentInstructions'
import type { PaymentInstructions as Instructions } from '../../api/types'

/**
 * Decision #110: the panel renders what the SERVER composed and nothing
 * else — the amount in the order's own currency, the account and purpose
 * line for a transfer, a promise when no account is configured, and
 * nothing at all when there are no instructions (a paid order).
 */
describe('PaymentInstructions', () => {
  const transfer: Instructions = {
    method: 'bank_transfer',
    amount_minor: 6400,
    currency: 'AMD',
    reference: 'MB-42',
    bank: { recipient: 'Mountain Breath', bank: 'Ameriabank', iban: 'AM00 0000 0000 0000 0000' },
  }

  it('cash: the amount to have ready, in the order’s currency', () => {
    render(<PaymentInstructions instructions={{ method: 'cash_on_delivery', amount_minor: 6400, currency: 'AMD' }} />)
    // formatMoney writes a non-breaking space before ֏; Testing Library
    // normalizes it to a plain one before matching.
    expect(screen.getByText(/6,400 ֏ ready in cash/)).toBeInTheDocument()
  })

  it('transfer: the account as a definition list, and the purpose line', () => {
    render(<PaymentInstructions instructions={transfer} />)
    expect(screen.getByText(/Transfer 6,400 ֏ to:/)).toBeInTheDocument()
    expect(screen.getByText('Ameriabank')).toBeInTheDocument()
    expect(screen.getByText('AM00 0000 0000 0000 0000')).toBeInTheDocument()
    expect(screen.getByText('MB-42')).toBeInTheDocument()
    expect(screen.getByText('We ship as soon as it clears.')).toBeInTheDocument()
  })

  it('transfer without a configured account: a promise, no blanks', () => {
    render(<PaymentInstructions instructions={{ ...transfer, bank: undefined }} />)
    expect(screen.getByText(/email you the account details/)).toBeInTheDocument()
    expect(screen.queryByText('IBAN')).not.toBeInTheDocument()
  })

  it('nothing to draw when the server sent no instructions', () => {
    const { container } = render(<PaymentInstructions instructions={undefined} />)
    expect(container).toBeEmptyDOMElement()
  })
})
