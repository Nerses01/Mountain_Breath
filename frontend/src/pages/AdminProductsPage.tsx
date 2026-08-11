import { useRef, useState } from 'react'
import { useFieldErrors } from '../i18n/useFieldErrors'
import { PREFIXED_LOCALES } from '../i18n/locales'
import { emptyTranslationDraft, localeLabel, translationPayload } from '../lib/translations'
import {
  useAdminProducts,
  useCategories,
  useCreateProduct,
  useMe,
  useUpdateProduct,
  useUpdateVariant,
  useUploadProductImage,
} from '../api/hooks'
import type { AdminProduct, NewVariantInput, ProductVariant } from '../api/types'
import { AdminNav } from '../components/AdminNav'
import { formatPrice } from '../lib/format'
import { ProductContentEditor } from './admin/ProductContentEditor'

// Admins type prices in major units ("1500.00"); the API speaks minor units.
// The conversion lives at this edge and nowhere else.
function toMinor(price: string): number {
  return Math.round(parseFloat(price.replace(',', '.')) * 100)
}

export function AdminProductsPage() {
  const me = useMe()

  if (me.isPending) return <Shell>Checking access…</Shell>
  if (!me.data || me.data.role !== 'admin') {
    return (
      <Shell>
        <p className="rounded-lg bg-red-50 p-4 text-red-600">
          This area requires an admin account.
        </p>
      </Shell>
    )
  }

  return (
    <Shell>
      <div className="flex items-center gap-6">
        <h2 className="text-xl font-bold text-stone-800">Admin — Products</h2>
        <AdminNav />
      </div>
      <CreateProductForm />
      <ProductList />
    </Shell>
  )
}

// --- creation form with dynamic variant rows -------------------------------

interface VariantDraft {
  sku: string
  label: string
  price: string // major units as typed; converted on submit
  stock: string
}

const emptyVariant: VariantDraft = { sku: '', label: '', price: '', stock: '0' }

