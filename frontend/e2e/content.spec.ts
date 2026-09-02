import { expect, test } from '@playwright/test'

// E9's definition of done, verbatim: "all five header links and every
// footer link resolve". This test clicks them all in a real browser — a
// route table can typecheck and still not mount, which is exactly what a
// unit test cannot see.
//
// The newsletter's other half ("a subscription requires confirming the
// email") is out of e2e reach for the same reason the password reset is:
// the middle step is an inbox, and CI has no Mailpit. The store lifecycle
// test owns the rule; the live Mailpit walk in the plan's verification
// note proved the hop between.
test('every header and footer link resolves to a real page', async ({ page }) => {
  await page.goto('/')

  // --- the five header destinations ---
  const nav = page.getByRole('banner')
  for (const [name, heading] of [
    ['Shop', 'The whole shelf'],
    ['Our hive', 'Our hive'],
    ['Benefits', 'What the hive does for you'],
    ['Journal', 'The journal'],
  ] as const) {
    await nav.getByRole('link', { name, exact: true }).click()
    await expect(page.getByRole('heading', { level: 1, name: heading })).toBeVisible()
  }
  await nav.getByRole('link', { name: 'Home', exact: true }).click()
  await expect(page.getByRole('heading', { name: /the hive gives/ })).toBeVisible()

  // --- a journal post opens from its list ---
  await page.goto('/journal')
  await page.getByRole('link', { name: 'The linden weeks' }).click()
  await expect(
    page.getByRole('heading', { level: 1, name: 'The linden weeks' }),
  ).toBeVisible()

  // --- the footer's company and legal links ---
  const footer = page.getByRole('contentinfo')
  for (const [name, heading] of [
    ['Harvest log', 'The journal'],
    ['Shipping', 'Shipping'],
    ['Contact', 'Contact'],
    ['Terms', 'Terms of sale'],
    ['Privacy', 'Privacy'],
  ] as const) {
    await footer.getByRole('link', { name, exact: true }).click()
    await expect(page.getByRole('heading', { level: 1, name: heading })).toBeVisible()
  }

  // --- the content follows the language, not just the chrome ---
  await page.goto('/hy/our-hive')
  await expect(page.getByRole('heading', { level: 1, name: 'Մեր փեթակը' })).toBeVisible()

  // --- the footer form answers with the honest half-promise ---
  await page.goto('/')
  await page.getByLabel('Harvest notes').fill(`e2e-news-${Date.now()}@test.local`)
  await page.getByRole('button', { name: 'Join' }).click()
  await expect(page.getByText(/check your inbox/)).toBeVisible()
})

// Decision #99: while the cursor rests on a product card, its photos cycle.
// The seed gives mountain-wildflower-honey three distinct SVG swatches, so
// "the visible photo changed" is a real pixel-level claim here, not a hope.
// Headless desktop Chromium reports (hover: hover) and no reduced-motion
// preference, which is exactly the device class the slideshow runs on.
test('product cards cycle their photos under the cursor', async ({ page }) => {
  await page.goto('/shop')

  const card = page
    .locator('article')
    .filter({ hasText: 'Mountain Wildflower Honey' })
    .first()
  const visiblePhoto = card.locator('img:not(.invisible)').first()

  const heroSrc = await visiblePhoto.getAttribute('src')
  expect(heroSrc).toContain('data:image/svg')

  // At rest only the hero is mounted — a grid of cards must not download
  // every photo of every product up front.
  await expect(card.locator('img')).toHaveCount(1)

  // Hover the CARD, not the img: the stretched-link overlay is the actual
  // hit target over the photo (the reason the handlers live on the
  // article), and Playwright's actionability check knows it.
  await card.hover()
  // The hover mounts the whole stack…
  await expect(card.locator('img')).toHaveCount(3)
  // …and within the ~0.9s interval the visible photo is a DIFFERENT one.
  await expect
    .poll(async () => visiblePhoto.getAttribute('src'), { timeout: 3000 })
    .not.toBe(heroSrc)

  // Leaving snaps back to the hero, so the grid at rest is always the
  // shop's chosen photos.
  await page.mouse.move(0, 0)
  await expect(visiblePhoto).toHaveAttribute('src', heroSrc!)
})
