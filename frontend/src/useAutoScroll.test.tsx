// The page-level half of the cycle: what suspends it and what freezes it. The
// pacing itself is covered in autoScroll.test.ts. jsdom has no layout, so the
// surface's scroll metrics and scrollTop are stubbed and the tick timer is
// driven by fake timers.

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, render } from '@testing-library/react'
import { START_HOLD_MS, TICK_MS } from './autoScroll'
import { useAutoScroll } from './useAutoScroll'

const OVERFLOW_PX = 1000
const STEPS = 20

// A scrollable panel wearing the cycle, optionally inside a backdrop and
// optionally with a second, later backdrop above it.
function Surface({ inDialog = false, covered = false }: { inDialog?: boolean; covered?: boolean }) {
  const autoScroll = useAutoScroll()
  const panel = (
    <div data-testid="surface" ref={autoScroll}>
      content
    </div>
  )
  return (
    <>
      {inDialog ? <div className="backdrop">{panel}</div> : panel}
      {covered && (
        <div className="backdrop">
          <div>a dialog on top</div>
        </div>
      )}
    </>
  )
}

// Stub the metrics jsdom never computes: the element overflows; scrollTop is a
// plain value rather than the always-zero one jsdom gives an element it never
// laid out.
function renderSurface(props: { inDialog?: boolean; covered?: boolean } = {}) {
  const view = render(<Surface {...props} />)
  const surface = view.getByTestId('surface')
  let scrollTop = 0
  Object.defineProperty(surface, 'scrollTop', {
    get: () => scrollTop,
    set: (value: number) => {
      scrollTop = value
    },
  })
  Object.defineProperty(surface, 'scrollHeight', { get: () => OVERFLOW_PX })
  Object.defineProperty(surface, 'clientHeight', { get: () => 0 })
  return { surface, view }
}

// Far enough to leave the start hold and take several descent steps.
const readPast = () => vi.advanceTimersByTime(START_HOLD_MS + TICK_MS * STEPS)

beforeEach(() => {
  vi.useFakeTimers()
  vi.stubGlobal('matchMedia', (query: string) => ({ matches: false, media: query }))
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
  cleanup()
})

describe('useAutoScroll', () => {
  it('holds still on open, then reads the surface down', () => {
    const { surface } = renderSurface()
    vi.advanceTimersByTime(START_HOLD_MS - TICK_MS)
    expect(surface.scrollTop).toBe(0)
    vi.advanceTimersByTime(TICK_MS * STEPS)
    expect(surface.scrollTop).toBeGreaterThan(0)
  })

  it('still reads for someone whose system has animation effects switched off', () => {
    // Windows drives prefers-reduced-motion from a general animation switch, so
    // the cycle must not be gated on it: doing so removes the feature outright
    // on a machine that has it turned off.
    vi.stubGlobal('matchMedia', (query: string) => ({ matches: true, media: query }))
    const { surface } = renderSurface()
    readPast()
    expect(surface.scrollTop).toBeGreaterThan(0)
  })

  it.each(['wheel', 'mousedown', 'touchstart', 'keydown', 'focusin'])(
    'suspends the cycle when the reader takes over with %s',
    (type) => {
      const { surface } = renderSurface()
      readPast()
      const taken = surface.scrollTop
      surface.dispatchEvent(new Event(type, { bubbles: true }))
      vi.advanceTimersByTime(TICK_MS * STEPS)
      expect(surface.scrollTop).toBe(taken)
    },
  )

  it('resumes on its own after the reader stops, rather than switching off', () => {
    const { surface } = renderSurface()
    readPast()
    surface.dispatchEvent(new Event('wheel', { bubbles: true }))
    const taken = surface.scrollTop
    // Long enough to outlast the suspension and take a few more steps.
    vi.advanceTimersByTime(START_HOLD_MS + TICK_MS * STEPS)
    expect(surface.scrollTop).toBeGreaterThan(taken)
  })

  it('freezes a surface sitting under another dialog; carries on when it closes', () => {
    const { surface, view } = renderSurface({ inDialog: true, covered: true })
    readPast()
    expect(surface.scrollTop).toBe(0)
    // The dialog above closes and the cycle picks up exactly where it was:
    // frozen means nothing was consumed, so the start hold it was part-way
    // through is still owed in full.
    view.rerender(<Surface inDialog covered={false} />)
    vi.advanceTimersByTime(TICK_MS * STEPS)
    expect(surface.scrollTop).toBe(0)
    readPast()
    expect(surface.scrollTop).toBeGreaterThan(0)
  })

  it('reads a surface that is itself the topmost dialog', () => {
    const { surface } = renderSurface({ inDialog: true })
    readPast()
    expect(surface.scrollTop).toBeGreaterThan(0)
  })

  it('holds a surface outside any dialog still while a dialog is open', () => {
    const { surface } = renderSurface({ covered: true })
    readPast()
    expect(surface.scrollTop).toBe(0)
  })

  it('stops ticking once the surface goes away', () => {
    const { view } = renderSurface()
    view.unmount()
    expect(vi.getTimerCount()).toBe(0)
  })
})
