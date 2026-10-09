import { describe, expect, it } from 'vitest'
import { enterActsAsSpace, keepsArrows, nextStop, ringMove } from './ring'

describe('ring: what a key means', () => {
  it('steps forward on Tab and on Right', () => {
    expect(ringMove('Tab', false)).toBe('forward')
    expect(ringMove('ArrowRight', false)).toBe('forward')
  })

  it('steps back on Shift+Tab and on Left', () => {
    expect(ringMove('Tab', true)).toBe('back')
    expect(ringMove('ArrowLeft', false)).toBe('back')
  })

  it('reads Left as back whether or not shift is held', () => {
    // Shift only reverses Tab. Making it reverse the arrows too would leave
    // Shift+Left meaning forward, which nothing would guess.
    expect(ringMove('ArrowLeft', true)).toBe('back')
    expect(ringMove('ArrowRight', true)).toBe('forward')
  })

  it('fires the focused stop on Enter and on Space', () => {
    expect(ringMove('Enter', false)).toBe('activate')
    expect(ringMove(' ', false)).toBe('activate')
  })

  it('leaves everything else alone', () => {
    for (const key of ['ArrowUp', 'ArrowDown', 'Escape', 'a', 'Home', 'PageDown']) {
      expect(ringMove(key, false)).toBe('none')
    }
  })
})

describe('ring: where a step lands', () => {
  const count = 4

  it('walks forward and back through the stops', () => {
    expect(nextStop(count, 0, 'forward')).toBe(1)
    expect(nextStop(count, 2, 'back')).toBe(1)
  })

  it('wraps at both ends, so the ring is a cycle rather than a dead end', () => {
    expect(nextStop(count, count - 1, 'forward')).toBe(0)
    expect(nextStop(count, 0, 'back')).toBe(count - 1)
  })

  it('enters at the end the step arrives from when nothing holds focus', () => {
    expect(nextStop(count, -1, 'forward')).toBe(0)
    expect(nextStop(count, -1, 'back')).toBe(count - 1)
  })

  it('answers no stop at all for an empty ring', () => {
    expect(nextStop(0, -1, 'forward')).toBe(-1)
    expect(nextStop(0, 0, 'back')).toBe(-1)
  })
})

describe('ring: controls that keep their arrows', () => {
  it('leaves a text field its caret', () => {
    expect(keepsArrows('INPUT', 'text')).toBe(true)
    expect(keepsArrows('textarea', null)).toBe(true)
  })

  it('leaves a date field its segments, which the arrows step', () => {
    expect(keepsArrows('INPUT', 'date')).toBe(true)
  })

  it('treats an input with no type as the text field it is', () => {
    expect(keepsArrows('input', null)).toBe(true)
  })

  it('takes the arrows back from a control with no caret', () => {
    expect(keepsArrows('INPUT', 'checkbox')).toBe(false)
    expect(keepsArrows('INPUT', 'radio')).toBe(false)
    expect(keepsArrows('BUTTON', null)).toBe(false)
    expect(keepsArrows('A', null)).toBe(false)
  })
})

describe('ring: Enter where only Space is answered', () => {
  it('is made to act on a checkbox and a radio', () => {
    expect(enterActsAsSpace('INPUT', 'checkbox')).toBe(true)
    expect(enterActsAsSpace('input', 'radio')).toBe(true)
  })

  it('is left alone on a button, which answers both already', () => {
    expect(enterActsAsSpace('BUTTON', null)).toBe(false)
    expect(enterActsAsSpace('INPUT', 'text')).toBe(false)
  })
})
