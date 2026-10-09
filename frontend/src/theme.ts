// The theme's own rules: which themes there are, what is remembered, what the
// button offers next. Pure, with no DOM and no React, so every rule is
// reachable from a test by calling it; useTheme drives it against the page.
//
// SymDiary opens dark and stays dark until somebody says otherwise (FR-073).
// It does not follow the Windows app mode: a window that changes under the
// reader because the desktop reached dusk is a surprise, while the choice here
// is one press away.

/** Theme is the two the page draws. There is no third. */
export type Theme = 'dark' | 'light'

/** The theme a window opens in when nothing has been remembered. */
export const defaultTheme: Theme = 'dark'

/** Where the choice is kept, in the window's own storage. */
export const themeKey = 'symdiary.theme'

/**
 * The theme a stored value means. Anything that is not one of the two is the
 * default: a value from a future version, a truncated write or a key somebody
 * edited by hand must not leave the page with no theme at all.
 */
export function storedTheme(raw: string | null): Theme {
  return raw === 'light' || raw === 'dark' ? raw : defaultTheme
}

/** The theme a press moves to. */
export function nextTheme(current: Theme): Theme {
  return current === 'dark' ? 'light' : 'dark'
}

// What the button says, keyed by the theme in USE: it names the one it would
// move to, matching its picture, so in the dark the button offers the sun.
const offers: Record<Theme, string> = {
  dark: 'Light mode',
  light: 'Dark mode',
}

/** The button's label while current is showing. */
export function themeLabel(current: Theme): string {
  return offers[current]
}
