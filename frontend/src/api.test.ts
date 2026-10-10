import { describe, expect, it, vi } from 'vitest'
import { api, noWindow, onProgress, sentence } from './api'
import { aState, installBridge, installEvents } from './bridge-fake'

describe('the bridge', () => {
  it('answers null and says so when the window is not there', async () => {
    const refused = vi.fn()
    expect(await api.state(refused)).toBeNull()
    expect(refused).toHaveBeenCalledWith(noWindow)
  })

  it('hands back what the facade answered', async () => {
    installBridge()
    const refused = vi.fn()
    expect(await api.state(refused)).toEqual(aState)
    expect(refused).not.toHaveBeenCalled()
  })

  it('turns a refusal into a sentence and answers null', async () => {
    installBridge({ SaveWebsite: vi.fn(() => Promise.reject(new Error('the site was not saved'))) })
    const refused = vi.fn()
    expect(await api.saveWebsite(0, 'https://example.org', [], refused)).toBeNull()
    expect(refused).toHaveBeenCalledWith('The site was not saved')
  })

  it('answers true for a call that answers nothing, null when refused', async () => {
    const bridge = installBridge({ DeleteWebsite: vi.fn(() => Promise.resolve()) })
    const refused = vi.fn()
    expect(await api.deleteWebsite(4, refused)).toBe(true)
    expect(bridge.DeleteWebsite).toHaveBeenCalledWith(4)

    installBridge({ Refresh: vi.fn(() => Promise.reject('a check is already running')) })
    expect(await api.refresh(refused)).toBeNull()
    expect(refused).toHaveBeenCalledWith('A check is already running')
  })

  it('passes every argument through to the facade', async () => {
    const bridge = installBridge({
      Propose: vi.fn(() => Promise.resolve(null)),
      ConfirmRepo: vi.fn(() => Promise.resolve('')),
      SaveInterval: vi.fn(() => Promise.resolve()),
      SavePeriod: vi.fn(() => Promise.resolve()),
      SaveUpdateCheck: vi.fn(() => Promise.resolve()),
      SaveStartWithWindows: vi.fn(() => Promise.resolve()),
      SaveSecret: vi.fn(() => Promise.resolve('')),
      RemoveSecret: vi.fn(() => Promise.resolve()),
      Donate: vi.fn(() => Promise.resolve()),
    })
    const refused = vi.fn()
    await api.overview(refused)
    await api.detail(3, refused)
    await api.propose('example.org', 2, refused)
    await api.confirmRepo('someone/Widget', refused)
    await api.settings(refused)
    await api.saveInterval(12, refused)
    await api.savePeriod(90, refused)
    await api.saveUpdateCheck(false, refused)
    await api.saveStartWithWindows(true, refused)
    await api.saveSecret('github', 'token', refused)
    await api.removeSecret('goatcounter', refused)
    await api.about(refused)
    await api.donate(refused)
    expect(bridge.Detail).toHaveBeenCalledWith(3)
    expect(bridge.Propose).toHaveBeenCalledWith('example.org', 2)
    expect(bridge.ConfirmRepo).toHaveBeenCalledWith('someone/Widget')
    expect(bridge.SaveInterval).toHaveBeenCalledWith(12)
    expect(bridge.SavePeriod).toHaveBeenCalledWith(90)
    expect(bridge.SaveUpdateCheck).toHaveBeenCalledWith(false)
    expect(bridge.SaveStartWithWindows).toHaveBeenCalledWith(true)
    expect(bridge.SaveSecret).toHaveBeenCalledWith('github', 'token')
    expect(bridge.RemoveSecret).toHaveBeenCalledWith('goatcounter')
    expect(bridge.Donate).toHaveBeenCalled()
    expect(refused).not.toHaveBeenCalled()
  })

  it('follows progress through the runtime; without one it does nothing', () => {
    expect(onProgress(vi.fn())()).toBeUndefined()
    const events = installEvents()
    const seen = vi.fn()
    onProgress(seen)
    events.send({ done: 1, total: 3, site: 'example.org' })
    expect(seen).toHaveBeenCalledWith({ done: 1, total: 3, site: 'example.org' })
  })

  it('leaves an empty reason alone', () => {
    expect(sentence('')).toBe('')
  })
})
