import { afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'
// Adds DOM matchers like .toBeInTheDocument() to expect().
import '@testing-library/jest-dom/vitest'

// Unmount rendered components between tests so they can't leak into
// each other.
afterEach(() => {
  cleanup()
})
