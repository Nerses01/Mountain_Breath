import { useRef, useState } from 'react'
import {
  useAdminProducts,
  useDeleteProductImage,
  useProduct,
  useSaveProductEditorial,
  useRelatedProducts,
  useSaveProductImages,
  useSaveProductRelated,
  useUpdateProduct,
  useUploadProductVideo,
} from '../../api/hooks'
import { useFieldErrors } from '../../i18n/useFieldErrors'
import type {
  AdminProduct,
  EditorialInput,
  ProductHighlight,
  ProductUsageCard,
} from '../../api/types'
import { PREFIXED_LOCALES } from '../../i18n/locales'
import { DEFAULT_LOCALE } from '../../i18n/useLocale'

/**
 * The admin's editor for everything E3 added to a product page: the gallery,
 * the "What it does" bullets, the usage cards, the notes and the curated
 * "Often taken together" list.
 *
 * DELIBERATELY IN ENGLISH ONLY, like the rest of the back office (E1.5's
 * scope decision): this is an internal tool for one family who share a
 * working language. What it EDITS is trilingual; the chrome around it is not.
 *
 * The editor reads through the PUBLIC product endpoint, one locale at a
 * time, rather than through an admin-shaped "give me every language at once"
 * read. That is a real trade: it costs one request per language tab, and it
 * means the editor sees exactly what a shopper sees, fallbacks included. The
 * alternative — a second read path with its own resolution rules — is how
 * an admin ends up showing content the storefront does not.
 */
export function ProductContentEditor({ product }: { product: AdminProduct }) {
  const [open, setOpen] = useState(false)

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="text-sm font-medium text-emerald-700 hover:underline"
      >
        edit page content →
      </button>
    )
  }

  return (
    <div className="mt-4 space-y-6 rounded-xl border border-stone-200 bg-stone-50 p-4">
      <div className="flex items-center justify-between">
        <h4 className="font-semibold text-stone-700">Page content</h4>
        <button
          type="button"
          onClick={() => setOpen(false)}
          className="text-sm text-stone-500 hover:underline"
        >
          close
        </button>
      </div>

      <MetadataEditor product={product} />
      <GalleryEditor product={product} />
      <VideoEditor product={product} />
      <EditorialEditor product={product} />
      <RelatedEditor product={product} />
    </div>
  )
}

// --- Metadata + notes ---------------------------------------------------

