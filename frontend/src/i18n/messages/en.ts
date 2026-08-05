/**
 * English messages — the reference locale.
 *
 * Namespaced per feature area rather than one flat file, so each future
 * phase adds its own section instead of everyone editing the same object
 * (docs/PLAN_ERA_2.md, E1.5). `common` is the app chrome; the rest arrive
 * with the phases that need them.
 *
 * Copy is taken from the design canvas verbatim where it exists.
 */
export const en = {
  common: {
    brand: 'Mountain Breath',
    tagline: 'Family apiary since 1974',
    nav: {
      home: 'Home',
      shop: 'Shop',
      ourHive: 'Our hive',
      benefits: 'Benefits',
      journal: 'Journal',
    },
    actions: {
      search: 'Search',
      wishlist: 'Wishlist',
      cart: 'Cart',
      account: 'Account',
      signIn: 'Sign in',
      signOut: 'Sign out',
    },
    language: {
      label: 'Language',
      change: 'Change language',
    },
    // ICU plural: i18next picks _one / _other from the count automatically,
    // which is why this is not string concatenation. Russian needs a third
    // form (_few) and Armenian's rules differ again — exactly the class of
    // bug the library exists to prevent.
    itemCount_one: '{{count}} item',
    itemCount_other: '{{count}} items',
    productCount_one: '{{count}} product',
    productCount_other: '{{count}} products',
  },
  // Validation codes from the API's `fields` envelope. The backend answers
  // with a code ("slug_format"), never a sentence, so the same 400 response
  // renders in whatever language the reader is using — see
  // backend/internal/domain/validation.go.
  //
  // `unknown` is the safety net: a code this catalogue has not learned yet
  // must still produce a sentence, not a raw identifier leaking into the UI.
  validation: {
    required: 'Required',
    slug_format: 'Use lowercase letters, digits and dashes — like "wildflower-honey"',
    email_format: 'Enter a valid email address',
    password_too_short: 'Use at least {{min}} characters',
    positive: 'Must be greater than zero',
    not_negative: 'Cannot be negative',
    variants_required: 'Add at least one size',
    unknown: 'This value is not valid',
  },
  footer: {
    blurb:
      'A family apiary on the high meadows. Honey, beeswax, propolis, royal jelly, pollen and venom, harvested by hand.',
    shop: 'Shop',
    company: 'Company',
    newsletter: {
      title: 'Harvest notes',
      blurb: 'What is flowering, what we are jarring. Once a month.',
      placeholder: 'you@email.com',
      submit: 'Join',
    },
    legal: {
      terms: 'Terms',
      privacy: 'Privacy',
      rights: '© {{year}} Mountain Breath Apiary',
    },
  },
} as const

export type Messages = typeof en
