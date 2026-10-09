// The ring against a real page: what the stops are, where a step lands and
// which controls are passed over. The rules themselves are covered in
// ring.test.ts; this is the half that touches the DOM.

import { afterEach, describe, expect, it } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { ringRoot, ringStops, useRing } from './useRing'

function Page({ withDialog = false }: { withDialog?: boolean }) {
  useRing()
  return (
    <div>
      <button type="button">First</button>
      <input type="text" aria-label="Words" />
      <button type="button" disabled>
        Inert
      </button>
      <button type="button">Last</button>
      {withDialog && (
        <div className="dialog">
          <button type="button">In the dialog</button>
          <button type="button">Close</button>
        </div>
      )}
    </div>
  )
}

afterEach(cleanup)

const named = (name: string) => screen.getByRole('button', { name })

describe('ringStops', () => {
  it('collects the enabled controls in the order they are drawn', () => {
    render(<Page />)
    const labels = ringStops(document).map((el) => el.textContent || el.getAttribute('aria-label'))
    expect(labels).toEqual(['First', 'Words', 'Last'])
  })

  it('passes over a disabled control, so the ring never stalls on a dead stop', () => {
    render(<Page />)
    expect(ringStops(document).map((el) => el.textContent)).not.toContain('Inert')
  })

  it('passes over a container, which is chrome rather than a control', () => {
    render(<Page />)
    expect(ringStops(document).some((el) => el.tagName === 'DIV')).toBe(false)
  })
})

describe('ringRoot', () => {
  it('is the whole page while no dialog is open', () => {
    render(<Page />)
    expect(ringRoot()).toBe(document)
  })

  it('is the dialog while one is open, so focus stays inside it', () => {
    render(<Page withDialog />)
    const root = ringRoot()
    expect(root).not.toBe(document)
    expect(ringStops(root).map((el) => el.textContent)).toEqual(['In the dialog', 'Close'])
  })
})

describe('the ring in use', () => {
  it('steps forward on Tab and on Right, which agree', () => {
    render(<Page />)
    named('First').focus()
    fireEvent.keyDown(window, { key: 'Tab' })
    expect(document.activeElement).toBe(screen.getByLabelText('Words'))

    named('First').focus()
    fireEvent.keyDown(window, { key: 'ArrowRight' })
    expect(document.activeElement).toBe(screen.getByLabelText('Words'))
  })

  it('steps back on Shift+Tab and on Left, which agree', () => {
    render(<Page />)
    named('Last').focus()
    fireEvent.keyDown(window, { key: 'Tab', shiftKey: true })
    expect(document.activeElement).toBe(screen.getByLabelText('Words'))

    named('Last').focus()
    // A text field keeps Left for its caret, so the step is made from a button.
    fireEvent.keyDown(window, { key: 'ArrowLeft' })
    expect(document.activeElement).toBe(screen.getByLabelText('Words'))
  })

  it('wraps at both ends', () => {
    render(<Page />)
    named('Last').focus()
    fireEvent.keyDown(window, { key: 'Tab' })
    expect(document.activeElement).toBe(named('First'))

    named('First').focus()
    fireEvent.keyDown(window, { key: 'Tab', shiftKey: true })
    expect(document.activeElement).toBe(named('Last'))
  })

  it('leaves a text field its arrows, so the caret is not stolen by the ring', () => {
    render(<Page />)
    const words = screen.getByLabelText('Words')
    words.focus()
    fireEvent.keyDown(window, { key: 'ArrowRight' })
    expect(document.activeElement).toBe(words)
    // Tab still leaves it, which is how a text field is left.
    fireEvent.keyDown(window, { key: 'Tab' })
    expect(document.activeElement).toBe(named('Last'))
  })

  it('enters the ring from nothing, which is how the window opens', () => {
    render(<Page />)
    ;(document.activeElement as HTMLElement)?.blur()
    fireEvent.keyDown(window, { key: 'Tab' })
    expect(document.activeElement).toBe(named('First'))
  })

  it('keeps the ring inside an open dialog', () => {
    render(<Page withDialog />)
    const close = named('Close')
    close.focus()
    fireEvent.keyDown(window, { key: 'Tab' })
    expect(document.activeElement).toBe(named('In the dialog'))
  })
})

describe('Enter where only Space is answered', () => {
  function Boxes() {
    useRing()
    return <input type="checkbox" aria-label="Chosen" />
  }

  it('ticks a checkbox on Enter, which the browser answers for Space alone', () => {
    render(<Boxes />)
    const box = screen.getByLabelText('Chosen') as HTMLInputElement
    box.focus()
    expect(box.checked).toBe(false)
    fireEvent.keyDown(window, { key: 'Enter' })
    expect(box.checked).toBe(true)
  })
})
