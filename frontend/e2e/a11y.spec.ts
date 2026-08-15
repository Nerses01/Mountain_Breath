import AxeBuilder from '@axe-core/playwright'
import { expect, test, type Page } from '@playwright/test'

// The axe half of E10's definition of done: these scans run in CI with the
// other journeys, so an accessibility regression FAILS THE BUILD — the
// difference between "we audited once" and "it stays audited".
//
// Scope: WCAG 2.1 A + AA rule tags, the storefront's key screens, in the
// states a visitor actually reaches. Violations print with their target
// selectors so a failure names the element, not just the rule.
async function expectNoViolations(page: Page) {
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
    .analyze()

  const summary = results.violations.map((v) => ({
    id: v.id,
    impact: v.impact,
    nodes: v.nodes.slice(0, 5).map((n) => n.target.join(' ')),
  }))
  expect(summary, JSON.stringify(summary, null, 2)).toEqual([])
}

test('home page has no WCAG A/AA violations', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: /the hive gives/ })).toBeVisible()
  await expectNoViolations(page)
})

test('shop page, with the facets loaded', async ({ page }) => {
  await page.goto('/shop')
  await expect(page.getByRole('link', { name: 'Mountain Wildflower Honey' })).toBeVisible()
  await expectNoViolations(page)
})

test('shop page with the mobile filter drawer open', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  await page.goto('/shop')
  await page.getByRole('button', { name: /Filters/ }).click()
  await expect(page.getByRole('dialog', { name: 'Filters' })).toBeVisible()
  await expectNoViolations(page)
})

test('product page, tabs and gallery included', async ({ page }) => {
  await page.goto('/products/mountain-wildflower-honey')
  await expect(
    page.getByRole('heading', { name: 'Mountain Wildflower Honey' }),
  ).toBeVisible()
  await expectNoViolations(page)
})

test('sign-in page, both provider buttons drawn', async ({ page }) => {
  await page.goto('/login')
  await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible()
  await expectNoViolations(page)
})

test('a content page and the journal', async ({ page }) => {
  await page.goto('/our-hive')
  await expect(page.getByRole('heading', { name: 'Our hive' })).toBeVisible()
  await expectNoViolations(page)

  await page.goto('/journal')
  await expect(page.getByRole('heading', { name: 'The journal' })).toBeVisible()
  await expectNoViolations(page)
})

test('an Armenian page carries the right lang and passes too', async ({ page }) => {
  await page.goto('/hy/shop')
  await expect(page.locator('html')).toHaveAttribute('lang', 'hy')
  await expect(page.getByRole('main').getByRole('article').first()).toBeVisible()
  await expectNoViolations(page)
})
