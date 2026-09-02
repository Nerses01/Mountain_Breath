import { describe, expect, it } from 'vitest'
import { emptyTranslationDraft, translationPayload } from './translations'

describe('emptyTranslationDraft', () => {
  it('covers every translatable language but not the default', () => {
    const draft = emptyTranslationDraft({ name: '' })

    // English lives in the parent `name` field, so it must not appear here —
    // the API rejects an "en" key outright.
    expect(Object.keys(draft).sort()).toEqual(['hy', 'ru'])
    expect(draft).not.toHaveProperty('en')
  })

  it('gives each language its own object', () => {
    const draft = emptyTranslationDraft({ name: '' })
    draft.hy.name = 'Մեղր'

    // A shared reference would make typing in one language echo into the
    // other, since the seed object is spread per locale.
    expect(draft.ru.name).toBe('')
  })
})

describe('translationPayload', () => {
  it('omits languages the user left blank', () => {
    const payload = translationPayload({
      hy: { name: 'Մեղր' },
      ru: { name: '' },
    })

    // The distinction this whole function exists for: ABSENT means "fall back
    // to English", while PRESENT-but-empty is a validation error that would
    // block the form.
    expect(payload).toEqual({ hy: { name: 'Մեղր' } })
    expect(payload).not.toHaveProperty('ru')
  })

  it('treats whitespace as blank', () => {
    expect(translationPayload({ hy: { name: '   ' }, ru: { name: '' } })).toBeUndefined()
  })

  it('returns undefined when nothing is filled in', () => {
    // undefined so the key is omitted from the JSON body entirely, rather
    // than sent as an empty object.
    expect(translationPayload({ hy: { name: '' }, ru: { name: '' } })).toBeUndefined()
  })

  it('drops a description with no name', () => {
    const payload = translationPayload({
      hy: { name: '', description: 'Նկարագրություն' },
      ru: { name: 'Мёд', description: '' },
    })

    // A description without a name is not a usable translation and the API
    // would reject it — dropping it here keeps the rest of the form working.
    expect(payload).toEqual({ ru: { name: 'Мёд', description: '' } })
  })

  it('keeps a filled name even when the description is empty', () => {
    const payload = translationPayload({ ru: { name: 'Мёд', description: '' } })

    expect(payload).toEqual({ ru: { name: 'Мёд', description: '' } })
  })
})
