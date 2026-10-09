import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { SettingsDialog } from './SettingsDialog'
import { installBridge, someSettings } from './bridge-fake'

function dialog() {
  const refused = vi.fn()
  const onClose = vi.fn()
  render(<SettingsDialog refused={refused} onClose={onClose} />)
  return { refused, onClose }
}

const keyBox = () => screen.getByLabelText('GoatCounter API key')
const keyField = () => keyBox().closest('.field') as HTMLElement
const tokenField = () => screen.getByLabelText('GitHub token (optional)').closest('.field') as HTMLElement
const button = (field: HTMLElement, name: string) =>
  Array.from(field.querySelectorAll('button')).find((b) => b.textContent === name) as HTMLButtonElement

describe('the settings', () => {
  it('shows each secret only as set or not', async () => {
    installBridge()
    dialog()
    expect(await screen.findByText('GoatCounter API key: not set')).toBeInTheDocument()
    expect(screen.getByText('GitHub token (optional): set')).toBeInTheDocument()
    expect(keyBox()).toHaveAttribute('type', 'password')
    expect(button(keyField(), 'Save')).toBeDisabled()
    expect(button(keyField(), 'Remove')).toBeDisabled()
    expect(button(tokenField(), 'Remove')).toBeEnabled()
  })

  it('saves a secret, says whether it works and empties the box', async () => {
    let saved = false
    const bridge = installBridge({
      Settings: vi.fn(() => Promise.resolve({ ...someSettings, goatCounterSet: saved })),
      SaveSecret: vi.fn(() => {
        saved = true
        return Promise.resolve('')
      }),
    })
    dialog()
    await screen.findByText('GoatCounter API key: not set')
    fireEvent.change(keyBox(), { target: { value: 'key' } })
    fireEvent.click(button(keyField(), 'Save'))
    expect(await screen.findByText('Saved; it works.')).toBeInTheDocument()
    expect(await screen.findByText('GoatCounter API key: set')).toBeInTheDocument()
    expect(bridge.SaveSecret).toHaveBeenCalledWith('goatcounter', 'key')
    expect(keyBox()).toHaveValue('')
  })

  it('says a saved secret did not work; nothing is said when the save is refused', async () => {
    installBridge({ SaveSecret: vi.fn(() => Promise.resolve('401 Unauthorized')) })
    dialog()
    await screen.findByText('GoatCounter API key: not set')
    fireEvent.change(keyBox(), { target: { value: 'wrong' } })
    fireEvent.click(button(keyField(), 'Save'))
    expect(await screen.findByText('Saved; it did not work: 401 Unauthorized')).toBeInTheDocument()

    installBridge({ SaveSecret: vi.fn(() => Promise.reject('the keychain is locked')) })
    const tokenBox = screen.getByLabelText('GitHub token (optional)')
    fireEvent.change(tokenBox, { target: { value: 'token' } })
    fireEvent.click(button(tokenField(), 'Save'))
    await waitFor(() => expect(tokenBox).toHaveValue('token'))
    expect(screen.getByText('Saved; it did not work: 401 Unauthorized')).toBeInTheDocument()
  })

  it('removes a secret', async () => {
    const bridge = installBridge({ RemoveSecret: vi.fn(() => Promise.resolve()) })
    dialog()
    await screen.findByText('GitHub token (optional): set')
    fireEvent.click(button(tokenField(), 'Remove'))
    await waitFor(() => expect(bridge.Settings).toHaveBeenCalledTimes(2))
    expect(bridge.RemoveSecret).toHaveBeenCalledWith('github')
  })

  it('saves the interval within its bounds and both switches as they change', async () => {
    const bridge = installBridge({
      SaveInterval: vi.fn(() => Promise.resolve()),
      SaveStartWithWindows: vi.fn(() => Promise.resolve()),
      SaveUpdateCheck: vi.fn(() => Promise.resolve()),
    })
    dialog()
    const interval = await screen.findByRole('spinbutton', { name: 'Check every (hours)' })
    expect(interval).toHaveValue(24)
    expect(interval).toHaveAttribute('min', '1')
    expect(interval).toHaveAttribute('max', '168')
    fireEvent.change(interval, { target: { value: '12' } })
    await waitFor(() => expect(bridge.SaveInterval).toHaveBeenCalledWith(12))

    fireEvent.click(screen.getByRole('checkbox', { name: 'Start with Windows' }))
    fireEvent.click(screen.getByRole('checkbox', { name: 'Check for a newer Visitron' }))
    await waitFor(() => expect(bridge.SaveUpdateCheck).toHaveBeenCalledWith(false))
    expect(bridge.SaveStartWithWindows).toHaveBeenCalledWith(true)
  })

  it('leaves the interval alone when the save is refused', async () => {
    const bridge = installBridge({ SaveInterval: vi.fn(() => Promise.reject('out of range')) })
    const { refused } = dialog()
    const interval = await screen.findByRole('spinbutton', { name: 'Check every (hours)' })
    fireEvent.change(interval, { target: { value: '500' } })
    await waitFor(() => expect(refused).toHaveBeenCalledWith('Out of range'))
    expect(bridge.Settings).toHaveBeenCalledTimes(1)
  })

  it('shows only Close when the settings could not be read', async () => {
    installBridge({ Settings: vi.fn(() => Promise.reject('no settings')) })
    const { refused, onClose } = dialog()
    await waitFor(() => expect(refused).toHaveBeenCalledWith('No settings'))
    expect(screen.queryByRole('spinbutton')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(onClose).toHaveBeenCalled()
  })
})
