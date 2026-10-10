import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AboutDialog, ConfirmDialog, LicenceDialog } from './Dialog'
import { anAbout } from './bridge-fake'

describe('About', () => {
  it('names the product, its author and every credit, leaving the licence and the check to the menu', () => {
    const onClose = vi.fn()
    render(<AboutDialog about={anAbout} onClose={onClose} />)
    const about = screen.getByRole('dialog', { name: 'Visitron 1.0.0' })
    expect(within(about).getByText('By Oliver Ernster')).toBeInTheDocument()
    expect(within(about).getByText('Go: BSD 3-Clause, The Go Authors')).toBeInTheDocument()
    expect(within(about).queryByText('GNU GENERAL PUBLIC LICENSE')).toBeNull()
    expect(within(about).queryByRole('button', { name: 'Check for updates' })).toBeNull()
    fireEvent.click(within(about).getByRole('button', { name: 'Close' }))
    expect(onClose).toHaveBeenCalled()
  })
})

describe('the licence', () => {
  it('shows the full text, opens on Close and closes', () => {
    const onClose = vi.fn()
    render(<LicenceDialog text={anAbout.licence} onClose={onClose} />)
    const licence = screen.getByRole('dialog', { name: 'Licence' })
    expect(within(licence).getByText('GNU GENERAL PUBLIC LICENSE')).toBeInTheDocument()
    expect(document.activeElement).toBe(within(licence).getByRole('button', { name: 'Close' }))
    fireEvent.click(document.activeElement as HTMLElement)
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
