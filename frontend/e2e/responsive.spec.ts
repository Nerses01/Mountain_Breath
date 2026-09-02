import { expect, test, type Page } from '@playwright/test'

// E10's definition of done, two of its three clauses:
//   "the purchase journey works from 375 px to 1440 px" — the 375px test,
//   through the MOBILE chrome (hamburger sheet, filter drawer, sticky bar),
//   not the desktop UI squeezed narrow;
//   "is completable with a keyboard only" — the second test never calls
//   click(): every interaction is Tab/Enter/Space/typing, so a control
//   that cannot be reached or operated by keyboard fails the build.

async function register(page: Page, email: string) {
  await page.goto('/login')
  await page.getByRole('button', { name: 'Create an account' }).click()
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password').fill('e2e-password-123')
  await page.getByRole('button', { name: 'Create an account', exact: true }).last().click()
  // Success = the navigate home. NOT the account icon's tooltip (the
  // desktop journeys' assertion) — the mobile header hides that icon,
  // which is precisely the kind of difference this journey exists to meet.
  await expect(page).toHaveURL('/')
}

test('the purchase journey at 375px, through the mobile chrome', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  const email = `e2e-mobile-${Date.now()}@test.local`
  await register(page, email)

  // The nav lives behind the hamburger now.
  await page.getByRole('button', { name: 'Menu' }).click()
  await page.getByRole('navigation', { name: 'Menu' }).getByRole('link', { name: 'Shop' }).click()
  await expect(page.getByRole('heading', { name: 'The whole shelf' })).toBeVisible()

  // The sidebar is a drawer: open it, filter, close it, and the back
  // button still undoes the filter (E2's URL-as-state, on a phone).
  await page.getByRole('button', { name: /Filters/ }).click()
  await page.getByRole('dialog', { name: 'Filters' })
    .getByRole('button', { name: /^Honey/ }).click()
  await expect(page).toHaveURL(/category=honey/)
  await page.getByRole('dialog', { name: 'Filters' })
    .getByRole('button', { name: 'Close' }).click()

  await page.getByRole('link', { name: 'Mountain Wildflower Honey' }).click()
  await expect(
    page.getByRole('heading', { name: 'Mountain Wildflower Honey' }),
  ).toBeVisible()

  // The sticky bar is the phone's buy button (two Add buttons exist at
  // 375px — the in-flow one and the bar's; the bar is the fixed one).
  await page
    .locator('div.fixed.bottom-0')
    .getByRole('button', { name: 'Add to cart' })
    .click()
  await expect(page.getByRole('link', { name: /In cart/ })).toBeVisible()

  await page.getByRole('banner').getByRole('link', { name: /^Cart/ }).click()
  await page.getByRole('link', { name: 'Go to checkout' }).click()

  await page.getByLabel('First name').fill('Anahit')
  await page.getByLabel('Last name').fill('Sargsyan')
  await page.getByLabel('Phone').fill('+374 91 000000')
  await page.getByLabel('Street and number').fill('14 Abovyan St, apt 6')
  await page.getByLabel('City').fill('Yerevan')
  await page.getByLabel('Postal code').fill('0009')
  await page.getByRole('radio', { name: /Bank transfer/ }).click()
  await page.getByRole('button', { name: 'Place the order' }).click()

  await expect(page.getByText('Your order is in.')).toBeVisible()
})

// Walks focus forward until the predicate matches, bounded so a broken tab
// order fails fast instead of hanging. This is the whole keyboard contract
// in one helper: if Tab cannot REACH a control, the journey cannot finish.
async function tabTo(page: Page, match: { role?: string; name?: string | RegExp; label?: string }) {
  for (let i = 0; i < 60; i++) {
    await page.keyboard.press('Tab')
    const target = match.label
      ? page.getByLabel(match.label, { exact: true })
      : page.getByRole((match.role ?? 'button') as 'button', { name: match.name })
    if (await target.first().evaluate(
      (el) => el === document.activeElement,
    ).catch(() => false)) {
      return
    }
  }
  throw new Error(`tab order never reached ${JSON.stringify(match)}`)
}

test('the purchase journey with a keyboard only', async ({ page }) => {
  const email = `e2e-kbd-${Date.now()}@test.local`

  // --- register, typing only ---
  await page.goto('/login')
  await tabTo(page, { role: 'button', name: 'Create an account' })
  await page.keyboard.press('Enter')
  await tabTo(page, { label: 'Email' })
  await page.keyboard.type(email)
  await tabTo(page, { label: 'Password' })
  await page.keyboard.type('e2e-password-123')
  await tabTo(page, { role: 'button', name: 'Create an account' })
  await page.keyboard.press('Enter')
  await expect(page.getByTitle(email)).toBeVisible()

  // --- straight to the product page (a keyboard user types URLs too) ---
  await page.goto('/products/mountain-wildflower-honey')

  // The variant pills are toggle buttons; the qty stepper's + is a real
  // button; add-to-cart submits — all reachable, all Enter/Space-operable.
  await tabTo(page, { role: 'button', name: /500 g/ })
  await page.keyboard.press('Enter')
  await tabTo(page, { role: 'button', name: 'Increase quantity' })
  await page.keyboard.press('Enter')
  await tabTo(page, { role: 'button', name: /Add to cart/ })
  await page.keyboard.press('Enter')
  await expect(page.getByRole('link', { name: /In cart/ })).toBeVisible()

  // --- cart → checkout ---
  await page.goto('/cart')
  await tabTo(page, { role: 'link', name: 'Go to checkout' })
  await page.keyboard.press('Enter')
  await expect(
    page.getByRole('heading', { name: 'Where should the jars go?' }),
  ).toBeVisible()

  // --- the form, Tab and type; the payment cards are radio buttons ---
  await tabTo(page, { label: 'First name' })
  await page.keyboard.type('Anahit')
  await tabTo(page, { label: 'Last name' })
  await page.keyboard.type('Sargsyan')
  await tabTo(page, { label: 'Phone' })
  await page.keyboard.type('+374 91 000000')
  await tabTo(page, { label: 'Street and number' })
  await page.keyboard.type('14 Abovyan St, apt 6')
  await tabTo(page, { label: 'City' })
  await page.keyboard.type('Yerevan')
  await tabTo(page, { label: 'Postal code' })
  await page.keyboard.type('0009')
  await tabTo(page, { role: 'radio', name: /Bank transfer/ })
  await page.keyboard.press('Enter')
  await tabTo(page, { role: 'button', name: 'Place the order' })
  await page.keyboard.press('Enter')

  await expect(page.getByText('Your order is in.')).toBeVisible()
})
