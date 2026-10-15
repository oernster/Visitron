import { fireEvent, render, screen } from '@testing-library/react'
import { beforeAll, describe, expect, it, vi } from 'vitest'
import { WorldDialog } from './WorldDialog'
import { installBridge, someVisitors } from './bridge-fake'
import type { Visitors } from './api'

function open(visitors: Visitors = someVisitors) {
  const bridge = installBridge({ WorldVisitors: vi.fn(() => Promise.resolve(visitors)) })
  const refused = vi.fn()
  const onClose = vi.fn()
  render(<WorldDialog refused={refused} onClose={onClose} />)
  return { bridge, refused, onClose }
}

const map = { name: 'World map of visitors by country' }

describe('the World visitors dialog', () => {
  // The dialog loads the map lazily; reading its shapes cold took longer than a
  // query waits when the whole suite runs, so it is loaded once beforehand.
  beforeAll(() => import('./WorldMap'))

  it('lists the visitors to every website by country, largest first, beside the map', async () => {
    const { bridge } = open()
    expect(await screen.findByRole('heading', { name: 'Visitors by country, 30 days' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'World visitors' })).toBeInTheDocument()
    expect(bridge.WorldVisitors).toHaveBeenCalledOnce()
    expect(screen.getByRole('table', { name: 'By country' })).toHaveTextContent('United Kingdom5France2')
    expect(await screen.findByRole('img', map)).toBeInTheDocument()
    expect(screen.getByText(/need not add up to page loads or to each website/)).toBeInTheDocument()
  })

  it('says there are no visitors yet and draws no map', async () => {
    open({ ...someVisitors, countries: [] })
    expect(await screen.findByText('No visitors in this period yet.')).toBeInTheDocument()
    expect(screen.queryByRole('img', map)).toBeNull()
  })

  it('says GoatCounter is not set up', async () => {
    open({ ...someVisitors, countries: [], noGoatCounter: true })
    expect(await screen.findByText(/need its account name and key in Settings/)).toBeInTheDocument()
    expect(screen.queryByRole('table', { name: 'By country' })).toBeNull()
  })

  it('names a failed read', async () => {
    open({ ...someVisitors, countries: [], problem: 'GoatCounter: offline' })
    expect(await screen.findByRole('alert')).toHaveTextContent('The countries could not be read: GoatCounter: offline')
    expect(screen.queryByRole('img', map)).toBeNull()
  })

  it('hands a refusal on and closes', async () => {
    installBridge({ WorldVisitors: vi.fn(() => Promise.reject(new Error('the history could not be read'))) })
    const refused = vi.fn()
    const onClose = vi.fn()
    render(<WorldDialog refused={refused} onClose={onClose} />)
    await vi.waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(refused).toHaveBeenCalledWith('The history could not be read')
  })

  it('closes from its Close button', async () => {
    const { onClose } = open()
    await screen.findByRole('table', { name: 'By country' })
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(onClose).toHaveBeenCalled()
  })
})