function CreateProductForm() {
  const categories = useCategories()
  const create = useCreateProduct()

  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [description, setDescription] = useState('')
  // Array state: the form's variant rows live in one array; every row edit
  // replaces the array (immutably), same rule as all React state.
  const [variants, setVariants] = useState<VariantDraft[]>([{ ...emptyVariant }])
  const [translations, setTranslations] = useState(() =>
    emptyTranslationDraft({ name: '', description: '' }),
  )

  // fieldErr keeps its name so the JSON-path call sites below
  // (`fieldErr('variants[0].sku')`) are untouched; it now resolves the
  // API's validation code into the reader's language.
  const { fieldError: fieldErr, formError } = useFieldErrors(create.error)

  function setVariant(i: number, patch: Partial<VariantDraft>) {
    setVariants(variants.map((v, idx) => (idx === i ? { ...v, ...patch } : v)))
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const payload = {
      category_id: Number(categoryId) || 0,
      slug,
      name,
      description,
      image_url: '',
      variants: variants.map(
        (v): NewVariantInput => ({
          sku: v.sku,
          label: v.label,
          price_minor: v.price ? toMinor(v.price) : 0,
          stock_qty: Number(v.stock) || 0,
        }),
      ),
      // Languages left blank are dropped entirely — absent means "fall back
      // to English", whereas present-but-empty is a validation error.
      translations: translationPayload(translations),
    }
    create.mutate(payload, {
      onSuccess: () => {
        setName('')
        setSlug('')
        setDescription('')
        setVariants([{ ...emptyVariant }])
        setTranslations(emptyTranslationDraft({ name: '', description: '' }))
      },
    })
  }

  return (
    <form
      onSubmit={onSubmit}
      className="mt-6 space-y-3 rounded-xl border border-stone-200 bg-white p-5"
    >
      <h3 className="font-medium text-stone-700">New product</h3>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <Input placeholder="Name (English)" value={name} onChange={setName} error={fieldErr('name')} />
        <Input placeholder="slug (e.g. berry-jam)" value={slug} onChange={setSlug} error={fieldErr('slug')} />
        <div>
          <select
            value={categoryId}
            onChange={(e) => setCategoryId(e.target.value)}
            className="w-full rounded-lg border border-stone-300 px-3 py-2 text-sm focus:border-emerald-600 focus:outline-none"
          >
            <option value="">— category —</option>
            {categories.data?.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
          <FieldError message={fieldErr('category_id')} />
        </div>
      </div>
      <Input placeholder="Description (English)" value={description} onChange={setDescription} />

      <fieldset className="rounded-lg border border-stone-200 p-3">
        <legend className="px-1 text-xs font-medium text-stone-500">
          Translations — optional, blank falls back to English
        </legend>
        <div className="space-y-3">
          {PREFIXED_LOCALES.map((locale) => (
            <div key={locale} className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <Input
                placeholder={`Name (${localeLabel(locale)})`}
                value={translations[locale]?.name ?? ''}
                onChange={(v) =>
                  setTranslations({
                    ...translations,
                    [locale]: { ...translations[locale], name: v },
                  })
                }
                error={fieldErr(`translations.${locale}.name`)}
              />
              <Input
                placeholder={`Description (${localeLabel(locale)})`}
                value={translations[locale]?.description ?? ''}
                onChange={(v) =>
                  setTranslations({
                    ...translations,
                    [locale]: { ...translations[locale], description: v },
                  })
                }
              />
            </div>
          ))}
        </div>
      </fieldset>

      <div className="space-y-2">
        <p className="text-sm font-medium text-stone-600">Variants</p>
        <FieldError message={fieldErr('variants')} />
        {variants.map((v, i) => (
          <div key={i} className="grid grid-cols-2 gap-2 sm:grid-cols-5">
            <Input placeholder="SKU" value={v.sku} onChange={(x) => setVariant(i, { sku: x })} error={fieldErr(`variants[${i}].sku`)} />
            <Input placeholder="Label (250 g)" value={v.label} onChange={(x) => setVariant(i, { label: x })} error={fieldErr(`variants[${i}].label`)} />
            <Input placeholder="Price (1500.00)" value={v.price} onChange={(x) => setVariant(i, { price: x })} error={fieldErr(`variants[${i}].price_minor`)} />
            <Input placeholder="Stock" value={v.stock} onChange={(x) => setVariant(i, { stock: x })} error={fieldErr(`variants[${i}].stock_qty`)} />
            <button
              type="button"
              onClick={() => setVariants(variants.filter((_, idx) => idx !== i))}
              disabled={variants.length === 1}
              className="text-stone-300 hover:text-red-500 disabled:opacity-30"
              title="Remove variant"
            >
              ✕
            </button>
          </div>
        ))}
        <button
          type="button"
          onClick={() => setVariants([...variants, { ...emptyVariant }])}
          className="text-sm font-medium text-emerald-700 hover:underline"
        >
          + add variant
        </button>
      </div>

      {formError && (
        <p className="rounded-lg bg-red-50 p-3 text-sm text-red-600">{formError}</p>
      )}
      {create.isSuccess && (
        <p className="rounded-lg bg-emerald-50 p-3 text-sm text-emerald-700">Product created.</p>
      )}

      <button
        type="submit"
        disabled={create.isPending}
        className="rounded-lg bg-emerald-700 px-5 py-2 text-sm font-medium text-white hover:bg-emerald-800 disabled:opacity-50"
      >
        {create.isPending ? 'Creating…' : 'Create product'}
      </button>
    </form>
  )
}

// --- existing products: toggle + inline variant editing --------------------

function ProductList() {
  const products = useAdminProducts()

  if (products.isPending) return <p className="mt-6 text-stone-400">Loading…</p>
  if (products.isError) return <p className="mt-6 text-red-600">Failed to load products.</p>

  return (
    <div className="mt-6 space-y-4">
      {products.data.items.map((p) => (
        <ProductRow key={p.id} product={p} />
      ))}
    </div>
  )
}

function ProductRow({ product }: { product: AdminProduct }) {
  const update = useUpdateProduct()

  function toggleActive() {
    update.mutate({
      id: product.id,
      data: {
        category_id: product.category_id,
        name: product.name,
        description: product.description,
        image_url: product.image_url,
        is_active: !product.is_active,
      },
    })
  }

  return (
    <article
      className={`rounded-xl border border-stone-200 bg-white p-5 ${product.is_active ? '' : 'opacity-60'}`}
    >
      <div className="flex flex-wrap items-center gap-3">
        <ImageSlot product={product} />
        <h3 className="font-semibold text-stone-800">{product.name}</h3>
        <span className="text-xs text-stone-400">{product.slug}</span>
        {!product.is_active && (
          <span className="rounded-full bg-stone-200 px-2 py-0.5 text-xs text-stone-500">
            inactive
          </span>
        )}
        <button
          type="button"
          onClick={toggleActive}
          disabled={update.isPending}
          className="ml-auto text-sm font-medium text-emerald-700 hover:underline disabled:opacity-50"
        >
          {product.is_active ? 'deactivate' : 'activate'}
        </button>
      </div>

      <div className="mt-3 space-y-2">
        {product.variants.map((v) => (
          <VariantEditor key={v.id} variant={v} />
        ))}
      </div>

      {/* E3: the editorial half of the product page. Collapsed by default —
          the list is for scanning stock and prices, and six expanded content
          editors would bury that. */}
      <div className="mt-3">
        <ProductContentEditor product={product} />
      </div>
    </article>
  )
}

function VariantEditor({ variant }: { variant: ProductVariant }) {
  const update = useUpdateVariant()
  const [price, setPrice] = useState((variant.price_minor / 100).toFixed(2))
  const [stock, setStock] = useState(String(variant.stock_qty))

  const dirty =
    toMinor(price) !== variant.price_minor || Number(stock) !== variant.stock_qty

  return (
    <div className="flex flex-wrap items-center gap-3 rounded-lg bg-stone-50 px-3 py-2 text-sm">
      <span className="w-24 font-medium text-stone-700">{variant.label}</span>
      <span className="w-28 text-xs text-stone-400">{variant.sku}</span>
      <label className="flex items-center gap-1">
        <span className="text-xs text-stone-400">price</span>
        <input
          value={price}
          onChange={(e) => setPrice(e.target.value)}
          className="w-24 rounded border border-stone-300 px-2 py-1 text-right"
        />
      </label>
      <label className="flex items-center gap-1">
        <span className="text-xs text-stone-400">stock</span>
        <input
          value={stock}
          onChange={(e) => setStock(e.target.value)}
          className="w-16 rounded border border-stone-300 px-2 py-1 text-right"
        />
      </label>
      {dirty && (
        <button
          type="button"
          disabled={update.isPending}
          onClick={() =>
            update.mutate({ id: variant.id, priceMinor: toMinor(price), stockQty: Number(stock) || 0 })
          }
          className="rounded bg-emerald-700 px-3 py-1 text-xs font-medium text-white hover:bg-emerald-800 disabled:opacity-50"
        >
          save
        </button>
      )}
      {!dirty && (
        <span className="text-xs text-stone-300">{formatPrice(variant.price_minor)}</span>
      )}
    </div>
  )
}

// ImageSlot: thumbnail that doubles as the upload button — clicking it opens
// the file picker via a hidden <input type="file">.
function ImageSlot({ product }: { product: AdminProduct }) {
  const upload = useUploadProductImage()
  const inputRef = useRef<HTMLInputElement>(null)

  // Upload failures (too large, wrong type) are whole-request errors rather
  // than per-field ones, so only formError applies here.
  const { formError } = useFieldErrors(upload.error)

  return (
    <div className="flex flex-col items-center">
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        disabled={upload.isPending}
        title={product.image_url ? 'Replace photo' : 'Add photo'}
        className="h-14 w-14 overflow-hidden rounded-lg border border-dashed border-stone-300 bg-stone-50 text-xs text-stone-400 hover:border-emerald-600 disabled:opacity-50"
      >
        {product.image_url ? (
          <img src={product.image_url} alt={product.name} className="h-full w-full object-cover" />
        ) : upload.isPending ? (
          '…'
        ) : (
          '+ 📷'
        )}
      </button>
      <input
        ref={inputRef}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0]
          if (file) upload.mutate({ id: product.id, file })
          e.target.value = '' // same file can be re-picked later
        }}
      />
      {formError && (
        <span className="mt-1 max-w-24 text-center text-xs text-red-600">{formError}</span>
      )}
    </div>
  )
}

// --- small shared bits ------------------------------------------------------

function Input({
  placeholder,
  value,
  onChange,
  error,
}: {
  placeholder: string
  value: string
  onChange: (v: string) => void
  error?: string
}) {
  return (
    <div>
      <input
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-lg border border-stone-300 px-3 py-2 text-sm focus:border-emerald-600 focus:outline-none"
      />
      <FieldError message={error} />
    </div>
  )
}

function FieldError({ message }: { message?: string }) {
  if (!message) return null
  return <span className="mt-1 block text-xs text-red-600">{message}</span>
}

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-4xl px-4 py-8">{children}</div>
}
