// Settings (FR-060 to FR-063). Each change is saved as it is made; the key
// and token are shown only as set or not (FR-062) and tried once on save.

import { useEffect, useState } from 'react'
import { api, type Refused, type SecretName, type Settings } from './api'
import { Modal } from './Modal'

interface Props {
  refused: Refused
  onClose: () => void
}

const secrets: { which: SecretName; label: string; set: keyof Settings }[] = [
  { which: 'goatcounter', label: 'GoatCounter API key', set: 'goatCounterSet' },
  { which: 'github', label: 'GitHub token (optional)', set: 'gitHubTokenSet' },
]

export function SettingsDialog({ refused, onClose }: Props) {
  const [settings, setSettings] = useState<Settings | null>(null)
  const [typed, setTyped] = useState<Record<SecretName, string>>({ goatcounter: '', github: '' })
  const [note, setNote] = useState('')
  const reload = () => void api.settings(refused).then((found) => found && setSettings(found))
  useEffect(reload, [refused])

  const saveSecret = async (which: SecretName) => {
    const problem = await api.saveSecret(which, typed[which], refused)
    if (problem === null) return
    setNote(problem ? `Saved; it did not work: ${problem}` : 'Saved; it works.')
    setTyped({ ...typed, [which]: '' })
    reload()
  }
  const removeSecret = async (which: SecretName) => {
    if (await api.removeSecret(which, refused)) reload()
  }
  const interval = async (hours: number) => {
    if (await api.saveInterval(hours, refused)) reload()
  }

  return (
    <Modal labelId="settings-title" role="dialog" onClose={onClose}>
      <h2 id="settings-title">Settings</h2>
      {settings && (
        <>
          {secrets.map(({ which, label, set }) => (
            <div className="field" key={which}>
              <span>
                {label}: {settings[set] ? 'set' : 'not set'}
              </span>
              <input type="password" autoComplete="off" value={typed[which]} aria-label={label}
                onChange={(e) => setTyped({ ...typed, [which]: e.target.value })} />
              <div className="actions">
                <button type="button" disabled={typed[which] === ''} onClick={() => void saveSecret(which)}>
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
        </>
      )}
      <div className="actions">
        <button type="button" onClick={onClose}>
          Close
        </button>
      </div>
    </Modal>
  )
}
