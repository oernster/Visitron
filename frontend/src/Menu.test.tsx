import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Menu, hoverCloseDelayMs, type MenuItem } from './Menu'

const chosen = { first: vi.fn(), second: vi.fn(), third: vi.fn() }
const items: MenuItem[] = [
  { label: 'First', onClick: chosen.first },
  { separator: true },
  { label: 'Second', onClick: chosen.second },
  { label: 'Third', onClick: chosen.third },
]

const trigger = () => screen.getByRole('button', { name: 'Help' })
const entry = (name: string) => screen.getByRole('menuitem', { name })
const shown = () => screen.queryByRole('menu', { name: 'Help' })

function renderMenu() {
  render(<Menu label="Help" icon="help.png" items={items} />)
}

afterEach(() => {
  vi.clearAllMocks()
  vi.useRealTimers()
})

describe('the menu, opening', () => {
  it('is closed until asked and says so on the trigger', () => {
    renderMenu()
    expect(shown()).toBeNull()
    expect(trigger()).toHaveAttribute('aria-haspopup', 'menu')
    expect(trigger()).toHaveAttribute('aria-expanded', 'false')
  })

  it.each(['ArrowDown', 'Enter', ' '])('opens on %j at the trigger with the first entry focused', (key) => {
    renderMenu()
    trigger().focus()
    fireEvent.keyDown(trigger(), { key })
    expect(shown()).not.toBeNull()
    expect(trigger()).toHaveAttribute('aria-expanded', 'true')
    expect(document.activeElement).toBe(entry('First'))
  })

  it('opens on a click leaving the focus where it was, then Down steps in', () => {
    renderMenu()
    trigger().focus()
    fireEvent.click(trigger())
    expect(shown()).not.toBeNull()
    expect(document.activeElement).toBe(trigger())
    fireEvent.keyDown(trigger(), { key: 'ArrowDown' })
    expect(document.activeElement).toBe(entry('First'))
  })

  it.each(['ArrowUp', 'Escape'])('retracts on %j at the trigger', (key) => {
    renderMenu()
    fireEvent.click(trigger())
    fireEvent.keyDown(trigger(), { key })
    expect(shown()).toBeNull()
  })

  it('heads the dropdown with its name and draws the divider', () => {
    renderMenu()
    fireEvent.click(trigger())
    const menu = shown() as HTMLElement
    expect(menu.querySelector('.menu-header')).toHaveTextContent('Help')
    expect(menu.querySelector('.menu-header')).toHaveAttribute('aria-hidden', 'true')
    expect(screen.getByRole('separator')).toHaveClass('menu-sep')
  })
})

describe('the menu, inside', () => {
  const openByKeyboard = () => {
    renderMenu()
    trigger().focus()
    fireEvent.keyDown(trigger(), { key: 'ArrowDown' })
  }

  it('walks the entries with Down and Up, wrapping at both ends and passing the divider', () => {
    openByKeyboard()
    fireEvent.keyDown(entry('First'), { key: 'ArrowDown' })
    expect(document.activeElement).toBe(entry('Second'))
    fireEvent.keyDown(entry('Second'), { key: 'ArrowDown' })
    fireEvent.keyDown(entry('Third'), { key: 'ArrowDown' })
    expect(document.activeElement).toBe(entry('First'))
    fireEvent.keyDown(entry('First'), { key: 'ArrowUp' })
    expect(document.activeElement).toBe(entry('Third'))
  })

  it('jumps to the ends with Home and End', () => {
    openByKeyboard()
    fireEvent.keyDown(entry('First'), { key: 'End' })
    expect(document.activeElement).toBe(entry('Third'))
    fireEvent.keyDown(entry('Third'), { key: 'Home' })
    expect(document.activeElement).toBe(entry('First'))
  })

  it.each(['Escape', 'Tab', 'ArrowLeft', 'ArrowRight'])('closes back onto the trigger on %j', (key) => {
    openByKeyboard()
    fireEvent.keyDown(entry('First'), { key })
    expect(shown()).toBeNull()
    expect(document.activeElement).toBe(trigger())
  })

  it('leaves every other key to the entry', () => {
    openByKeyboard()
    fireEvent.keyDown(entry('First'), { key: 'a' })
    expect(shown()).not.toBeNull()
    expect(document.activeElement).toBe(entry('First'))
  })

  it('keeps every key it answers from the window, so the ring does not act on it too', () => {
    const heard = vi.fn()
    window.addEventListener('keydown', heard)
    try {
      openByKeyboard()
      for (const key of ['ArrowDown', 'ArrowUp', 'Home', 'End', 'ArrowRight']) {
        fireEvent.keyDown(document.activeElement as HTMLElement, { key })
      }
      expect(heard).not.toHaveBeenCalled()
    } finally {
      window.removeEventListener('keydown', heard)
    }
  })

  it('runs the chosen entry and closes', () => {
    openByKeyboard()
    fireEvent.click(entry('Second'))
    expect(chosen.second).toHaveBeenCalledTimes(1)
    expect(chosen.first).not.toHaveBeenCalled()
    expect(shown()).toBeNull()
  })
})

describe('the menu and the pointer', () => {
  it('closes on a click outside it but not on one inside', () => {
    renderMenu()
    fireEvent.click(trigger())
    fireEvent.mouseDown(shown() as HTMLElement)
    expect(shown()).not.toBeNull()
    fireEvent.mouseDown(document.body)
    expect(shown()).toBeNull()
  })

  it('opens on hover and closes once the pointer has been gone for the grace period', () => {
    vi.useFakeTimers()
    renderMenu()
    const menu = trigger().parentElement as HTMLElement
    fireEvent.mouseEnter(menu)
    expect(shown()).not.toBeNull()
    fireEvent.mouseLeave(menu)
    act(() => vi.advanceTimersByTime(hoverCloseDelayMs - 1))
    expect(shown()).not.toBeNull()
    act(() => vi.advanceTimersByTime(1))
    expect(shown()).toBeNull()
  })

  it('stays open when the pointer comes back within the grace period', () => {
    vi.useFakeTimers()
    renderMenu()
    const menu = trigger().parentElement as HTMLElement
    fireEvent.mouseEnter(menu)
    fireEvent.mouseLeave(menu)
    fireEvent.mouseEnter(menu)
    act(() => vi.advanceTimersByTime(hoverCloseDelayMs))
    expect(shown()).not.toBeNull()
  })
})
