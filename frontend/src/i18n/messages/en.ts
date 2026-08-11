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
      close: 'Close',
    },
    language: {
      label: 'Language',
      change: 'Change language',
    },
    currency: {
      label: 'Currency',
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
    one_primary_image: 'Choose exactly one main image',
    rating_range: 'Choose a rating from 1 to 5 stars',
    // The LIMIT lives in the backend, the SENTENCE lives here — which is why
    // the code carries no number.
    too_long: 'This is too long',
    invalid_status: 'Not a valid status',
    unknown: 'This value is not valid',
  },
  // Shared UI states. These belong in `common` rather than being repeated
  // per page: "Loading…" said five different ways in five files is five
  // things to translate and five chances to drift.
  state: {
    loading: 'Loading…',
    loadFailed: 'Something went wrong. Please try again.',
    empty: 'Nothing here yet.',
    signInRequired: 'Please <1>sign in</1> to continue.',
  },
  catalog: {
    searchPlaceholder: 'Search the shop… typos and prefixes welcome',
    resultCount_one: '{{count}} product',
    resultCount_other: '{{count}} products',
    resultsFor: '{{count}} results for “{{query}}”',
    outOfStock: 'Out of stock',
    stockLeft: '{{count}} left',
    addToCart: 'Add to cart',
    add: 'Add',
    all: 'All',
    back: '← Back to the catalogue',
    notFound: 'This product does not exist (anymore?).',
    size: 'Size',
    inStock: '{{count}} in stock',
    inCart: 'In cart: {{count}}',
    signInToBuy: 'Sign in to buy',
    adding: 'Adding…',

    // ---- Shop page (E2) ------------------------------------------------
    shopTitle: 'The whole shelf',
    shopBlurb:
      'Six products, one meadow. Everything here was taken from our own hives this season.',
    allProducts: 'All hive products',
    sortLabel: 'Sort products',
    sortPrefix: 'Sort:',
    sort: {
      // "Most loved" deliberately still means bought-most, not rated-highest
      // — see the note on DefaultProductSort in the Go domain layer.
      popular: 'Most loved',
      rating: 'Best rated',
      price_asc: 'Price, low to high',
      price_desc: 'Price, high to low',
      newest: 'Newest',
    },
    facet: {
      category: 'Category',
      benefit: 'Good for',
      price: 'Price',
    },
    clearFilters: 'Clear filters',
    noResults: 'Nothing on the shelf matches that.',
    noResultsHint: 'Try fewer filters, or a wider price range.',
    noResultsFor: 'Nothing found for “{{query}}”.',
    seeAllResults_one: 'See {{count}} result',
    seeAllResults_other: 'See all {{count}} results',
    pagination: 'Pages',
    nextPage: 'Next page',
    // "500 g · Energy" — the card's one-line summary.
    sizeAndBenefit: '{{size}} · {{benefit}}',
    priceFrom: 'from {{price}}',
    help: {
      title: 'Not sure where to start?',
      blurb: 'Tell us what you want to feel better about and we will point at one jar.',
      cta: 'Ask a beekeeper',
    },
    // Badge KEYS from products.badge (migration 000009). The database stores
    // the key and the wording lives here, so a badge reads correctly in all
    // three languages and adding a fourth needs no backend change.
    badge: {
      best_seller: 'Best seller',
      new: 'New',
      cold_chain: 'Cold chain',
      for_makers: 'For makers',
      immunity: 'Immunity',
      protein: 'Protein',
    },
  },
  // ---- Product page (E3) -----------------------------------------------
  // Everything a product page says that is NOT the product's own content.
  // The bullets, usage cards and notes come from the API, because the family
  // writes those; these are the labels around them.
  product: {
    whatItDoes: 'What it does',
    size: 'Size',
    addToCartWithPrice: 'Add to cart — {{price}}',
    related: 'Often taken together',
    gallery: {
      label: 'Product images',
      image: 'Image {{n}}',
    },
    meta: {
      harvest: 'Harvest',
      shipping: 'Shipping',
      lab: 'Lab report',
      batch: 'Batch {{batch}}',
    },
    tabs: {
      label: 'Product details',
      howToTakeIt: 'How to take it',
      storage: 'Storage',
      reviews: 'Reviews ({{count}})',
    },
    rating: {
      // The accessible name of the star row. A number read aloud is worth
      // more than five glyph names.
      outOf: '{{rating}} out of 5',
      outOfWithCount_one: '{{rating}} out of 5, {{count}} review',
      outOfWithCount_other: '{{rating}} out of 5, {{count}} reviews',
      count_one: '({{count}} review)',
      count_other: '({{count}} reviews)',
      none: 'No reviews yet',
    },
    reviews: {
      empty: 'No reviews yet. If you have tried it, yours would be the first.',
      formTitle: 'Write a review',
      yourRating: 'Your rating',
      title: 'Title',
      body: 'Your review',
      submit: 'Submit review',
      thanks: 'Thank you — your review will appear once we have read it.',
      notPurchased: 'You can review a product once it has been delivered to you.',
      pagination: 'Review pages',
    },
  },
  // ---- Home page (E2) --------------------------------------------------
  // Copy about the SHOP, not about the catalog: it is translated here rather
  // than stored in the database because nobody edits it from the admin and
  // it changes with the design, not with the stock.
  home: {
    hero: {
      eyebrow: 'Keep on buzzing',
      titleAccent: 'Everything',
      title: 'the hive gives',
      blurb:
        'Honey, beeswax, propolis, royal jelly, pollen and venom — harvested by our family on the high meadows, and nothing else added on the way to you.',
      primaryCta: 'Shop the hive',
      secondaryCta: 'Meet the beekeepers',
      imageSlot: 'hero image — honey jar + comb',
      stamp: { raw: 'Raw', unfiltered: 'unfiltered' },
    },
    stats: {
      altitude: { value: '1,400 m', label: 'Meadow altitude' },
      hives: { value: '210', label: 'Hives in the family' },
      generations: { value: '3 gen.', label: 'Of beekeeping' },
    },
    harvest: {
      title: 'How we harvest',
      blurb:
        'We take only the surplus comb, spin it cold, and jar it the same week. No heating above hive temperature, no filtering out the pollen, no blending across seasons.',
      link: 'Read the harvest log',
    },
    benefits: {
      title: 'What the hive',
      titleAccent: 'does for you',
      blurb:
        'Every jar and tincture on this shelf earns its place by what it does in the body. Here is the short version, with sources on each product page.',
      link: 'See all benefits',
      items: {
        energy: { lead: 'Honey and pollen for', emphasis: 'steady natural energy' },
        defense: { lead: 'Propolis for', emphasis: 'antimicrobial defense' },
        vitality: { lead: 'Royal jelly for', emphasis: 'vitality and skin' },
        balms: { lead: 'Beeswax for', emphasis: 'balms, creams and candles' },
      },
    },
    shelf: {
      eyebrow: 'The shelf',
      title: 'Six gifts of the hive',
      action: 'Shop all',
    },
    story: {
      eyebrow: 'Our family',
      title: 'My grandfather kept nine hives. We keep two hundred and ten.',
      blurb:
        'Same meadow, same slow method, a lot more jars. If you ever come up the valley in July, we will put you in a veil and let you lift a frame yourself.',
      link: 'Visit the apiary',
      imageSlot: 'photo — beekeeper holding a frame',
    },
  },
  cart: {
    title: 'Your cart',
    empty: 'Your cart is empty.',
    browse: 'Browse the catalogue',
    signInRequired: 'Please <1>sign in</1> to use the cart.',
    each: 'each',
    remove: 'Remove',
    increase: 'Increase quantity',
    decrease: 'Decrease quantity',
    total: 'Total',
    checkout: 'Checkout',
    placingOrder: 'Placing order…',
  },
  account: {
    orderNumber: 'Order #{{id}}',
    status: {
      pending: 'Pending',
      confirmed: 'Confirmed',
      shipped: 'Shipped',
      delivered: 'Delivered',
      cancelled: 'Cancelled',
    },
    ordersTitle: 'Your orders',
    noOrders: 'No orders yet.',
    signInRequired: 'Please <1>sign in</1> to see your orders.',
    signIn: 'Sign in',
    createAccount: 'Create account',
    email: 'Email',
    password: 'Password',
    admin: 'Admin',
    signOut: 'Sign out',
    working: 'Working…',
  },
  footer: {
    blurb:
      'A family apiary on the high meadows. Honey, beeswax, propolis, royal jelly, pollen and venom, harvested by hand.',
    shop: 'Shop',
    company: 'Company',
    // The Shop column is built from the category list, so it needs no keys.
    // These four are content pages E9 builds; the labels translate now.
    companyLinks: {
      ourHive: 'Our hive',
      harvestLog: 'Harvest log',
      shipping: 'Shipping',
      contact: 'Contact',
    },
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
