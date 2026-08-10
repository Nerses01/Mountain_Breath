import { expect, test } from '@playwright/test'

// The full customer journey in a real browser: register → shop → filter →
// product page → add to cart → checkout → order visible.
// Each run registers a fresh user, so the test is repeatable.
//
// Rewritten for E2. Three things moved under it, and every one of them is a
// change the unit tests could not have caught:
//   - `/` is the designed home page now; the product listing lives at /shop.
//   - the seed is the six hive products (decision #1) — Wild Thyme Tea, which
//     this test used to buy, no longer exists.
//   - the header's cart control is a pill reading "Cart · N", not "🧺 Cart".
test('a new customer can buy a product end to end', async ({ page }) => {
  const email = `e2e-${Date.now()}@test.local`

  // --- register ---
  await page.goto('/')
  await page.getByRole('link', { name: 'Sign in' }).click()
  await page.getByRole('button', { name: 'Create account' }).click()
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password').fill('e2e-password-123')
  // scope to the form: the mode tab above it has the same label
  await page.locator('form').getByRole('button', { name: 'Create account' }).click()

  // logged in: the header's account link is titled with our email
  await expect(page.getByTitle(email)).toBeVisible()

  // --- the faceted shop ---
  await page.getByRole('banner').getByRole('link', { name: 'Shop' }).click()
  await expect(page.getByRole('heading', { name: 'The whole shelf' })).toBeVisible()

  // Filtering is a URL change, not component state — which is exactly what
  // makes it assertable from outside the app.
  await page.getByRole('button', { name: /^Propolis/ }).click()
  await expect(page).toHaveURL(/category=propolis/)
  await expect(page.getByRole('link', { name: 'Raw Propolis Tincture' })).toBeVisible()

  // ...and the back button undoes it, which a useState implementation would
  // fail while passing every click-based test.
  await page.goBack()
  await expect(page).not.toHaveURL(/category=propolis/)

  // --- browse to a product (seeded with plenty of stock) ---
  await page.getByRole('link', { name: 'Mountain Wildflower Honey' }).click()
  await expect(
    page.getByRole('heading', { name: 'Mountain Wildflower Honey' }),
  ).toBeVisible()

  // --- add to cart ---
  await page.getByRole('button', { name: 'Add to cart' }).click()
  // the button flips to its "in cart" state — derived from the cart query
  await page.getByRole('link', { name: /In cart/ }).click()

  // --- cart & checkout ---
  await expect(page.getByRole('heading', { name: 'Your cart' })).toBeVisible()
  await expect(page.getByText('Mountain Wildflower Honey')).toBeVisible()
  await page.getByRole('button', { name: 'Checkout' }).click()

  // --- order confirmed ---
  await expect(page.getByRole('heading', { name: 'Your orders' })).toBeVisible()
  await expect(page.getByText(/Order #\d+/)).toBeVisible()
  await expect(page.getByText('Pending')).toBeVisible()
  await expect(page.getByText(/1 × Mountain Wildflower Honey/)).toBeVisible()

  // --- and the cart is empty again ---
  await page.getByRole('banner').getByRole('link', { name: /^Cart/ }).click()
  await expect(page.getByText('Your cart is empty.')).toBeVisible()
})
