// The DOM half of the focus ring: what the stops are on the page as it stands,
// and the one key listener that steps between them. Ported from PigeonPost's
// focusRing.ts, which scopes the ring to the topmost dialog and collects the
// same set the browser steps with Tab.

import { useEffect } from 'react'
import { enterActsAsSpace, keepsArrows, nextStop, ringMove } from './ring'

// What the browser itself treats as tabbable. Anything explicitly taken out of
// the tab order is left out here too, so the ring and Tab never disagree.
const STOP_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

/**
 * The container the ring is scoped to: the topmost open dialog where one is
 * showing, so focus stays inside it, otherwise the whole document.
 */
export function ringRoot(): ParentNode {
  const dialogs = document.querySelectorAll<HTMLElement>('.dialog')
  return dialogs.length > 0 ? dialogs[dialogs.length - 1] : document
}

/**
 * The visible, enabled stops inside root, in document order. A control that is
 * hidden or has not been laid out is skipped, matching what Tab would do.
 *
 * jsdom lays nothing out, so `getClientRects` is empty for everything there. It
 * is therefore asked only when the browser has actually measured something,
 * which leaves the filter honest in a real window and inert under a test.
 */
export function ringStops(root: ParentNode): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(STOP_SELECTOR)).filter((el) => {
    if (el.tabIndex < 0 || el.hasAttribute('disabled')) {
      return false
    }
    if (el.getClientRects().length > 0) {
      return true
    }
    // Nothing was measured. Fall back to the styles a test can set, rather
    // than dropping every stop on the floor.
    const style = getComputedStyle(el)
    return style.visibility !== 'hidden' && style.display !== 'none'
  })
}

/** Move focus one stop forward or back, wrapping at both ends. */
export function step(move: 'forward' | 'back'): void {
  const stops = ringStops(ringRoot())
  const at = stops.indexOf(document.activeElement as HTMLElement)
  const next = nextStop(stops.length, at, move)
  if (next >= 0) {
    stops[next].focus()
  }
}

/**
 * Give the page the house keyboard ring for as long as it is mounted.
 *
 * Tab and Right step forward, Shift+Tab and Left step back; both wrap. The
 * arrows are answered here rather than left to the browser, which has no
 * opinion about them, so they step the ring from anywhere; a field holding text
 * keeps them for its caret and is left with Tab instead.
 *
 * Tab is answered too rather than left alone, because the browser's own Tab
 * walks out of the page furniture and into the window, which a dialog must not
 * allow and the main window has no use for.
 */
export function useRing(): void {
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const active = document.activeElement as HTMLElement | null
      const move = ringMove(event.key, event.shiftKey)
      if (move === 'activate') {
        if (
          event.key === 'Enter' &&
          active &&
          enterActsAsSpace(active.tagName, active.getAttribute('type'))
        ) {
          // A checkbox answers Space alone, so Enter on one does nothing at
          // all. The house rule is that the two keys agree everywhere.
          event.preventDefault()
          active.click()
        }
        return
      }
      if (move === 'none') {
        return
      }
      if (
        event.key !== 'Tab' &&
        active &&
        keepsArrows(active.tagName, active.getAttribute('type'))
      ) {
        return
      }
      if (event.key !== 'Tab' && active?.getAttribute('aria-expanded') === 'true') {
        // An open dropdown owns its arrows until it closes; stepping the ring
        // out from under the reader mid-choice is not what the key meant.
        return
      }
      event.preventDefault()
      step(move)
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])
}
