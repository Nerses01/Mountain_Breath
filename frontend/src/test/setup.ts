import { afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'
// Adds DOM matchers like .toBeInTheDocument() to expect().
import '@testing-library/jest-dom/vitest'
// Initialise i18next once for the whole suite. Any component calling
// useTranslation/useLocale needs a real instance — without it react-i18next
// hands back a stub whose changeLanguage is not a function, and the failure
// surfaces in whichever component happened to be translated most recently
// rather than where the real problem is. Doing it here means no test has to
// remember.
import '../i18n'

// Unmount rendered components between tests so they can't leak into
// each other.
afterEach(() => {
  cleanup()
})
