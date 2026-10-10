import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { CloseChoiceDialog } from './CloseChoiceDialog'

function dialog() {
  const handlers = { onMinimise: vi.fn(), onQuit: vi.fn(), onCancel: vi.fn() }
  render(<CloseChoiceDialog {...handlers} />)
  return handlers
}

describe('the close choice', () => {
  it('opens on Minimise to tray, so a stray Enter never quits', () => {
    const { onMinimise, onQuit } = dialog()
    expect(screen.getByRole('alertdialog', { name: 'Close Visitron' })).toBeInTheDocument()
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Minimise to tray' }))
    expect(screen.queryByRole('button', { name: 'Go back' })).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Minimise to tray' }))
    expect(onMinimise).toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Quit' }))
    expect(onQuit).toHaveBeenCalled()
  })

  it('cancels on Escape', () => {
    const { onCancel } = dialog()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onCancel).toHaveBeenCalled()
  })

  it('cancels on its cross, once per click, from the mouse or the keyboard', () => {
    const { onCancel, onQuit, onMinimise } = dialog()
    const cross = screen.getByRole('button', { name: 'Close this dialog' })
    fireEvent.mouseDown(cross)
    fireEvent.click(cross, { detail: 1 })
    expect(onCancel).toHaveBeenCalledTimes(1)
    fireEvent.click(cross, { detail: 0 })
    expect(onCancel).toHaveBeenCalledTimes(2)
    expect(onQuit).not.toHaveBeenCalled()
    expect(onMinimise).not.toHaveBeenCalled()
  })

  it('warns of open work and opens on Go back', () => {
    const work = document.createElement('div')
    work.className = 'dialog'
    document.body.appendChild(work)
    try {
      const { onCancel } = dialog()
      expect(screen.getByRole('alert')).toHaveTextContent(/anything unsaved in it may be lost/)
      const back = screen.getByRole('button', { name: 'Go back' })
      expect(document.activeElement).toBe(back)
      fireEvent.click(back)
      expect(onCancel).toHaveBeenCalled()
    } finally {
      work.remove()
    }
  })
})
