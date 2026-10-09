// The window's own storage, which keeps what belongs to this window rather than
// to the record: the theme (FR-073) and the last appointment (FR-046).
//
// Storage can refuse both reading and writing: a window opened with site data
// blocked answers by throwing rather than by answering nothing. Nothing kept
// here is worth a dead page, so a refusal reads as nothing held and a write
// that is refused lasts only as long as the window.

/** The value kept under key; null when there is none or storage refuses. */
export function readStored(key: string): string | null {
  try {
    return window.localStorage.getItem(key)
  } catch {
    return null
  }
}

/** keepStored keeps value under key where it can; null forgets it. */
export function keepStored(key: string, value: string | null): void {
  try {
    if (value === null) {
      window.localStorage.removeItem(key)
    } else {
      window.localStorage.setItem(key, value)
    }
  } catch {
    // Nothing to do and nothing to say: the page already shows the value.
  }
}
