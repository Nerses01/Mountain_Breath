import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { ProductImage } from '../../api/types'
import { cx } from '../../lib/cx'

/**
 * The product page's hero image plus its thumbnail strip.
 *
 * No carousel library. The whole behaviour is "show one of N images", and
 * the accessible version of that is a well-known pattern the platform
 * already supports — a tablist, where each thumbnail is a tab and the hero
 * is its panel.
 *
 * The keyboard contract that pattern requires, and which a row of plain
 * buttons would get wrong:
 *
 *  - ONE tab stop for the whole strip, not one per thumbnail. A gallery of
 *    five images should not cost a keyboard user five presses to walk past.
 *    That is `tabIndex={0}` on the selected thumb and `-1` on the rest —
 *    "roving tabindex".
 *  - Arrow keys move between thumbnails, Home/End jump to the ends.
 *  - Moving focus SELECTS, so the hero follows the arrow keys directly
 *    rather than requiring an extra Enter.
 *
 * Focus is moved imperatively after a key press because the DOM decides
 * where focus lives; React state alone would update the highlight and leave
 * the browser focused on the old element.
 */
export function Gallery({ images, productName }: { images: ProductImage[]; productName: string }) {
  const { t } = useTranslation()
  const [active, setActive] = useState(0)
  const thumbRefs = useRef<(HTMLButtonElement | null)[]>([])
  // Only steal focus after a KEY press, never on first render — a gallery
  // that grabs focus on page load throws a screen reader into the middle of
  // the page.
  const shouldFocus = useRef(false)

  useEffect(() => {
    if (shouldFocus.current) {
      thumbRefs.current[active]?.focus()
      shouldFocus.current = false
    }
  }, [active])

  // The design draws a hatched placeholder where the photograph goes. The
  // shop has no photography yet, so an empty gallery keeps the page's
  // geometry rather than collapsing the left column.
  if (images.length === 0) {
    return (
      <div
        aria-hidden
        className="flex h-130 items-center justify-center rounded-2xl bg-panel text-center font-mono text-xs uppercase tracking-label text-ink-muted"
      >
        {productName}
      </div>
    )
  }

  const move = (next: number) => {
    shouldFocus.current = true
    // Wrap around: from the last thumbnail, Right goes back to the first.
    setActive((next + images.length) % images.length)
  }

  const onKeyDown = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowRight':
      case 'ArrowDown':
        e.preventDefault()
        move(active + 1)
        break
      case 'ArrowLeft':
      case 'ArrowUp':
        e.preventDefault()
        move(active - 1)
        break
      case 'Home':
        e.preventDefault()
        move(0)
        break
      case 'End':
        e.preventDefault()
        move(images.length - 1)
        break
    }
  }

  const current = images[active]

  return (
    <div className="flex flex-col gap-4">
      {images.map((img, i) => (
        <div
          key={img.id}
          id={`gallery-panel-${img.id}`}
          role="tabpanel"
          aria-labelledby={`gallery-tab-${img.id}`}
          // hidden rather than unmounted: the panel has to exist for the
          // tab's aria-controls to point at something real.
          hidden={i !== active}
          className="h-130 overflow-hidden rounded-2xl bg-panel"
        >
          <img
            src={img.url}
            // Alt from the API, in the reader's language — the only
            // description of this photo a non-sighted customer gets.
            alt={img.alt}
            className="size-full object-cover"
          />
        </div>
      ))}

      {images.length > 1 && (
        <div
          role="tablist"
          aria-label={t('product:gallery.label')}
          onKeyDown={onKeyDown}
          className="grid grid-cols-4 gap-3.5"
        >
          {images.map((img, i) => (
            <button
              key={img.id}
              ref={(el) => {
                thumbRefs.current[i] = el
              }}
              id={`gallery-tab-${img.id}`}
              type="button"
              role="tab"
              aria-selected={i === active}
              aria-controls={`gallery-panel-${img.id}`}
              // Roving tabindex: the strip is ONE tab stop.
              tabIndex={i === active ? 0 : -1}
              onClick={() => setActive(i)}
              className={cx(
                'h-24 overflow-hidden rounded-lg bg-panel transition',
                i === active
                  ? 'border-2 border-brand'
                  : 'border-2 border-transparent hover:border-line-strong',
              )}
            >
              {/* Decorative: the tab's accessible name comes from the alt
                  text below, and the panel repeats it. */}
              <img src={img.url} alt="" className="size-full object-cover" />
              <span className="sr-only">{img.alt || t('product:gallery.image', { n: i + 1 })}</span>
            </button>
          ))}
        </div>
      )}

      {/* Announced on change, so a screen reader user learns the hero moved
          without having to go looking for it. */}
      <p aria-live="polite" className="sr-only">
        {current.alt}
      </p>
    </div>
  )
}
