/**
 * Armenian messages.
 *
 * NOT type-constrained to `Messages` on purpose: plural rules differ per
 * language, so the key SETS legitimately differ (Russian needs _few/_many,
 * English does not). i18next resolves any missing key against `en` via
 * fallbackLng, so a gap degrades to English rather than rendering blank.
 *
 * ⚠️ Machine-assisted translation pending native review — particularly the
 * apiary vocabulary, which is specialist and easy to get subtly wrong. The
 * brand name stays in Latin script, as brand names normally do.
 */
export const hy = {
  common: {
    brand: 'Mountain Breath',
    tagline: 'Ընտանեկան մեղվանոց 1974 թվականից',
    nav: {
      home: 'Գլխավոր',
      shop: 'Խանութ',
      ourHive: 'Մեր փեթակը',
      benefits: 'Օգուտները',
      journal: 'Օրագիր',
    },
    actions: {
      search: 'Որոնել',
      wishlist: 'Ցանկագիր',
      cart: 'Զամբյուղ',
      account: 'Հաշիվ',
      signIn: 'Մուտք',
      signOut: 'Ելք',
    },
    language: {
      label: 'Լեզու',
      change: 'Փոխել լեզուն',
    },
    itemCount_one: '{{count}} ապրանք',
    itemCount_other: '{{count}} ապրանք',
    productCount_one: '{{count}} ապրանք',
    productCount_other: '{{count}} ապրանք',
  },
  footer: {
    blurb:
      'Ընտանեկան մեղվանոց բարձր լեռների մարգագետիններում։ Մեղր, մեղրամոմ, պրոպոլիս, մեղվի կաթ, ծաղկափոշի և մեղվի թույն՝ ձեռքով հավաքված։',
    shop: 'Խանութ',
    company: 'Ընկերություն',
    newsletter: {
      title: 'Բերքի նշումներ',
      blurb: 'Ինչ է ծաղկում, ինչ ենք տարայավորում։ Ամիսը մեկ անգամ։',
      placeholder: 'you@email.com',
      submit: 'Միանալ',
    },
    legal: {
      terms: 'Պայմաններ',
      privacy: 'Գաղտնիություն',
      rights: '© {{year}} Mountain Breath մեղվանոց',
    },
  },
}