function MetadataEditor({ product }: { product: AdminProduct }) {
  const update = useUpdateProduct()
  // Reads the ENGLISH detail, because these fields' English copy lives on
  // the product's default-locale translation row.
  const detail = useProduct(product.slug)

  const [labBatch, setLabBatch] = useState<string | null>(null)
  const [coldChain, setColdChain] = useState<boolean | null>(null)
  const [notes, setNotes] = useState<Record<string, string> | null>(null)

  if (detail.isPending) return <p className="text-sm text-stone-400">Loading…</p>
  if (detail.isError || !detail.data) return null
  const d = detail.data

  // `?? server value` throughout: the field is uncontrolled-from-the-server
  // until it is first typed into, so a background refetch cannot yank what
  // the admin is halfway through writing.
  const current = {
    lab_batch: labBatch ?? d.lab_batch,
    is_cold_chain: coldChain ?? d.is_cold_chain,
    disclaimer: notes?.disclaimer ?? d.disclaimer,
    storage_note: notes?.storage_note ?? d.storage_note,
    harvest_note: notes?.harvest_note ?? d.harvest_note,
    shipping_note: notes?.shipping_note ?? d.shipping_note,
  }

  const setNote = (key: string, value: string) =>
    setNotes((n) => ({ ...(n ?? {}), [key]: value }))

  return (
    <section className="space-y-3">
      <h5 className="text-sm font-semibold uppercase tracking-wide text-stone-500">
        Metadata (English)
      </h5>

      <div className="grid gap-3 sm:grid-cols-2">
        <label className="text-sm">
          <span className="mb-1 block text-stone-600">Lab batch</span>
          <input
            value={current.lab_batch}
            onChange={(e) => setLabBatch(e.target.value)}
            placeholder="RJ-0626"
            className="w-full rounded-lg border border-stone-300 px-3 py-2"
          />
        </label>
        <label className="flex items-center gap-2 self-end text-sm text-stone-600">
          <input
            type="checkbox"
            checked={current.is_cold_chain}
            onChange={(e) => setColdChain(e.target.checked)}
          />
          Cold chain (E6 charges chilled shipping from this)
        </label>
      </div>

      {(
        [
          ['harvest_note', 'Harvest note', 'June 2026, Hive 41'],
          ['shipping_note', 'Shipping note', 'Chilled, 2–4 days'],
          ['disclaimer', 'Disclaimer', 'Not a medicine…'],
          ['storage_note', 'Storage (tab body)', ''],
        ] as const
      ).map(([key, label, placeholder]) => (
        <label key={key} className="block text-sm">
          <span className="mb-1 block text-stone-600">{label}</span>
          <textarea
            value={current[key]}
            onChange={(e) => setNote(key, e.target.value)}
            placeholder={placeholder}
            rows={key === 'storage_note' || key === 'disclaimer' ? 2 : 1}
            className="w-full rounded-lg border border-stone-300 px-3 py-2"
          />
        </label>
      ))}

      <button
        type="button"
        disabled={update.isPending}
        onClick={() =>
          update.mutate({
            id: product.id,
            data: {
              category_id: product.category_id,
              name: product.name,
              description: product.description,
              is_active: product.is_active,
              ...current,
            },
          })
        }
        className="rounded-lg bg-emerald-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
      >
        {update.isPending ? 'Saving…' : 'Save metadata'}
      </button>
    </section>
  )
}

// --- Gallery ------------------------------------------------------------

function GalleryEditor({ product }: { product: AdminProduct }) {
  const detail = useProduct(product.slug)
  const save = useSaveProductImages()
  const remove = useDeleteProductImage()

  if (!detail.data) return null
  const images = detail.data.images

  if (images.length === 0) {
    return (
      <section className="space-y-2">
        <h5 className="text-sm font-semibold uppercase tracking-wide text-stone-500">Gallery</h5>
        <p className="text-sm text-stone-400">
          No images yet — upload one with the image button above. The first upload becomes
          the hero automatically.
        </p>
      </section>
    )
  }

  return (
    <section className="space-y-2">
      <h5 className="text-sm font-semibold uppercase tracking-wide text-stone-500">Gallery</h5>
      <ul className="space-y-2">
        {images.map((img, i) => (
          <li key={img.id} className="flex items-center gap-3 rounded-lg bg-white p-2">
            <img src={img.url} alt="" className="size-12 rounded object-cover" />
            <span className="text-xs text-stone-400">#{i + 1}</span>

            {/* Radio, not a checkbox: exactly one hero, and a radio group is
                what "one of these" means to a screen reader. The database
                agrees via a partial unique index. */}
            <label className="flex items-center gap-1.5 text-sm text-stone-600">
              <input
                type="radio"
                name={`primary-${product.id}`}
                checked={img.is_primary}
                onChange={() =>
                  save.mutate({
                    id: product.id,
                    images: images.map((other) => ({
                      id: other.id,
                      is_primary: other.id === img.id,
                      alt: {},
                    })),
                  })
                }
              />
              hero
            </label>

            <button
              type="button"
              disabled={i === 0 || save.isPending}
              onClick={() => {
                // Reorder by swapping with the previous entry; the ARRAY
                // ORDER is what the API turns into sort_order.
                const next = [...images]
                ;[next[i - 1], next[i]] = [next[i], next[i - 1]]
                save.mutate({
                  id: product.id,
                  images: next.map((o) => ({ id: o.id, is_primary: o.is_primary, alt: {} })),
                })
              }}
              className="text-sm text-stone-500 hover:underline disabled:opacity-30"
            >
              ↑
            </button>

            <AltEditor productId={product.id} imageId={img.id} images={images} />

            <button
              type="button"
              disabled={remove.isPending}
              onClick={() => remove.mutate({ productId: product.id, imageId: img.id })}
              className="ml-auto text-sm text-red-600 hover:underline disabled:opacity-50"
            >
              delete
            </button>
          </li>
        ))}
      </ul>
      {save.isError && <p className="text-sm text-red-600">Could not save the gallery.</p>}
    </section>
  )
}

