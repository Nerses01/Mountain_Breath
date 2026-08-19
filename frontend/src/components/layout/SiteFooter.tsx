import { useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useCatalogFacets, useSubscribeNewsletter } from '../../api/hooks'
import { useFieldErrors } from '../../i18n/useFieldErrors'
import { useLocale } from '../../i18n/useLocale'
import { scrollToTopState } from '../../lib/scrollToTopState'
import { Button } from '../ui/Button'
import { CurrencySwitcher } from '../ui/CurrencySwitcher'
import { LanguageSwitcher } from '../ui/LanguageSwitcher'

/**
 * The design's four-column footer on bark, plus the bottom bar.
 *
 * Body-size text uses --color-ink-on-dark-soft rather than the design's
 * #a98a74: the mock sets that colour at 13px in the bottom bar, where it
 * measures 4.2:1 against the bark background and fails AA (token block in
 * src/index.css has the table).
 *
 * The language and currency switchers are BACK in the bottom bar (decision
 * #98, reversing #90): the settings screen that replaced them exists only
 * behind a sign-in, which left an anonymous visitor with no control at all
 * — no way to switch currency, and language only by hand-editing the URL.
 * That is #90's own "anonymous traffic suffers" reversal condition, met.
 * The two-controls-disagree worry #90 named cannot occur: both surfaces
 * read and write the same single sources (the URL for language, the
 * useCurrency context for currency).
 */
export function SiteFooter() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const year = new Date().getFullYear()

  // E1.5 hardcoded these as English literals and E2 found them still English
  // on the Russian home page — "no hardcoded string left in JSX" is easy to
  // believe about a file until someone reads it in another language.
  //
  // The Shop column is the real category list, so it translates itself and
  // each entry is a working filter link. Four of six, matching the mock's
  // column length; the shop page is the place that lists all of them.
  //
  // From the FACETS endpoint, not /categories, and E3 found out why by
  // looking at the rendered footer: /categories returns every row in the
  // table, including Era I's herbal-tea and coffee, which still exist
  // because deactivated products that old orders reference cannot be
  // deleted. The footer was linking to two filters that return nothing.
  // Facets already answers the narrower question the footer is actually
  // asking — "categories with something in them" — and the shop page has
  // usually cached it already.
  const facets = useCatalogFacets({})
  const shopLinks = (facets.data?.categories ?? []).slice(0, 4)

  // E9: the company column goes live. "Harvest log" is the design's own
  // name for what the nav calls the Journal — one page, the mock's two
  // labels, so both point at /journal rather than inventing a second page
  // to satisfy a synonym.
  const companyLinks = [
    { key: 'ourHive', to: '/our-hive' },
    { key: 'harvestLog', to: '/journal' },
    { key: 'shipping', to: '/shipping' },
    { key: 'contact', to: '/contact' },
  ] as const

  return (
    <footer className="mt-16 bg-bark">
      <div className="mx-auto flex max-w-360 flex-col gap-8 px-6 py-12 lg:px-14">
        <div className="grid gap-10 md:grid-cols-2 lg:grid-cols-[1.4fr_1fr_1fr_1.2fr]">
          <div className="flex flex-col gap-3">
            <span className="font-display text-lg font-extrabold tracking-wide text-ink-on-dark">
              {t('common:brand')}
            </span>
            <p className="max-w-xs text-sm leading-relaxed text-ink-on-dark-soft">
              {t('footer:blurb')}
            </p>
          </div>

          <FooterColumn title={t('footer:shop')}>
            {shopLinks.map((c) => (
              <li key={c.slug}>
                <Link
                  to={`${localePath('/shop')}?category=${c.slug}`}
                  // The reader is at the BOTTOM of a page asking for a fresh
                  // one — and from /shop this is a query-only change that
                  // ScrollToTop would otherwise leave put (the sidebar's
                  // filters rely on that). The state carries the intent.
                  state={scrollToTopState}
                  className="text-sm text-ink-on-dark-soft hover:text-ink-on-dark"
                >
                  {c.name}
                </Link>
              </li>
            ))}
          </FooterColumn>

          <FooterColumn title={t('footer:company')}>
            {companyLinks.map(({ key, to }) => (
              <li key={key}>
                <Link
                  to={localePath(to)}
                  // Also for the "already on that page" click, which changes
                  // no URL at all and would otherwise do nothing visible.
                  state={scrollToTopState}
                  className="text-sm text-ink-on-dark-soft transition hover:text-ink-on-dark"
                >
                  {t(`footer:companyLinks.${key}`)}
                </Link>
              </li>
            ))}
          </FooterColumn>

          <NewsletterSignup />
        </div>

        <div className="flex flex-wrap items-center justify-between gap-4 border-t border-bark-soft pt-5 text-xs text-ink-on-dark-soft">
          <p>{t('footer:legal.rights', { year })}</p>
          <div className="flex flex-wrap items-center gap-6">
            <Link
              to={localePath('/terms')}
              state={scrollToTopState}
              className="transition hover:text-ink-on-dark"
            >
              {t('footer:legal.terms')}
            </Link>
            <Link
              to={localePath('/privacy')}
              state={scrollToTopState}
              className="transition hover:text-ink-on-dark"
            >
              {t('footer:legal.privacy')}
            </Link>
            <CurrencySwitcher />
            <LanguageSwitcher />
          </div>
        </div>
      </div>
    </footer>
  )
}

