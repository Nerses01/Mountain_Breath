import { Link, useParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useLocale } from '../i18n/useLocale'
import { journalPost, journalPosts } from '../content'
import { Markdown } from '../components/Markdown'

/**
 * The journal (E9) — the design's "Harvest log", one page under two of the
 * mock's labels. The list is cards in the shop grid's spirit; a post is the
 * prose styles with its date. All of it ships in the bundle (decision #3),
 * so there is no loading state to design: content pages cannot be slow.
 */
export function JournalPage() {
  const { t } = useTranslation()
  const { locale, localePath } = useLocale()
  const posts = journalPosts(locale)

  return (
    <div className="mx-auto max-w-360 px-6 py-12 lg:px-14">
      <span className="font-display text-2xs font-bold uppercase tracking-eyebrow text-ink-faint">
        {t('journal:eyebrow')}
      </span>
      <h1 className="mt-2 font-display text-display-md font-extrabold text-ink">
        {t('journal:title')}
      </h1>
      <p className="mt-2 max-w-xl text-base text-ink-body">{t('journal:blurb')}</p>

      <div className="mt-8 grid gap-5 md:grid-cols-2 lg:grid-cols-3">
        {posts.map((post) => (
          <article key={post.slug} className="flex h-full flex-col gap-3 rounded-xl bg-card p-6">
            <time dateTime={post.date} className="text-xs text-ink-muted">
              {formatDate(post.date, locale)}
            </time>
            <h2 className="font-display text-lg font-bold text-ink">
              <Link
                to={localePath(`/journal/${post.slug}`)}
                className="hover:text-brand-ink"
              >
                {post.title}
              </Link>
            </h2>
            <p className="text-sm leading-relaxed text-ink-soft">{post.teaser}</p>
            <Link
              to={localePath(`/journal/${post.slug}`)}
              className="mt-auto pt-2 text-sm font-semibold text-brand-ink hover:underline"
            >
              {t('journal:readMore')}
            </Link>
          </article>
        ))}
      </div>
    </div>
  )
}

export function JournalPostPage() {
  const { t } = useTranslation()
  const { locale, localePath } = useLocale()
  const { slug = '' } = useParams()
  const post = journalPost(slug, locale)

  if (!post) {
    return (
      <div className="mx-auto max-w-360 px-6 py-12 lg:px-14">
        <p className="text-ink-body">{t('journal:notFound')}</p>
        <Link
          to={localePath('/journal')}
          className="mt-3 inline-block text-sm font-semibold text-brand-ink hover:underline"
        >
          {t('journal:backToList')}
        </Link>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-360 px-6 py-12 lg:px-14">
      <time dateTime={post.date} className="text-sm text-ink-muted">
        {formatDate(post.date, locale)}
      </time>
      <div className="mt-3">
        <Markdown markdown={`# ${post.title}\n\n${post.body}`} />
      </div>
      <Link
        to={localePath('/journal')}
        className="mt-8 inline-block text-sm font-semibold text-brand-ink hover:underline"
      >
        {t('journal:backToList')}
      </Link>
    </div>
  )
}

// Intl renders the date in the reader's language from the frontmatter's ISO
// string — the one piece of "content" the markdown does not own.
function formatDate(iso: string, locale: string): string {
  if (!iso) return ''
  return new Date(iso + 'T00:00:00').toLocaleDateString(locale, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}
