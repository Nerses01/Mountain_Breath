import { useLocale } from '../i18n/useLocale'
import { pageMarkdown, type PageSlug } from '../content'
import { Markdown } from '../components/Markdown'

/**
 * One component, six pages (E9): Our hive, Benefits, Shipping, Contact,
 * Terms, Privacy — the route decides the slug, the locale decides the
 * file, the markdown decides everything else. The design has no mock for
 * any of these (the canvas draws six storefront screens), so the layout is
 * ours: the prose styles on the panel background, a comfortable column.
 */
export function ContentPage({ slug }: { slug: PageSlug }) {
  const { locale } = useLocale()
  // The slug union makes a missing file a build-time impossibility for
  // English; other locales fall back per file inside pageMarkdown.
  const markdown = pageMarkdown(slug, locale) ?? ''

  return (
    <div className="mx-auto max-w-360 px-6 py-12 lg:px-14">
      <Markdown markdown={markdown} />
    </div>
  )
}
