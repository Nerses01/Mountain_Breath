import { useTranslation } from 'react-i18next'
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
  const year = new Date().getFullYear()

  const shopLinks = ['Honey', 'Beeswax', 'Propolis', 'Royal jelly']
  const companyLinks = ['Our hive', 'Harvest log', 'Shipping', 'Contact']

  return (
    <footer className="mt-16 bg-bark">
      <div className="mx-auto flex max-w-[1440px] flex-col gap-8 px-6 py-12 lg:px-14">
        <div className="grid gap-10 md:grid-cols-2 lg:grid-cols-[1.4fr_1fr_1fr_1.2fr]">
          <div className="flex flex-col gap-3">
            <span className="font-display text-lg font-extrabold tracking-wide text-ink-on-dark">
              {t('common:brand')}
            </span>
            <p className="max-w-xs text-sm leading-relaxed text-ink-on-dark-soft">
              {t('footer:blurb')}
            </p>
          </div>

          <FooterColumn title={t('footer:shop')} items={shopLinks} />
          <FooterColumn title={t('footer:company')} items={companyLinks} />

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

function FooterColumn({ title, items }: { title: string; items: string[] }) {
  return (
    <div className="flex flex-col gap-3">
      <h2 className="font-display text-sm font-bold uppercase tracking-label text-honey">
        {title}
      </h2>
      <ul className="flex flex-col gap-3">
        {items.map((item) => (
          <li key={item} className="text-sm text-ink-on-dark-soft">
            {item}
          </li>
        ))}
      </ul>
    </div>
  )
}
