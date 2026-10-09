// The gentle self-reading cycle for long help content: hold still on open, read
// down slowly, hold at the tail, rewind fast, repeat; step aside the moment the
// reader takes over. This file is the state machine only, pure and free of
// the DOM, so every phase is reachable from a test by calling the tick rather
// than by waiting. useAutoScroll drives it against a real element on a timer.
//
// The pace belongs to the application, never to one surface: if a surface seems
// to need a different pace, the pace is wrong everywhere. Ported from
// PigeonPost, which carries the same constants as the PySide6 apps.

// The clock. Every hold counts down in whole ticks, so a wait and a movement
// share one granularity.
export const TICK_MS = 40

// The stillness before the first descent: the reader orients before anything
// moves. It is the opening phase's wait rather than a special case, so it costs
// no extra state.
export const START_HOLD_MS = 5000

// The descent is one pixel every second tick. The divider is a countdown of
// ticks rather than a slower timer, so holds keep their TICK_MS granularity.
export const DESCENT_PX = 1
export const DESCENT_TICKS_PER_STEP = 2

// Long enough to finish reading the tail before the rewind takes it away.
export const BOTTOM_HOLD_MS = 5000

// A reposition, not a reading pass, so it travels fast. Never read at this pace
// and never rewind at the reading pace.
export const REWIND_PX = 15

// The breath before the next pass.
export const TOP_HOLD_MS = 2000

// The stillness required after any manual reading input before the cycle picks
// up again, from wherever the reader left it. Manual input suspends the cycle;
// it never switches it off.
export const MANUAL_HOLD_MS = 2500

// Two reading movements and three holds. A manual hold is a hold like the
// others, differing only in what it resumes into.
export type AutoScrollPhase = 'down' | 'pauseBottom' | 'up' | 'pauseTop' | 'manual'

export interface AutoScrollState {
  phase: AutoScrollPhase
  /** What is left of the current hold; ignored by the movement phases. */
  waitMs: number
  /** Counts down to the next descent pixel. */
  ticksToStep: number
}

/** The part of a scrollable element the machine reads: where it sits, how far it can go. */
export interface ScrollView {
  scrollTop: number
  maxScrollTop: number
}

/**
 * The state a surface opens in: the top hold seeded with the start hold, so a
 * fresh surface stands still before it first moves.
 */
export function initialAutoScrollState(): AutoScrollState {
  return { phase: 'pauseTop', waitMs: START_HOLD_MS, ticksToStep: DESCENT_TICKS_PER_STEP }
}

/**
 * The state a manual reading input puts the cycle into: a hold, from which it
 * resumes at the reader's own position rather than restarting.
 */
export function suspended(state: AutoScrollState): AutoScrollState {
  return { ...state, phase: 'manual', waitMs: MANUAL_HOLD_MS }
}

/**
 * Advance the cycle by one tick and report how far the surface should move,
 * already clamped to its bounds. Content that does not overflow consumes
 * nothing: no wait counts down and no phase changes, so attaching the cycle to
 * a surface that currently fits is free and correct.
 */
export function autoScrollTick(
  state: AutoScrollState,
  view: ScrollView,
): { state: AutoScrollState; delta: number } {
  if (view.maxScrollTop <= 0) {
    return { state, delta: 0 }
  }
  if (state.phase === 'down') {
    return descend(state, view)
  }
  if (state.phase === 'up') {
    return rewind(state, view)
  }
  return hold(state, view)
}

// Count the current wait down; when it runs out, choose the direction to resume
// in: after the bottom hold the rewind, after a manual hold whatever is
// left to do from where the reader stopped (a reader who scrolled to the very
// end has only the rewind left), otherwise the reading pass.
function hold(state: AutoScrollState, view: ScrollView): { state: AutoScrollState; delta: number } {
  const waitMs = state.waitMs - TICK_MS
  if (waitMs > 0) {
    return { state: { ...state, waitMs }, delta: 0 }
  }
  if (state.phase === 'pauseBottom') {
    return { state: { ...state, phase: 'up', waitMs: 0 }, delta: 0 }
  }
  if (state.phase === 'manual' && view.scrollTop >= view.maxScrollTop) {
    return { state: { ...state, phase: 'up', waitMs: 0 }, delta: 0 }
  }
  return { state: { phase: 'down', waitMs: 0, ticksToStep: DESCENT_TICKS_PER_STEP }, delta: 0 }
}

// Advance the reading pass by a pixel every DESCENT_TICKS_PER_STEP ticks, then
// hand over to the bottom hold on arrival.
function descend(
  state: AutoScrollState,
  view: ScrollView,
): { state: AutoScrollState; delta: number } {
  const ticksToStep = state.ticksToStep - 1
  if (ticksToStep > 0) {
    return { state: { ...state, ticksToStep }, delta: 0 }
  }
  const remaining = view.maxScrollTop - view.scrollTop
  if (remaining <= DESCENT_PX) {
    return {
      state: { phase: 'pauseBottom', waitMs: BOTTOM_HOLD_MS, ticksToStep: DESCENT_TICKS_PER_STEP },
      delta: Math.max(0, remaining),
    }
  }
  return { state: { ...state, ticksToStep: DESCENT_TICKS_PER_STEP }, delta: DESCENT_PX }
}

// Travel back at the repositioning pace, then hand over to the top hold on
// arrival.
function rewind(
  state: AutoScrollState,
  view: ScrollView,
): { state: AutoScrollState; delta: number } {
  if (view.scrollTop <= REWIND_PX) {
    return {
      state: { phase: 'pauseTop', waitMs: TOP_HOLD_MS, ticksToStep: DESCENT_TICKS_PER_STEP },
      delta: -view.scrollTop,
    }
  }
  return { state, delta: -REWIND_PX }
}
