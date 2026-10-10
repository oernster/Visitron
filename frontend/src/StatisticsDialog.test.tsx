import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { StatisticsDialog } from './StatisticsDialog'
import { installBridge, someStatistics } from './bridge-fake'
import type { Statistics } from './api'

function open(stats: Statistics = someStatistics) {
  const bridge = installBridge({ Statistics: vi.fn(() => Promise.resolve(stats)) })
  const refused = vi.fn()
  const onClose = vi.fn()
  render(<StatisticsDialog id={1} refused={refused} onClose={onClose} />)
  return { bridge, refused, onClose }
}

describe('the Statistics dialog', () => {
  it('shows the downloads to date and the visitors by country, largest first', async () => {
    const { bridge } = open()
    expect(await screen.findByRole('heading', { name: 'example.org' })).toBeInTheDocument()
    expect(bridge.Statistics).toHaveBeenCalledWith(1)
    expect(screen.getByRole('table', { name: 'By platform' })).toHaveTextContent('Windows70macOS10')
    expect(screen.getByRole('table', { name: 'By repository' })).toHaveTextContent('someone/Widget80')
    expect(screen.getByRole('table', { name: 'By release' })).toHaveTextContent('v1.0.080')
    expect(screen.getByRole('heading', { name: 'Visitors by country, 30 days' })).toBeInTheDocument()
    expect(screen.getByRole('table', { name: 'By country' })).toHaveTextContent('United Kingdom5France2')
  })

  it('says GoatCounter is not set up and keeps the downloads', async () => {
    open({ ...someStatistics, countries: [], noGoatCounter: true })
    expect(await screen.findByText(/need its account name and key in Settings/)).toBeInTheDocument()
    expect(screen.getByRole('table', { name: 'By platform' })).toBeInTheDocument()
    expect(screen.queryByRole('table', { name: 'By country' })).toBeNull()
  })

  it('names a failed country read and keeps the downloads', async () => {
    open({ ...someStatistics, countries: [], countriesProblem: 'GoatCounter: offline' })
    expect(await screen.findByRole('alert')).toHaveTextContent('The countries could not be read: GoatCounter: offline')
    expect(screen.getByRole('table', { name: 'By platform' })).toBeInTheDocument()
  })

  it('says when there are no visitors and no downloads yet', async () => {
    open({ ...someStatistics, byPlatform: [], byRepo: [], byRelease: [], countries: [] })
    expect(await screen.findByText('No visitors in this period yet.')).toBeInTheDocument()
    expect(screen.getByText('No downloads yet.')).toBeInTheDocument()
    expect(screen.queryByRole('table', { name: 'By release' })).toBeNull()
  })

  it('hands a refusal on and closes', async () => {
    installBridge({ Statistics: vi.fn(() => Promise.reject(new Error('that website is no longer recorded'))) })
    const refused = vi.fn()
    const onClose = vi.fn()
    render(<StatisticsDialog id={1} refused={refused} onClose={onClose} />)
    await vi.waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(refused).toHaveBeenCalledWith('That website is no longer recorded')
  })

  it('closes from its Close button', async () => {
    const { onClose } = open()
    await screen.findByRole('heading', { name: 'example.org' })
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(onClose).toHaveBeenCalled()
  })
})