/**
 * E9: the footer form goes live with double opt-in. The success copy is the
 * honest half-promise — "one click left" — because typing an address here
 * subscribes nobody; the emailed link does. The 204 arrives whatever the
 * address's history, so this component CANNOT know more than that, and its
 * copy does not pretend to.
 */
function NewsletterSignup() {
  const { t } = useTranslation()
  const [email, setEmail] = useState('')
  const subscribe = useSubscribeNewsletter()
  const errors = useFieldErrors(subscribe.error)

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!email.trim()) return
    subscribe.mutate(email)
  }

  return (
    <div className="flex flex-col gap-3">
      <h2 className="font-display text-sm font-bold uppercase tracking-label text-honey">
        {t('footer:newsletter.title')}
      </h2>
      <p className="text-sm leading-relaxed text-ink-on-dark-soft">
        {t('footer:newsletter.blurb')}
      </p>

      {subscribe.isSuccess ? (
        <p role="status" className="mt-1 rounded-2xl bg-honey/15 p-4 text-sm text-honey">
          {t('footer:newsletter.sent')}
        </p>
      ) : (
        <form className="mt-1 flex flex-col gap-2" onSubmit={onSubmit} noValidate>
          <div className="flex gap-2">
            <label htmlFor="newsletter-email" className="sr-only">
              {t('footer:newsletter.title')}
            </label>
            <input
              id="newsletter-email"
              type="email"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value)
                subscribe.reset()
              }}
              placeholder={t('footer:newsletter.placeholder')}
              aria-invalid={Boolean(errors.fieldError('email')) || undefined}
              className="min-w-0 flex-1 rounded-full bg-bark-soft px-4 py-3 text-sm text-ink-on-dark placeholder:text-ink-on-dark-soft"
            />
            <Button type="submit" variant="honey" disabled={subscribe.isPending}>
              {t('footer:newsletter.submit')}
            </Button>
          </div>
          {errors.fieldError('email') && (
            <p role="alert" className="text-xs text-danger">
              {errors.fieldError('email')}
            </p>
          )}
        </form>
      )}
    </div>
  )
}

function FooterColumn({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-3">
      <h2 className="font-display text-sm font-bold uppercase tracking-label text-honey">
        {title}
      </h2>
      <ul className="flex flex-col gap-3">{children}</ul>
    </div>
  )
}
