/**
 * Russian messages.
 *
 * Note the plural keys: Russian selects between _one (1 товар), _few
 * (2 товара) and _many (5 товаров) by rules that also depend on the tens
 * digit — 21 takes _one, 22 takes _few, 25 takes _many. Hand-rolled
 * `count === 1 ? a : b` string building cannot express that, which is the
 * concrete reason this project took an i18n dependency at all.
 *
 * ⚠️ Machine-assisted translation pending native review.
 */
export const ru = {
  common: {
    brand: 'Mountain Breath',
    tagline: 'Семейная пасека с 1974 года',
    nav: {
      home: 'Главная',
      shop: 'Магазин',
      ourHive: 'Наша пасека',
      benefits: 'Польза',
      journal: 'Журнал',
    },
    actions: {
      search: 'Поиск',
      wishlist: 'Избранное',
      cart: 'Корзина',
      account: 'Аккаунт',
      signIn: 'Войти',
      signOut: 'Выйти',
    },
    language: {
      label: 'Язык',
      change: 'Сменить язык',
    },
    itemCount_one: '{{count}} товар',
    itemCount_few: '{{count}} товара',
    itemCount_many: '{{count}} товаров',
    productCount_one: '{{count}} продукт',
    productCount_few: '{{count}} продукта',
    productCount_many: '{{count}} продуктов',
  },
  state: {
    loading: 'Загрузка…',
    loadFailed: 'Что-то пошло не так. Попробуйте ещё раз.',
    empty: 'Здесь пока ничего нет.',
    signInRequired: 'Пожалуйста, <1>войдите</1>, чтобы продолжить.',
  },
  catalog: {
    searchPlaceholder: 'Поиск по магазину…',
    resultCount_one: '{{count}} продукт',
    resultCount_few: '{{count}} продукта',
    resultCount_many: '{{count}} продуктов',
    resultsFor: 'Результатов по запросу «{{query}}»: {{count}}',
    outOfStock: 'Нет в наличии',
    stockLeft: 'Осталось {{count}}',
    addToCart: 'В корзину',
    all: 'Все',
    back: '← Назад в каталог',
    notFound: 'Такого продукта больше нет.',
    size: 'Размер',
    inStock: 'В наличии: {{count}}',
    inCart: 'В корзине: {{count}}',
    signInToBuy: 'Войдите, чтобы купить',
    adding: 'Добавляем…',
  },
  cart: {
    title: 'Ваша корзина',
    empty: 'Корзина пуста.',
    browse: 'Перейти в каталог',
    signInRequired: 'Пожалуйста, <1>войдите</1>, чтобы пользоваться корзиной.',
    each: 'за штуку',
    remove: 'Удалить',
    increase: 'Увеличить количество',
    decrease: 'Уменьшить количество',
    total: 'Итого',
    checkout: 'Оформить заказ',
    placingOrder: 'Оформляем заказ…',
  },
  account: {
    orderNumber: 'Заказ №{{id}}',
    status: {
      pending: 'В ожидании',
      confirmed: 'Подтверждён',
      shipped: 'Отправлен',
      delivered: 'Доставлен',
      cancelled: 'Отменён',
    },
    ordersTitle: 'Ваши заказы',
    noOrders: 'Заказов пока нет.',
    signInRequired: 'Пожалуйста, <1>войдите</1>, чтобы увидеть заказы.',
    signIn: 'Войти',
    createAccount: 'Создать аккаунт',
    email: 'Электронная почта',
    password: 'Пароль',
    admin: 'Админ',
    signOut: 'Выйти',
    working: 'Выполняется…',
  },
  validation: {
    required: 'Обязательное поле',
    slug_format:
      'Используйте строчные буквы, цифры и дефисы — например «wildflower-honey»',
    email_format: 'Введите корректный адрес электронной почты',
    password_too_short: 'Не менее {{min}} символов',
    positive: 'Должно быть больше нуля',
    not_negative: 'Не может быть отрицательным',
    variants_required: 'Добавьте хотя бы один размер',
    unknown: 'Некорректное значение',
  },
  footer: {
    blurb:
      'Семейная пасека на высокогорных лугах. Мёд, воск, прополис, маточное молочко, пыльца и яд — всё собрано вручную.',
    shop: 'Магазин',
    company: 'Компания',
    newsletter: {
      title: 'Заметки о сборе',
      blurb: 'Что цветёт и что мы разливаем по банкам. Раз в месяц.',
      placeholder: 'you@email.com',
      submit: 'Подписаться',
    },
    legal: {
      terms: 'Условия',
      privacy: 'Конфиденциальность',
      rights: '© {{year}} Пасека Mountain Breath',
    },
  },
}
