// Settings (FR-060 to FR-063). Each change is saved as it is made; the key
// and token are tried once on save. Each box holds the stored secret, hidden
// until its eye is pressed and hidden again whenever the dialog opens
// (Amendment 8).

import { useCallback, useEffect, useState } from 'react'
import { api, type Refused, type SecretName, type Settings } from './api'
import { onEnter } from './keys'
import { Modal } from './Modal'
import { secretHelp } from './secretHelp'
import { useAutoScroll } from './useAutoScroll'

interface Props {
  refused: Refused
  onClose: () => void
}

// The eye on a secret's box: an emoji, so it needs no picture of its own.
const eye = '\u{1F441}'

const none: Record<SecretName, string> = { goatcounter: '', github: '' }
const hidden: Record<SecretName, boolean> = { goatcounter: false, github: false }

export function SettingsDialog({ refused, onClose }: Props) {
  const [settings, setSettings] = useState<Settings | null>(null)
  const [stored, setStored] = useState(none)
  const [typed, setTyped] = useState(none)
  const [shown, setShown] = useState(hidden)
  const [note, setNote] = useState('')
  // The steps under each secret make Settings taller than a short window, so
  // the body scrolls and Close stays pinned beneath it, as in the Guide.
  const autoScroll = useAutoScroll()
  // The GoatCounter account name as typed; null until the settings first arrive, so a
  // later reload never overwrites what the owner is part way through typing.
  const [site, setSite] = useState<string | null>(null)
  const reload = useCallback(
    () =>
      void api.settings(refused).then((found) => {
        if (!found) return
        setSettings(found)
        setSite((s) => s ?? found.goatCounterSite)
      }),
    [refused],
  )
  const loadSecret = useCallback(
    (which: SecretName) =>
      void api.secret(which, refused).then((value) => {
        if (value === null) return
        setStored((s) => ({ ...s, [which]: value }))
        setTyped((t) => ({ ...t, [which]: value }))
      }),
    [refused],
  )
  useEffect(() => {
    reload()
    secretHelp.forEach(({ which }) => loadSecret(which))
  }, [reload, loadSecret])

  // Save and Enter apply a box only when it holds something new; one rule
  // for both, so the key never does what the button refuses.
  const canSave = (which: SecretName) => typed[which] !== '' && typed[which] !== stored[which]
  const canSaveSite = () => !!site && site !== settings?.goatCounterSite
  const saveSecret = async (which: SecretName) => {
    const problem = await api.saveSecret(which, typed[which], refused)
    if (problem === null) return
    setNote(problem ? `Saved; it did not work: ${problem}` : 'Saved; it works.')
    reload()
    loadSecret(which)
  }
  // Saving the site answers why a stored key did not work there; the box then
  // shows the code as kept, whatever form it was typed in (Amendment 11).
  const saveSite = async () => {
    const problem = await api.saveGoatCounterSite(site ?? '', refused)
    if (problem === null) return
    setNote(problem ? `Account name saved; the key did not work there: ${problem}` : 'Account name saved.')
    const found = await api.settings(refused)
    if (!found) return
    setSettings(found)
    setSite(found.goatCounterSite)
  }
  const removeSecret = async (which: SecretName) => {
    if (!(await api.removeSecret(which, refused))) return
    reload()
    loadSecret(which)
  }
  const interval = async (hours: number) => {
    if (await api.saveInterval(hours, refused)) reload()
  }

  return (
    <Modal labelId="settings-title" role="dialog" onClose={onClose} pinnedActions>
      <h2 id="settings-title">Settings</h2>
      {settings && (
        <div className="dialog-body" ref={autoScroll}>
          {secretHelp.map(({ which, label, set, why, steps }) => (
            <div className="field" key={which}>
              <span>
                {label}: {settings[set] ? 'set' : 'not set'}
              </span>
              <p className="secret-help">{why}</p>
              <ol className="secret-steps">
                {steps.map((step) => (
                  <li key={step}>{step}</li>
                ))}
              </ol>
              {which === 'goatcounter' && (
                <>
                  <span>GoatCounter account name: {settings.goatCounterSite || 'not set'}</span>
                  <div className="secret-entry">
                    <input type="text" autoComplete="off" spellCheck={false} aria-label="GoatCounter account name"
                      placeholder="youraccount" value={site ?? ''}
                      onChange={(e) => setSite(e.target.value)}
                      onKeyDown={onEnter(() => void saveSite(), canSaveSite())} />
                    <button type="button" disabled={!canSaveSite()}
                      onClick={() => void saveSite()}>
                      Save name
                    </button>
                  </div>
                </>
              )}
              <div className="secret-entry">
                <input type={shown[which] ? 'text' : 'password'} autoComplete="off" spellCheck={false}
                  value={typed[which]} aria-label={label}
                  onChange={(e) => setTyped({ ...typed, [which]: e.target.value })}
                  onKeyDown={onEnter(() => void saveSecret(which), canSave(which))} />
                <button type="button" className="reveal" aria-label={`Show ${label}`}
                  aria-pressed={shown[which]} title={shown[which] ? 'Hide' : 'Show'}
                  onClick={() => setShown({ ...shown, [which]: !shown[which] })}>
                  {eye}
                </button>
              </div>
              <div className="actions">
                <button type="button" disabled={!canSave(which)}
                  onClick={() => void saveSecret(which)}>
                  Save
                </button>
                <button type="button" disabled={!settings[set]} onClick={() => void removeSecret(which)}>
                  Remove
                </button>
              </div>
            </div>
          ))}
          {note && <p role="status">{note}</p>}
          <label className="field">
            Check every (hours)
            <input type="number" min={settings.minInterval} max={settings.maxInterval}
              value={settings.intervalHours}
              onChange={(e) => void interval(Number(e.target.value))} />
          </label>
          <label className="check">
            <input type="checkbox" checked={settings.startWithWindows}
              onChange={(e) => void api.saveStartWithWindows(e.target.checked, refused).then(reload)} />
            Start with Windows
          </label>
          <label className="check">
            <input type="checkbox" checked={settings.updateCheck}
              onChange={(e) => void api.saveUpdateCheck(e.target.checked, refused).then(reload)} />
            Check for a newer Visitron
          </label>
        </div>
      )}
      <div className="actions">
        <button type="button" onClick={onClose}>
          Close
        </button>
      </div>
    </Modal>
  )
}
