import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { WebsiteDialog } from './WebsiteDialog'
import { aProposal, installBridge } from './bridge-fake'

function dialog(editing?: { id: number; url: string }) {
  const refused = vi.fn()
  const onSaved = vi.fn()
  const onClose = vi.fn()
  render(<WebsiteDialog editing={editing} refused={refused} onSaved={onSaved} onClose={onClose} />)
  return { refused, onSaved, onClose }
}

const address = () => screen.getByRole('textbox', { name: 'Website address' })
const find = () => screen.getByRole('button', { name: 'Find repositories' })
const save = () => screen.getByRole('button', { name: 'Save' })

describe('adding a website', () => {
  it('finds the repositories, keeps the ticks and saves the ticked ones', async () => {
    const bridge = installBridge({
      Propose: vi.fn(() => Promise.resolve(aProposal)),
      SaveWebsite: vi.fn(() => Promise.resolve(7)),
    })
    const { onSaved } = dialog()
    expect(screen.getByRole('heading', { name: 'Add website' })).toBeInTheDocument()
    expect(find()).toBeDisabled()
    expect(save()).toBeDisabled()

    fireEvent.change(address(), { target: { value: 'symdiary.com' } })
    fireEvent.click(find())
    const kept = await screen.findByRole('checkbox', { name: 'oernster/SymDiary' })
    expect(bridge.Propose).toHaveBeenCalledWith('symdiary.com', 0)
    expect(kept).toBeChecked()
    const site = screen.getByRole('checkbox', { name: 'oernster/SymDiary-site' })
    expect(site).not.toBeChecked()

    fireEvent.click(site)
    fireEvent.click(kept)
    fireEvent.click(save())
    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(bridge.SaveWebsite).toHaveBeenCalledWith(0, 'https://symdiary.com', ['oernster/SymDiary-site'])
  })

  it('finds on Enter and forgets the proposal when the address changes', async () => {
    installBridge({ Propose: vi.fn(() => Promise.resolve(aProposal)) })
    dialog()
    fireEvent.change(address(), { target: { value: 'symdiary.com' } })
    fireEvent.keyDown(address(), { key: 'Enter' })
    await screen.findByRole('checkbox', { name: 'oernster/SymDiary' })
    fireEvent.keyDown(address(), { key: 'a' })
    fireEvent.change(address(), { target: { value: 'symdiary.co' } })
    expect(screen.queryByRole('checkbox')).toBeNull()
    expect(save()).toBeDisabled()
  })

  it('does nothing on Enter in an empty address, as the disabled button would', () => {
    const bridge = installBridge({ Propose: vi.fn(() => Promise.resolve(aProposal)) })
    dialog()
    fireEvent.keyDown(address(), { key: 'Enter' })
    expect(bridge.Propose).not.toHaveBeenCalled()
    expect(address()).toHaveAttribute('placeholder', 'symdiary.com')
  })

  it('says when the site could not be read and still takes a repository by hand', async () => {
    const bridge = installBridge({
      Propose: vi.fn(() => Promise.resolve({ ...aProposal, found: [], ticked: [], problem: 'no answer' })),
      ConfirmRepo: vi.fn(() => Promise.resolve('oernster/SymDiary')),
    })
    dialog()
    fireEvent.change(address(), { target: { value: 'symdiary.com' } })
    fireEvent.click(find())
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'The site could not be read: no answer. Add its repositories by hand.',
    )
    const typed = screen.getByRole('textbox', { name: 'Add a repository by hand' })
    fireEvent.change(typed, { target: { value: 'SymDiary' } })
    fireEvent.keyDown(typed, { key: 'Enter' })
    expect(await screen.findByRole('checkbox', { name: 'oernster/SymDiary' })).toBeChecked()
    expect(bridge.ConfirmRepo).toHaveBeenCalledWith('SymDiary')
    expect(typed).toHaveValue('')

    // The same repository again neither lists nor ticks it twice.
    fireEvent.change(typed, { target: { value: 'oernster/SymDiary' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add repository' }))
    await waitFor(() => expect(typed).toHaveValue(''))
    expect(screen.getAllByRole('checkbox')).toHaveLength(1)
  })

  it('shows a refused find in the dialog and leaves a refused repository untyped', async () => {
    installBridge({ Propose: vi.fn(() => Promise.reject('not a web address')) })
    dialog()
    fireEvent.change(address(), { target: { value: '::' } })
    fireEvent.click(find())
    expect(await screen.findByRole('alert')).toHaveTextContent('Not a web address')
    expect(find()).toBeEnabled()

    installBridge({
      Propose: vi.fn(() => Promise.resolve(aProposal)),
      ConfirmRepo: vi.fn(() => Promise.reject('no such repository')),
    })
    fireEvent.click(find())
    const typed = await screen.findByRole('textbox', { name: 'Add a repository by hand' })
    expect(screen.queryByRole('alert')).toBeNull()
    fireEvent.change(typed, { target: { value: 'nobody/nothing' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add repository' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('No such repository')
    expect(typed).toHaveValue('nobody/nothing')
  })

  it('stays open when the save is refused and closes on Cancel', async () => {
    installBridge({
      Propose: vi.fn(() => Promise.resolve(aProposal)),
      SaveWebsite: vi.fn(() => Promise.reject('the store is closed')),
    })
    const { refused, onSaved, onClose } = dialog()
    fireEvent.change(address(), { target: { value: 'symdiary.com' } })
    fireEvent.click(find())
    await screen.findByRole('checkbox', { name: 'oernster/SymDiary' })
    fireEvent.click(save())
    await waitFor(() => expect(refused).toHaveBeenCalledWith('The store is closed'))
    expect(onSaved).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onClose).toHaveBeenCalled()
  })
})

describe('editing a website', () => {
  it('opens on the saved address and proposes against the website being edited', async () => {
    const bridge = installBridge({
      Propose: vi.fn(() => Promise.resolve(aProposal)),
      SaveWebsite: vi.fn(() => Promise.resolve(1)),
    })
    const { onSaved } = dialog({ id: 1, url: 'https://symdiary.com' })
    expect(screen.getByRole('heading', { name: 'Edit website' })).toBeInTheDocument()
    expect(address()).toHaveValue('https://symdiary.com')
    fireEvent.click(find())
    await screen.findByRole('checkbox', { name: 'oernster/SymDiary' })
    expect(bridge.Propose).toHaveBeenCalledWith('https://symdiary.com', 1)
    fireEvent.click(save())
    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(bridge.SaveWebsite).toHaveBeenCalledWith(1, 'https://symdiary.com', ['oernster/SymDiary'])
  })
})
