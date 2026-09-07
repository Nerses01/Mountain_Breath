/**
 * The icon files under public/ are generated (scripts/generate-icons.ts) and
 * committed. Pairs of artefacts that must agree — a generator and its
 * output, index.html and the files it points at, a manifest and the sizes
 * it declares — rot silently unless something checks them, so this does.
 * (The same idea as the Postman-collection rule, one directory over.)
 */
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { encodeIco, GLYPH, HONEY, iconSvg, INK, pngSize } from './icons.ts'

// Paths hang off the working directory, not import.meta.url: under Vitest's
// jsdom environment the latter is an http:// URL, and Vitest already takes
// the frontend directory (where vite.config.ts lives) as the root.
const FRONTEND = process.cwd()
const inPublic = (name: string) => join(FRONTEND, 'public', name)
const read = (name: string) => readFileSync(inPublic(name))

describe('site icons', () => {
  it('icon.svg on disk is exactly what the generator produces', () => {
    // core.autocrlf checks text files out with CRLF on Windows; compare content, not line endings.
    const onDisk = read('icon.svg').toString('utf8').replace(/\r\n/g, '\n')
    expect(onDisk).toBe(iconSvg())
  })

  it('draws with the brand tokens from index.css', () => {
    const css = readFileSync(join(FRONTEND, 'src', 'index.css'), 'utf8')
    expect(css).toContain(`--color-honey: ${HONEY};`)
    expect(css).toContain(`--color-ink: ${INK};`)
  })

  it('the M is mirror-symmetric about its centre line', () => {
    const key = ([x, y]: readonly [number, number]) => `${x},${y}`
    const corners = new Set(GLYPH.points.map(key))
    for (const [x, y] of GLYPH.points) {
      expect(corners.has(key([GLYPH.width - x, y])), `${x},${y} has no twin`).toBe(true)
    }
  })

  it('index.html and the manifest point at files that exist, at the sizes they declare', () => {
    const html = readFileSync(join(FRONTEND, 'index.html'), 'utf8')
    const hrefs = [
      ...html.matchAll(/<link rel="(?:icon|apple-touch-icon|manifest)" href="\/([^"]+)"/g),
    ].map((m) => m[1])
    expect(hrefs).toEqual(['favicon.ico', 'icon.svg', 'apple-touch-icon.png', 'manifest.webmanifest'])
    for (const href of hrefs) {
      expect(existsSync(inPublic(href)), `public/${href} missing`).toBe(true)
    }

    expect(pngSize(read('apple-touch-icon.png'))).toEqual({ width: 180, height: 180 })

    const manifest: { icons: { src: string; sizes: string }[] } = JSON.parse(
      read('manifest.webmanifest').toString('utf8'),
    )
    expect(manifest.icons.length).toBeGreaterThan(0)
    for (const icon of manifest.icons) {
      const { width, height } = pngSize(read(icon.src.replace(/^\//, '')))
      expect(`${width}x${height}`, icon.src).toBe(icon.sizes)
    }
  })

  it('favicon.ico holds 16, 32 and 48 px PNG entries whose directory matches the images', () => {
    const ico = read('favicon.ico')
    const view = new DataView(ico.buffer, ico.byteOffset, ico.byteLength)
    expect(view.getUint16(0, true)).toBe(0) // reserved
    expect(view.getUint16(2, true)).toBe(1) // icon, not cursor
    const count = view.getUint16(4, true)
    const sizes: number[] = []
    for (let i = 0; i < count; i++) {
      const entry = 6 + 16 * i
      const length = view.getUint32(entry + 8, true)
      const offset = view.getUint32(entry + 12, true)
      const { width, height } = pngSize(ico.subarray(offset, offset + length))
      expect([ico[entry], ico[entry + 1]]).toEqual([width, height])
      sizes.push(width)
    }
    expect(sizes).toEqual([16, 32, 48])
  })

  it('encodeIco lays out the directory per the spec and refuses a size mismatch', () => {
    // A PNG header is all pngSize reads, so a 24-byte stub stands in for an image.
    const stub = (size: number, fill: number) => {
      const png = new Uint8Array(24).fill(fill)
      const view = new DataView(png.buffer)
      view.setUint32(0, 0x89504e47)
      view.setUint32(12, 0x49484452)
      view.setUint32(16, size)
      view.setUint32(20, size)
      return png
    }
    const ico = encodeIco([
      { size: 16, png: stub(16, 0xaa) },
      { size: 256, png: stub(256, 0xbb) },
    ])
    const view = new DataView(ico.buffer)
    expect(ico.byteLength).toBe(6 + 2 * 16 + 2 * 24)
    expect(view.getUint16(4, true)).toBe(2)
    expect([ico[6], ico[7]]).toEqual([16, 16])
    expect([ico[22], ico[23]]).toEqual([0, 0]) // 256 does not fit a byte: 0 by convention
    expect(view.getUint16(6 + 6, true)).toBe(32) // bits per pixel
    expect(view.getUint32(6 + 8, true)).toBe(24) // first image's length
    expect(view.getUint32(6 + 12, true)).toBe(38) // ...starts right after the directory
    expect(view.getUint32(22 + 12, true)).toBe(62) // the second follows the first
    expect(ico[38 + 4]).toBe(0xaa) // byte 4 of a stub is untouched by its header
    expect(ico[62 + 4]).toBe(0xbb)
    expect(() => encodeIco([{ size: 32, png: stub(16, 0) }])).toThrow(/declared 32/)
  })
})
