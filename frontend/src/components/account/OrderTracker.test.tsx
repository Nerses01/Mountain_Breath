import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { OrderTracker } from './OrderTracker'
import type { Order, OrderEvent, OrderStatus } from '../../api/types'

/**
 * Decision #2's mapping, pinned: the tracker draws the REAL state machine
 * (Placed → Confirmed → Shipped → Delivered), marks exactly the current
 * step with aria-current="step", dates steps only from RECORDED events,
 * and renders cancellation as a band, not a fifth step.
 */

function order(status: OrderStatus, events: OrderEvent[]): Order {
  return {
    id: 7,
    status,
    created_at: '2026-08-14T10:00:00Z',
    items: [],
    subtotal_minor: 0,
    shipping_minor: 0,
    discount_minor: 0,
    tax_minor: 0,
    total_minor: 0,
    member_discount_minor: 0,
    promo_discount_minor: 0,
    payment_method: 'card',
    payment_status: 'unpaid',
    leave_with_neighbour: false,
    currency: 'USD',
    events,
    has_cold_chain: false,
  }
}

function renderTracker(o: Order) {
  return render(
    <MemoryRouter>
      <OrderTracker order={o} />
    </MemoryRouter>,
  )
}

const pendingEvent: OrderEvent = { status: 'pending', created_at: '2026-08-14T10:00:00Z' }

describe('OrderTracker', () => {
  // status → which of the four labels should carry aria-current="step";
  // null = none (delivered: every step is done).
  const cases: Array<{ status: OrderStatus; current: string | null }> = [
    { status: 'pending', current: 'Placed' },
    { status: 'confirmed', current: 'Confirmed' },
    { status: 'shipped', current: 'Shipped' },
    { status: 'delivered', current: null },
  ]

  for (const { status, current } of cases) {
    it(`${status}: four steps, aria-current on ${current ?? 'none'}`, () => {
      renderTracker(order(status, [pendingEvent]))

      const steps = screen.getAllByRole('listitem')
      expect(steps).toHaveLength(4)

      const marked = steps.filter((li) => li.getAttribute('aria-current') === 'step')
      if (current === null) {
        expect(marked).toHaveLength(0)
      } else {
        expect(marked).toHaveLength(1)
        expect(marked[0]).toHaveTextContent(current)
      }
    })
  }

  it('dates come only from recorded events — unrecorded steps show a dash', () => {
    renderTracker(
      order('shipped', [
        pendingEvent,
        { status: 'confirmed', created_at: '2026-08-15T10:00:00Z' },
      ]),
    )

    const steps = screen.getAllByRole('listitem')
    // Placed and Confirmed have recorded dates; Shipped happened (it IS the
    // current status) but was never recorded — honesty over invention.
    expect(steps[0]).toHaveTextContent(/Aug/)
    expect(steps[1]).toHaveTextContent(/Aug/)
    expect(steps[2]).toHaveTextContent('—')
    expect(steps[3]).toHaveTextContent('—')
  })

  it('cancelled renders the band instead of steps', () => {
    renderTracker(
      order('cancelled', [
        pendingEvent,
        { status: 'cancelled', created_at: '2026-08-16T10:00:00Z' },
      ]),
    )

    expect(screen.queryAllByRole('listitem')).toHaveLength(0)
    expect(screen.getByText(/cancelled/)).toBeInTheDocument()
  })
})
