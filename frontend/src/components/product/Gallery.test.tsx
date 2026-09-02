import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { Gallery } from './Gallery'
import type { ProductImage, ProductVideo } from '../../api/types'

const images: ProductImage[] = [
  { id: 1, url: '/uploads/a.jpg', alt: 'A jar of royal jelly', is_primary: true },
  { id: 2, url: '/uploads/b.jpg', alt: 'The texture, close up', is_primary: false },
  { id: 3, url: '/uploads/c.jpg', alt: 'The jar in a hand', is_primary: false },
]

const video: ProductVideo = { id: 9, url: '/uploads/clip.mp4', alt: 'Harvest clip' }

const tabs = () => screen.getAllByRole('tab')
const strip = () => screen.getByRole('tablist')

/**
 * These assert the KEYBOARD CONTRACT, which is the only reason the gallery
 * is a tablist rather than three buttons and a picture. A mouse-only test
 * would pass against a version that is unusable without one.
 */
describe('Gallery', () => {
  it('shows the first image and marks its thumbnail selected', () => {
    render(<Gallery images={images} productName="Fresh Royal Jelly" />)

    expect(tabs()[0]).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByAltText('A jar of royal jelly')).toBeVisible()
  })

  it('is ONE tab stop, not one per thumbnail', () => {
    render(<Gallery images={images} productName="Fresh Royal Jelly" />)

    // Roving tabindex: walking past a five-image gallery must not cost a
    // keyboard user five presses.
    expect(tabs()[0]).toHaveAttribute('tabindex', '0')
    expect(tabs()[1]).toHaveAttribute('tabindex', '-1')
    expect(tabs()[2]).toHaveAttribute('tabindex', '-1')
  })

  it('moves with arrow keys, and moving selects', () => {
    render(<Gallery images={images} productName="Fresh Royal Jelly" />)

    fireEvent.keyDown(strip(), { key: 'ArrowRight' })
    expect(tabs()[1]).toHaveAttribute('aria-selected', 'true')
    // Selection follows focus — no extra Enter to actually see the image.
    expect(screen.getByAltText('The texture, close up')).toBeVisible()

    fireEvent.keyDown(strip(), { key: 'ArrowLeft' })
    expect(tabs()[0]).toHaveAttribute('aria-selected', 'true')
  })

  it('wraps around at both ends', () => {
    render(<Gallery images={images} productName="Fresh Royal Jelly" />)

    fireEvent.keyDown(strip(), { key: 'ArrowLeft' })
    expect(tabs()[2]).toHaveAttribute('aria-selected', 'true')

    fireEvent.keyDown(strip(), { key: 'ArrowRight' })
    expect(tabs()[0]).toHaveAttribute('aria-selected', 'true')
  })

  it('jumps to the ends with Home and End', () => {
    render(<Gallery images={images} productName="Fresh Royal Jelly" />)

    fireEvent.keyDown(strip(), { key: 'End' })
    expect(tabs()[2]).toHaveAttribute('aria-selected', 'true')

    fireEvent.keyDown(strip(), { key: 'Home' })
    expect(tabs()[0]).toHaveAttribute('aria-selected', 'true')
  })

  it('moves DOM focus, not just the highlight', () => {
    render(<Gallery images={images} productName="Fresh Royal Jelly" />)

    fireEvent.keyDown(strip(), { key: 'ArrowRight' })

    // The distinction that matters: React state alone would repaint the
    // selection and leave the browser focused on the old thumbnail.
    expect(document.activeElement).toBe(tabs()[1])
  })

  it('does not steal focus on first render', () => {
    render(<Gallery images={images} productName="Fresh Royal Jelly" />)

    // A gallery that grabs focus on page load drops a screen-reader user
    // into the middle of the document.
    expect(document.activeElement).toBe(document.body)
  })

  it('each panel is labelled by its own tab', () => {
    render(<Gallery images={images} productName="Fresh Royal Jelly" />)

    const tab = tabs()[0]
    const panelId = tab.getAttribute('aria-controls')
    expect(document.getElementById(panelId!)).toHaveAttribute('aria-labelledby', tab.id)
  })

  it('renders a placeholder, and no tablist, when there are no images', () => {
    render(<Gallery images={[]} productName="Fresh Royal Jelly" />)

    // The shop has no photography yet; the slot keeps the page's geometry
    // instead of collapsing the left column.
    expect(screen.getByText('Fresh Royal Jelly')).toBeInTheDocument()
    expect(screen.queryByRole('tablist')).not.toBeInTheDocument()
  })

  it('shows no thumbnail strip for a single image', () => {
    render(<Gallery images={[images[0]]} productName="Fresh Royal Jelly" />)

    // A tablist of one is navigation with nowhere to go.
    expect(screen.queryByRole('tablist')).not.toBeInTheDocument()
    expect(screen.getByAltText('A jar of royal jelly')).toBeVisible()
  })

  // ── The video tab (decision #99) ───────────────────────────────────────

  it('renders the video as the LAST tab, reachable by keyboard', () => {
    render(<Gallery images={images} video={video} productName="Fresh Royal Jelly" />)

    // Same strip, one more tab — the keyboard contract covers it for free.
    expect(tabs()).toHaveLength(4)
    expect(tabs()[3]).toHaveTextContent('Harvest clip')

    fireEvent.keyDown(strip(), { key: 'End' })
    expect(tabs()[3]).toHaveAttribute('aria-selected', 'true')

    const clip = document.querySelector('video')
    expect(clip).toHaveAttribute('src', '/uploads/clip.mp4')
    // The polite defaults: nothing plays or sounds until the visitor asks.
    expect(clip).toHaveAttribute('controls')
    expect(clip).toHaveProperty('muted', true)
    expect(clip).not.toHaveAttribute('autoplay')
  })

  it('pauses a playing video when another tab is selected', () => {
    // jsdom has no media pipeline: `paused` is forced to false to simulate
    // a clip mid-playback, and pause() is spied both to assert and to keep
    // "Not implemented" noise out of the run.
    const playing = vi
      .spyOn(HTMLMediaElement.prototype, 'paused', 'get')
      .mockReturnValue(false)
    const pause = vi
      .spyOn(HTMLMediaElement.prototype, 'pause')
      .mockImplementation(() => {})
    render(<Gallery images={images} video={video} productName="Fresh Royal Jelly" />)

    fireEvent.keyDown(strip(), { key: 'End' }) // onto the video
    pause.mockClear()
    fireEvent.keyDown(strip(), { key: 'Home' }) // away from it

    // Sound and motion from a hidden panel is a bug in any language.
    expect(pause).toHaveBeenCalled()
    pause.mockRestore()
    playing.mockRestore()
  })

  it('a lone video renders without a thumbnail strip', () => {
    render(<Gallery images={[]} video={video} productName="Fresh Royal Jelly" />)

    expect(screen.queryByRole('tablist')).not.toBeInTheDocument()
    expect(document.querySelector('video')).toHaveAttribute('src', '/uploads/clip.mp4')
  })
})
