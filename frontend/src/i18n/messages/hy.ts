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
  state: {
    loading: 'Բեռնվում է…',
    loadFailed: 'Սխալ տեղի ունեցավ։ Փորձեք կրկին։',
    empty: 'Այստեղ դեռ ոչինչ չկա։',
    signInRequired: 'Խնդրում ենք <1>մուտք գործել</1> շարունակելու համար։',
  },
  catalog: {
    searchPlaceholder: 'Որոնեք խանութում…',
    resultCount_one: '{{count}} ապրանք',
    resultCount_other: '{{count}} ապրանք',
    resultsFor: '{{count}} արդյունք «{{query}}» հարցման համար',
    outOfStock: 'Առկա չէ',
    stockLeft: 'Մնացել է {{count}}',
    addToCart: 'Ավելացնել զամբյուղ',
    all: 'Բոլորը',
    back: '← Վերադառնալ կատալոգ',
    notFound: 'Այս ապրանքը գոյություն չունի։',
    size: 'Չափս',
    inStock: 'Առկա է {{count}}',
    inCart: 'Զամբյուղում՝ {{count}}',
    signInToBuy: 'Մուտք գործեք գնելու համար',
    adding: 'Ավելացվում է…',
  },
  cart: {
    title: 'Ձեր զամբյուղը',
    empty: 'Զամբյուղը դատարկ է։',
    browse: 'Դիտել կատալոգը',
    signInRequired: 'Խնդրում ենք <1>մուտք գործել</1> զամբյուղն օգտագործելու համար։',
    each: 'հատը',
    remove: 'Հեռացնել',
    increase: 'Ավելացնել քանակը',
    decrease: 'Նվազեցնել քանակը',
    total: 'Ընդամենը',
    checkout: 'Պատվիրել',
    placingOrder: 'Պատվերը մշակվում է…',
  },
  account: {
    orderNumber: 'Պատվեր #{{id}}',
    status: {
      pending: 'Սպասման մեջ',
      confirmed: 'Հաստատված',
      shipped: 'Ուղարկված',
      delivered: 'Առաքված',
      cancelled: 'Չեղարկված',
    },
    ordersTitle: 'Ձեր պատվերները',
    noOrders: 'Դեռ պատվերներ չկան։',
    signInRequired: 'Խնդրում ենք <1>մուտք գործել</1> պատվերները տեսնելու համար։',
    signIn: 'Մուտք',
    createAccount: 'Ստեղծել հաշիվ',
    email: 'Էլ. հասցե',
    password: 'Գաղտնաբառ',
    admin: 'Ադմին',
    signOut: 'Ելք',
    working: 'Կատարվում է…',
  },
  validation: {
    required: 'Պարտադիր է',
    slug_format:
      'Օգտագործեք փոքրատառեր, թվեր և գծիկներ — օրինակ՝ «wildflower-honey»',
    email_format: 'Մուտքագրեք վավեր էլ. հասցե',
    password_too_short: 'Օգտագործեք առնվազն {{min}} նիշ',
    positive: 'Պետք է մեծ լինի զրոյից',
    not_negative: 'Չի կարող բացասական լինել',
    variants_required: 'Ավելացրեք առնվազն մեկ չափս',
    unknown: 'Այս արժեքը վավեր չէ',
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
