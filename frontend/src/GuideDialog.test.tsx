import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { GuideDialog } from './GuideDialog'
import { guideSections } from './guideContent'

describe('the guide', () => {
  it('shows every section with each control named', () => {
    render(<GuideDialog onClose={vi.fn()} onAbout={vi.fn()} />)
    expect(screen.getByRole('dialog', { name: 'How Visitron works' })).toBeInTheDocument()
    for (const section of guideSections) {
      expect(screen.getByRole('heading', { name: section.heading })).toBeInTheDocument()
      for (const entry of section.entries ?? []) {
        expect(screen.getByText(entry.name)).toBeInTheDocument()
      }
    }
  })

  it('says what the warning on Refresh means', () => {
    render(<GuideDialog onClose={vi.fn()} onAbout={vi.fn()} />)
    expect(screen.getByText(/Refresh carries a warning sign/)).toBeInTheDocument()
  })

  it('leads to About and closes', () => {
    const onAbout = vi.fn()
    const onClose = vi.fn()
    render(<GuideDialog onClose={onClose} onAbout={onAbout} />)
    fireEvent.click(screen.getByRole('button', { name: 'About' }))
    expect(onAbout).toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(onClose).toHaveBeenCalled()
  })
})
