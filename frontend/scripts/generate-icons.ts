/**
 * Regenerates every icon file under public/ from scripts/icons.ts:
 *
 *   cd frontend && node scripts/generate-icons.ts
 *
 * No build step: Node 24 strips the type annotations itself, and the
 * tsconfig's `erasableSyntaxOnly` is the guarantee that these files only use
 * syntax that CAN be stripped (no enums, no parameter properties).
 * Playwright's Chromium — installed anyway for the e2e suite — does the
 * rasterising, so there is no image library to add, and the PNGs come from
 * the same engine that will display the SVG.
 *
 * What gets written, and who reads it (index.html wires them up):
 *   icon.svg               vector, rounded, transparent corners — Chrome, Firefox
 *   favicon.ico            16 + 32 + 48 px — Safari, older browsers, Google's
 *                          crawler, and every client that probes /favicon.ico
 *   apple-touch-icon.png   180 px, FULL BLEED — iOS clips its own corner radius
 *                          and composites transparency over black
 *   icon-192.png           rounded — Android home screen and install prompts,
 *   icon-512.png           via manifest.webmanifest
 *
 * The outputs are committed: the generator runs when the logo changes, not
 * on every build, and icons.test.ts pins the committed files to this source.
 */
import { writeFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { chromium } from '@playwright/test'
import { encodeIco, iconSvg } from './icons.ts'

const PUBLIC = new URL('../public/', import.meta.url)

const browser = await chromium.launch()

/** Rasterise an SVG at `size` px on a transparent background. */
async function rasterise(svg: string, size: number): Promise<Uint8Array> {
  const page = await browser.newPage({
    viewport: { width: size, height: size },
    deviceScaleFactor: 1,
  })
  const src = `data:image/svg+xml;base64,${Buffer.from(svg).toString('base64')}`
  // 'load' fires once the image has decoded, and the image is all there is.
  await page.setContent(
    `<!doctype html><style>body{margin:0}img{display:block;width:${size}px;height:${size}px}</style><img src="${src}">`,
    { waitUntil: 'load' },
  )
  const png = await page.screenshot({ omitBackground: true })
  await page.close()
  return png
}

const rounded = iconSvg()
const fullBleed = iconSvg({ radius: 0 })

await writeFile(new URL('icon.svg', PUBLIC), rounded)

const entries: { size: number; png: Uint8Array }[] = []
for (const size of [16, 32, 48]) {
  entries.push({ size, png: await rasterise(rounded, size) })
}
await writeFile(new URL('favicon.ico', PUBLIC), encodeIco(entries))

await writeFile(new URL('apple-touch-icon.png', PUBLIC), await rasterise(fullBleed, 180))
await writeFile(new URL('icon-192.png', PUBLIC), await rasterise(rounded, 192))
await writeFile(new URL('icon-512.png', PUBLIC), await rasterise(rounded, 512))

await browser.close()
console.log(`icons written to ${fileURLToPath(PUBLIC)}`)
