import { useTranslation } from 'react-i18next'
import { ApiError } from '../api/client'

/** Mirrors backend/internal/domain.PasswordMinLength. */
const PASSWORD_MIN_LENGTH = 8

/**
 * Turns the API's `fields` envelope into readable sentences.
 *
 * The backend answers validation failures with CODES ("slug_format"), never
 * prose, so one 400 response can render in any language. This hook is the
 * one place that maps a code to a sentence — every form calls it rather than
 * printing `err.fields[x]` raw, which would show a reader "slug_format".
 *
 * Unknown codes fall back to a generic sentence instead of leaking the
 * identifier: a backend that adds a code before this catalogue learns it
 * should look imprecise, not broken.
 */
export function useFieldErrors(error: unknown) {
  const { t } = useTranslation()
  const apiError = error instanceof ApiError ? error : null

  /** The message for one field, or undefined if that field is fine. */
  function fieldError(name: string): string | undefined {
    const code = apiError?.fields?.[name]
    if (!code) return undefined

    // i18next returns the fallback when the key is missing, so an unknown
    // code lands on the generic sentence rather than rendering itself.
    return t([`validation:${code}`, 'validation:unknown'], {
      min: PASSWORD_MIN_LENGTH,
    })
  }

  return {
    /** True when the request failed for a reason not tied to one field. */
    hasFormError: Boolean(apiError && !apiError.fields),
    formError: apiError && !apiError.fields ? apiError.message : undefined,
    fieldError,
  }
}
