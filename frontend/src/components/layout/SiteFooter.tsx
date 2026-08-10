import type { ReactNode } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useCategories } from '../../api/hooks'
import { useLocale } from '../../i18n/useLocale'
import { Button } from '../ui/Button'
import { LanguageSwitcher } from '../ui/LanguageSwitcher'

/**
 * The design's four-column footer on bark, plus the bottom bar.
 *
 * Body-size text uses --color-ink-on-dark-soft rather than the design's
 * #a98a74: the mock sets that colour at 13px in the bottom bar, where it
 * measures 4.2:1 against the bark background and fails AA (token block in
 * src/index.css has the table).
 *
 * The language switcher sits in the bottom bar beside the slot E5's currency
 * switcher will take — following the design's own habit of putting
 * locale-shaped controls there ("USD / AMD") rather than inventing a header
 * position the mock gives no guidance for.
 */
export function SiteFooter() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const year = new Date().getFullYear()

  // E1.5 hardcoded these as English literals and E2 found them still English
  // on the Russian home page — "no hardcoded string left in JSX" is easy to
  // believe about a file until someone reads it in another language.
  //
  // The Shop column is now the real category list, so it translates itself
  // and each entry is a working filter link. Four of six, matching the mock's
  // column length; the shop page is the place that lists all of them.
  const categories = useCategories()
  const shopLinks = (categories.data ?? []).slice(0, 4)

  // Company links stay plain text until E9 builds those pages — but their
  // LABELS are translatable either way.
  const companyLinks = ['ourHive', 'harvestLog', 'shipping', 'contact'] as const

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
              <li key={c.id}>
                <Link
                  to={`${localePath('/shop')}?category=${c.slug}`}
                  className="text-sm text-ink-on-dark-soft hover:text-ink-on-dark"
                >
                  {c.name}
                </Link>
              </li>
            ))}
          </FooterColumn>

          <FooterColumn title={t('footer:company')}>
            {companyLinks.map((key) => (
              <li key={key} className="text-sm text-ink-on-dark-soft">
                {t(`footer:companyLinks.${key}`)}
              </li>
            ))}
          </FooterColumn>

          <div className="flex flex-col gap-3">
            <h2 className="font-display text-sm font-bold uppercase tracking-label text-honey">
              {t('footer:newsletter.title')}
            </h2>
            <p className="text-sm leading-relaxed text-ink-on-dark-soft">
              {t('footer:newsletter.blurb')}
            </p>
            {/* Inert until E9 wires double opt-in; kept as a real <form> so
                the markup does not have to be rebuilt then. */}
            <form
              className="mt-1 flex gap-2"
              onSubmit={(e) => e.preventDefault()}
            >
              <label htmlFor="newsletter-email" className="sr-only">
                {t('footer:newsletter.title')}
              </label>
              <input
                id="newsletter-email"
                type="email"
                placeholder={t('footer:newsletter.placeholder')}
                className="min-w-0 flex-1 rounded-full bg-bark-soft px-4 py-3 text-sm text-ink-on-dark placeholder:text-ink-on-dark-soft"
              />
              <Button type="submit" variant="honey">
                {t('footer:newsletter.submit')}
              </Button>
            </form>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-4 border-t border-bark-soft pt-5 text-xs text-ink-on-dark-soft">
          <p>{t('footer:legal.rights', { year })}</p>
          <div className="flex flex-wrap items-center gap-6">
            <span>{t('footer:legal.terms')}</span>
            <span>{t('footer:legal.privacy')}</span>
            <LanguageSwitcher />
          </div>
        </div>
      </div>
    </footer>
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
