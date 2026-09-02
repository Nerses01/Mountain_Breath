import { expect, test } from '@playwright/test'

// The E8 account journey in a real browser: register → heart a product →
// wishlist shows it → save a cart line for later → the address book.
//
// The password-reset flow is deliberately NOT here: its middle step is an
// email, and the e2e stack has no mailbox to read (Mailpit runs in dev but
// not in CI). The flow is covered end to end at the store layer (token
// single-use, expiry, session revocation) and at the handler layer (the
// mail's link carries the stored token) — what a browser test would add is
// the one hop those already prove between them.
test('hearts, save-for-later and the address book survive a session', async ({ page }) => {
  const email = `e2e-account-${Date.now()}@test.local`

  // --- register ---
  await page.goto('/login')
  await page.getByRole('button', { name: 'Create an account' }).click()
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password').fill('e2e-password-123')
  await page.getByRole('button', { name: 'Create an account', exact: true }).last().click()
  await expect(page.getByTitle(email)).toBeVisible()

  // --- heart a product from its page ---
  await page.goto('/products/mountain-wildflower-honey')
  await expect(
    page.getByRole('heading', { name: 'Mountain Wildflower Honey' }),
  ).toBeVisible()
  // The buy-box heart is the first on the page (related cards carry their
  // own further down).
  const heart = page.getByRole('button', { name: 'Wishlist' }).first()
  await heart.click()
  await expect(heart).toHaveAttribute('aria-pressed', 'true')

  // ...and put a jar in the cart for the save-for-later step. Five
  // add-buttons render here (buy box + four related-card quick-adds); the
  // buy box is the only one that quotes the price in its label.
  await page.getByRole('button', { name: /^Add to cart — / }).click()
  await expect(page.getByRole('link', { name: /In cart/ })).toBeVisible()

  // --- the wishlist page lists the saved card ---
  // (A3: the page heading is "Wishlist" per the canvas; the role+level keeps
  // this from matching the header's Wishlist link or the heart buttons.)
  await page.getByRole('banner').getByRole('link', { name: 'Wishlist' }).click()
  await expect(page.getByRole('heading', { name: 'Wishlist', level: 1 })).toBeVisible()
  await expect(
    page.getByRole('link', { name: 'Mountain Wildflower Honey' }),
  ).toBeVisible()

  // --- save-for-later moves the cart line out ---
  await page.getByRole('banner').getByRole('link', { name: /^Cart/ }).click()
  await page.getByRole('button', { name: 'Save for later' }).click()
  await expect(page.getByText('Your cart is empty.')).toBeVisible()

  // --- the address book ---
  // A1 replaced the header's "Account" link with the menu-button pill (the
  // user's name, title = email); "Addresses" is one of its menuitems and
  // the page it opens is the canvas's "Address book".
  await page.getByTitle(email).click()
  await page.getByRole('menuitem', { name: 'Addresses' }).click()
  await expect(page.getByRole('heading', { name: 'Addresses', level: 1 })).toBeVisible()
  await page.getByRole('button', { name: 'Add an address' }).click()
  await page.getByLabel('Label').fill('Home')
  await page.getByLabel('First name').fill('Anahit')
  await page.getByLabel('Last name').fill('Sargsyan')
  await page.getByLabel('Phone').fill('+374 91 000000')
  await page.getByLabel('Street and number').fill('14 Abovyan St, apt 6')
  await page.getByLabel('City').fill('Yerevan')
  await page.getByLabel('Postal code').fill('0009')
  await page.getByRole('button', { name: 'Save address' }).click()

  // The first entry became the default whatever the checkbox said — the
  // checkout's prefill needs one to stand on. Scoped to main: the header's
  // "Home" nav link would otherwise also match the label text.
  await expect(page.getByRole('main').getByText('Home')).toBeVisible()
  // exact: the page subtitle also contains the word ("The default one…").
  await expect(page.getByText('Default', { exact: true })).toBeVisible()
  await expect(page.getByText('14 Abovyan St, apt 6')).toBeVisible()
})
