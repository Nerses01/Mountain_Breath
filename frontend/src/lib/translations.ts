import { PREFIXED_LOCALES, type Locale } from '../i18n/locales'

/**
 * A form's per-language draft: every non-default locale always has an entry
 * while editing, even when empty, so the inputs stay controlled.
 */
export type TranslationDraft<T> = Record<string, T>

/** Builds a blank draft — one empty entry per translatable language. */
export function emptyTranslationDraft<T>(empty: T): TranslationDraft<T> {
  return Object.fromEntries(PREFIXED_LOCALES.map((l) => [l, { ...empty }]))
}

/**
 * Turns a draft into the request payload, dropping any language the user
 * left entirely blank.
 *
 * This is the whole point of the function. The backend treats a PRESENT
 * translation with an empty name as a validation error (`required`) — sending
 * `{hy: {name: ""}}` would block the form — while an ABSENT language simply
 * falls back to English. "Left blank" and "explicitly empty" are different
 * statements, and only the form knows which one the user meant.
 *
 * Returns undefined when nothing survives, so the field is omitted from the
 * JSON body rather than sent as `{}`.
 */
export function translationPayload<T extends Record<string, string>>(
  draft: TranslationDraft<T>,
): Record<string, T> | undefined {
  const filled: Record<string, T> = {}

  for (const [locale, text] of Object.entries(draft)) {
    // A language counts as "filled in" only if its NAME is set. A description
    // without a name is not a usable translation, and the backend would
    // reject it — better to drop it here than to fail the whole submit.
    if (text.name?.trim()) {
      filled[locale] = text
    }
  }

  return Object.keys(filled).length > 0 ? filled : undefined
}

/** Human label for a language, for form headings. */
export function localeLabel(locale: Locale | string): string {
  return String(locale).toUpperCase()
}