// --- Video -------------------------------------------------------------

/**
 * The single video slot (decision #99). One slot means the two states are
 * "empty → upload" and "filled → preview + delete"; there is no reorder, no
 * hero flag, no per-slot list. Replacing is delete-then-upload through the
 * SAME delete mutation the photos use, because the video is a gallery row
 * with kind='video' — one less endpoint, one less cache to invalidate.
 *
 * Exported for its test: the parent editor only renders behind useProduct,
 * and the section's two states are worth asserting in isolation.
 */
export function VideoEditor({ product }: { product: AdminProduct }) {
  const detail = useProduct(product.slug)
  const upload = useUploadProductVideo()
  const remove = useDeleteProductImage()
  const inputRef = useRef<HTMLInputElement>(null)

  // Upload failures (too large, wrong type, slot taken) are whole-request
  // errors — same shape as the photo slot's.
  const { formError } = useFieldErrors(upload.error)

  if (!detail.data) return null
  const video = detail.data.video

  return (
    <section className="space-y-2">
      <h5 className="text-sm font-semibold uppercase tracking-wide text-stone-500">Video</h5>

      {video ? (
        <div className="flex items-center gap-3 rounded-lg bg-white p-2">
          {/* preload="metadata": the admin sees the first frame and the
              duration without the browser pulling the whole clip into a
              list that may show six of these. */}
          <video
            src={video.url}
            preload="metadata"
            muted
            controls
            className="h-24 rounded"
            aria-label={video.alt || 'Product video'}
          />
          <p className="text-xs text-stone-400">
            Shown as the last gallery tab on the product page. To replace it,
            delete it and upload another.
          </p>
          <button
            type="button"
            disabled={remove.isPending}
            onClick={() => remove.mutate({ productId: product.id, imageId: video.id })}
            className="ml-auto text-sm text-red-600 hover:underline disabled:opacity-50"
          >
            delete video
          </button>
        </div>
      ) : (
        <>
          <button
            type="button"
            onClick={() => inputRef.current?.click()}
            disabled={upload.isPending}
            className="rounded-lg border border-dashed border-stone-300 px-4 py-2 text-sm text-stone-500 hover:border-emerald-600 disabled:opacity-50"
          >
            {upload.isPending ? 'Uploading…' : '+ 🎬 upload video (MP4/WebM, max 50 MB)'}
          </button>
          <input
            ref={inputRef}
            type="file"
            accept="video/mp4,video/webm"
            className="hidden"
            data-testid="video-file-input"
            onChange={(e) => {
              const file = e.target.files?.[0]
              if (file) upload.mutate({ id: product.id, file })
              e.target.value = '' // same file can be re-picked later
            }}
          />
        </>
      )}

      {formError && <p className="text-sm text-red-600">{formError}</p>}
      {remove.isError && <p className="text-sm text-red-600">Could not delete the video.</p>}
    </section>
  )
}

function AltEditor({
  productId,
  imageId,
  images,
}: {
  productId: number
  imageId: number
  images: { id: number; is_primary: boolean; alt: string }[]
}) {
  const save = useSaveProductImages()
  const [alt, setAlt] = useState<Record<string, string>>({})
  const [open, setOpen] = useState(false)

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="text-sm text-stone-500 hover:underline"
      >
        alt text
      </button>
    )
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      {/* "en" is a peer here, unlike product translations: an image's alt has
          no parent field to hold the English in. */}
      {[DEFAULT_LOCALE, ...PREFIXED_LOCALES].map((locale) => (
        <input
          key={locale}
          value={alt[locale] ?? ''}
          onChange={(e) => setAlt((a) => ({ ...a, [locale]: e.target.value }))}
          placeholder={`alt (${locale})`}
          className="w-36 rounded border border-stone-300 px-2 py-1 text-sm"
        />
      ))}
      <button
        type="button"
        onClick={() => {
          save.mutate({
            id: productId,
            images: images.map((o) => ({
              id: o.id,
              is_primary: o.is_primary,
              alt: o.id === imageId ? alt : {},
            })),
          })
          setOpen(false)
        }}
        className="text-sm font-medium text-emerald-700 hover:underline"
      >
        save alt
      </button>
    </div>
  )
}

