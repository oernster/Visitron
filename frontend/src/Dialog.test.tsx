import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AboutDialog, ConfirmDialog } from './Dialog'
import { anAbout } from './bridge-fake'

describe('About', () => {
  it('names the product, its author, every credit and the licence', () => {
    const onClose = vi.fn()
    render(<AboutDialog about={anAbout} onClose={onClose} />)
    const about = screen.getByRole('dialog', { name: 'Visitron 1.0.0' })
    expect(within(about).getByText('By Oliver Ernster')).toBeInTheDocument()
    expect(within(about).getByText('Go: BSD 3-Clause, The Go Authors')).toBeInTheDocument()
    expect(within(about).getByText('GNU GENERAL PUBLIC LICENSE')).toBeInTheDocument()
    fireEvent.click(within(about).getByRole('button', { name: 'Close' }))
    expect(onClose).toHaveBeenCalled()
  })
})

describe('the confirmation', () => {
  it('opens on Cancel, so a stray Enter is the safe answer', () => {
    const onConfirm = vi.fn()
    const onCancel = vi.fn()
    render(<ConfirmDialog text="Delete it?" confirmLabel="Delete" onConfirm={onConfirm} onCancel={onCancel} />)
    expect(screen.getByRole('alertdialog', { name: 'Delete it?' })).toBeInTheDocument()
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Cancel' }))
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(onConfirm).toHaveBeenCalled()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onCancel).toHaveBeenCalled()
  })
})
