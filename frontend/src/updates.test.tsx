import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { UpdateDialog, checkEveryMs, firstCheckMs, useUpdateCheck } from './updates'
import { anOffer, current, installBridge } from './bridge-fake'
import type { Update } from './api'

// Harness draws the hook with its dialog, as the shell does, plus the manual
// check's button.
function Harness() {
  const updates = useUpdateCheck()
  return (
    <>
      <button type="button" onClick={updates.checkNow}>
        Check now
      </button>
      {updates.found && <UpdateDialog name="Visitron" found={updates.found} onClose={updates.dismiss} />}
    </>
  )
}

afterEach(() => {
  vi.useRealTimers()
})

async function offered() {
  vi.useFakeTimers()
  render(<Harness />)
  await act(() => vi.advanceTimersByTimeAsync(firstCheckMs))
  vi.useRealTimers()
  return screen.getByRole('dialog', { name: 'Update available' })
}

describe('the automatic check', () => {
  it('checks 3 seconds after the page loads, then once a day', async () => {
    const bridge = installBridge()
    vi.useFakeTimers()
    render(<Harness />)
    await act(() => vi.advanceTimersByTimeAsync(firstCheckMs - 1))
    expect(bridge.CheckForUpdates).not.toHaveBeenCalled()
    await act(() => vi.advanceTimersByTimeAsync(1))
    expect(bridge.CheckForUpdates).toHaveBeenCalledWith(false)
    await act(() => vi.advanceTimersByTimeAsync(checkEveryMs))
    expect(bridge.CheckForUpdates).toHaveBeenCalledTimes(2)
  })

  it('stays silent unless it has an offer, a refusal included', async () => {
    for (const answer of [current, { ...current, outcome: 'off' } as Update, { ...anOffer, outcome: 'skipped' } as Update]) {
      installBridge({ CheckForUpdates: vi.fn(() => Promise.resolve(answer)) })
      vi.useFakeTimers()
      const { unmount } = render(<Harness />)
      await act(() => vi.advanceTimersByTimeAsync(firstCheckMs))
      expect(screen.queryByRole('dialog')).toBeNull()
      unmount()
      vi.useRealTimers()
    }
    installBridge({ CheckForUpdates: vi.fn(() => Promise.reject('offline')) })
    vi.useFakeTimers()
    render(<Harness />)
    await act(() => vi.advanceTimersByTimeAsync(firstCheckMs))
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('offers Download first, then Skip this version and Later', async () => {
    installBridge({ CheckForUpdates: vi.fn(() => Promise.resolve(anOffer)) })
    const dialog = await offered()
    expect(dialog).toHaveTextContent('Visitron 1.1.0 is available. You are running 1.0.0.')
    const buttons = Array.from(dialog.querySelectorAll('button')).map((b) => b.textContent)
    // The close cross comes last (Amendment 10), so it never takes the first stop.
    expect(buttons).toEqual(['Download', 'Skip this version', 'Later', '×'])
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Download' }))
  })
})

describe('acting on an offer', () => {
  it('hands the download to the facade and closes', async () => {
    const bridge = installBridge({
      CheckForUpdates: vi.fn(() => Promise.resolve(anOffer)),
      DownloadUpdate: vi.fn(() => Promise.resolve()),
    })
    await offered()
    fireEvent.click(screen.getByRole('button', { name: 'Download' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(bridge.DownloadUpdate).toHaveBeenCalled()
  })

  it('skips the offered version and closes', async () => {
    const bridge = installBridge({
      CheckForUpdates: vi.fn(() => Promise.resolve(anOffer)),
      SkipUpdate: vi.fn(() => Promise.resolve()),
    })
    await offered()
    fireEvent.click(screen.getByRole('button', { name: 'Skip this version' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(bridge.SkipUpdate).toHaveBeenCalled()
  })

  it('stays open to say why a download or a skip was refused', async () => {
    installBridge({
      CheckForUpdates: vi.fn(() => Promise.resolve(anOffer)),
      DownloadUpdate: vi.fn(() => Promise.reject('no browser opener is wired')),
    })
    await offered()
    fireEvent.click(screen.getByRole('button', { name: 'Download' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('No browser opener is wired')
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('closes on Later, changing nothing', async () => {
    const bridge = installBridge({ CheckForUpdates: vi.fn(() => Promise.resolve(anOffer)) })
    await offered()
    fireEvent.click(screen.getByRole('button', { name: 'Later' }))
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(bridge.DownloadUpdate).not.toHaveBeenCalled()
    expect(bridge.SkipUpdate).not.toHaveBeenCalled()
  })
})

describe('the check asked for', () => {
  it('tells every outcome it can find', async () => {
    const answers: [Update | 'refused', string][] = [
      [current, 'You are running the latest version.'],
      [{ ...current, outcome: 'none' }, 'No release of Visitron has been published yet.'],
      [{ ...current, outcome: 'uncomparable', running: '0.0.0-dev' }, 'built from source as 0.0.0-dev'],
      [{ ...current, outcome: 'unreachable' }, 'could not reach GitHub'],
      ['refused', 'could not reach GitHub'],
    ]
    for (const [answer, words] of answers) {
      const bridge = installBridge({
        CheckForUpdates: vi.fn(() => (answer === 'refused' ? Promise.reject('offline') : Promise.resolve(answer))),
      })
      const { unmount } = render(<Harness />)
      fireEvent.click(screen.getByRole('button', { name: 'Check now' }))
      const dialog = await screen.findByRole('dialog', { name: 'Check for updates' })
      expect(dialog).toHaveTextContent(words)
      expect(bridge.CheckForUpdates).toHaveBeenCalledWith(true)
      fireEvent.click(screen.getByRole('button', { name: 'Close' }))
      expect(screen.queryByRole('dialog')).toBeNull()
      unmount()
    }
  })
})