// --- Highlights and usage cards, per locale -----------------------------

const LOCALES = [DEFAULT_LOCALE, ...PREFIXED_LOCALES] as const

function EditorialEditor({ product }: { product: AdminProduct }) {
  const [locale, setLocale] = useState<string>(DEFAULT_LOCALE)
  const save = useSaveProductEditorial()

  return (
    <section className="space-y-3">
      <div className="flex items-center gap-3">
        <h5 className="text-sm font-semibold uppercase tracking-wide text-stone-500">
          Page copy
        </h5>
        {/* One tab per language. Saving sends ONLY the open tab's locale, so
            editing Armenian cannot wipe English — the store leaves absent
            locales alone. */}
        <div className="flex gap-1">
          {LOCALES.map((l) => (
            <button
              key={l}
              type="button"
              onClick={() => setLocale(l)}
              className={
                l === locale
                  ? 'rounded bg-stone-800 px-2 py-1 text-xs font-medium text-white'
                  : 'rounded px-2 py-1 text-xs text-stone-500 hover:bg-stone-200'
              }
            >
              {l}
            </button>
          ))}
        </div>
      </div>

      {/* key={locale} remounts the form on a tab switch, so each language
          starts from ITS OWN server state instead of inheriting the draft
          left in the previous tab. */}
      <EditorialForm
        key={locale}
        product={product}
        locale={locale}
        onSave={(content) => save.mutate({ id: product.id, content: { [locale]: content } })}
        saving={save.isPending}
      />
      {save.isError && <p className="text-sm text-red-600">Could not save the copy.</p>}
    </section>
  )
}

