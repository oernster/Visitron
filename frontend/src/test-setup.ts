// Shared set-up for the front-end suite: each test starts with no bridge, so a
// test that wants the window says so with installBridge.

import '@testing-library/jest-dom/vitest'
import { afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'

afterEach(() => {
  cleanup()
  delete (window as unknown as { go?: unknown }).go
})
