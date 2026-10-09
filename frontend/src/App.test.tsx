import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { App } from './App'
import { aState, anOverview, installBridge, installEvents, noWindowShown } from './bridge-fake'

const band = () => screen.getByRole('navigation', { name: 'Visitron' })
const bandButton = (name: string | RegExp) => within(band()).getByRole('button', { name })

describe('the shell', () => {
  it('lists the websites and says when they were last checked', async () => {
    installBridge()
    render(<App />)
    expect(await screen.findByText('symdiary.com')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('Last checked 9 Oct 2026 20:00.')
    expect(bandButton('Edit website')).toBeDisabled()
    expect(bandButton('Delete website')).toBeDisabled()
  })

  it('opens on nothing: the first focus is the sink, not a control', async () => {
    installBridge()
    render(<App />)
    await screen.findByText('symdiary.com')
    expect(document.activeElement).toHaveClass('focus-sink')
  })

  it('shows the selected website in detail and enables Edit and Delete', async () => {
    const bridge = installBridge()
    render(<App />)
    fireEvent.click(await screen.findByText('symdiary.com'))
    expect(await screen.findByRole('heading', { name: 'symdiary.com' })).toBeInTheDocument()
    expect(bridge.Detail).toHaveBeenCalledWith(1)
    expect(bandButton('Edit website')).toBeEnabled()
    expect(bandButton('Delete website')).toBeEnabled()
  })

  it('shows the problem when the data could not be opened', async () => {
    installBridge({ State: vi.fn(() => Promise.resolve({ ...aState, problem: 'The data could not be opened.' })) })
    render(<App />)
    expect(await screen.findByRole('alert')).toHaveTextContent('The data could not be opened.')
  })

  it('says a failed check, a missing key and a site never checked', async () => {
    installBridge({
      Overview: vi.fn(() => Promise.resolve({
        ...anOverview, lastSuccess: '', lastFailure: '9 Oct 2026 21:00', failure: 'offline', noKey: true,
      })),
    })
    render(<App />)
    expect(await screen.findByText(/Not checked yet\./)).toHaveTextContent(
      'Not checked yet. The check at 9 Oct 2026 21:00 failed: offline Page loads need a GoatCounter key in Settings.',
    )
  })

  it('says the window is missing when there is no facade; Dismiss clears it', async () => {
    render(<App />)
    expect(await screen.findAllByText(noWindowShown)).not.toHaveLength(0)
    fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }))
    expect(screen.queryByText(noWindowShown)).toBeNull()
  })
})

describe('the check', () => {
  it('follows a running check and reloads the list when it ends', async () => {
    const events = installEvents()
    const bridge = installBridge({ Refresh: vi.fn(() => Promise.resolve()) })
    render(<App />)
    await screen.findByText('symdiary.com')
    fireEvent.click(bandButton('Refresh'))
    expect(bridge.Refresh).toHaveBeenCalled()

    act(() => events.send({ done: 0, total: 2, site: 'symdiary.com' }))
    expect(screen.getByRole('status')).toHaveTextContent('Checking symdiary.com (1 of 2)')
    expect(bandButton('Checking')).toBeDisabled()

    act(() => events.send({ done: 2, total: 0, site: '' }))
    await waitFor(() => expect(bridge.Overview).toHaveBeenCalledTimes(2))
    expect(bandButton('Refresh')).toBeEnabled()
  })

  it('puts the warning on Refresh only while the last check has failed', async () => {
    let failed = true
    const events = installEvents()
    installBridge({
      Overview: vi.fn(() => Promise.resolve(failed
        ? { ...anOverview, lastFailure: '9 Oct 2026 21:00', failure: 'offline' }
        : anOverview)),
    })
    render(<App />)
    await screen.findByText('symdiary.com')
    const refresh = bandButton('Refresh')
    expect(refresh).toHaveAttribute('title', 'The check at 9 Oct 2026 21:00 failed: offline')
    expect(refresh.querySelector('img.band-badge')).not.toBeNull()

    failed = false
    act(() => events.send({ done: 1, total: 0, site: '' }))
    await waitFor(() => expect(bandButton('Refresh').querySelector('img.band-badge')).toBeNull())
    expect(bandButton('Refresh')).not.toHaveAttribute('title')
  })

  it('holds Refresh while the facade says a check is running', async () => {
    installBridge({ Overview: vi.fn(() => Promise.resolve({ ...anOverview, running: true })) })
    render(<App />)
    await screen.findByText('symdiary.com')
    expect(bandButton('Checking')).toBeDisabled()
  })

  it('saves a new period and reloads', async () => {
    const bridge = installBridge({ SavePeriod: vi.fn(() => Promise.resolve()) })
    render(<App />)
    fireEvent.change(await screen.findByRole('combobox', { name: 'Period' }), { target: { value: '7' } })
    await waitFor(() => expect(bridge.Overview).toHaveBeenCalledTimes(2))
    expect(bridge.SavePeriod).toHaveBeenCalledWith(7)
  })

  it('keeps the list when a new period is refused', async () => {
    const bridge = installBridge({ SavePeriod: vi.fn(() => Promise.reject('no')) })
    render(<App />)
    fireEvent.change(await screen.findByRole('combobox', { name: 'Period' }), { target: { value: '7' } })
    expect(await screen.findByText('No')).toBeInTheDocument()
    expect(bridge.Overview).toHaveBeenCalledTimes(1)
  })
})