function EditorialForm({
  product,
  locale,
  onSave,
  saving,
}: {
  product: AdminProduct
  locale: string
  onSave: (content: EditorialInput) => void
  saving: boolean
}) {
  // The public read, in this locale. Its fallback means an untranslated tab
  // opens pre-filled with the English — which is what the shopper currently
  // sees, and the right starting point for a translator.
  const detail = useProduct(product.slug)
  const [highlights, setHighlights] = useState<ProductHighlight[] | null>(null)
  const [cards, setCards] = useState<ProductUsageCard[] | null>(null)

  if (!detail.data) return null
  const h = highlights ?? detail.data.highlights
  const c = cards ?? detail.data.usage_cards

  return (
    <div className="space-y-3">
      <p className="text-xs text-stone-400">
        Editing <strong>{locale}</strong>. Empty rows are rejected; remove them instead.
      </p>

      <div className="space-y-2">
        <span className="text-sm text-stone-600">“What it does” bullets</span>
        {h.map((item, i) => (
          <div key={i} className="flex gap-2">
            <input
              value={item.text}
              onChange={(e) =>
                setHighlights(h.map((x, j) => (i === j ? { text: e.target.value } : x)))
              }
              className="flex-1 rounded-lg border border-stone-300 px-3 py-2 text-sm"
            />
            <button
              type="button"
              onClick={() => setHighlights(h.filter((_, j) => j !== i))}
              className="text-sm text-red-600 hover:underline"
            >
              remove
            </button>
          </div>
        ))}
        <button
          type="button"
          onClick={() => setHighlights([...h, { text: '' }])}
          className="text-sm text-emerald-700 hover:underline"
        >
          + bullet
        </button>
      </div>

      <div className="space-y-2">
        <span className="text-sm text-stone-600">Usage cards</span>
        {c.map((card, i) => (
          <div key={i} className="space-y-1 rounded-lg bg-white p-2">
            <div className="flex gap-2">
              <input
                value={card.kicker}
                placeholder="Morning"
                onChange={(e) =>
                  setCards(c.map((x, j) => (i === j ? { ...x, kicker: e.target.value } : x)))
                }
                className="w-32 rounded border border-stone-300 px-2 py-1 text-sm"
              />
              <input
                value={card.title}
                placeholder="A grain of rice"
                onChange={(e) =>
                  setCards(c.map((x, j) => (i === j ? { ...x, title: e.target.value } : x)))
                }
                className="flex-1 rounded border border-stone-300 px-2 py-1 text-sm"
              />
              <button
                type="button"
                onClick={() => setCards(c.filter((_, j) => j !== i))}
                className="text-sm text-red-600 hover:underline"
              >
                remove
              </button>
            </div>
            <textarea
              value={card.body}
              rows={2}
              onChange={(e) =>
                setCards(c.map((x, j) => (i === j ? { ...x, body: e.target.value } : x)))
              }
              className="w-full rounded border border-stone-300 px-2 py-1 text-sm"
            />
          </div>
        ))}
        <button
          type="button"
          onClick={() => setCards([...c, { kicker: '', title: '', body: '' }])}
          className="text-sm text-emerald-700 hover:underline"
        >
          + card
        </button>
      </div>

      <button
        type="button"
        disabled={saving}
        onClick={() => onSave({ highlights: h, usage_cards: c })}
        className="rounded-lg bg-emerald-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
      >
        {saving ? 'Saving…' : `Save ${locale} copy`}
      </button>
    </div>
  )
}

// --- Related products ---------------------------------------------------

function RelatedEditor({ product }: { product: AdminProduct }) {
  const all = useAdminProducts()
  // `curated: true` — the admin's own list, NOT the computed panel. Reading
  // the storefront's version here would pre-tick a list nobody chose, and
  // saving it would freeze a dynamic panel into a static one. The empty
  // picker was worse still: for a product that IS curated, one click on
  // Save would have wiped the curation.
  const curated = useRelatedProducts(product.slug, true)
  const save = useSaveProductRelated()
  const [picked, setPicked] = useState<number[] | null>(null)

  if (!all.data || curated.isPending) return null

  // Inactive products are filtered out: the storefront read skips them, so
  // curating one produces a pairing that can never render.
  const others = all.data.items.filter((p) => p.id !== product.id && p.is_active)
  const selection = picked ?? (curated.data ?? []).map((p) => p.id)

  return (
    <section className="space-y-2">
      <h5 className="text-sm font-semibold uppercase tracking-wide text-stone-500">
        Often taken together
      </h5>
      <p className="text-xs text-stone-400">
        {selection.length === 0
          ? 'Nothing curated — the shop computes this panel from shared benefits, then popularity. Ticking anything replaces that entirely.'
          : 'Curated. Untick everything and save to hand the panel back to the computed list.'}
      </p>
      <div className="flex flex-wrap gap-2">
        {others.map((p) => {
          const on = selection.includes(p.id)
          return (
            <button
              key={p.id}
              type="button"
              aria-pressed={on}
              onClick={() =>
                setPicked(
                  on ? selection.filter((id) => id !== p.id) : [...selection, p.id],
                )
              }
              className={
                on
                  ? 'rounded-full bg-stone-800 px-3 py-1 text-sm text-white'
                  : 'rounded-full bg-white px-3 py-1 text-sm text-stone-600 ring-1 ring-stone-200'
              }
            >
              {p.name}
            </button>
          )
        })}
      </div>
      <button
        type="button"
        disabled={save.isPending}
        onClick={() => save.mutate({ id: product.id, relatedIds: selection })}
        className="rounded-lg bg-emerald-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
      >
        {save.isPending ? 'Saving…' : 'Save related'}
      </button>
    </section>
  )
}
