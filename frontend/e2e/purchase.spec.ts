import { expect, test } from '@playwright/test'

// The full customer journey in a real browser: register → browse →
// product page → add to cart → checkout → order visible.
// Each run registers a fresh user, so the test is repeatable.
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

  // logged in: the header shows our email
  await expect(page.getByText(email)).toBeVisible()

  // --- browse to a product (seeded with plenty of stock) ---
  await page.getByRole('link', { name: /Wild Thyme Tea/ }).click()
  await expect(page.getByRole('heading', { name: 'Wild Thyme Tea' })).toBeVisible()

  // --- add to cart ---
  await page.getByRole('button', { name: 'Add to cart' }).click()
  // the button flips to its "in cart" state — derived from the cart query
  await page.getByRole('link', { name: /In cart/ }).click()

  // --- cart & checkout ---
  await expect(page.getByRole('heading', { name: 'Your cart' })).toBeVisible()
  await expect(page.getByText('Wild Thyme Tea')).toBeVisible()
  await page.getByRole('button', { name: 'Checkout' }).click()

  // --- order confirmed ---
  await expect(page.getByRole('heading', { name: 'Your orders' })).toBeVisible()
  await expect(page.getByText(/Order #\d+/)).toBeVisible()
  await expect(page.getByText('pending')).toBeVisible()
  await expect(page.getByText(/1 × Wild Thyme Tea/)).toBeVisible()

  // --- and the cart is empty again ---
  await page.getByRole('link', { name: '🧺 Cart' }).click()
  await expect(page.getByText('Empty.')).toBeVisible()
})
