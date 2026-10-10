import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { DetailPane } from './DetailPane'
import { aDetail } from './bridge-fake'

describe('the detail pane', () => {
  it('names the website without its scheme and charts it, leaving the tables to Statistics', () => {
    render(<DetailPane detail={aDetail} />)
    expect(screen.getByRole('heading', { name: 'example.org' })).toBeInTheDocument()
    expect(screen.getByText('Downloads to date: 80')).toBeInTheDocument()
    expect(screen.getByText('Page loads per day, 10 in all')).toBeInTheDocument()
    expect(screen.getByText('Downloads per day, 3 in all')).toBeInTheDocument()
    expect(screen.queryByRole('table', { name: 'By platform' })).toBeNull()
  })
})
