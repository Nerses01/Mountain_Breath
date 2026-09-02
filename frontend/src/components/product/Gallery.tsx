import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { ProductImage, ProductVideo } from '../../api/types'
import { cx } from '../../lib/cx'

/**
 * The product page's hero image plus its thumbnail strip.
 *
 * No carousel library. The whole behaviour is "show one of N media", and
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
 *
 * Decision #99 added the video as the LAST tab: same strip, same keyboard
 * contract, one more panel. It never autoplays — the visitor presses play —
 * and switching to another tab pauses it, because sound and motion from a
 * hidden panel is a bug in any language.
 */
export function Gallery({
  images,
  video = null,
  productName,
}: {
  images: ProductImage[]
  video?: ProductVideo | null
  productName: string
}) {
  const { t } = useTranslation()
  const [active, setActive] = useState(0)
  const thumbRefs = useRef<(HTMLButtonElement | null)[]>([])
  const videoRef = useRef<HTMLVideoElement>(null)
  // Only steal focus after a KEY press, never on first render — a gallery
  // that grabs focus on page load throws a screen reader into the middle of
  // the page.
  const shouldFocus = useRef(false)

  // One flat list of tabs, photos first, the clip last. A discriminated
  // union, so each render site switch()es on `kind` instead of probing
  // fields — the TypeScript spelling of a tagged variant.
  type Tab =
    | { kind: 'image'; id: number; url: string; alt: string }
    | { kind: 'video'; id: number; url: string; alt: string }
  const tabs: Tab[] = [
    ...images.map((img): Tab => ({ kind: 'image', id: img.id, url: img.url, alt: img.alt })),
    ...(video
      ? [{ kind: 'video', id: video.id, url: video.url, alt: video.alt } as Tab]
      : []),
  ]
  const videoIndex = video ? tabs.length - 1 : -1

  useEffect(() => {
    if (shouldFocus.current) {
      thumbRefs.current[active]?.focus()
      shouldFocus.current = false
    }
  }, [active])

  // Pause on switch-away. An effect rather than an onClick side effect: the
  // active index can change from clicks AND arrow keys, and this covers
  // every path with one rule. The paused check keeps this from "pausing" a
  // video that never played (which is also the mount state).
  useEffect(() => {
    const clip = videoRef.current
    if (active !== videoIndex && clip && !clip.paused) {
      clip.pause()
    }
  }, [active, videoIndex])

  // The design draws a hatched placeholder where the photograph goes. The
  // shop has no photography yet, so an empty gallery keeps the page's
  // geometry rather than collapsing the left column.
  if (tabs.length === 0) {
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
    setActive((next + tabs.length) % tabs.length)
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
        move(tabs.length - 1)
        break
    }
  }

  const current = tabs[active]
  const videoLabel = (alt: string) => alt || t('product:gallery.video')

  return (
    <div className="flex flex-col gap-4">
      {tabs.map((tab, i) => (
        <div
          key={tab.id}
          id={`gallery-panel-${tab.id}`}
          role="tabpanel"
          aria-labelledby={`gallery-tab-${tab.id}`}
          // hidden rather than unmounted: the panel has to exist for the
          // tab's aria-controls to point at something real — and unmounting
          // the video would silently reset its playback position.
          hidden={i !== active}
          className="h-130 overflow-hidden rounded-2xl bg-panel"
        >
          {tab.kind === 'image' ? (
            <img
              src={tab.url}
              // Alt from the API, in the reader's language — the only
              // description of this photo a non-sighted customer gets.
              alt={tab.alt}
              className="size-full object-cover"
            />
          ) : (
            // muted + playsInline: the polite defaults — no sound until the
            // visitor asks, no fullscreen hijack on iOS. preload="metadata"
            // fetches dimensions and duration, not the clip.
            <video
              ref={videoRef}
              src={tab.url}
              controls
              muted
              playsInline
              preload="metadata"
              aria-label={videoLabel(tab.alt)}
              className="size-full object-cover"
            />
          )}
        </div>
      ))}

      {tabs.length > 1 && (
        <div
          role="tablist"
          aria-label={t('product:gallery.label')}
          onKeyDown={onKeyDown}
          className="grid grid-cols-4 gap-3.5"
        >
          {tabs.map((tab, i) => (
            <button
              key={tab.id}
              ref={(el) => {
                thumbRefs.current[i] = el
              }}
              id={`gallery-tab-${tab.id}`}
              type="button"
              role="tab"
              aria-selected={i === active}
              aria-controls={`gallery-panel-${tab.id}`}
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
              {tab.kind === 'image' ? (
                <>
                  {/* Decorative: the tab's accessible name comes from the
                      alt text below, and the panel repeats it. */}
                  <img src={tab.url} alt="" className="size-full object-cover" />
                  <span className="sr-only">
                    {tab.alt || t('product:gallery.image', { n: i + 1 })}
                  </span>
                </>
              ) : (
                <>
                  {/* A play glyph, not a frame grab: pulling a poster frame
                      would cost a video request per card in the strip. */}
                  <span aria-hidden className="flex size-full items-center justify-center">
                    <svg viewBox="0 0 24 24" className="size-8 fill-ink-muted">
                      <path d="M8 5v14l11-7z" />
                    </svg>
                  </span>
                  <span className="sr-only">{videoLabel(tab.alt)}</span>
                </>
              )}
            </button>
          ))}
        </div>
      )}

      {/* Announced on change, so a screen reader user learns the hero moved
          without having to go looking for it. */}
      <p aria-live="polite" className="sr-only">
        {current.kind === 'video' ? videoLabel(current.alt) : current.alt}
      </p>
    </div>
  )
}
