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

  // --- cart: the summary card quotes what checkout will charge ---
  await expect(page.getByRole('heading', { name: 'Your cart' })).toBeVisible()
  await expect(page.getByText('Mountain Wildflower Honey').first()).toBeVisible()
  await expect(page.getByText('Subtotal')).toBeVisible()
  await page.getByRole('link', { name: 'Go to checkout' }).click()

  // --- checkout (E6): the address form under its own chrome ---
  await expect(
    page.getByRole('heading', { name: 'Where should the jars go?' }),
  ).toBeVisible()

  // An empty submit fails CLIENT-side: field errors appear, no navigation.
  await page.getByRole('button', { name: 'Place the order' }).click()
  await expect(page.getByRole('alert').first()).toBeVisible()
  await expect(page).toHaveURL(/checkout/)

  await page.getByLabel('First name').fill('Anahit')
  await page.getByLabel('Last name').fill('Sargsyan')
  await page.getByLabel('Phone').fill('+374 91 000000')
  await page.getByLabel('Street and number').fill('14 Abovyan St, apt 6')
  await page.getByLabel('City').fill('Yerevan')
  await page.getByLabel('Postal code').fill('0009')
  await page.getByRole('radio', { name: /Bank transfer/ }).click()
  await page.getByRole('button', { name: 'Place the order' }).click()

  // --- confirmation: the receipt with the snapshot ---
  await expect(page.getByText('Your order is in.')).toBeVisible()
  await expect(page.getByRole('heading', { name: /Order #\d+/ })).toBeVisible()
  await expect(page.getByText('14 Abovyan St, apt 6')).toBeVisible()
  await expect(page.getByText('Bank transfer')).toBeVisible()
  await expect(page.getByText('Payment pending')).toBeVisible()
  // The breakdown balances on screen: subtotal + shipping = total, and the
  // "Includes VAT" line is display-only, contained in the subtotal.
  await expect(page.getByText('Includes VAT')).toBeVisible()

  // --- it appears in the order history too ---
  await page.getByRole('link', { name: '← All your orders' }).click()
  await expect(page.getByRole('heading', { name: 'Your orders' })).toBeVisible()
  await expect(page.getByText(/1 × Mountain Wildflower Honey/)).toBeVisible()

  // --- and the cart is empty again ---
  await page.getByRole('banner').getByRole('link', { name: /^Cart/ }).click()
  await expect(page.getByText('Your cart is empty.')).toBeVisible()
})
