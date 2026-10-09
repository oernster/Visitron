import { describe, expect, it } from 'vitest'
import { defaultTheme, nextTheme, storedTheme, themeLabel } from './theme'

describe('the theme rules', () => {
  it('opens dark when nothing has been remembered', () => {
    expect(defaultTheme).toBe('dark')
    expect(storedTheme(null)).toBe('dark')
  })

  it('takes back what it wrote', () => {
    expect(storedTheme('dark')).toBe('dark')
    expect(storedTheme('light')).toBe('light')
  })

  it('falls back to the default rather than trusting a stored value', () => {
    // A value from a later version, a truncated write or a hand edit. None of
    // them may leave the page wearing no theme at all.
    for (const stored of ['', 'Dark', 'system', 'sepia', '{"theme":"light"}']) {
      expect(storedTheme(stored)).toBe(defaultTheme)
    }
  })

  it('moves to the other one and back', () => {
    expect(nextTheme('dark')).toBe('light')
    expect(nextTheme('light')).toBe('dark')
    expect(nextTheme(nextTheme('dark'))).toBe('dark')
  })

  it('names the theme the press would move to, never the one in use', () => {
    // The label says the same thing as the picture: in the dark, the sun.
    expect(themeLabel('dark')).toBe('Light mode')
    expect(themeLabel('light')).toBe('Dark mode')
  })
})
