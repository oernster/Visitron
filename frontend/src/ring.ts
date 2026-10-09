// The focus ring's own rules: which key means what, where a step lands, which
// controls keep their arrows for themselves. Pure, with no DOM and no
// React, so every rule is reachable from a test by calling it; useRing drives
// it against the real page.
//
// The model is the house one. Tab and Right both step forward, Shift+Tab and
// Left both step back, the ring wraps at either end; the horizontal arrows are
// tested first so they step the ring everywhere rather than being trapped
// by whatever has focus.

/** What a key press means to the ring. */
export type RingMove = 'forward' | 'back' | 'activate' | 'none'

/** The keys that step the ring forward or backward, plus those that fire a stop. */
const FORWARD_KEYS = ['Tab', 'ArrowRight']
const BACK_KEYS = ['ArrowLeft']
const ACTIVATE_KEYS = ['Enter', ' ']

/**
 * What a key press means. Shift+Tab is a step back rather than forward, which
 * is why the modifier is read here rather than by the caller.
 */
export function ringMove(key: string, shiftKey: boolean): RingMove {
  if (key === 'Tab') {
    return shiftKey ? 'back' : 'forward'
  }
  if (FORWARD_KEYS.includes(key)) {
    return 'forward'
  }
  if (BACK_KEYS.includes(key)) {
    return 'back'
  }
  if (ACTIVATE_KEYS.includes(key)) {
    return 'activate'
  }
  return 'none'
}

/**
 * Where a step lands, wrapping at both ends. `from` is -1 when nothing in the
 * ring holds focus, which enters at the end the step arrives from: the first
 * stop going forward, the last coming back. Answers -1 for an empty ring, so a
 * caller with nothing to focus is told rather than given a wrong index.
 */
export function nextStop(count: number, from: number, move: 'forward' | 'back'): number {
  if (count <= 0) {
    return -1
  }
  const step = move === 'forward' ? 1 : -1
  if (from < 0) {
    return move === 'forward' ? 0 : count - 1
  }
  return (from + step + count) % count
}

// The input types that hold text a caret moves through. A date field is one of
// these on purpose: its arrows step the segment under the caret, which is the
// same claim on the keys.
const TEXT_INPUT_TYPES = ['text', 'search', 'email', 'url', 'tel', 'password', 'number', 'date', 'datetime-local', 'time', 'month', 'week']

/**
 * Whether a control keeps the horizontal arrows for itself. A field holding
 * text keeps them for its caret, so it is left with Tab rather than with Left
 * and Right; that is the one place the arrows do not step the ring.
 */
export function keepsArrows(tagName: string, type: string | null): boolean {
  const tag = tagName.toLowerCase()
  if (tag === 'textarea') {
    return true
  }
  if (tag !== 'input') {
    return false
  }
  return TEXT_INPUT_TYPES.includes((type ?? 'text').toLowerCase())
}

/**
 * Whether Enter should be made to act like Space on this control. A button
 * handles both already; a checkbox or a radio answers Space alone, so Enter on
 * one does nothing at all unless it is given this.
 */
export function enterActsAsSpace(tagName: string, type: string | null): boolean {
  if (tagName.toLowerCase() !== 'input') {
    return false
  }
  const kind = (type ?? '').toLowerCase()
  return kind === 'checkbox' || kind === 'radio'
}
