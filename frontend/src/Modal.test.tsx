import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { Modal } from './Modal'

describe('the dialog shell', () => {
  it('gives every dialog a cross that closes it, without opening on the cross', () => {
    const onClose = vi.fn()
    render(
      <Modal labelId="t" role="dialog" onClose={onClose}>
        <h2 id="t">A dialog</h2>
        <button type="button">Its own answer</button>
      </Modal>,
    )
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Its own answer' }))
    fireEvent.mouseDown(screen.getByRole('button', { name: 'Close this dialog' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
