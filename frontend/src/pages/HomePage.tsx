import { useTranslation } from 'react-i18next'
import { useProducts, useQuickAdd } from '../api/hooks'
import { ProductCard } from '../components/ProductCard'
import {
  ArrowRightIcon,
  ButtonLink,
  Card,
  CheckIcon,
  SectionHeading,
  Stat,
} from '../components/ui'
import { useLocale } from '../i18n/useLocale'
import { usePageMeta } from '../lib/usePageMeta'

/**
 * The design's Home screen: hero with a stat strip, the "How we harvest" /
 * "What the hive does for you" band, six product cards, and the family story.
 *
 * The products come from the API — no hardcoded product copy anywhere on
 * this page, which is the phase's own requirement. Everything else IS copy
 * (the hero headline, the harvest story), and copy lives in the message
 * catalogues where it can be translated, not in the database: it describes
 * the shop rather than the catalog, and nobody edits it from the admin.
 */
export function HomePage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()

  // Six cards, most loved first — the same default the Shop page opens with,
  // so the two pages agree about what "the shelf" looks like.
  const products = useProducts({ perPage: 6, sort: 'popular' })

  // undefined while signed out, which the card renders as a disabled Add.
  const quickAdd = useQuickAdd()

  // The home page IS the brand — bare title, the hero's blurb as the
  // description a search result shows.
  usePageMeta({ description: t('home:hero.blurb') })

  return (
    <div className="mx-auto max-w-360 px-6 lg:px-14">
      {/* ── Hero ─────────────────────────────────────────────────────── */}
      <section className="grid items-center gap-10 py-12 lg:grid-cols-2 lg:py-14">
        <div className="flex flex-col gap-5.5">
          <p className="font-display text-sm font-bold uppercase tracking-eyebrow text-ink-soft">
            {t('home:hero.eyebrow')}
          </p>

          {/* The mock breaks this headline across two lines and colours the
              first word. <Trans> is not needed — there is no link inside, so
              two spans and a <br> read more plainly than a component map.

              The explicit {' '} is not a typo. A <br> contributes nothing to
              the accessible name, so without it a screen reader announces
              "Everythingthe hive gives" as one word. The space is invisible
              on screen (a trailing space before a line break collapses) and
              audible where it matters. */}
          {/* E10: the 68px hero size is a 1440px number — at 375px it fits
              four characters a line. Step down through the scale instead of
              scaling linearly, so each breakpoint uses a size the type
              system already tuned. */}
          <h1 className="font-display text-display-lg font-extrabold text-ink md:text-display-xl">
            <span className="text-brand-ink">{t('home:hero.titleAccent')}</span>{' '}
            <br />
            {t('home:hero.title')}
          </h1>

          <p className="max-w-120 text-lg leading-relaxed text-ink-body text-pretty">
            {t('home:hero.blurb')}
          </p>

          <div className="flex flex-wrap items-center gap-4 pt-1.5">
            <ButtonLink to={localePath('/shop')} size="lg">
              {t('home:hero.primaryCta')}
              <ArrowRightIcon className="ml-2.5 inline size-4" />
            </ButtonLink>
            {/* E10 audit find: E9 built /our-hive and this stayed disabled —
                the cost of "inert until the page exists" comments is that
                someone must remember them. Live now. */}
            <ButtonLink to={localePath('/our-hive')} variant="ghost" size="lg">
              {t('home:hero.secondaryCta')}
            </ButtonLink>
          </div>

          <div className="mt-2.5 flex flex-wrap gap-9 border-t border-line pt-4.5">
            <Stat value={t('home:stats.altitude.value')} label={t('home:stats.altitude.label')} />
            <Stat value={t('home:stats.hives.value')} label={t('home:stats.hives.label')} />
            <Stat
              value={t('home:stats.generations.value')}
              label={t('home:stats.generations.label')}
            />
          </div>
        </div>

        {/* The hero image slot. The mock draws a cut-out honey jar over a
            radial glow; with no photography yet, the glow and the frame keep
            the composition and the label says what belongs here. */}
        <div
          aria-hidden
          className="relative flex min-h-100 items-center justify-center lg:min-h-120"
        >
          <span className="absolute size-90 rounded-full bg-radial-[at_35%_30%] from-honey to-brand opacity-90" />
          {/* The mock cuts a photographed jar out over this glow. With no
              photography yet the slot has to read as a placeholder ON the
              glow, not vanish into it — hence the diagonal hatch (the mock's
              own placeholder texture) and a boxed label rather than a
              translucent panel, which the orange swallowed whole. */}
          {/* aspect-ratio, not h-full: a percentage height resolves against
              the parent's HEIGHT, and this parent only has a min-height, so
              h-full collapsed the frame to a thin band across the glow. An
              aspect ratio derives the height from the width instead — and
              the ratio is the mock's own 470×440 slot. */}
          <span
            className="relative flex aspect-47/44 w-full max-w-118 items-center justify-center rounded-2xl"
            style={{
              backgroundImage:
                'repeating-linear-gradient(135deg, rgb(255 255 255 / 0.55) 0 12px, rgb(255 255 255 / 0.25) 12px 24px)',
            }}
          >
            <span className="rounded-md bg-card/85 px-4 py-2 text-center font-mono text-xs uppercase tracking-label text-ink-muted">
              {t('home:hero.imageSlot')}
            </span>
          </span>
          <span className="absolute right-1.5 top-8 flex size-29 flex-col items-center justify-center gap-0.5 rounded-full border-[1.5px] border-dashed border-brand bg-card">
            <span className="font-display text-sm font-bold text-ink">
              {t('home:hero.stamp.raw')}
            </span>
            <span className="text-2xs tracking-label text-ink-muted">
              {t('home:hero.stamp.unfiltered')}
            </span>
            <span className="font-display text-base font-extrabold text-brand-ink">100%</span>
          </span>
        </div>
      </section>

      {/* ── Harvest + benefits band ──────────────────────────────────── */}
      <section className="grid gap-8 pb-14 lg:grid-cols-[340px_1fr]">
        <Card tone="bark" className="flex flex-col gap-4 p-8">
          <span
            aria-hidden
            className="flex size-13 items-center justify-center rounded-lg bg-honey/20 text-2xl"
          >
            🍯
          </span>
          <h2 className="font-display text-display-xs font-bold text-ink-on-dark">
            {t('home:harvest.title')}
          </h2>
          <p className="text-base leading-relaxed text-ink-on-dark-body">
            {t('home:harvest.blurb')}
          </p>
          <p className="mt-auto font-display text-base font-semibold text-honey">
            {t('home:harvest.link')} →
          </p>
        </Card>

        <Card className="grid gap-8 p-8 lg:grid-cols-2 lg:p-9">
          <div className="flex flex-col gap-3.5">
            <h2 className="font-display text-display-xs font-bold text-ink">
              {t('home:benefits.title')}{' '}
              <span className="text-brand-ink">{t('home:benefits.titleAccent')}</span>
            </h2>
            <p className="text-base leading-relaxed text-ink-body">
              {t('home:benefits.blurb')}
            </p>
            <p className="mt-auto font-display text-base font-semibold text-brand-ink">
              {t('home:benefits.link')} →
            </p>
          </div>

          <ul className="flex flex-col gap-3.5">
            {(['energy', 'defense', 'vitality', 'balms'] as const).map((key) => (
              <li key={key} className="flex items-start gap-3">
                <span
                  aria-hidden
                  className="mt-0.5 flex size-5.5 shrink-0 items-center justify-center rounded-full bg-honey text-ink"
                >
                  <CheckIcon className="size-3" />
                </span>
                <p className="text-base text-ink-strong">
                  {t(`home:benefits.items.${key}.lead`)}{' '}
                  <strong className="font-semibold text-ink">
                    {t(`home:benefits.items.${key}.emphasis`)}
                  </strong>
                </p>
              </li>
            ))}
          </ul>
        </Card>
      </section>

      {/* ── The shelf ────────────────────────────────────────────────── */}
      <section className="flex flex-col gap-6.5 pb-15">
        <SectionHeading
          eyebrow={t('home:shelf.eyebrow')}
          title={t('home:shelf.title')}
          action={{ label: t('home:shelf.action'), to: localePath('/shop') }}
        />

        {products.isError && (
          <p className="rounded-lg bg-card p-4 text-danger">
            {t('common:state.loadFailed')}
          </p>
        )}
        {products.isPending && <p className="text-ink-muted">{t('common:state.loading')}</p>}

        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 xl:grid-cols-3">
          {products.data?.items.map((p) => (
            <ProductCard key={p.id} product={p} layout="feature" onAdd={quickAdd} />
          ))}
        </div>
      </section>

      {/* ── Story band ───────────────────────────────────────────────── */}
      <section className="mb-14 grid items-center gap-10 rounded-2xl bg-honey p-6 md:p-11 lg:grid-cols-[1fr_380px]">
        <div className="flex flex-col gap-3.5">
          <p className="font-display text-xs font-bold uppercase tracking-eyebrow text-ink-strong">
            {t('home:story.eyebrow')}
          </p>
          <h2 className="font-display text-display-sm font-extrabold text-ink">
            {t('home:story.title')}
          </h2>
          <p className="max-w-140 text-base leading-relaxed text-ink-strong">
            {t('home:story.blurb')}
          </p>
          <p className="flex items-center gap-2.5 pt-1 font-display text-base font-semibold text-ink">
            {t('home:story.link')} <span aria-hidden>→</span>
          </p>
        </div>
        <span
          aria-hidden
          className="flex h-60 items-center justify-center rounded-lg bg-card/40 text-center font-mono text-2xs uppercase tracking-label text-ink-strong"
        >
          {t('home:story.imageSlot')}
        </span>
      </section>
    </div>
  )
}
