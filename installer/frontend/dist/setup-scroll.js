// The gentle self-reading cycle, for the one surface in setup long enough to
// need it: hold still on open, read down slowly, hold at the tail, rewind
// fast, repeat; step aside the moment the reader takes over.
//
// This is a port of the application's own frontend/src/autoScroll.ts and
// useAutoScroll.ts, kept deliberately close to them so the two can be read
// side by side. It is a copy rather than an import because the setup page has
// no build step and cannot reach a TypeScript module; Fulcrum's installer
// carries its own copy of the same scroller for the same reason. What must
// never differ is the pace: the constants below are the application's; one
// surface needing a different pace would mean the pace is wrong everywhere.

// The clock. Every hold counts down in whole ticks, so a wait and a movement
// share one granularity.
const TICK_MS = 40

// The stillness before the first descent: the reader orients before anything
// moves. It is the opening phase's wait rather than a special case.
const START_HOLD_MS = 5000

// The descent is one pixel every second tick. The divider is a countdown of
// ticks rather than a slower timer, so holds keep their TICK_MS granularity.
const DESCENT_PX = 1
const DESCENT_TICKS_PER_STEP = 2

// Long enough to finish reading the tail before the rewind takes it away.
const BOTTOM_HOLD_MS = 5000

// A reposition, not a reading pass, so it travels fast. Never read at this
// pace and never rewind at the reading pace.
const REWIND_PX = 15

// The breath before the next pass.
const TOP_HOLD_MS = 2000

// The stillness required after any manual reading input before the cycle picks
// up again, from wherever the reader left it. Manual input suspends the cycle;
// it never switches it off.
const MANUAL_HOLD_MS = 2500

// The inputs that count as reading by hand. mousedown covers a press on the
// native scrollbar as well as on the words; focusin covers the keyboard
// arriving in the surface, which is somebody about to read, exactly like a
// scroll.
const MANUAL_EVENTS = ['wheel', 'mousedown', 'touchstart', 'keydown', 'focusin']

/** The state a surface opens in: the top hold seeded with the start hold, so a
 *  fresh surface stands still before it first moves. */
function initialAutoScrollState() {
    return {phase: 'pauseTop', waitMs: START_HOLD_MS, ticksToStep: DESCENT_TICKS_PER_STEP}
}

/** The state a manual reading input puts the cycle into: a hold, from which it
 *  resumes at the reader's own position rather than restarting. */
function suspended(state) {
    return {...state, phase: 'manual', waitMs: MANUAL_HOLD_MS}
}

/** Advance the cycle by one tick and report how far the surface should move,
 *  already clamped. Content that does not overflow consumes nothing, so
 *  attaching the cycle to a surface that currently fits is free and correct. */
function autoScrollTick(state, view) {
    if (view.maxScrollTop <= 0) {
        return {state, delta: 0}
    }
    if (state.phase === 'down') {
        return descend(state, view)
    }
    if (state.phase === 'up') {
        return rewind(state, view)
    }
    return hold(state, view)
}

// Count the current wait down; when it runs out, choose the direction to
// resume in: after the bottom hold the rewind, after a manual hold whatever is
// left to do from where the reader stopped (a reader who scrolled to the very
// end has only the rewind left), otherwise the reading pass.
function hold(state, view) {
    const waitMs = state.waitMs - TICK_MS
    if (waitMs > 0) {
        return {state: {...state, waitMs}, delta: 0}
    }
    if (state.phase === 'pauseBottom') {
        return {state: {...state, phase: 'up', waitMs: 0}, delta: 0}
    }
    if (state.phase === 'manual' && view.scrollTop >= view.maxScrollTop) {
        return {state: {...state, phase: 'up', waitMs: 0}, delta: 0}
    }
    return {state: {phase: 'down', waitMs: 0, ticksToStep: DESCENT_TICKS_PER_STEP}, delta: 0}
}

// Advance the reading pass by a pixel every DESCENT_TICKS_PER_STEP ticks, then
// hand over to the bottom hold on arrival.
function descend(state, view) {
    const ticksToStep = state.ticksToStep - 1
    if (ticksToStep > 0) {
        return {state: {...state, ticksToStep}, delta: 0}
    }
    const remaining = view.maxScrollTop - view.scrollTop
    if (remaining <= DESCENT_PX) {
        return {
            state: {phase: 'pauseBottom', waitMs: BOTTOM_HOLD_MS, ticksToStep: DESCENT_TICKS_PER_STEP},
            delta: Math.max(0, remaining),
        }
    }
    return {state: {...state, ticksToStep: DESCENT_TICKS_PER_STEP}, delta: DESCENT_PX}
}

// Travel back at the repositioning pace, then hand over to the top hold on
// arrival.
function rewind(state, view) {
    if (view.scrollTop <= REWIND_PX) {
        return {
            state: {phase: 'pauseTop', waitMs: TOP_HOLD_MS, ticksToStep: DESCENT_TICKS_PER_STEP},
            delta: -view.scrollTop,
        }
    }
    return {state, delta: -REWIND_PX}
}

/**
 * Give a scrollable element the cycle above and answer the way to stop it.
 *
 * The caller holds the answer and calls it when the surface goes, which in
 * setup means leaving the licence screen. Nothing here watches for that: a
 * screen stack has no unmount to hook, so the one place that knows the screen
 * has changed is the one place that stops the timer. Coming back starts a
 * fresh cycle, start hold and all, which is right: somebody returning to a
 * licence is starting to read it again.
 *
 * The cycle is NOT gated on prefers-reduced-motion. On Windows that query
 * follows the general Animation effects switch, which people turn off for
 * performance rather than for motion sensitivity, so gating on it would
 * silently remove the feature. A reader who does not want it touches the pane.
 */
function readItself(node) {
    let state = initialAutoScrollState()
    const takeOver = () => {
        state = suspended(state)
    }
    for (const type of MANUAL_EVENTS) {
        node.addEventListener(type, takeOver, {passive: true})
    }
    const timer = window.setInterval(() => {
        const view = {
            scrollTop: node.scrollTop,
            maxScrollTop: node.scrollHeight - node.clientHeight,
        }
        const stepped = autoScrollTick(state, view)
        state = stepped.state
        if (stepped.delta !== 0) {
            node.scrollTop = view.scrollTop + stepped.delta
        }
    }, TICK_MS)
    return () => {
        window.clearInterval(timer)
        for (const type of MANUAL_EVENTS) {
            node.removeEventListener(type, takeOver)
        }
    }
}
