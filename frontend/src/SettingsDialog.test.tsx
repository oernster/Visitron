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
  it('says where the key and the token come from', async () => {
    installBridge()
    dialog()
    expect(await screen.findByText('Choose your username in the top menu, then API.')).toBeInTheDocument()
    expect(screen.getByText(/Fine-grained tokens, Generate new token/)).toBeInTheDocument()
    expect(screen.getAllByRole('listitem')).toHaveLength(9)
    // Generic for every owner: no account is named, only the form one takes.
    expect(document.body).not.toHaveTextContent(/oernster/i)
    expect(screen.getByText(/such as yourname\.goatcounter\.com/)).toBeInTheDocument()
  })

  it('keeps the GoatCounter site as its code and says whether a stored key works there', async () => {
    let kept = ''
    const bridge = installBridge({
      Settings: vi.fn(() => Promise.resolve({ ...someSettings, goatCounterSite: kept })),
      SaveGoatCounterSite: vi.fn(() => {
        kept = 'someone'
        return Promise.resolve('401 Unauthorized')
      }),
    })
    dialog()
    expect(await screen.findByText('GoatCounter site: not set')).toBeInTheDocument()
    const siteBox = screen.getByLabelText('GoatCounter site')
    const save = screen.getByRole('button', { name: 'Save site' })
    expect(save).toBeDisabled()
    fireEvent.change(siteBox, { target: { value: 'https://Someone.goatcounter.com/' } })
    fireEvent.click(save)
    expect(await screen.findByText('Site saved; the key did not work there: 401 Unauthorized')).toBeInTheDocument()
    expect(bridge.SaveGoatCounterSite).toHaveBeenCalledWith('https://Someone.goatcounter.com/')
    expect(await screen.findByText('GoatCounter site: someone')).toBeInTheDocument()
    expect(siteBox).toHaveValue('someone')
    expect(save).toBeDisabled()
  })

  it('applies the site and a secret on Enter only when the Save button would', async () => {
    const bridge = installBridge({
      SaveGoatCounterSite: vi.fn(() => Promise.resolve('')),
      SaveSecret: vi.fn(() => Promise.resolve('')),
    })
    dialog()
    const siteBox = await screen.findByLabelText('GoatCounter site')
    fireEvent.keyDown(siteBox, { key: 'Enter' })
    expect(bridge.SaveGoatCounterSite).not.toHaveBeenCalled()
    fireEvent.change(siteBox, { target: { value: 'someone' } })
    fireEvent.keyDown(siteBox, { key: 'Enter' })
    await waitFor(() => expect(bridge.SaveGoatCounterSite).toHaveBeenCalledWith('someone'))

    const tokenBox = screen.getByLabelText('GitHub token (optional)')
    await waitFor(() => expect(tokenBox).toHaveValue('a-github-token'))
    fireEvent.keyDown(tokenBox, { key: 'Enter' })
    expect(bridge.SaveSecret).not.toHaveBeenCalled()
    fireEvent.change(keyBox(), { target: { value: 'new-key' } })
    fireEvent.keyDown(keyBox(), { key: 'Enter' })
    await waitFor(() => expect(bridge.SaveSecret).toHaveBeenCalledWith('goatcounter', 'new-key'))
  })

  it('says nothing when a site is refused; the refusal goes to the window', async () => {
    installBridge({ SaveGoatCounterSite: vi.fn(() => Promise.reject('not a site')) })
    const { refused } = dialog()
    const siteBox = await screen.findByLabelText('GoatCounter site')
    fireEvent.change(siteBox, { target: { value: 'bad site' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save site' }))
    await waitFor(() => expect(refused).toHaveBeenCalledWith('Not a site'))
    expect(siteBox).toHaveValue('bad site')
    expect(screen.queryByText(/Site saved/)).toBeNull()
  })

  it('holds each stored secret hidden until its eye is pressed', async () => {
    installBridge()
    dialog()
    expect(await screen.findByText('GoatCounter API key: not set')).toBeInTheDocument()
    expect(screen.getByText('GitHub token (optional): set')).toBeInTheDocument()
    const tokenBox = screen.getByLabelText('GitHub token (optional)')
    await waitFor(() => expect(tokenBox).toHaveValue('a-github-token'))
    expect(tokenBox).toHaveAttribute('type', 'password')
    expect(keyBox()).toHaveAttribute('type', 'password')

    const eye = screen.getByRole('button', { name: 'Show GitHub token (optional)' })
    fireEvent.click(eye)
    expect(tokenBox).toHaveAttribute('type', 'text')
    expect(eye).toHaveAttribute('aria-pressed', 'true')
    expect(keyBox()).toHaveAttribute('type', 'password')
    fireEvent.click(eye)
    expect(tokenBox).toHaveAttribute('type', 'password')

    expect(button(keyField(), 'Save')).toBeDisabled()
    expect(button(tokenField(), 'Save')).toBeDisabled()
    expect(button(keyField(), 'Remove')).toBeDisabled()
    expect(button(tokenField(), 'Remove')).toBeEnabled()
  })

  it('saves a secret, says whether it works and keeps it in the box', async () => {
    let saved = ''
    const bridge = installBridge({
      Settings: vi.fn(() => Promise.resolve({ ...someSettings, goatCounterSet: saved !== '' })),
      Secret: vi.fn((which: string) => Promise.resolve(which === 'goatcounter' ? saved : '')),
      SaveSecret: vi.fn((_: string, value: string) => {
        saved = value
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
    await waitFor(() => expect(button(keyField(), 'Save')).toBeDisabled())
    expect(keyBox()).toHaveValue('key')
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