describe('closing the window', () => {
  it('asks, then minimises to the tray or quits as chosen', async () => {
    const events = installEvents()
    const bridge = installBridge({
      MinimiseToTray: vi.fn(() => Promise.resolve()),
      RequestQuit: vi.fn(() => Promise.resolve()),
    })
    render(<App />)
    await screen.findByText('symdiary.com')

    act(() => events.close())
    fireEvent.click(screen.getByRole('button', { name: 'Minimise to tray' }))
    await waitFor(() => expect(bridge.MinimiseToTray).toHaveBeenCalled())
    expect(screen.queryByRole('alertdialog')).toBeNull()

    act(() => events.close())
    fireEvent.click(screen.getByRole('button', { name: 'Quit' }))
    await waitFor(() => expect(bridge.RequestQuit).toHaveBeenCalled())
  })

  it('stays open when the choice is cancelled', async () => {
    const events = installEvents()
    installBridge()
    render(<App />)
    await screen.findByText('symdiary.com')
    act(() => events.close())
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('alertdialog')).toBeNull()
  })
})

describe('the band', () => {
  it('orders the band as FR-040 states', async () => {
    installBridge()
    render(<App />)
    await screen.findByText('symdiary.com')
    const groups = band().querySelectorAll('.band-group')
    const labels = (group: Element) => Array.from(group.querySelectorAll('button')).map((b) => b.textContent)
    expect(labels(groups[0])).toEqual(['Add website', 'Edit website', 'Delete website', 'Refresh', 'Settings'])
    expect(labels(groups[1])).toEqual(['Donate', 'Light mode', 'Help'])
    expect(groups[1].children[1]).toHaveClass('band-separator')
  })

  it('opens Add website and closes it again', async () => {
    installBridge()
    render(<App />)
    await screen.findByText('symdiary.com')
    fireEvent.click(bandButton('Add website'))
    expect(screen.getByRole('heading', { name: 'Add website' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByRole('heading', { name: 'Add website' })).toBeNull()
  })

  it('edits the selected website and reloads once it is saved', async () => {
    const bridge = installBridge({
      Propose: vi.fn(() => Promise.resolve({ url: 'https://symdiary.com', found: [], ticked: [], problem: '' })),
      SaveWebsite: vi.fn(() => Promise.resolve(1)),
    })
    render(<App />)
    fireEvent.click(await screen.findByText('symdiary.com'))
    fireEvent.click(bandButton('Edit website'))
    expect(screen.getByRole('textbox', { name: 'Website address' })).toHaveValue('https://symdiary.com')
    fireEvent.click(screen.getByRole('button', { name: 'Find repositories' }))
    await screen.findByText(/Downloads count from the ticked repositories/)
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(screen.queryByRole('heading', { name: 'Edit website' })).toBeNull())
    expect(bridge.Overview).toHaveBeenCalledTimes(2)
  })

  it('asks before deleting, then deletes and clears the selection', async () => {
    const bridge = installBridge({ DeleteWebsite: vi.fn(() => Promise.resolve()) })
    render(<App />)
    fireEvent.click(await screen.findByText('symdiary.com'))
    fireEvent.click(bandButton('Delete website'))
    expect(screen.getByText('Delete https://symdiary.com and its download history?')).toBeInTheDocument()
    fireEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(bridge.DeleteWebsite).toHaveBeenCalledWith(1))
    await waitFor(() => expect(bandButton('Delete website')).toBeDisabled())
    expect(screen.queryByRole('alertdialog')).toBeNull()
  })

  it('keeps the website when the delete is cancelled or refused', async () => {
    installBridge({ DeleteWebsite: vi.fn(() => Promise.reject('the store is closed')) })
    render(<App />)
    fireEvent.click(await screen.findByText('symdiary.com'))
    fireEvent.click(bandButton('Delete website'))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByRole('alertdialog')).toBeNull()

    fireEvent.click(bandButton('Delete website'))
    fireEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Delete' }))
    expect(await screen.findByText('The store is closed')).toBeInTheDocument()
    expect(bandButton('Delete website')).toBeEnabled()
  })

  it('opens Settings and reloads the list when it closes', async () => {
    const bridge = installBridge()
    render(<App />)
    await screen.findByText('symdiary.com')
    fireEvent.click(bandButton('Settings'))
    await screen.findByText('GoatCounter API key: not set')
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    await waitFor(() => expect(bridge.Overview).toHaveBeenCalledTimes(2))
  })

  it('opens the guide, then About from it', async () => {
    installBridge()
    render(<App />)
    await screen.findByText('symdiary.com')
    fireEvent.click(bandButton('Help'))
    expect(screen.getByRole('heading', { name: 'How Visitron works' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'About' }))
    expect(await screen.findByRole('heading', { name: 'Visitron 1.0.0' })).toBeInTheDocument()
    fireEvent.click(within(screen.getByRole('dialog', { name: 'Visitron 1.0.0' })).getByRole('button', { name: 'Close' }))
    expect(screen.queryByRole('heading', { name: 'Visitron 1.0.0' })).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(screen.queryByRole('heading', { name: 'How Visitron works' })).toBeNull()
  })

  it('asks Go for the donation page and swaps the theme', async () => {
    const bridge = installBridge({ Donate: vi.fn(() => Promise.resolve()) })
    render(<App />)
    await screen.findByText('symdiary.com')
    fireEvent.click(bandButton('Donate'))
    expect(bridge.Donate).toHaveBeenCalled()
    fireEvent.click(bandButton('Light mode'))
    expect(bandButton('Dark mode')).toBeInTheDocument()
  })
})
