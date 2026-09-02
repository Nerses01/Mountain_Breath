import { beforeEach, describe, expect, it } from 'vitest'
import { renderHook } from '@testing-library/react'
import { ApiError } from '../api/client'
import { useFieldErrors } from './useFieldErrors'
import i18n from './index'

function errorWith(fields: Record<string, string>) {
  return new ApiError(400, 'validation_failed', 'one or more fields are invalid', fields)
}

describe('useFieldErrors', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en')
  })

  it('renders a validation code as a sentence', () => {
    const { result } = renderHook(() =>
      useFieldErrors(errorWith({ slug: 'slug_format' })),
    )

    // The API sends "slug_format"; a reader must never see that.
    expect(result.current.fieldError('slug')).toContain('lowercase')
    expect(result.current.fieldError('slug')).not.toBe('slug_format')
  })

  it('translates the same code per language', async () => {
    await i18n.changeLanguage('ru')
    const { result } = renderHook(() =>
      useFieldErrors(errorWith({ name: 'required' })),
    )

    // The whole point of codes: one 400 response, three languages.
    expect(result.current.fieldError('name')).toBe('Обязательное поле')
    await i18n.changeLanguage('en')
  })

  it('interpolates the password minimum', () => {
    const { result } = renderHook(() =>
      useFieldErrors(errorWith({ password: 'password_too_short' })),
    )

    expect(result.current.fieldError('password')).toBe('Use at least 8 characters')
  })

  it('falls back to a generic sentence for an unknown code', () => {
    const { result } = renderHook(() =>
      useFieldErrors(errorWith({ slug: 'code_added_after_this_catalogue' })),
    )

    // A backend that adds a code first should look imprecise, not broken —
    // the raw identifier must not reach the UI.
    expect(result.current.fieldError('slug')).toBe('This value is not valid')
  })

  it('resolves JSON-path field names', () => {
    const { result } = renderHook(() =>
      useFieldErrors(errorWith({ 'variants[0].price_minor': 'positive' })),
    )

    expect(result.current.fieldError('variants[0].price_minor')).toBe(
      'Must be greater than zero',
    )
  })

  it('returns undefined for fields that are fine', () => {
    const { result } = renderHook(() =>
      useFieldErrors(errorWith({ slug: 'required' })),
    )

    expect(result.current.fieldError('name')).toBeUndefined()
  })

  it('separates whole-form errors from field errors', () => {
    const conflict = new ApiError(409, 'slug_taken', 'a category with this slug already exists')
    const { result } = renderHook(() => useFieldErrors(conflict))

    expect(result.current.formError).toBe('a category with this slug already exists')
    expect(result.current.fieldError('slug')).toBeUndefined()
  })

  it('ignores non-API errors', () => {
    const { result } = renderHook(() => useFieldErrors(new Error('network down')))

    expect(result.current.formError).toBeUndefined()
    expect(result.current.fieldError('slug')).toBeUndefined()
  })
})
