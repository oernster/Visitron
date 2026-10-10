// Enter in a text box applies it, as its button would (Amendment 12): the one
// home for that rule, so every box keeps it the same way.

import type { KeyboardEvent } from 'react'

/** onEnter answers a key handler that runs act on Enter, when allowed. */
export function onEnter(act: () => void, allowed = true) {
  return (event: KeyboardEvent) => {
    if (event.key !== 'Enter' || !allowed) return
    event.preventDefault()
    act()
  }
}
