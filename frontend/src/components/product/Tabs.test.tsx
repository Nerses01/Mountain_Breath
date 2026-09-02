import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router'
import { Tabs, type Tab } from './Tabs'

const tabs: Tab[] = [
  { id: 'how-to-take-it', label: 'How to take it', panel: <p>Under the tongue.</p> },
  { id: 'storage', label: 'Storage', panel: <p>Keep refrigerated.</p> },
]

function HashProbe() {
  const { hash } = useLocation()
  return <output data-testid="hash">{hash}</output>
}

function renderTabs(initialEntry = '/products/fresh-royal-jelly') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Tabs tabs={tabs} label="Product details" />
      <HashProbe />
    </MemoryRouter>,
  )
}

const tabButtons = () => screen.getAllByRole('tab')
const strip = () => screen.getByRole('tablist')
const hash = () => screen.getByTestId('hash').textContent

describe('Tabs', () => {
  it('opens on the first tab when the URL has no hash', () => {
    renderTabs()

    expect(tabButtons()[0]).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByText('Under the tongue.')).toBeVisible()
  })

  it('opens on the tab named by the URL hash', () => {
    // The whole point of putting it in the hash: a tab is linkable.
    renderTabs('/products/fresh-royal-jelly#storage')

    expect(tabButtons()[1]).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByText('Keep refrigerated.')).toBeVisible()
    expect(screen.getByText('Under the tongue.')).not.toBeVisible()
  })

  it('writes the hash when a tab is clicked, so the view is shareable', () => {
    renderTabs()

    fireEvent.click(tabButtons()[1])

    expect(hash()).toBe('#storage')
    expect(screen.getByText('Keep refrigerated.')).toBeVisible()
  })

  it('ignores a hash that names no tab', () => {
    // Some other anchor on the page, or a hand-edited URL — the tabs should
    // fall back to the first rather than showing none.
    renderTabs('/products/fresh-royal-jelly#reviews')

    expect(tabButtons()[0]).toHaveAttribute('aria-selected', 'true')
  })

  it('moves with arrow keys and selects as it goes', () => {
    renderTabs()

    fireEvent.keyDown(strip(), { key: 'ArrowRight' })
    expect(tabButtons()[1]).toHaveAttribute('aria-selected', 'true')
    expect(document.activeElement).toBe(tabButtons()[1])

    // Wraps.
    fireEvent.keyDown(strip(), { key: 'ArrowRight' })
    expect(tabButtons()[0]).toHaveAttribute('aria-selected', 'true')
  })

  it('is one tab stop', () => {
    renderTabs()

    expect(tabButtons()[0]).toHaveAttribute('tabindex', '0')
    expect(tabButtons()[1]).toHaveAttribute('tabindex', '-1')
  })

  it('makes each panel focusable', () => {
    renderTabs()

    // A panel may hold nothing focusable, and without this a keyboard user
    // could select a tab but never reach or scroll its contents.
    for (const panel of screen.getAllByRole('tabpanel', { hidden: true })) {
      expect(panel).toHaveAttribute('tabindex', '0')
    }
  })

  it('links each panel to its tab both ways', () => {
    renderTabs()

    const tab = tabButtons()[0]
    const panelId = tab.getAttribute('aria-controls')
    expect(document.getElementById(panelId!)).toHaveAttribute('aria-labelledby', tab.id)
  })
})
