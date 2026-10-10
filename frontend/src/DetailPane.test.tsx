import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { DetailPane } from './DetailPane'
import { aDetail } from './bridge-fake'

describe('the detail pane', () => {
  it('names the website without its scheme and shows every total', () => {
    render(<DetailPane detail={aDetail} />)
    expect(screen.getByRole('heading', { name: 'example.org' })).toBeInTheDocument()
    expect(screen.getByText('Downloads to date: 80')).toBeInTheDocument()
    expect(screen.getByRole('table', { name: 'By platform' })).toHaveTextContent('Windows70macOS10')
    expect(screen.getByRole('table', { name: 'By repository' })).toHaveTextContent('someone/Widget80')
    expect(screen.getByRole('table', { name: 'By release' })).toHaveTextContent('v1.0.080')
    expect(screen.getByText('Page loads per day, 10 in all')).toBeInTheDocument()
    expect(screen.getByText('Downloads per day, 3 in all')).toBeInTheDocument()
  })

  it('leaves out a total with nothing in it', () => {
    render(<DetailPane detail={{ ...aDetail, byRelease: [] }} />)
    expect(screen.queryByRole('table', { name: 'By release' })).toBeNull()
  })
})
